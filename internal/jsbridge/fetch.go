package jsbridge

import (
	"bufio"
	"context"
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
					bodyReader = strings.NewReader(*req.Body)
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

					result := map[string]interface{}{
						"status":     resp.StatusCode,
						"statusText": resp.Status,
						"body":       string(respBody),
						"headers":    respHeaders,
						"url":        respURL,
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

const fetchJS = `
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
  delete(n) { delete this._m[n.toLowerCase()]; }
  entries() { return Object.entries(this._m)[Symbol.iterator](); }
  keys() { return Object.keys(this._m)[Symbol.iterator](); }
  values() { return Object.values(this._m)[Symbol.iterator](); }
  [Symbol.iterator]() { return this.entries(); }
  forEach(fn) { Object.entries(this._m).forEach(([k,v]) => fn(v,k,this)); }
};

globalThis.Response = class Response {
  constructor(body, init) {
    if (body && typeof body === 'object' && body.body) {
      // Streaming response from Go — body is {status, headers, url, body: ReadableStream}
      this._body = null;
      this._bodyStream = body.body;
      this.status = body.status || 200;
      this.ok = this.status >= 200 && this.status < 300;
      this.statusText = body.statusText || '';
      this.headers = new Headers(body.headers);
      this.url = body.url || '';
    } else {
      this._body = body != null ? String(body) : '';
      this._bodyStream = null;
      init = init || {};
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
  get body() {
    if (this._bodyStream) return this._bodyStream;
    const text = this._body;
    if (typeof ReadableStream === 'undefined') return null;
    return new ReadableStream({
      start(controller) {
        if (text && text.length > 0) {
          if (typeof TextEncoder !== 'undefined') {
            controller.enqueue(new TextEncoder().encode(text));
          } else {
            controller.enqueue(text);
          }
        }
        controller.close();
      }
    });
  }
  async text() {
    this.bodyUsed = true;
    if (this._body !== null) return this._body;
    // Read from stream
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
      return this._body;
    }
    return '';
  }
  async json() { return JSON.parse(await this.text()); }
  async arrayBuffer() {
    this.bodyUsed = true;
    const text = await this.text();
    const enc = new TextEncoder();
    return enc.encode(text).buffer;
  }
  clone() {
    if (this._bodyStream) throw new Error('Cannot clone a streaming response');
    return new Response(this._body, {
      status: this.status, statusText: this.statusText, headers: this.headers._m
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
      opts.headers.forEach((v, k) => { headers[k] = v; });
    } else if (typeof opts.headers === 'object') {
      for (const [k, v] of Object.entries(opts.headers)) headers[k] = String(v);
    }
  }
  const body = opts.body != null ? String(opts.body) : null;

  // Always use streaming mode — Go decides based on response Content-Type
  // whether to read the full body or deliver chunks incrementally.
  // The AI SDK sets stream:true in the body and the server responds with
  // Content-Type: text/event-stream for SSE.
  const raw = await __go_fetch(JSON.stringify({ url, method, headers, body }));
  if (typeof raw === 'string') {
    // Non-streaming response — full body returned as JSON
    const data = JSON.parse(raw);
    const resp = new Response(data.body, {
      status: data.status, statusText: data.statusText, headers: data.headers
    });
    resp.url = data.url || url;
    return resp;
  } else {
    // Streaming response — Response object with ReadableStream body
    return new Response(raw);
  }
};
`
