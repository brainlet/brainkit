package jsbridge

import quickjs "github.com/buke/quickjs-go"

// ErrorCompatPolyfill provides Error.captureStackTrace and Response.json.
// Error.captureStackTrace is V8-specific, used by pg-pool and other Node.js libs.
// Response.json is a static method some SDK providers expect.
type ErrorCompatPolyfill struct{}

func ErrorCompat() *ErrorCompatPolyfill { return &ErrorCompatPolyfill{} }

func (p *ErrorCompatPolyfill) Name() string { return "errorcompat" }

func (p *ErrorCompatPolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, `
// Error.captureStackTrace — V8-specific, used by pg-pool
if (!Error.captureStackTrace) {
  Error.captureStackTrace = function(err, constructorOpt) {
    if (err && !err.stack) {
      err.stack = new Error().stack || "";
    }
  };
}

// global alias — required by pg npm package
if (typeof global === "undefined") {
  globalThis.global = globalThis;
}

// Response.json static — some SDK providers use Response.json()
if (typeof Response !== "undefined" && !Response.json) {
  Response.json = function(data, init) {
    var body = JSON.stringify(data);
    var headers = new Headers(init && init.headers);
    if (!headers.has("content-type")) headers.set("content-type", "application/json");
    return new Response(body, {
      status: (init && init.status) || 200,
      statusText: (init && init.statusText) || "OK",
      headers: headers,
    });
  };
}
`)
}
