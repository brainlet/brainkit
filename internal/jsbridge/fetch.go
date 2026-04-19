package jsbridge

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	quickjs "github.com/buke/quickjs-go"
)

// FetchSpanHook is called around each fetch request for tracing.
// start is called before the request, returns a finish function called after.
type FetchSpanHook func(method, url string) (finish func(statusCode int, err error))

// FetchPolyfill provides globalThis.fetch with async non-blocking I/O.
// HTTP calls run in tracked goroutines via Bridge.Go().
// For SSE/streaming responses, chunks are delivered incrementally
// via ReadableStream backed by Go goroutine reads.
type FetchPolyfill struct {
	client   *http.Client
	bridge   *Bridge // set during Setup
	spanHook FetchSpanHook
}

// FetchOption configures a FetchPolyfill.
type FetchOption func(*FetchPolyfill)

// FetchClient sets the HTTP client used for requests.
func FetchClient(c *http.Client) FetchOption {
	return func(p *FetchPolyfill) { p.client = c }
}

// FetchWithSpanHook sets a tracing hook called around each fetch request.
func FetchWithSpanHook(hook FetchSpanHook) FetchOption {
	return func(p *FetchPolyfill) { p.spanHook = hook }
}

// Fetch creates a fetch polyfill.
func Fetch(opts ...FetchOption) *FetchPolyfill {
	p := &FetchPolyfill{client: http.DefaultClient}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *FetchPolyfill) Name() string { return "fetch" }

// SetBridge is called by the bridge after construction to give polyfills
// access to Bridge.Go() for tracked goroutines.
func (p *FetchPolyfill) SetBridge(b *Bridge) { p.bridge = b }

