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
    isAnyArrayBuffer: function(v) {
      return v instanceof ArrayBuffer ||
        (typeof SharedArrayBuffer !== "undefined" && v instanceof SharedArrayBuffer);
    },
    isArrayBufferView: function(v) { return ArrayBuffer.isView(v); },
    isBoxedPrimitive: function(v) {
      if (v === null || typeof v !== "object") return false;
      var tag = Object.prototype.toString.call(v);
      return tag === "[object String]" || tag === "[object Number]" ||
        tag === "[object Boolean]" || tag === "[object BigInt]" ||
        tag === "[object Symbol]";
    },
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
    Object.defineProperty(ctor.prototype, "constructor", {
      value: ctor,
      writable: true,
      configurable: true,
      enumerable: false,
    });
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

  function brainkitUnsupportedBoundary(opts) {
    opts = opts || {};
    var api = opts.api || opts.importPath || "unsupported";
    var boundaryClass = opts.boundaryClass || "unsupported-node-api";
    var err = new Error(
      api + ": unsupported Brainkit boundary (" + boundaryClass + "). " +
      "Owner: " + (opts.suggestedOwner || opts.owner || "jsbridge") + "."
    );
    err.name = opts.name || "BrainkitUnsupportedBoundaryError";
    err.code = opts.code || "BRAINKIT_UNSUPPORTED_BOUNDARY";
    err.importPath = opts.importPath || "";
    err.api = api;
    err.boundaryClass = boundaryClass;
    err.suggestedOwner = opts.suggestedOwner || opts.owner || "jsbridge";
    err.owner = opts.owner || "internal/jsbridge.NodeCompat";
    err.packageName = opts.packageName || "";
    err.packageVersion = opts.packageVersion || "";
    return err;
  }
  if (typeof globalThis.__brainkit_unsupported_boundary !== "function") {
    globalThis.__brainkit_unsupported_boundary = brainkitUnsupportedBoundary;
  }

  var asyncContext = globalThis.__brainkit_async_context;
  if (!asyncContext) {
    (function() {
      var instances = new Set();
      var stores = new Map();

      function register(instance) {
        instances.add(instance);
        return instance;
      }
      function capture() {
        var snapshot = new Map();
        instances.forEach(function(instance) {
          if (stores.has(instance)) snapshot.set(instance, stores.get(instance));
        });
        return snapshot;
      }
      function restore(snapshot) {
        instances.forEach(function(instance) {
          if (snapshot.has(instance)) stores.set(instance, snapshot.get(instance));
          else stores.delete(instance);
        });
      }
      function runSnapshot(snapshot, fn, thisArg, args) {
        if (typeof fn !== "function") return fn;
        var prev = capture();
        restore(snapshot || new Map());
        try {
          return fn.apply(thisArg, args || []);
        } finally {
          restore(prev);
        }
      }
      function bind(fn, snapshot) {
        if (typeof fn !== "function") return fn;
        if (fn.__brainkitAsyncContextBound) return fn;
        var captured = snapshot || capture();
        function bound() {
          return runSnapshot(captured, fn, this, arguments);
        }
        try {
          Object.defineProperty(bound, "__brainkitAsyncContextBound", { value: true });
          Object.defineProperty(bound, "_orig", { value: fn._orig || fn, configurable: true });
        } catch (_) {
          bound.__brainkitAsyncContextBound = true;
          bound._orig = fn._orig || fn;
        }
        return bound;
      }
      function wrapPromise(promise, snapshot) {
        if (!promise || typeof promise.then !== "function" || promise.__brainkitAsyncContextWrapped) return promise;
        var captured = snapshot || capture();
        var originalThen = promise.then;
        var originalCatch = promise.catch;
        var originalFinally = promise.finally;
        try {
          Object.defineProperty(promise, "__brainkitAsyncContextWrapped", { value: true });
          Object.defineProperty(promise, "then", {
            configurable: true,
            value: function(onFulfilled, onRejected) {
              return wrapPromise(originalThen.call(this, bind(onFulfilled, captured), bind(onRejected, captured)), captured);
            },
          });
          if (typeof originalCatch === "function") {
            Object.defineProperty(promise, "catch", {
              configurable: true,
              value: function(onRejected) {
                return wrapPromise(originalCatch.call(this, bind(onRejected, captured)), captured);
              },
            });
          }
          if (typeof originalFinally === "function") {
            Object.defineProperty(promise, "finally", {
              configurable: true,
              value: function(onFinally) {
                return wrapPromise(originalFinally.call(this, bind(onFinally, captured)), captured);
              },
            });
          }
        } catch (_) {
          return promise;
        }
        return promise;
      }

      asyncContext = {
        register: register,
        hasStore: function(instance) { return stores.has(instance); },
        getStore: function(instance) { return stores.get(instance); },
        setStore: function(instance, store) {
          register(instance);
          stores.set(instance, store);
        },
        deleteStore: function(instance) { stores.delete(instance); },
        capture: capture,
        runSnapshot: runSnapshot,
        bind: bind,
        wrapPromise: wrapPromise,
      };
      globalThis.__brainkit_async_context = asyncContext;
    })();
  }

  function patchCallbackTimer(name) {
    var original = globalThis[name];
    if (typeof original !== "function" || original.__brainkitAsyncContextPatched) return;
    function patched(fn) {
      var args = Array.prototype.slice.call(arguments);
      if (typeof args[0] === "function") args[0] = asyncContext.bind(args[0]);
      return original.apply(this, args);
    }
    patched.__brainkitAsyncContextPatched = true;
    patched.__brainkitAsyncContextOriginal = original;
    globalThis[name] = patched;
  }
  patchCallbackTimer("setTimeout");
  patchCallbackTimer("setInterval");
  patchCallbackTimer("setImmediate");
  patchCallbackTimer("queueMicrotask");
  if (globalThis.timersPromises && !globalThis.timersPromises.__brainkitAsyncContextPatched) {
    if (typeof globalThis.timersPromises.setTimeout === "function") {
      var originalTimerPromiseTimeout = globalThis.timersPromises.setTimeout;
      globalThis.timersPromises.setTimeout = function() {
        return asyncContext.wrapPromise(originalTimerPromiseTimeout.apply(this, arguments));
      };
    }
    if (typeof globalThis.timersPromises.setInterval === "function") {
      var originalTimerPromiseInterval = globalThis.timersPromises.setInterval;
      globalThis.timersPromises.setInterval = function() {
        var iterable = originalTimerPromiseInterval.apply(this, arguments);
        if (!iterable || typeof iterable[Symbol.asyncIterator] !== "function") return iterable;
        var originalIterator = iterable[Symbol.asyncIterator];
        iterable[Symbol.asyncIterator] = function() {
          var iterator = originalIterator.apply(this, arguments);
          if (iterator && typeof iterator.next === "function") {
            var originalNext = iterator.next;
            iterator.next = function() {
              return asyncContext.wrapPromise(originalNext.apply(this, arguments));
            };
          }
          return iterator;
        };
        return iterable;
      };
    }
    globalThis.timersPromises.__brainkitAsyncContextPatched = true;
  }

  function patchEventEmitter(Emitter) {
    if (!Emitter || !Emitter.prototype || Emitter.prototype.__brainkitAsyncContextPatched) return;
    function patchRegister(method) {
      var original = Emitter.prototype[method];
      if (typeof original !== "function") return;
      Emitter.prototype[method] = function(ev, fn) {
        if (typeof fn === "function") {
          fn = asyncContext.bind(fn);
        }
        return original.call(this, ev, fn);
      };
    }
    patchRegister("on");
    patchRegister("addListener");
    patchRegister("prependListener");
    Emitter.prototype.__brainkitAsyncContextPatched = true;
  }
  patchEventEmitter(globalThis.EventEmitter);

  var noop = function() {};
  var diagnosticsChannels = new Map();
  class DiagnosticsChannel {
    constructor(name) {
      this.name = String(name || "");
      this._subscribers = [];
    }
    subscribe(fn) {
      if (typeof fn !== "function") throw new TypeError("diagnostics_channel subscriber must be a function");
      if (this._subscribers.indexOf(fn) < 0) this._subscribers.push(fn);
      return this;
    }
    unsubscribe(fn) {
      this._subscribers = this._subscribers.filter(function(existing) { return existing !== fn && existing._orig !== fn; });
      return this;
    }
    publish(message) {
      var list = this._subscribers.slice();
      for (var i = 0; i < list.length; i++) {
        list[i](message, this.name);
      }
    }
    bindStore() { return this; }
    runStores(context, fn, thisArg) {
      if (typeof context === "function") {
        return context.apply(fn, Array.prototype.slice.call(arguments, 2));
      }
      if (typeof fn === "function") {
        return fn.apply(thisArg, Array.prototype.slice.call(arguments, 3));
      }
      return undefined;
    }
  }
  Object.defineProperty(DiagnosticsChannel.prototype, "hasSubscribers", {
    configurable: true,
    get: function() { return this._subscribers.length > 0; },
  });
  function channel(name) {
    var key = String(name || "");
    var existing = diagnosticsChannels.get(key);
    if (existing) return existing;
    var created = new DiagnosticsChannel(key);
    diagnosticsChannels.set(key, created);
    return created;
  }
  function tracingChannel(name) {
    var base = String(name || "");
    var trace = {
      start: channel(base + ":start"),
      end: channel(base + ":end"),
      asyncStart: channel(base + ":asyncStart"),
      asyncEnd: channel(base + ":asyncEnd"),
      error: channel(base + ":error"),
      subscribe: function(subscribers) {
        if (typeof subscribers === "function") {
          this.start.subscribe(subscribers);
          this.end.subscribe(subscribers);
          this.asyncStart.subscribe(subscribers);
          this.asyncEnd.subscribe(subscribers);
          this.error.subscribe(subscribers);
        } else if (subscribers && typeof subscribers === "object") {
          if (subscribers.start) this.start.subscribe(subscribers.start);
          if (subscribers.end) this.end.subscribe(subscribers.end);
          if (subscribers.asyncStart) this.asyncStart.subscribe(subscribers.asyncStart);
          if (subscribers.asyncEnd) this.asyncEnd.subscribe(subscribers.asyncEnd);
          if (subscribers.error) this.error.subscribe(subscribers.error);
        }
        return this;
      },
      unsubscribe: function(subscribers) {
        if (typeof subscribers === "function") {
          this.start.unsubscribe(subscribers);
          this.end.unsubscribe(subscribers);
          this.asyncStart.unsubscribe(subscribers);
          this.asyncEnd.unsubscribe(subscribers);
          this.error.unsubscribe(subscribers);
        } else if (subscribers && typeof subscribers === "object") {
          if (subscribers.start) this.start.unsubscribe(subscribers.start);
          if (subscribers.end) this.end.unsubscribe(subscribers.end);
          if (subscribers.asyncStart) this.asyncStart.unsubscribe(subscribers.asyncStart);
          if (subscribers.asyncEnd) this.asyncEnd.unsubscribe(subscribers.asyncEnd);
          if (subscribers.error) this.error.unsubscribe(subscribers.error);
        }
        return this;
      },
    };
    Object.defineProperty(trace, "hasSubscribers", {
      configurable: true,
      get: function() {
        return trace.start.hasSubscribers || trace.end.hasSubscribers || trace.asyncStart.hasSubscribers || trace.asyncEnd.hasSubscribers || trace.error.hasSubscribers;
      },
    });
    return trace;
  }
  globalThis.diagnostics_channel = {
    channel: channel,
    tracingChannel: tracingChannel,
    hasSubscribers: function(name) { return channel(name).hasSubscribers; },
    subscribe: function(name, fn) { return channel(name).subscribe(fn); },
    unsubscribe: function(name, fn) { return channel(name).unsubscribe(fn); },
    Channel: DiagnosticsChannel,
  };

  class AsyncLocalStorage {
    constructor() { asyncContext.register(this); }
    getStore() { return asyncContext.getStore(this); }
    run(store, fn) {
      asyncContext.register(this);
      var prevHas = asyncContext.hasStore(this);
      var prev = asyncContext.getStore(this);
      asyncContext.setStore(this, store);
      try {
        var args = Array.prototype.slice.call(arguments, 2);
        return fn.apply(null, args);
      } finally {
        if (prevHas) asyncContext.setStore(this, prev);
        else asyncContext.deleteStore(this);
      }
    }
    enterWith(store) { asyncContext.setStore(this, store); }
    disable() { asyncContext.deleteStore(this); }
    static bind(fn) { return asyncContext.bind(fn); }
    static snapshot() {
      var snapshot = asyncContext.capture();
      return function(fn) {
        var args = Array.prototype.slice.call(arguments, 1);
        return asyncContext.runSnapshot(snapshot, fn, this, args);
      };
    }
  }
  class AsyncResource {
    constructor(type) {
      this.type = type;
      this._asyncContextSnapshot = asyncContext.capture();
    }
    runInAsyncScope(fn, thisArg) {
      var args = Array.prototype.slice.call(arguments, 2);
      return asyncContext.runSnapshot(this._asyncContextSnapshot, fn, thisArg, args);
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
    constructor() {
      throw brainkitUnsupportedBoundary({
        api: "worker_threads.Worker",
        importPath: "worker_threads",
        boundaryClass: "worker",
        suggestedOwner: "future worker/sidecar runtime profile",
        owner: "internal/jsbridge.NodeCompat",
      });
    }
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

  var builtinModuleNames = [
    "assert", "assert/strict", "async_hooks", "buffer", "child_process",
    "crypto", "diagnostics_channel", "dns", "dns/promises", "events",
    "fs", "fs/promises", "http", "https", "net", "os", "path",
    "path/posix", "perf_hooks", "process", "querystring", "stream",
    "stream/promises", "stream/web", "string_decoder", "timers",
    "timers/promises", "tls", "url", "util", "util/types",
    "worker_threads", "zlib",
  ];
  function normalizeBuiltinSpecifier(specifier) {
    specifier = String(specifier || "");
    return specifier.indexOf("node:") === 0 ? specifier.slice(5) : specifier;
  }
  function makeBufferModule() {
    var B = globalThis.Buffer;
    if (!B) return undefined;
    function NodeBuffer(arg, encodingOrOffset, length) {
      if (typeof arg === "number") return B.alloc(arg);
      return B.from(arg, encodingOrOffset, length);
    }
    for (var key in B) {
      try { NodeBuffer[key] = B[key]; } catch (_) {}
    }
    NodeBuffer.prototype = Uint8Array.prototype;
    return { Buffer: NodeBuffer, default: { Buffer: NodeBuffer } };
  }
  function getBuiltinModule(specifier) {
    switch (normalizeBuiltinSpecifier(specifier)) {
    case "assert":
    case "assert/strict":
      return globalThis.assert;
    case "async_hooks":
      return globalThis.async_hooks;
    case "buffer":
      return makeBufferModule();
    case "child_process":
      return globalThis.child_process;
    case "crypto":
      return globalThis.crypto;
    case "diagnostics_channel":
      return globalThis.diagnostics_channel;
    case "dns":
      return globalThis.dns;
    case "dns/promises":
      return globalThis.dns && globalThis.dns.promises;
    case "events":
      return { EventEmitter: globalThis.EventEmitter, default: globalThis.EventEmitter };
    case "fs":
      return globalThis.fs;
    case "fs/promises":
      return globalThis.fs && globalThis.fs.promises;
    case "http":
      return globalThis.http;
    case "https":
      return globalThis.https;
    case "module":
      return globalThis.node_module;
    case "net":
      return globalThis.net;
    case "os":
      return globalThis.os;
    case "path":
    case "path/posix":
      return globalThis.path;
    case "perf_hooks":
      return globalThis.perf_hooks;
    case "process":
      return globalThis.process;
    case "querystring":
      return globalThis.querystring;
    case "stream":
      return globalThis.stream;
    case "stream/promises":
      return globalThis.stream && globalThis.stream.promises;
    case "stream/web":
      return {
        ReadableStream: globalThis.ReadableStream,
        WritableStream: globalThis.WritableStream,
        TransformStream: globalThis.TransformStream,
      };
    case "string_decoder":
      return { StringDecoder: globalThis.StringDecoder };
    case "timers":
      return {
        setTimeout: globalThis.setTimeout,
        clearTimeout: globalThis.clearTimeout,
        setInterval: globalThis.setInterval,
        clearInterval: globalThis.clearInterval,
        setImmediate: globalThis.setImmediate,
        clearImmediate: globalThis.clearImmediate,
      };
    case "timers/promises":
      return globalThis.timersPromises;
    case "tls":
      return globalThis.tls;
    case "url":
      return globalThis.node_url;
    case "util":
      return globalThis.util;
    case "util/types":
      return globalThis.utilTypes;
    case "worker_threads":
      return globalThis.worker_threads;
    case "zlib":
      return globalThis.zlib;
    default:
      return undefined;
    }
  }
  function requireUnsupportedBoundary(specifier) {
    throw brainkitUnsupportedBoundary({
      api: "module.createRequire(" + String(specifier) + ")",
      importPath: String(specifier || ""),
      boundaryClass: "dynamic-require",
      suggestedOwner: "future npm resolver package patch or adapter",
      owner: "internal/jsbridge.NodeCompat",
    });
  }
  function createRequire() {
    function require(specifier) {
      var mod = getBuiltinModule(specifier);
      if (mod !== undefined) return mod;
      return requireUnsupportedBoundary(specifier);
    }
    require.resolve = function(specifier) {
      if (globalThis.node_module.isBuiltin(specifier)) return normalizeBuiltinSpecifier(specifier);
      return requireUnsupportedBoundary(specifier);
    };
    require.cache = {};
    require.extensions = {};
    require.main = undefined;
    return require;
  }
  class Module {}
  globalThis.node_module = {
    createRequire: createRequire,
    builtinModules: builtinModuleNames.slice(),
    isBuiltin: function(specifier) {
      var id = normalizeBuiltinSpecifier(specifier);
      return builtinModuleNames.indexOf(id) >= 0 || id === "module";
    },
    syncBuiltinESMExports: function() {},
    Module: Module,
    default: null,
  };
  globalThis.node_module.default = globalThis.node_module;
  if (globalThis.process && typeof globalThis.process.getBuiltinModule !== "function") {
    globalThis.process.getBuiltinModule = getBuiltinModule;
  }

  function unavailable(name, boundaryClass, suggestedOwner, importPath) {
    return function() {
      throw brainkitUnsupportedBoundary({
        api: name,
        importPath: importPath || "",
        boundaryClass: boundaryClass || "unsupported-node-api",
        suggestedOwner: suggestedOwner || "jsbridge",
        owner: "internal/jsbridge.NodeCompat",
      });
    };
  }
  var BaseEmitter = globalThis.EventEmitter || class {
    constructor() { this._events = {}; }
    on(name, fn) {
      (this._events[name] || (this._events[name] = [])).push(fn);
      return this;
    }
    once(name, fn) {
      var self = this;
      function wrapped() {
        self.off(name, wrapped);
        return fn.apply(this, arguments);
      }
      return this.on(name, wrapped);
    }
    off(name, fn) {
      var list = this._events[name] || [];
      this._events[name] = list.filter(function(existing) { return existing !== fn; });
      return this;
    }
    removeListener(name, fn) { return this.off(name, fn); }
    emit(name) {
      var args = Array.prototype.slice.call(arguments, 1);
      var list = (this._events[name] || []).slice();
      for (var i = 0; i < list.length; i++) list[i].apply(this, args);
      return list.length > 0;
    }
  };
  patchEventEmitter(BaseEmitter);
  function normalizeHeaderName(name) {
    return String(name || "").toLowerCase();
  }
  function copyHeaders(headers) {
    var out = {};
    if (!headers) return out;
    if (typeof headers.forEach === "function") {
      headers.forEach(function(value, key) { out[normalizeHeaderName(key)] = String(value); });
      return out;
    }
    for (var key in headers) {
      if (Object.prototype.hasOwnProperty.call(headers, key)) {
        out[normalizeHeaderName(key)] = String(headers[key]);
      }
    }
    return out;
  }
  function appendRawHeaders(headers) {
    var raw = [];
    for (var key in headers) {
      if (Object.prototype.hasOwnProperty.call(headers, key)) {
        raw.push(key, headers[key]);
      }
    }
    return raw;
  }
  function parseRequestArgs(input, options, defaultProtocol) {
    options = options || {};
    var url;
    if (typeof input === "string") {
      url = new URL(input);
    } else if (input && typeof input.href === "string") {
      url = new URL(input.href);
    } else {
      var source = input && typeof input === "object" ? input : {};
      var protocol = options.protocol || source.protocol || defaultProtocol;
      var host = options.hostname || options.host || source.hostname || source.host || "localhost";
      if (options.port || source.port) {
        var hostWithoutPort = String(host).replace(/:\d+$/, "");
        host = hostWithoutPort + ":" + (options.port || source.port);
      }
      var path = options.path || source.path || source.pathname || "/";
      if (source.search && String(path).indexOf("?") < 0) path += source.search;
      url = new URL(protocol + "//" + host + path);
    }
    var method = String(options.method || (input && input.method) || "GET").toUpperCase();
    var headers = copyHeaders(input && input.headers);
    var optionHeaders = copyHeaders(options.headers);
    for (var key in optionHeaders) headers[key] = optionHeaders[key];
    return {
      url: url,
      href: url.href || String(url),
      method: method,
      headers: headers,
    };
  }
  function chunkToBody(chunks) {
    if (!chunks.length) return undefined;
    if (chunks.length === 1) return chunks[0];
    if (typeof Buffer !== "undefined") {
      return Buffer.concat(chunks.map(function(chunk) {
        return Buffer.isBuffer(chunk) ? chunk : Buffer.from(String(chunk));
      }));
    }
    return chunks.map(function(chunk) { return String(chunk); }).join("");
  }
  function normalizeChunk(chunk) {
    if (chunk === undefined || chunk === null) return "";
    if (typeof Buffer !== "undefined" && Buffer.isBuffer(chunk)) return chunk;
    if (chunk instanceof Uint8Array) {
      return typeof Buffer !== "undefined" ? Buffer.from(chunk) : chunk;
    }
    return String(chunk);
  }
  class IncomingMessage extends BaseEmitter {
    constructor(response, url) {
      super();
      this.statusCode = response.status;
      this.statusMessage = response.statusText || "";
      this.headers = copyHeaders(response.headers);
      this.rawHeaders = appendRawHeaders(this.headers);
      this.url = url;
      this.method = undefined;
      this.complete = false;
      this.readable = true;
      this._response = response;
      this._encoding = null;
    }
    setEncoding(encoding) {
      this._encoding = encoding || "utf8";
      return this;
    }
    resume() {
      if (!this._started) this._start();
      return this;
    }
    pipe(dest) {
      this.on("data", function(chunk) {
        if (dest && typeof dest.write === "function") dest.write(chunk);
      });
      this.on("end", function() {
        if (dest && typeof dest.end === "function") dest.end();
      });
      this.resume();
      return dest;
    }
    _emitChunk(chunk) {
      if (this._encoding && chunk && typeof chunk.toString === "function") {
        this.emit("data", chunk.toString(this._encoding));
      } else {
        this.emit("data", chunk);
      }
    }
    async _start() {
      if (this._started) return;
      this._started = true;
      try {
        var body = this._response.body;
        if (body && typeof body.getReader === "function") {
          var reader = body.getReader();
          for (;;) {
            var item = await reader.read();
            if (item.done) break;
            this._emitChunk(normalizeChunk(item.value));
          }
        } else if (typeof this._response.arrayBuffer === "function") {
          var data = await this._response.arrayBuffer();
          if (data && data.byteLength > 0) this._emitChunk(normalizeChunk(new Uint8Array(data)));
        }
        this.complete = true;
        this.emit("end");
        this.emit("close");
      } catch (err) {
        this.emit("error", err);
        this.emit("close");
      }
    }
  }
  class ClientRequest extends BaseEmitter {
    constructor(state) {
      super();
      this.method = state.method;
      this.path = state.url.pathname + state.url.search;
      this.host = state.url.host;
      this.protocol = state.url.protocol;
      this.aborted = false;
      this.destroyed = false;
      this.writableEnded = false;
      this._url = state.href;
      this._headers = state.headers;
      this._chunks = [];
      this._controller = typeof AbortController !== "undefined" ? new AbortController() : null;
    }
    setHeader(name, value) {
      this._headers[normalizeHeaderName(name)] = String(value);
      return this;
    }
    getHeader(name) {
      return this._headers[normalizeHeaderName(name)];
    }
    removeHeader(name) {
      delete this._headers[normalizeHeaderName(name)];
      return this;
    }
    write(chunk, _encoding, cb) {
      if (this.writableEnded) throw new Error("write after end");
      this._chunks.push(normalizeChunk(chunk));
      if (typeof cb === "function") cb();
      return true;
    }
    end(chunk, encoding, cb) {
      if (typeof encoding === "function") { cb = encoding; encoding = undefined; }
      if (chunk !== undefined && chunk !== null) this.write(chunk, encoding);
      this.writableEnded = true;
      if (typeof cb === "function") cb();
      this._start();
      return this;
    }
    abort() {
      this.aborted = true;
      if (this._controller) this._controller.abort();
      this.emit("abort");
      this.emit("close");
    }
    destroy(err) {
      this.destroyed = true;
      if (this._controller) this._controller.abort();
      if (err) this.emit("error", err);
      this.emit("close");
      return this;
    }
    setTimeout(ms, cb) {
      if (typeof setTimeout === "function") {
        var self = this;
        setTimeout(function() {
          if (typeof cb === "function") cb();
          self.emit("timeout");
        }, ms);
      }
      return this;
    }
    setNoDelay() { return this; }
    setSocketKeepAlive() { return this; }
    async _start() {
      if (this._started) return;
      this._started = true;
      try {
        if (typeof fetch !== "function") throw new Error("http.request: fetch is not available in QuickJS");
        var init = { method: this.method, headers: this._headers };
        var body = chunkToBody(this._chunks);
        if (body !== undefined && this.method !== "GET" && this.method !== "HEAD") init.body = body;
        if (this._controller) init.signal = this._controller.signal;
        var response = await fetch(this._url, init);
        var message = new IncomingMessage(response, this._url);
        this.emit("response", message);
        Promise.resolve().then(function() { message._start(); });
      } catch (err) {
        this.emit("error", err);
        this.emit("close");
      }
    }
  }
  function makeRequest(defaultProtocol) {
    return function request(input, options, cb) {
      if (typeof options === "function") { cb = options; options = {}; }
      var state = parseRequestArgs(input, options || {}, defaultProtocol);
      var req = new ClientRequest(state);
      if (typeof cb === "function") req.on("response", cb);
      return req;
    };
  }
  function makeGet(request) {
    return function get(input, options, cb) {
      var req = request(input, options, cb);
      req.end();
      return req;
    };
  }
  class HTTPAgent {
    constructor(options) { this.options = options || {}; }
  }
  var httpRequest = makeRequest("http:");
  globalThis.http = {
    createServer: unavailable("http.createServer", "server-listener", "modules/gateway or future server-listener runtime profile", "http"),
    request: httpRequest,
    get: makeGet(httpRequest),
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
    constructor(options) { this.options = options || {}; }
  }
  var httpsRequest = makeRequest("https:");
  globalThis.https = {
    createServer: unavailable("https.createServer", "server-listener", "modules/gateway or future server-listener runtime profile", "https"),
    request: httpsRequest,
    get: makeGet(httpsRequest),
    Agent: HTTPSAgent,
    globalAgent: new HTTPSAgent(),
  };
})();
`
