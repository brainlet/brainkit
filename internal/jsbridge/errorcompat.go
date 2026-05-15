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

// Error.prepareStackTrace — V8-specific. Packages such as Stagehand inspect
// callsite objects to discover their current module path. QuickJS exposes stack
// text, so provide a best-effort callsite object view when prepareStackTrace is
// installed.
(function() {
  if (Error.__brainkitPrepareStackTracePatched) return;
  var NativeError = Error;

  function parseCallsites(stackText) {
    var lines = String(stackText || "").split("\n").slice(1);
    return lines.map(function(line) {
      var text = String(line || "").trim();
      var fn = null;
      var file = null;
      var lineNo = null;
      var colNo = null;
      var match = text.match(/^at\s+(.*?)\s+\((.*?):(\d+):(\d+)\)$/) ||
        text.match(/^at\s+(.*?):(\d+):(\d+)$/);
      if (match) {
        if (match.length === 5) {
          fn = match[1] || null;
          file = match[2] || null;
          lineNo = Number(match[3]);
          colNo = Number(match[4]);
        } else {
          file = match[1] || null;
          lineNo = Number(match[2]);
          colNo = Number(match[3]);
        }
      }
      return {
        getFileName: function() { return file; },
        getScriptNameOrSourceURL: function() { return file; },
        getFunctionName: function() { return fn; },
        getMethodName: function() { return null; },
        getLineNumber: function() { return lineNo; },
        getColumnNumber: function() { return colNo; },
        isNative: function() { return file === "native"; },
        isEval: function() { return file === "<eval>" || text.indexOf("<eval>") >= 0; },
        toString: function() { return text; },
      };
    });
  }

  function BrainkitError(message) {
    var err = new NativeError(message);
    var stackText = err.stack || "";
    try {
      Object.setPrototypeOf(err, new.target ? new.target.prototype : BrainkitError.prototype);
    } catch (_) {}
    Object.defineProperty(err, "stack", {
      configurable: true,
      get: function() {
        if (typeof BrainkitError.prepareStackTrace === "function") {
          return BrainkitError.prepareStackTrace(err, parseCallsites(stackText));
        }
        return stackText;
      },
      set: function(value) { stackText = value; },
    });
    return err;
  }
  BrainkitError.prototype = NativeError.prototype;
  try { Object.setPrototypeOf(BrainkitError, NativeError); } catch (_) {}
  Object.getOwnPropertyNames(NativeError).forEach(function(name) {
    if (name === "length" || name === "name" || name === "prototype" || name === "prepareStackTrace") return;
    try {
      Object.defineProperty(BrainkitError, name, Object.getOwnPropertyDescriptor(NativeError, name));
    } catch (_) {}
  });
  var prepareStackTraceValue = NativeError.prepareStackTrace;
  Object.defineProperty(BrainkitError, "prepareStackTrace", {
    configurable: true,
    get: function() { return prepareStackTraceValue; },
    set: function(value) { prepareStackTraceValue = value; },
  });
  BrainkitError.__brainkitPrepareStackTracePatched = true;
  globalThis.Error = BrainkitError;
})();

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