func (p *FetchPolyfill) Setup(ctx *quickjs.Context) error {
	client := p.client
	polyfill := p

	ctx.Globals().Set("__go_fetch", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) == 0 {
			return ctx.ThrowError(fmt.Errorf("fetch: missing request argument"))
		}

		reqJSON := args[0].ToString()

		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			polyfill.bridge.Go(func(goCtx context.Context) {
				var req struct {
					URL     string            `json:"url"`
					Method  string            `json:"method"`
					Headers map[string]string `json:"headers"`
					Body    *string           `json:"body"`
				}
				if err := json.Unmarshal([]byte(reqJSON), &req); err != nil {
					ctx.Schedule(func(ctx *quickjs.Context) {
						errVal := ctx.NewError(fmt.Errorf("fetch: invalid request: %w", err))
						defer errVal.Free()
						reject(errVal)
					})
					return
				}

				var bodyReader io.Reader
				if req.Body != nil && *req.Body != "" {
					// JS side signals binary bodies (multipart, Blob, ArrayBuffer)
					// by base64-encoding the payload + setting
					// x-brainkit-body-encoding: base64. Decode here and strip
					// the marker so it never reaches the wire.
					if enc := req.Headers["x-brainkit-body-encoding"]; enc == "base64" {
						raw, decErr := base64.StdEncoding.DecodeString(*req.Body)
						if decErr != nil {
							ctx.Schedule(func(ctx *quickjs.Context) {
								errVal := ctx.NewError(fmt.Errorf("fetch: decode binary body: %w", decErr))
								defer errVal.Free()
								reject(errVal)
							})
							return
						}
						bodyReader = bytes.NewReader(raw)
						delete(req.Headers, "x-brainkit-body-encoding")
					} else {
						bodyReader = strings.NewReader(*req.Body)
					}
				}

				httpReq, err := http.NewRequestWithContext(goCtx, req.Method, req.URL, bodyReader)
				if err != nil {
					ctx.Schedule(func(ctx *quickjs.Context) {
						errVal := ctx.NewError(fmt.Errorf("fetch: %w", err))
						defer errVal.Free()
						reject(errVal)
					})
					return
				}
				for k, v := range req.Headers {
					httpReq.Header.Set(k, v)
				}

				// Tracing span hook (if configured)
				var finishSpan func(int, error)
				if polyfill.spanHook != nil {
					finishSpan = polyfill.spanHook(req.Method, req.URL)
				}

				resp, err := client.Do(httpReq)
				if err != nil {
					if finishSpan != nil {
						finishSpan(0, err)
					}
					if goCtx.Err() != nil {
						return // bridge closing — don't schedule
					}
					ctx.Schedule(func(ctx *quickjs.Context) {
						errVal := ctx.NewError(fmt.Errorf("fetch: %w", err))
						defer errVal.Free()
						reject(errVal)
					})
					return
				}

				if finishSpan != nil {
					finishSpan(resp.StatusCode, nil)
				}

				respHeaders := make(map[string]string)
				for k, v := range resp.Header {
					respHeaders[strings.ToLower(k)] = strings.Join(v, ", ")
				}
				headersJSON, _ := json.Marshal(respHeaders)
				respURL := resp.Request.URL.String()

				// Detect SSE streaming from response Content-Type
				contentType := resp.Header.Get("Content-Type")
				streaming := strings.Contains(contentType, "text/event-stream") ||
					strings.Contains(contentType, "text/plain") && resp.TransferEncoding != nil

				if streaming {
					// SSE/streaming mode: deliver chunks incrementally via ReadableStream.
					// The Response.body is a ReadableStream backed by this goroutine.
					streamID := fmt.Sprintf("s%p", resp)

					ctx.Schedule(func(ctx *quickjs.Context) {
						// Create a ReadableStream with a controller we can push to
						ctx.Eval(fmt.Sprintf(`
							globalThis.__stream_ctrl_%s = null;
							globalThis.__stream_%s = new ReadableStream({
								start(controller) {
									globalThis.__stream_ctrl_%s = controller;
								}
							});
						`, streamID, streamID, streamID))

						// Build Response with the streaming body
						ctx.Eval(fmt.Sprintf(`
							globalThis.__stream_resp_%s = {
								status: %d,
								statusText: %q,
								headers: %s,
								url: %q,
								body: globalThis.__stream_%s,
							};
						`, streamID, resp.StatusCode, resp.Status, string(headersJSON), respURL, streamID))

						// Resolve the fetch Promise with the Response
						resolveJS := ctx.Eval(fmt.Sprintf(`globalThis.__stream_resp_%s`, streamID))
						resolve(resolveJS)
					})

					// Read body incrementally in this goroutine
					reader := bufio.NewReaderSize(resp.Body, 4096)
					buf := make([]byte, 4096)
					for {
						n, readErr := reader.Read(buf)
						if goCtx.Err() != nil {
							resp.Body.Close()
							return // bridge closing
						}
						if n > 0 {
							chunk := string(buf[:n])
							ctx.Schedule(func(ctx *quickjs.Context) {
								ctx.Eval(fmt.Sprintf(
									`globalThis.__stream_ctrl_%s?.enqueue(new TextEncoder().encode(%q))`,
									streamID, chunk,
								))
							})
						}
						if readErr != nil {
							resp.Body.Close()
							ctx.Schedule(func(ctx *quickjs.Context) {
								ctx.Eval(fmt.Sprintf(
									`globalThis.__stream_ctrl_%s?.close(); delete globalThis.__stream_ctrl_%s; delete globalThis.__stream_%s; delete globalThis.__stream_resp_%s`,
									streamID, streamID, streamID, streamID,
								))
							})
							break
						}
					}
				} else {
					// Non-streaming mode: read full body, resolve once.
					defer resp.Body.Close()
					respBody, err := io.ReadAll(resp.Body)
					if err != nil {
						if goCtx.Err() != nil {
							return
						}
						ctx.Schedule(func(ctx *quickjs.Context) {
							errVal := ctx.NewError(fmt.Errorf("fetch: read body: %w", err))
							defer errVal.Free()
							reject(errVal)
						})
						return
					}

					// Binary responses (audio, images, octet-stream, ...) cannot
					// survive Go-string → JSON-marshal → JS-string because
					// invalid UTF-8 bytes get rewritten as U+FFFD. Base64
					// the body and let the JS side decode it back.
					bodyStr := ""
					bodyEncoding := ""
					if isTextContentType(contentType) {
						bodyStr = string(respBody)
					} else {
						bodyStr = base64.StdEncoding.EncodeToString(respBody)
						bodyEncoding = "base64"
					}
					result := map[string]interface{}{
						"status":       resp.StatusCode,
						"statusText":   resp.Status,
						"body":         bodyStr,
						"bodyEncoding": bodyEncoding,
						"headers":      respHeaders,
						"url":          respURL,
					}
					b, _ := json.Marshal(result)
					resultJSON := string(b)

					ctx.Schedule(func(ctx *quickjs.Context) {
						resolve(ctx.NewString(resultJSON))
					})
				}
			})
		})
	}))

	return evalJS(ctx, fetchJS)
}

