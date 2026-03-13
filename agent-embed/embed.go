package agentembed

import (
	_ "embed"
	"fmt"

	quickjs "github.com/buke/quickjs-go"

	"github.com/brainlet/brainkit/jsbridge"
)

//go:embed agent_embed_bundle.js
var bundleSource string

// LoadBundle evaluates the agent-embed bundle in the given bridge.
// After loading, globalThis.__agent_embed is available with Agent, createTool, and Mastra.
func LoadBundle(b *jsbridge.Bridge) error {
	// The Mastra bundle and AI SDK providers reference Node.js and browser globals
	// that don't exist in QuickJS. The esbuild node-stub plugin handles import-time
	// resolution, but runtime code still accesses these globals directly.
	// Set up everything before loading the bundle.
	setup, err := b.Eval("agent-embed-setup.js", runtimeGlobalsJS)
	if err != nil {
		return fmt.Errorf("agent-embed: setup globals: %w", err)
	}
	setup.Free()

	val, err := b.Eval("agent-embed-bundle.js", bundleSource, quickjs.EvalAwait(true))
	if err != nil {
		return fmt.Errorf("agent-embed: load bundle: %w", err)
	}
	val.Free()
	return nil
}

const runtimeGlobalsJS = `
// ─── process ────────────────────────────────────────────────────────────
// Node.js process global. The jsbridge Process() polyfill provides a Proxy-based
// process.env backed by real Go os.Getenv/Setenv; here we ensure the rest of the
// API surface exists so that code like process.version.substring(0) doesn't crash.
(function() {
  var p = globalThis.process || {};

  // env: if jsbridge Process() polyfill was loaded, this is already a Proxy.
  // Otherwise provide a plain object that can be mutated by Generate().
  if (!p.env) p.env = { NODE_ENV: "production" };

  // version / versions — AI SDK reads process.version.substring(0)
  if (!p.version) p.version = "v20.0.0";
  if (!p.versions) p.versions = { node: "20.0.0" };

  // Core fields
  if (!p.platform) p.platform = "linux";
  if (!p.arch) p.arch = "x64";
  if (p.pid === undefined) p.pid = 1;
  if (!p.argv) p.argv = [];
  if (!p.argv0) p.argv0 = "";
  if (!p.execArgv) p.execArgv = [];
  if (!p.execPath) p.execPath = "/usr/local/bin/node";
  if (!p.title) p.title = "node";
  if (!p.release) p.release = { name: "node" };
  if (!p.config) p.config = {};

  // Functions
  if (!p.cwd) p.cwd = function() { return "/"; };
  if (!p.chdir) p.chdir = function() {};
  if (!p.exit) p.exit = function() {};
  if (!p.kill) p.kill = function() {};
  if (!p.abort) p.abort = function() {};
  if (!p.umask) p.umask = function() { return 0o22; };
  if (!p.uptime) p.uptime = function() { return 0; };
  if (!p.cpuUsage) p.cpuUsage = function() { return { user: 0, system: 0 }; };
  if (!p.memoryUsage) p.memoryUsage = function() {
    return { rss: 0, heapTotal: 0, heapUsed: 0, external: 0, arrayBuffers: 0 };
  };
  if (!p.hrtime) {
    p.hrtime = function(prev) {
      var now = Date.now();
      var s = Math.floor(now / 1000);
      var ns = (now % 1000) * 1e6;
      if (prev) {
        s -= prev[0];
        ns -= prev[1];
        if (ns < 0) { s--; ns += 1e9; }
      }
      return [s, ns];
    };
    p.hrtime.bigint = function() { return BigInt(Date.now()) * BigInt(1e6); };
  }

  // nextTick — critical for Mastra's async tool execution pipeline
  if (!p.nextTick) p.nextTick = function(fn) {
    var args = [];
    for (var i = 1; i < arguments.length; i++) args.push(arguments[i]);
    queueMicrotask(function() { fn.apply(null, args); });
  };

  // Streams
  if (!p.stdout) p.stdout = {
    write: function() { return true; },
    isTTY: false,
    columns: 80,
    rows: 24,
    on: function() { return this; },
    once: function() { return this; },
    emit: function() { return false; },
  };
  if (!p.stderr) p.stderr = {
    write: function() { return true; },
    isTTY: false,
    columns: 80,
    rows: 24,
    on: function() { return this; },
    once: function() { return this; },
    emit: function() { return false; },
  };
  if (!p.stdin) p.stdin = {
    isTTY: false,
    on: function() { return this; },
    once: function() { return this; },
    resume: function() { return this; },
    pause: function() { return this; },
    read: function() { return null; },
  };

  // Event-related
  if (!p.on) p.on = function() { return p; };
  if (!p.once) p.once = function() { return p; };
  if (!p.off) p.off = function() { return p; };
  if (!p.emit) p.emit = function() { return false; };
  if (!p.removeListener) p.removeListener = function() { return p; };
  if (!p.removeAllListeners) p.removeAllListeners = function() { return p; };
  if (!p.listeners) p.listeners = function() { return []; };
  if (!p.listenerCount) p.listenerCount = function() { return 0; };
  if (!p.addListener) p.addListener = p.on;
  if (!p.prependListener) p.prependListener = p.on;
  if (!p.prependOnceListener) p.prependOnceListener = p.once;

  // Feature detection
  if (!p.features) p.features = {};
  if (!p.allowedNodeEnvironmentFlags) p.allowedNodeEnvironmentFlags = new Set();

  globalThis.process = p;
})();

// ─── navigator ──────────────────────────────────────────────────────────
// Various libraries check navigator for environment detection.
if (typeof navigator === "undefined") {
  globalThis.navigator = {
    userAgent: "Mozilla/5.0 (compatible; QuickJS/0.1; Go)",
    language: "en-US",
    languages: ["en-US", "en"],
    platform: "Linux x86_64",
    hardwareConcurrency: 1,
    onLine: true,
    cookieEnabled: false,
    maxTouchPoints: 0,
    mediaDevices: {},
    permissions: {},
    clipboard: {},
    locks: { request: function() { return Promise.resolve(); } },
  };
}

// ─── performance ────────────────────────────────────────────────────────
// AI SDK and telemetry code use performance.now() and performance.timeOrigin.
if (typeof performance === "undefined") {
  var _perfStart = Date.now();
  globalThis.performance = {
    now: function() { return Date.now() - _perfStart; },
    timeOrigin: _perfStart,
    mark: function() {},
    measure: function() {},
    getEntriesByName: function() { return []; },
    getEntriesByType: function() { return []; },
    clearMarks: function() {},
    clearMeasures: function() {},
  };
}

// ─── Buffer ─────────────────────────────────────────────────────────────
// Node.js Buffer — used by crypto stubs, HTTP parsing, and various SDK internals.
if (typeof Buffer === "undefined") {
  var _Buffer = {
    from: function(v, enc) {
      if (v instanceof Uint8Array || v instanceof ArrayBuffer) {
        return new Uint8Array(v);
      }
      if (typeof v === "string") {
        if (enc === "base64") {
          var bin = atob(v);
          var arr = new Uint8Array(bin.length);
          for (var i = 0; i < bin.length; i++) arr[i] = bin.charCodeAt(i);
          return arr;
        }
        if (enc === "hex") {
          var arr = new Uint8Array(v.length / 2);
          for (var i = 0; i < v.length; i += 2) arr[i / 2] = parseInt(v.substr(i, 2), 16);
          return arr;
        }
        return new TextEncoder().encode(v);
      }
      if (Array.isArray(v)) {
        return new Uint8Array(v);
      }
      return new Uint8Array(0);
    },
    alloc: function(n, fill) {
      var b = new Uint8Array(n);
      if (fill !== undefined) b.fill(typeof fill === "number" ? fill : 0);
      return b;
    },
    allocUnsafe: function(n) { return new Uint8Array(n); },
    allocUnsafeSlow: function(n) { return new Uint8Array(n); },
    isBuffer: function(obj) { return false; },
    isEncoding: function(enc) {
      return ["utf8", "utf-8", "ascii", "latin1", "binary", "hex", "base64", "ucs2", "ucs-2", "utf16le", "utf-16le"]
        .indexOf((enc || "").toLowerCase()) !== -1;
    },
    byteLength: function(str, enc) {
      if (typeof str === "string") return new TextEncoder().encode(str).length;
      if (str instanceof Uint8Array || str instanceof ArrayBuffer) return str.byteLength || str.length;
      return 0;
    },
    concat: function(bufs, totalLength) {
      if (!totalLength) {
        totalLength = 0;
        for (var i = 0; i < bufs.length; i++) totalLength += bufs[i].length;
      }
      var r = new Uint8Array(totalLength);
      var off = 0;
      for (var i = 0; i < bufs.length; i++) {
        r.set(bufs[i], off);
        off += bufs[i].length;
      }
      return r;
    },
    compare: function(a, b) {
      var len = Math.min(a.length, b.length);
      for (var i = 0; i < len; i++) {
        if (a[i] < b[i]) return -1;
        if (a[i] > b[i]) return 1;
      }
      return a.length < b.length ? -1 : a.length > b.length ? 1 : 0;
    },
  };
  globalThis.Buffer = _Buffer;
}

// ─── Scheduling ─────────────────────────────────────────────────────────
// queueMicrotask, setImmediate, clearImmediate — used by Mastra's async
// tool execution pipeline and various SDK internals.
if (typeof queueMicrotask === "undefined") {
  globalThis.queueMicrotask = function(fn) { Promise.resolve().then(fn); };
}
if (typeof setImmediate === "undefined") {
  globalThis.setImmediate = function(fn) {
    var args = [];
    for (var i = 1; i < arguments.length; i++) args.push(arguments[i]);
    Promise.resolve().then(function() { fn.apply(null, args); });
    return 0;
  };
}
if (typeof clearImmediate === "undefined") {
  globalThis.clearImmediate = function() {};
}
if (typeof setInterval === "undefined") {
  // Mastra doesn't use setInterval, but some SDK code may reference it.
  globalThis.setInterval = function() { return 0; };
  globalThis.clearInterval = function() {};
}

// ─── Event utilities ────────────────────────────────────────────────────
// EventTarget — used by AbortSignal inheritance in some SDK code.
if (typeof EventTarget === "undefined") {
  globalThis.EventTarget = class EventTarget {
    constructor() { this._listeners = {}; }
    addEventListener(type, fn) {
      (this._listeners[type] = this._listeners[type] || []).push(fn);
    }
    removeEventListener(type, fn) {
      var a = this._listeners[type];
      if (a) this._listeners[type] = a.filter(function(f) { return f !== fn; });
    }
    dispatchEvent(event) {
      var a = this._listeners[event.type];
      if (a) a.forEach(function(fn) { fn(event); });
      return true;
    }
  };
}

// Event constructor
if (typeof Event === "undefined") {
  globalThis.Event = class Event {
    constructor(type, opts) {
      this.type = type;
      this.bubbles = !!(opts && opts.bubbles);
      this.cancelable = !!(opts && opts.cancelable);
      this.defaultPrevented = false;
      this.target = null;
      this.currentTarget = null;
      this.timeStamp = Date.now();
    }
    preventDefault() { this.defaultPrevented = true; }
    stopPropagation() {}
    stopImmediatePropagation() {}
  };
}

// CustomEvent
if (typeof CustomEvent === "undefined") {
  globalThis.CustomEvent = class CustomEvent extends Event {
    constructor(type, opts) {
      super(type, opts);
      this.detail = opts && opts.detail !== undefined ? opts.detail : null;
    }
  };
}

// ─── Web APIs ───────────────────────────────────────────────────────────
// Headers — some SDK code constructs Headers directly.
// The jsbridge Fetch polyfill already provides a full Headers implementation,
// but if it hasn't been loaded yet or if code runs before fetch init, we need a fallback.
if (typeof Headers === "undefined") {
  globalThis.Headers = class Headers {
    constructor(init) {
      this._map = {};
      if (init instanceof Headers) {
        init.forEach(function(v, k) { this.set(k, v); }.bind(this));
      } else if (init && typeof init === "object") {
        Object.keys(init).forEach(function(k) { this.set(k, init[k]); }.bind(this));
      }
    }
    get(name) { var v = this._map[name.toLowerCase()]; return v !== undefined ? v : null; }
    set(name, value) { this._map[name.toLowerCase()] = String(value); }
    has(name) { return name.toLowerCase() in this._map; }
    delete(name) { delete this._map[name.toLowerCase()]; }
    append(name, value) {
      var k = name.toLowerCase();
      this._map[k] = this._map[k] ? this._map[k] + ", " + value : String(value);
    }
    forEach(fn) {
      var self = this;
      Object.keys(this._map).forEach(function(k) { fn(self._map[k], k, self); });
    }
    entries() { var self = this; return Object.keys(this._map).map(function(k) { return [k, self._map[k]]; })[Symbol.iterator](); }
    keys() { return Object.keys(this._map)[Symbol.iterator](); }
    values() { var self = this; return Object.keys(this._map).map(function(k) { return self._map[k]; })[Symbol.iterator](); }
    [Symbol.iterator]() { return this.entries(); }
  };
}

// Response.json() static method — some SDK providers use Response.json()
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

"ok";
`
