package jsbridge

import quickjs "github.com/buke/quickjs-go"

// NodeCompatPolyfill provides pure JavaScript Node compatibility surfaces that
// do not need host resources.
type NodeCompatPolyfill struct{}

// NodeCompat creates pure JavaScript Node compatibility globals.
func NodeCompat() *NodeCompatPolyfill { return &NodeCompatPolyfill{} }

func (p *NodeCompatPolyfill) Name() string { return "nodecompat" }

func (p *NodeCompatPolyfill) Setup(ctx *quickjs.Context) error {
	return evalJS(ctx, nodeCompatJS)
}

const nodeCompatJS = `
(function() {
  "use strict";

  if (typeof Object.hasOwn !== "function") {
    Object.defineProperty(Object, "hasOwn", {
      configurable: true,
      writable: true,
      value: function hasOwn(object, property) {
        if (object === null || object === undefined) {
          throw new TypeError("Object.hasOwn called on null or undefined");
        }
        return Object.prototype.hasOwnProperty.call(Object(object), property);
      },
    });
  }

  function fail(msg) { throw new Error(msg || "Assertion failed"); }

  function assert(value, msg) {
    if (!value) fail(msg);
  }
  assert.ok = assert;
  assert.strictEqual = function(a, b, msg) {
    if (a !== b) fail(msg || String(a) + " !== " + String(b));
  };
  assert.deepStrictEqual = function(a, b, msg) {
    if (JSON.stringify(a) !== JSON.stringify(b)) fail(msg || "Not deeply equal");
  };
  assert.throws = function(fn, msg) {
    var threw = false;
    try {
      fn();
    } catch (_) {
      threw = true;
    }
    if (!threw) fail(msg || "Expected to throw");
  };
  assert.fail = fail;
  globalThis.assert = assert;

  function querystringParse(str) {
    var out = {};
    String(str || "").split("&").forEach(function(pair) {
      if (!pair) return;
      var eq = pair.indexOf("=");
      var key = eq >= 0 ? pair.slice(0, eq) : pair;
      var value = eq >= 0 ? pair.slice(eq + 1) : "";
      out[decodeURIComponent(key.replace(/\+/g, " "))] = decodeURIComponent(value.replace(/\+/g, " "));
    });
    return out;
  }
  function querystringStringify(obj) {
    return Object.keys(obj || {}).map(function(key) {
      return encodeURIComponent(key) + "=" + encodeURIComponent(obj[key]);
    }).join("&");
  }
  globalThis.querystring = {
    parse: querystringParse,
    stringify: querystringStringify,
    encode: querystringStringify,
    decode: querystringParse,
  };

  class StringDecoder {
    constructor(encoding) {
      this.encoding = encoding || "utf-8";
      this._decoder = new TextDecoder(this.encoding);
    }
    write(buf) {
      return this._decoder.decode(buf instanceof Uint8Array ? buf : new Uint8Array(buf), { stream: true });
    }
    end(buf) {
      if (buf) return this._decoder.decode(buf instanceof Uint8Array ? buf : new Uint8Array(buf));
      return this._decoder.decode();
    }
  }
  globalThis.StringDecoder = StringDecoder;

  var utilTypes = {
    isUint8Array: function(v) { return v instanceof Uint8Array; },
    isArrayBuffer: function(v) { return v instanceof ArrayBuffer; },
    isDate: function(v) { return v instanceof Date || (v !== null && typeof v === "object" && typeof v.getTime === "function" && typeof v.toISOString === "function"); },
    isRegExp: function(v) { return v instanceof RegExp; },
    isMap: function(v) { return v instanceof Map; },
    isSet: function(v) { return v instanceof Set; },
    isTypedArray: function(v) { return ArrayBuffer.isView(v) && !(v instanceof DataView); },
    isPromise: function(v) { return !!v && (typeof v === "object" || typeof v === "function") && typeof v.then === "function"; },
  };
  globalThis.utilTypes = utilTypes;

  function promisify(fn) {
    return function() {
      var args = Array.prototype.slice.call(arguments);
      return new Promise(function(resolve, reject) {
        args.push(function(err, result) {
          if (err) reject(err);
          else resolve(result);
        });
        fn.apply(null, args);
      });
    };
  }

  function inherits(ctor, superCtor) {
    ctor.prototype = Object.create(superCtor.prototype);
    ctor.prototype.constructor = ctor;
  }

  function format(fmt) {
    var args = Array.prototype.slice.call(arguments, 1);
    var i = 0;
    return String(fmt).replace(/%[sdj%]/g, function(token) {
      if (token === "%%") return "%";
      if (i >= args.length) return token;
      var value = args[i++];
      if (token === "%j") {
        try { return JSON.stringify(value); } catch (_) { return "[Circular]"; }
      }
      return String(value);
    });
  }

  globalThis.util = {
    promisify: promisify,
    inherits: inherits,
    deprecate: function(fn) { return fn; },
    types: utilTypes,
    inspect: globalThis.__util_inspect || function(value) {
      try { return JSON.stringify(value); } catch (_) { return String(value); }
    },
    format: format,
    TextEncoder: globalThis.TextEncoder,
    TextDecoder: globalThis.TextDecoder,
  };

  class PerformanceObserver {
    constructor(callback) { this.callback = callback; }
    observe() {}
    disconnect() {}
    takeRecords() { return []; }
  }
  function monitorEventLoopDelay() {
    return {
      enable: function() { return this; },
      disable: function() { return this; },
      percentile: function() { return 0; },
      reset: function() {},
      min: 0,
      max: 0,
      mean: 0,
      stddev: 0,
    };
  }
  globalThis.perf_hooks = {
    performance: globalThis.performance,
    PerformanceObserver: PerformanceObserver,
    monitorEventLoopDelay: monitorEventLoopDelay,
  };

  var noop = function() {};
  function channel() {
    return {
      subscribe: noop,
      unsubscribe: noop,
      publish: noop,
      hasSubscribers: false,
      bindStore: noop,
      runStores: noop,
    };
  }
  class DiagnosticsChannel {
    constructor() { this.hasSubscribers = false; }
    subscribe() {}
    unsubscribe() {}
    publish() {}
    bindStore() {}
    runStores() {}
  }
  globalThis.diagnostics_channel = {
    channel: channel,
    tracingChannel: function() {
      return {
        start: channel(),
        end: channel(),
        asyncStart: channel(),
        asyncEnd: channel(),
        error: channel(),
        subscribe: noop,
        unsubscribe: noop,
        hasSubscribers: false,
      };
    },
    hasSubscribers: function() { return false; },
    subscribe: noop,
    unsubscribe: noop,
    Channel: DiagnosticsChannel,
  };

  class AsyncLocalStorage {
    constructor() { this._store = undefined; }
    getStore() { return this._store; }
    run(store, fn) {
      var prev = this._store;
      this._store = store;
      try {
        var args = Array.prototype.slice.call(arguments, 2);
        return fn.apply(null, args);
      } finally {
        this._store = prev;
      }
    }
    enterWith(store) { this._store = store; }
    disable() { this._store = undefined; }
  }
  class AsyncResource {
    constructor(type) { this.type = type; }
    runInAsyncScope(fn, thisArg) {
      var args = Array.prototype.slice.call(arguments, 2);
      return fn.apply(thisArg, args);
    }
    emitDestroy() { return this; }
    asyncId() { return 0; }
    triggerAsyncId() { return 0; }
  }
  globalThis.async_hooks = {
    createHook: function() { return { enable: function() { return this; }, disable: function() { return this; } }; },
    executionAsyncId: function() { return 0; },
    triggerAsyncId: function() { return 0; },
    executionAsyncResource: function() { return {}; },
    AsyncLocalStorage: AsyncLocalStorage,
    AsyncResource: AsyncResource,
  };

  class Worker {
    constructor() { throw new Error("Worker threads not available in QuickJS"); }
  }
  class MessageChannel {
    constructor() { this.port1 = {}; this.port2 = {}; }
  }
  class MessagePort {}
  globalThis.worker_threads = {
    isMainThread: true,
    parentPort: null,
    workerData: undefined,
    threadId: 0,
    Worker: Worker,
    MessageChannel: MessageChannel,
    MessagePort: MessagePort,
  };

  function unavailable(name) {
    return function() { throw new Error(name + ": not available in QuickJS"); };
  }
  class HTTPAgent {
    constructor() {}
  }
  globalThis.http = {
    createServer: unavailable("http.createServer"),
    request: unavailable("http.request"),
    get: unavailable("http.get"),
    Agent: HTTPAgent,
    globalAgent: new HTTPAgent(),
    METHODS: ["GET", "HEAD", "POST", "PUT", "DELETE", "CONNECT", "OPTIONS", "TRACE", "PATCH"],
    STATUS_CODES: {
      200: "OK",
      201: "Created",
      204: "No Content",
      301: "Moved Permanently",
      302: "Found",
      304: "Not Modified",
      400: "Bad Request",
      401: "Unauthorized",
      403: "Forbidden",
      404: "Not Found",
      500: "Internal Server Error",
    },
  };
  class HTTPSAgent {
    constructor() {}
  }
  globalThis.https = {
    createServer: unavailable("https.createServer"),
    request: unavailable("https.request"),
    get: unavailable("https.get"),
    Agent: HTTPSAgent,
    globalAgent: new HTTPSAgent(),
  };
})();
`