// isTextContentType returns true when the response Content-Type is
// known to be UTF-8 safe (so the body survives Go-string +
// JSON-marshal). Anything else gets base64'd to preserve binary.
func isTextContentType(ct string) bool {
	if ct == "" {
		// Empty bodies and unknown types are safest as text — they
		// either round-trip (ASCII) or are short enough that
		// caller can override.
		return true
	}
	low := strings.ToLower(ct)
	if strings.HasPrefix(low, "text/") {
		return true
	}
	if strings.HasPrefix(low, "application/json") ||
		strings.HasPrefix(low, "application/xml") ||
		strings.HasPrefix(low, "application/javascript") ||
		strings.HasPrefix(low, "application/x-www-form-urlencoded") ||
		strings.HasPrefix(low, "application/x-ndjson") ||
		strings.HasPrefix(low, "application/ld+json") {
		return true
	}
	if strings.Contains(low, "+json") || strings.Contains(low, "+xml") {
		return true
	}
	return false
}

const fetchJS = `
// FormData — minimal polyfill for multipart/form-data bodies.
// The OpenAI Node SDK + many HTTP clients check
// ` + "`body instanceof FormData`" + ` during request serialization, which
// throws ReferenceError when FormData is undefined. Shape
// matches the Fetch spec: append / get / getAll / has / set /
// delete / entries / keys / values / forEach.
globalThis.FormData = class FormData {
  constructor() {
    // entries is an ordered list of {name, value, filename} so
    // duplicate appends preserve order + multi-value semantics.
    this._e = [];
  }
  append(name, value, filename) {
    this._e.push({ name: String(name), value: value, filename: filename });
  }
  set(name, value, filename) {
    const k = String(name);
    this._e = this._e.filter(x => x.name !== k);
    this._e.push({ name: k, value: value, filename: filename });
  }
  delete(name) { const k = String(name); this._e = this._e.filter(x => x.name !== k); }
  has(name) { const k = String(name); return this._e.some(x => x.name === k); }
  get(name) {
    const k = String(name);
    const hit = this._e.find(x => x.name === k);
    return hit ? hit.value : null;
  }
  getAll(name) {
    const k = String(name);
    return this._e.filter(x => x.name === k).map(x => x.value);
  }
  *entries() { for (const e of this._e) yield [e.name, e.value]; }
  *keys()    { for (const e of this._e) yield e.name; }
  *values()  { for (const e of this._e) yield e.value; }
  [Symbol.iterator]() { return this.entries(); }
  forEach(fn) { for (const e of this._e) fn(e.value, e.name, this); }
};

// Blob — minimal spec-shape polyfill. Many SDKs wrap upload
// bytes in a Blob; the polyfill carries the bytes verbatim for
// round-tripping through FormData + fetch.
if (typeof globalThis.Blob === 'undefined' || globalThis.Blob === null) {
  globalThis.Blob = class Blob {
    constructor(parts, options) {
      this._parts = parts || [];
      this.type = (options && options.type) || '';
      this.size = 0;
      for (const p of this._parts) {
        if (typeof p === 'string') this.size += p.length;
        else if (p && typeof p.length === 'number') this.size += p.length;
        else if (p && typeof p.byteLength === 'number') this.size += p.byteLength;
      }
    }
    arrayBuffer() {
      const bufs = this._parts.map(p => {
        if (typeof p === 'string') return new TextEncoder().encode(p);
        if (p && p.buffer) return new Uint8Array(p.buffer, p.byteOffset || 0, p.byteLength);
        if (p && typeof p.length === 'number') return new Uint8Array(p);
        return new Uint8Array(0);
      });
      const total = bufs.reduce((a, b) => a + b.byteLength, 0);
      const out = new Uint8Array(total);
      let off = 0;
      for (const b of bufs) { out.set(b, off); off += b.byteLength; }
      return Promise.resolve(out.buffer);
    }
    text() {
      return this.arrayBuffer().then(ab => new TextDecoder().decode(ab));
    }
    stream() {
      // OpenAI SDK's multipart serializer reads file.stream() to
      // pipe contents into the request. Wrap the arrayBuffer in a
      // single-chunk ReadableStream so the existing Fetch path
      // drains cleanly.
      const buf = this;
      return new ReadableStream({
        start(controller) {
          buf.arrayBuffer().then(ab => {
            controller.enqueue(new Uint8Array(ab));
            controller.close();
          });
        },
      });
    }
    slice(start, end, contentType) {
      return new Blob([], { type: contentType || this.type });
    }
  };
}

// File — extends Blob with a name + lastModified.
if (typeof globalThis.File === 'undefined' || globalThis.File === null) {
  globalThis.File = class File extends globalThis.Blob {
    constructor(parts, name, options) {
      super(parts, options);
      this.name = String(name || '');
      this.lastModified = (options && options.lastModified) || Date.now();
    }
  };
}

globalThis.Headers = class Headers {
  constructor(init) {
    this._m = {};
    if (init) {
      if (typeof init.forEach === 'function') {
        init.forEach((v, k) => { this._m[k.toLowerCase()] = String(v); });
      } else if (typeof init === 'object') {
        for (const [k, v] of Object.entries(init)) {
          this._m[k.toLowerCase()] = String(v);
        }
      }
    }
  }
  get(n) { return this._m[n.toLowerCase()] || null; }
  has(n) { return n.toLowerCase() in this._m; }
  set(n, v) { this._m[n.toLowerCase()] = String(v); }
  // append adds a value to an existing header (comma-joined per
  // Fetch spec) or sets it when absent. Required by the OpenAI
  // Node SDK's Headers accumulator and any other lib that
  // follows the real fetch Headers contract.
  append(n, v) {
    const k = n.toLowerCase();
    if (k in this._m) {
      this._m[k] = this._m[k] + ", " + String(v);
    } else {
      this._m[k] = String(v);
    }
  }
  delete(n) { delete this._m[n.toLowerCase()]; }
  getSetCookie() {
    // Spec method: return all set-cookie values as an array.
    // brainkit keeps a single string keyed on set-cookie; split
    // on the pattern browsers use when round-tripping.
    const v = this._m["set-cookie"];
    if (!v) return [];
    return String(v).split(/, (?=[^;]+?=)/);
  }
  entries() { return Object.entries(this._m)[Symbol.iterator](); }
  keys() { return Object.keys(this._m)[Symbol.iterator](); }
  values() { return Object.values(this._m)[Symbol.iterator](); }
  [Symbol.iterator]() { return this.entries(); }
  forEach(fn) { Object.entries(this._m).forEach(([k,v]) => fn(v,k,this)); }
};

globalThis.Response = class Response {
  constructor(body, init) {
    this._formData = null;
    if (body && typeof body === 'object' && body.body) {
      // Streaming response from Go — body is {status, headers, url, body: ReadableStream}
      this._body = null;
      this._bodyStream = body.body;
      this._bodyEncoding = '';
      this.status = body.status || 200;
      this.ok = this.status >= 200 && this.status < 300;
      this.statusText = body.statusText || '';
      this.headers = new Headers(body.headers);
      this.url = body.url || '';
    } else if (typeof FormData !== 'undefined' && body instanceof FormData) {
      // FormData body — serialize lazily in text()/arrayBuffer(). The
      // OpenAI SDK probes FormData support with
      //   data.toString() === (await new Response(data).text())
      // so text() MUST differ from "[object Object]". Using String(body)
      // at construction time would collapse the two to the same value
      // and trick the SDK into thinking file uploads are unsupported.
      this._body = null;
      this._bodyStream = null;
      this._bodyEncoding = '';
      this._formData = body;
      init = init || {};
      this.status = init.status || 200;
      this.ok = this.status >= 200 && this.status < 300;
      this.statusText = init.statusText || '';
      this.headers = new Headers(init.headers);
      this.url = '';
    } else {
      this._body = body != null ? String(body) : '';
      this._bodyStream = null;
      init = init || {};
      // bodyEncoding === 'base64' means the body is base64-encoded
      // raw bytes (binary response from Go that wouldn't survive
      // JSON-marshal as a string). Decode lazily in arrayBuffer / body.
      this._bodyEncoding = init.bodyEncoding || '';
      this.status = init.status || 200;
      this.ok = this.status >= 200 && this.status < 300;
      this.statusText = init.statusText || '';
      this.headers = new Headers(init.headers);
      this.url = '';
    }
    this.type = 'basic';
    this.redirected = false;
    this.bodyUsed = false;
  }
  async _materializeFormData() {
    if (!this._formData) return;
    const fd = this._formData;
    this._formData = null;
    const boundary = '----brainkitResponseBoundary' + Math.random().toString(16).slice(2);
    const enc = new TextEncoder();
    const parts = [];
    for (const [name, value] of fd.entries()) {
      let header;
      let bytes;
      if (value && typeof value === 'object' && typeof value.arrayBuffer === 'function') {
        const fname = value.name || name;
        const ctype = value.type || 'application/octet-stream';
        header = 'Content-Disposition: form-data; name="' + name +
                 '"; filename="' + fname + '"\r\n' +
                 'Content-Type: ' + ctype + '\r\n\r\n';
        bytes = new Uint8Array(await value.arrayBuffer());
      } else {
        header = 'Content-Disposition: form-data; name="' + name + '"\r\n\r\n';
        bytes = enc.encode(String(value));
      }
      parts.push(enc.encode('--' + boundary + '\r\n' + header));
      parts.push(bytes);
      parts.push(enc.encode('\r\n'));
    }
    parts.push(enc.encode('--' + boundary + '--\r\n'));
    const total = parts.reduce((n, b) => n + b.byteLength, 0);
    const out = new Uint8Array(total);
    let off = 0;
    for (const b of parts) { out.set(b, off); off += b.byteLength; }
    // Store as base64 so _bytes() can round-trip without re-encoding.
    let bin = '';
    for (let i = 0; i < out.byteLength; i++) bin += String.fromCharCode(out[i]);
    this._body = btoa(bin);
    this._bodyEncoding = 'base64';
    if (!this.headers.get('content-type')) {
      this.headers.set('content-type', 'multipart/form-data; boundary=' + boundary);
    }
  }
  _bytes() {
    // Return the body as a Uint8Array. Handles both base64-encoded
    // binary and plain text (utf-8 encoded).
    if (this._bodyEncoding === 'base64') {
      const bin = atob(this._body || '');
      const u8 = new Uint8Array(bin.length);
      for (let i = 0; i < bin.length; i++) u8[i] = bin.charCodeAt(i) & 0xFF;
      return u8;
    }
    return new TextEncoder().encode(this._body || '');
  }
  get body() {
    if (this._bodyStream) return this._bodyStream;
    if (typeof ReadableStream === 'undefined') return null;
    const bytes = this._bytes();
    return new ReadableStream({
      start(controller) {
        if (bytes.byteLength > 0) controller.enqueue(bytes);
        controller.close();
      }
    });
  }
  async text() {
    this.bodyUsed = true;
    if (this._formData) await this._materializeFormData();
    if (this._bodyStream) {
      const reader = this._bodyStream.getReader();
      const chunks = [];
      const decoder = new TextDecoder();
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        chunks.push(typeof value === 'string' ? value : decoder.decode(value));
      }
      this._body = chunks.join('');
      this._bodyStream = null;
      this._bodyEncoding = '';
      return this._body;
    }
    if (this._bodyEncoding === 'base64') {
      // Caller asked for text on a binary body — best-effort decode
      // through TextDecoder so utf-8 sequences come out right.
      return new TextDecoder().decode(this._bytes());
    }
    return this._body || '';
  }
  async json() { return JSON.parse(await this.text()); }
  async arrayBuffer() {
    this.bodyUsed = true;
    if (this._formData) await this._materializeFormData();
    if (this._bodyStream) {
      const reader = this._bodyStream.getReader();
      const chunks = [];
      let total = 0;
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        const u8 = value instanceof Uint8Array ? value :
                   typeof value === 'string' ? new TextEncoder().encode(value) :
                   new Uint8Array(value);
        chunks.push(u8);
        total += u8.byteLength;
      }
      const out = new Uint8Array(total);
      let off = 0;
      for (const c of chunks) { out.set(c, off); off += c.byteLength; }
      this._bodyStream = null;
      return out.buffer;
    }
    return this._bytes().buffer;
  }
  clone() {
    if (this._bodyStream) throw new Error('Cannot clone a streaming response');
    return new Response(this._body, {
      status: this.status, statusText: this.statusText, headers: this.headers._m,
      bodyEncoding: this._bodyEncoding
    });
  }
};

globalThis.Request = class Request {
  constructor(input, init) {
    this.url = typeof input === 'string' ? input : input.url;
    init = init || {};
    this.method = init.method || 'GET';
    this.headers = new Headers(init.headers);
    this._body = init.body || null;
  }
  async text() { return this._body || ''; }
  async json() { return JSON.parse(this._body); }
};

// _serializeBody turns whatever the caller passed in as the
// request body into a string (or base64-wrapped-string for
// binary multipart) the Go fetch side can forward. Handles:
//   - FormData → multipart/form-data with a generated boundary;
//     sets Content-Type on the headers in place. Supports
//     Blob/File parts (reads .arrayBuffer() for the bytes).
//   - URLSearchParams → x-www-form-urlencoded string.
//   - Blob / ArrayBuffer / TypedArray → base64-wrapped string
//     with a Content-Type default.
//   - string / everything else → String(body).
// The returned object is { body, headers } — headers are the
// possibly-augmented request headers.
async function _serializeBody(body, headers) {
  if (body == null) return { body: null, headers };
  if (typeof body === 'string') return { body, headers };
  if (typeof FormData !== 'undefined' && body instanceof FormData) {
    const boundary = '----brainkitBoundary' + Math.random().toString(16).slice(2);
    const enc = new TextEncoder();
    const parts = [];
    for (const [name, value] of body.entries()) {
      let header;
      let bytes;
      if (value && typeof value === 'object' && typeof value.arrayBuffer === 'function') {
        const fname = value.name || name;
        const ctype = value.type || 'application/octet-stream';
        header = 'Content-Disposition: form-data; name="' + name +
                 '"; filename="' + fname + '"\r\n' +
                 'Content-Type: ' + ctype + '\r\n\r\n';
        bytes = new Uint8Array(await value.arrayBuffer());
      } else {
        header = 'Content-Disposition: form-data; name="' + name + '"\r\n\r\n';
        bytes = enc.encode(String(value));
      }
      parts.push(enc.encode('--' + boundary + '\r\n' + header));
      parts.push(bytes);
      parts.push(enc.encode('\r\n'));
    }
    parts.push(enc.encode('--' + boundary + '--\r\n'));
    const total = parts.reduce((n, b) => n + b.byteLength, 0);
    const out = new Uint8Array(total);
    let off = 0;
    for (const b of parts) { out.set(b, off); off += b.byteLength; }
    // Base64 encode so the body survives the JSON hop to Go.
    let bin = '';
    for (let i = 0; i < out.byteLength; i++) bin += String.fromCharCode(out[i]);
    headers['content-type'] = 'multipart/form-data; boundary=' + boundary;
    headers['x-brainkit-body-encoding'] = 'base64';
    return { body: btoa(bin), headers };
  }
  if (typeof URLSearchParams !== 'undefined' && body instanceof URLSearchParams) {
    if (!headers['content-type']) headers['content-type'] = 'application/x-www-form-urlencoded;charset=UTF-8';
    return { body: body.toString(), headers };
  }
  if (typeof Blob !== 'undefined' && body instanceof Blob) {
    if (!headers['content-type']) headers['content-type'] = body.type || 'application/octet-stream';
    const ab = await body.arrayBuffer();
    const u8 = new Uint8Array(ab);
    let bin = '';
    for (let i = 0; i < u8.byteLength; i++) bin += String.fromCharCode(u8[i]);
    headers['x-brainkit-body-encoding'] = 'base64';
    return { body: btoa(bin), headers };
  }
  if (body instanceof ArrayBuffer || ArrayBuffer.isView(body)) {
    const u8 = body instanceof ArrayBuffer ? new Uint8Array(body) :
               new Uint8Array(body.buffer, body.byteOffset, body.byteLength);
    if (!headers['content-type']) headers['content-type'] = 'application/octet-stream';
    let bin = '';
    for (let i = 0; i < u8.byteLength; i++) bin += String.fromCharCode(u8[i]);
    headers['x-brainkit-body-encoding'] = 'base64';
    return { body: btoa(bin), headers };
  }
  return { body: String(body), headers };
}

globalThis.fetch = async (input, init) => {
  const url = typeof input === 'string' ? input : (input && input.url) || String(input);
  const opts = init || {};
  if (typeof input !== 'string' && input) {
    if (!opts.method && input.method) opts.method = input.method;
    if (!opts.headers && input.headers) opts.headers = input.headers;
    if (!opts.body && input._body) opts.body = input._body;
  }
  const method = opts.method || 'GET';
  const headers = {};
  if (opts.headers) {
    if (typeof opts.headers.forEach === 'function') {
      opts.headers.forEach((v, k) => { headers[k.toLowerCase()] = v; });
    } else if (typeof opts.headers === 'object') {
      for (const [k, v] of Object.entries(opts.headers)) headers[k.toLowerCase()] = String(v);
    }
  }
  const serialized = await _serializeBody(opts.body, headers);
  const body = serialized.body;
  Object.assign(headers, serialized.headers);

  // Always use streaming mode — Go decides based on response Content-Type
  // whether to read the full body or deliver chunks incrementally.
  // The AI SDK sets stream:true in the body and the server responds with
  // Content-Type: text/event-stream for SSE.
  const raw = await __go_fetch(JSON.stringify({ url, method, headers, body }));
  if (typeof raw === 'string') {
    // Non-streaming response — full body returned as JSON
    const data = JSON.parse(raw);
    const resp = new Response(data.body, {
      status: data.status, statusText: data.statusText, headers: data.headers,
      bodyEncoding: data.bodyEncoding || ''
    });
    resp.url = data.url || url;
    return resp;
  } else {
    // Streaming response — Response object with ReadableStream body
    return new Response(raw);
  }
};
`
