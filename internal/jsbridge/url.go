package jsbridge

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	quickjs "github.com/buke/quickjs-go"
)

// URLPolyfill provides globalThis.URL and globalThis.URLSearchParams.
type URLPolyfill struct{}

// URL creates a URL polyfill.
func URL() *URLPolyfill { return &URLPolyfill{} }

func (p *URLPolyfill) Name() string { return "url" }

func (p *URLPolyfill) Setup(ctx *quickjs.Context) error {
	ctx.Globals().Set("__go_url_parse", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		if len(args) == 0 {
			return ctx.ThrowError(fmt.Errorf("URL: missing argument"))
		}
		raw := args[0].ToString()

		if len(args) > 1 && !args[1].IsUndefined() && !args[1].IsNull() {
			base, err := url.Parse(args[1].ToString())
			if err != nil {
				return ctx.ThrowError(fmt.Errorf("URL: invalid base: %w", err))
			}
			ref, err := url.Parse(raw)
			if err != nil {
				return ctx.ThrowError(fmt.Errorf("URL: invalid url: %w", err))
			}
			raw = base.ResolveReference(ref).String()
		}

		u, err := url.Parse(raw)
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("URL: invalid url %q: %w", raw, err))
		}

		username := ""
		password := ""
		if u.User != nil {
			username = u.User.Username()
			password, _ = u.User.Password()
		}

		result := map[string]string{
			"href":     u.String(),
			"protocol": u.Scheme + ":",
			"hostname": u.Hostname(),
			"host":     u.Host,
			"port":     u.Port(),
			"pathname": u.Path,
			"search":   "",
			"hash":     "",
			"origin":   u.Scheme + "://" + u.Host,
			"username": username,
			"password": password,
		}
		if u.RawQuery != "" {
			result["search"] = "?" + u.RawQuery
		}
		if u.Fragment != "" {
			result["hash"] = "#" + u.Fragment
		}

		b, _ := json.Marshal(result)
		return ctx.ParseJSON(string(b))
	}))

	ctx.Globals().Set("__go_url_search_params", ctx.NewFunction(func(ctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		qs := ""
		if len(args) > 0 {
			qs = strings.TrimPrefix(args[0].ToString(), "?")
		}
		vals, err := url.ParseQuery(qs)
		if err != nil {
			return ctx.ThrowError(fmt.Errorf("URLSearchParams: %w", err))
		}
		var pairs [][]string
		for k, vs := range vals {
			for _, v := range vs {
				pairs = append(pairs, []string{k, v})
			}
		}
		b, _ := json.Marshal(pairs)
		return ctx.ParseJSON(string(b))
	}))

	return evalJS(ctx, urlJS)
}

const urlJS = `
globalThis.URL = class URL {
  constructor(url, base) {
    const p = __go_url_parse(url, base);
    this.href = p.href; this.protocol = p.protocol;
    this.hostname = p.hostname; this.host = p.host;
    this.port = p.port; this.pathname = p.pathname;
    this.search = p.search; this.hash = p.hash;
    this.origin = p.origin; this.username = p.username;
    this.password = p.password;
    this.searchParams = new URLSearchParams(this.search);
  }
  toString() { return this.href; }
  toJSON() { return this.href; }
};

globalThis.URLSearchParams = class URLSearchParams {
  constructor(init) {
    this._p = [];
    if (typeof init === 'string') {
      const pairs = __go_url_search_params(init);
      if (pairs) this._p = pairs;
    } else if (init && typeof init === 'object') {
      for (const [k, v] of Object.entries(init)) {
        this._p.push([String(k), String(v)]);
      }
    }
  }
  get(n) { const e = this._p.find(([k]) => k === n); return e ? e[1] : null; }
  getAll(n) { return this._p.filter(([k]) => k === n).map(([,v]) => v); }
  has(n) { return this._p.some(([k]) => k === n); }
  set(n, v) { this.delete(n); this._p.push([n, String(v)]); }
  append(n, v) { this._p.push([n, String(v)]); }
  delete(n) { this._p = this._p.filter(([k]) => k !== n); }
  toString() {
    return this._p.map(([k,v]) => encodeURIComponent(k)+'='+encodeURIComponent(v)).join('&');
  }
  entries() { return this._p[Symbol.iterator](); }
  keys() { return this._p.map(([k]) => k)[Symbol.iterator](); }
  values() { return this._p.map(([,v]) => v)[Symbol.iterator](); }
  [Symbol.iterator]() { return this.entries(); }
  forEach(fn) { this._p.forEach(([k,v]) => fn(v,k,this)); }
};
`
