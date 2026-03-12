package jsbridge

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/fastschema/qjs"
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

func (p *FetchPolyfill) Setup(ctx *qjs.Context) error {
	ctx.SetFunc("__go_fetch_sync", func(this *qjs.This) (*qjs.Value, error) {
		args := this.Args()
		if len(args) == 0 {
			return nil, fmt.Errorf("fetch: missing request argument")
		}

		var req struct {
			URL     string            `json:"url"`
			Method  string            `json:"method"`
			Headers map[string]string `json:"headers"`
			Body    *string           `json:"body"`
		}
		if err := json.Unmarshal([]byte(args[0].String()), &req); err != nil {
			return nil, fmt.Errorf("fetch: invalid request: %w", err)
		}

		var bodyReader io.Reader
		if req.Body != nil && *req.Body != "" {
			bodyReader = strings.NewReader(*req.Body)
		}

		httpReq, err := http.NewRequest(req.Method, req.URL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("fetch: %w", err)
		}
		for k, v := range req.Headers {
			httpReq.Header.Set(k, v)
		}

		resp, err := p.client.Do(httpReq)
		if err != nil {
			return nil, fmt.Errorf("fetch: %w", err)
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("fetch: read body: %w", err)
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
		return this.Context().NewString(string(b)), nil
	})

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
  const raw = __go_fetch_sync(JSON.stringify({ url, method, headers, body }));
  const data = JSON.parse(raw);
  const resp = new Response(data.body, {
    status: data.status, statusText: data.statusText, headers: data.headers
  });
  resp.url = data.url || url;
  return resp;
};
`
