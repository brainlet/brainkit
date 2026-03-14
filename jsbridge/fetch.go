package jsbridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	quickjs "github.com/buke/quickjs-go"
)

// FetchPolyfill provides globalThis.fetch, Headers, Response, and Request.
type FetchPolyfill struct {
	client *http.Client
}

// FetchOption configures a FetchPolyfill.
type FetchOption func(*FetchPolyfill)

// FetchClient sets the HTTP client used for requests.
func FetchClient(c *http.Client) FetchOption {
	return func(p *FetchPolyfill) { p.client = c }
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

func (p *FetchPolyfill) Setup(ctx *quickjs.Context) error {
	client := p.client

	// Async fetch: returns a Promise, HTTP runs in a separate goroutine.
	// The bridge is NOT held during the HTTP call — other scheduled work
	// (bus deliveries, other Promise resolutions) can run via ProcessJobs().
	ctx.Globals().Set("__go_fetch", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) == 0 {
			return ctx.ThrowError(fmt.Errorf("fetch: missing request argument"))
		}

		reqJSON := args[0].ToString()

		return ctx.NewPromise(func(resolve, reject func(*quickjs.Value)) {
			go func() {
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

				httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)
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

				resp, err := client.Do(httpReq)
				if err != nil {
					ctx.Schedule(func(ctx *quickjs.Context) {
						errVal := ctx.NewError(fmt.Errorf("fetch: %w", err))
						defer errVal.Free()
						reject(errVal)
					})
					return
				}
				defer resp.Body.Close()

				respBody, err := io.ReadAll(resp.Body)
				if err != nil {
					ctx.Schedule(func(ctx *quickjs.Context) {
						errVal := ctx.NewError(fmt.Errorf("fetch: read body: %w", err))
						defer errVal.Free()
						reject(errVal)
					})
					return
				}

				respHeaders := make(map[string]string)
				for k, v := range resp.Header {
					respHeaders[strings.ToLower(k)] = strings.Join(v, ", ")
				}

				result := map[string]interface{}{
					"status":     resp.StatusCode,
					"statusText": resp.Status,
					"body":       string(respBody),
					"headers":    respHeaders,
					"url":        resp.Request.URL.String(),
				}
				b, _ := json.Marshal(result)
				resultJSON := string(b)

				ctx.Schedule(func(ctx *quickjs.Context) {
					resolve(ctx.NewString(resultJSON))
				})
			}()
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
    this._body = body != null ? String(body) : '';
    init = init || {};
    this.status = init.status || 200;
    this.ok = this.status >= 200 && this.status < 300;
    this.statusText = init.statusText || '';
    this.headers = new Headers(init.headers);
    this.type = 'basic';
    this.url = '';
    this.redirected = false;
    this.bodyUsed = false;
  }
  get body() {
    const text = this._body;
    if (typeof ReadableStream === 'undefined') return null;
    return new ReadableStream({
      start(controller) {
        if (text.length > 0) {
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
  async text() { this.bodyUsed = true; return this._body; }
  async json() { this.bodyUsed = true; return JSON.parse(this._body); }
  async arrayBuffer() {
    this.bodyUsed = true;
    const enc = new TextEncoder();
    return enc.encode(this._body).buffer;
  }
  clone() {
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
  const raw = await __go_fetch(JSON.stringify({ url, method, headers, body }));
  const data = JSON.parse(raw);
  const resp = new Response(data.body, {
    status: data.status, statusText: data.statusText, headers: data.headers
  });
  resp.url = data.url || url;
  return resp;
};
`
