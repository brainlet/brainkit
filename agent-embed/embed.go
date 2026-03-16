package agentembed

import (
	_ "embed"
	"fmt"

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

	val, err := b.EvalAsync("agent-embed-bundle.js", bundleSource)
	if err != nil {
		return fmt.Errorf("agent-embed: load bundle: %w", err)
	}
	val.Free()
	return nil
}

const runtimeGlobalsJS = `
// ─── global ─────────────────────────────────────────────────────────────
// Node.js "global" — alias for globalThis. Required by the pg npm package.
if (typeof global === "undefined") {
  globalThis.global = globalThis;
}

// ─── require shim ───────────────────────────────────────────────────────
// The AI SDK bundle has dynamic require("@opentelemetry/api") that can't
// be resolved at build time. Provide a no-op shim with OpenTelemetry stubs.
if (typeof require === "undefined") {
  var _noopSpan = {
    setAttribute: function() { return this; },
    setAttributes: function() { return this; },
    addEvent: function() { return this; },
    setStatus: function() { return this; },
    end: function() {},
    isRecording: function() { return false; },
    recordException: function() {},
    updateName: function() { return this; },
    spanContext: function() { return { traceId: "", spanId: "", traceFlags: 0 }; },
  };
  var _noopTracer = {
    startSpan: function() { return _noopSpan; },
    startActiveSpan: function(name, optionsOrFn, fnOrUndef) {
      var fn = typeof optionsOrFn === "function" ? optionsOrFn : fnOrUndef;
      if (typeof fn === "function") return fn(_noopSpan);
      return _noopSpan;
    },
  };
  var _otelStub = {
    trace: {
      getTracer: function() { return _noopTracer; },
      setSpan: function(ctx) { return ctx; },
      getSpan: function() { return _noopSpan; },
      getActiveSpan: function() { return undefined; },
    },
    context: {
      active: function() { return {}; },
      with: function(ctx, fn) { return fn(); },
      bind: function(ctx, fn) { return fn; },
    },
    SpanStatusCode: { UNSET: 0, OK: 1, ERROR: 2 },
    SpanKind: { INTERNAL: 0, SERVER: 1, CLIENT: 2 },
    diag: { debug: function() {}, info: function() {}, warn: function() {}, error: function() {} },
    propagation: {},
    metrics: { getMeter: function() { return {}; } },
  };
  // IMPORTANT: This require is captured by esbuild's internal _I at bundle start.
  // Any dynamic require("zod/v4") from the AI SDK goes through this function.
  // The AI SDK's toJSONSchema resolver caches on first call, so we use a Proxy
  // that defers to globalThis.__zod_v4_module (set by entry.mjs) on property access.
  // The AI SDK resolves toJSONSchema via dynamic require("zod/v4").
  // It caches the result on first call. We return a wrapper that defers
  // to entry.mjs's real zodV4 module. The toJSONSchema function itself
  // is a deferred thunk that calls the real one when invoked.
  var _zodV4Wrapper = {
    toJSONSchema: function() {
      var real = globalThis.__zod_v4_module;
      if (real && typeof real.toJSONSchema === "function") {
        return real.toJSONSchema.apply(real, arguments);
      }
      throw new Error("toJSONSchema not yet available");
    },
  };
  globalThis.require = function(mod) {
    if (mod === "@opentelemetry/api") return _otelStub;
    if (mod === "zod/v4" || mod === "zod") {
      return globalThis.__zod_v4_module || _zodV4Wrapper;
    }
    // LSP dependencies — pre-loaded in entry.mjs
    if (mod === "vscode-jsonrpc/node" || mod === "vscode-jsonrpc") {
      return globalThis.__vscode_jsonrpc_node || {};
    }
    if (mod === "vscode-languageserver-protocol") {
      return globalThis.__vscode_lsp_protocol || {};
    }
    // execa polyfill — backed by Go spawn bridge
    if (mod === "execa") {
      return { execa: globalThis.__execa_polyfill || function() { throw new Error("execa not available"); } };
    }
    return {};
  };
}

// ─── Error.captureStackTrace ────────────────────────────────────────────
// V8-specific API used by pg-pool and other Node.js libraries.
// QuickJS doesn't have it — provide a no-op shim.
if (!Error.captureStackTrace) {
  Error.captureStackTrace = function(err, constructorOpt) {
    if (err && !err.stack) {
      err.stack = new Error().stack || "";
    }
  };
}

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
// Node.js Buffer — pg-protocol needs .write(), .copy(), .writeInt32BE(), .slice(), .toString(), etc.
// We extend Uint8Array with Buffer methods so objects work as both typed arrays and Buffers.
if (typeof Buffer === "undefined") {
  (function() {
    var _te = new TextEncoder();
    var _td = new TextDecoder();

    function _addBufferMethods(buf) {
      if (buf._isBuffer) return buf;
      buf._isBuffer = true;

      buf.write = function(string, offset, lengthOrEnc, encoding) {
        if (typeof offset === 'string') { encoding = offset; offset = 0; }
        if (typeof lengthOrEnc === 'string') { encoding = lengthOrEnc; lengthOrEnc = undefined; }
        offset = offset || 0;
        var bytes = _te.encode(string);
        var len = lengthOrEnc !== undefined ? Math.min(bytes.length, lengthOrEnc) : bytes.length;
        for (var i = 0; i < len && (offset + i) < buf.length; i++) buf[offset + i] = bytes[i];
        return len;
      };

      buf.copy = function(target, targetStart, sourceStart, sourceEnd) {
        targetStart = targetStart || 0;
        sourceStart = sourceStart || 0;
        sourceEnd = sourceEnd || buf.length;
        for (var i = sourceStart; i < sourceEnd && (targetStart + i - sourceStart) < target.length; i++) {
          target[targetStart + i - sourceStart] = buf[i];
        }
        return sourceEnd - sourceStart;
      };

      buf.writeInt32BE = function(value, offset) {
        offset = offset || 0;
        buf[offset]     = (value >>> 24) & 0xff;
        buf[offset + 1] = (value >>> 16) & 0xff;
        buf[offset + 2] = (value >>> 8) & 0xff;
        buf[offset + 3] = value & 0xff;
        return offset + 4;
      };

      buf.writeUInt32BE = buf.writeInt32BE;

      buf.writeInt16BE = function(value, offset) {
        offset = offset || 0;
        buf[offset]     = (value >>> 8) & 0xff;
        buf[offset + 1] = value & 0xff;
        return offset + 2;
      };

      buf.writeUInt16BE = buf.writeInt16BE;

      buf.readInt32BE = function(offset) {
        offset = offset || 0;
        return (buf[offset] << 24) | (buf[offset+1] << 16) | (buf[offset+2] << 8) | buf[offset+3];
      };

      buf.readUInt32BE = function(offset) {
        offset = offset || 0;
        return ((buf[offset] << 24) | (buf[offset+1] << 16) | (buf[offset+2] << 8) | buf[offset+3]) >>> 0;
      };

      buf.readInt16BE = function(offset) {
        offset = offset || 0;
        var val = (buf[offset] << 8) | buf[offset+1];
        return val > 0x7FFF ? val - 0x10000 : val;
      };

      buf.readUInt16BE = function(offset) {
        offset = offset || 0;
        return (buf[offset] << 8) | buf[offset+1];
      };

      // Little-endian methods (used by MongoDB/BSON)
      buf.readInt32LE = function(offset) {
        offset = offset || 0;
        return buf[offset] | (buf[offset+1] << 8) | (buf[offset+2] << 16) | (buf[offset+3] << 24);
      };

      buf.readUInt32LE = function(offset) {
        offset = offset || 0;
        return (buf[offset] | (buf[offset+1] << 8) | (buf[offset+2] << 16) | (buf[offset+3] << 24)) >>> 0;
      };

      buf.readInt16LE = function(offset) {
        offset = offset || 0;
        var val = buf[offset] | (buf[offset+1] << 8);
        return val > 0x7FFF ? val - 0x10000 : val;
      };

      buf.readUInt16LE = function(offset) {
        offset = offset || 0;
        return buf[offset] | (buf[offset+1] << 8);
      };

      buf.writeInt32LE = function(value, offset) {
        offset = offset || 0;
        buf[offset] = value & 0xff;
        buf[offset + 1] = (value >>> 8) & 0xff;
        buf[offset + 2] = (value >>> 16) & 0xff;
        buf[offset + 3] = (value >>> 24) & 0xff;
        return offset + 4;
      };

      buf.writeUInt32LE = buf.writeInt32LE;

      buf.writeInt16LE = function(value, offset) {
        offset = offset || 0;
        buf[offset] = value & 0xff;
        buf[offset + 1] = (value >>> 8) & 0xff;
        return offset + 2;
      };

      buf.writeUInt16LE = buf.writeInt16LE;

      buf.readFloatLE = function(offset) {
        offset = offset || 0;
        var tmp = new Uint8Array(4);
        for (var i = 0; i < 4; i++) tmp[i] = buf[offset + i];
        return new DataView(tmp.buffer).getFloat32(0, true);
      };

      buf.readDoubleLE = function(offset) {
        offset = offset || 0;
        var tmp = new Uint8Array(8);
        for (var i = 0; i < 8; i++) tmp[i] = buf[offset + i];
        return new DataView(tmp.buffer).getFloat64(0, true);
      };

      buf.writeFloatLE = function(value, offset) {
        offset = offset || 0;
        var tmp = new Uint8Array(4);
        new DataView(tmp.buffer).setFloat32(0, value, true);
        for (var i = 0; i < 4; i++) buf[offset + i] = tmp[i];
        return offset + 4;
      };

      buf.writeDoubleLE = function(value, offset) {
        offset = offset || 0;
        var tmp = new Uint8Array(8);
        new DataView(tmp.buffer).setFloat64(0, value, true);
        for (var i = 0; i < 8; i++) buf[offset + i] = tmp[i];
        return offset + 8;
      };

      buf.readUInt8 = function(offset) { return buf[offset || 0]; };
      buf.writeUInt8 = function(value, offset) { buf[offset || 0] = value & 0xff; return (offset || 0) + 1; };

      var origSlice = buf.slice.bind(buf);
      buf.slice = function(start, end) {
        return _addBufferMethods(origSlice(start, end));
      };

      buf.subarray = function(start, end) {
        return _addBufferMethods(Uint8Array.prototype.subarray.call(buf, start, end));
      };

      buf.toString = function(encoding, start, end) {
        start = start || 0;
        end = end !== undefined ? end : buf.length;
        var sub = buf.subarray(start, end);
        encoding = (encoding || 'utf8').toLowerCase();
        if (encoding === 'utf8' || encoding === 'utf-8') {
          return _td.decode(sub);
        }
        if (encoding === 'hex') {
          var hex = '';
          for (var i = 0; i < sub.length; i++) hex += (sub[i] < 16 ? '0' : '') + sub[i].toString(16);
          return hex;
        }
        if (encoding === 'base64') {
          // Pure-JS base64 — btoa goes through Go which truncates at null bytes
          var _c = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
          var b64 = '';
          for (var i = 0; i < sub.length; i += 3) {
            var a0 = sub[i], a1 = i+1 < sub.length ? sub[i+1] : 0, a2 = i+2 < sub.length ? sub[i+2] : 0;
            b64 += _c[(a0>>2)&0x3f] + _c[((a0<<4)|(a1>>4))&0x3f];
            b64 += (i+1 < sub.length) ? _c[((a1<<2)|(a2>>6))&0x3f] : '=';
            b64 += (i+2 < sub.length) ? _c[a2&0x3f] : '=';
          }
          return b64;
        }
        if (encoding === 'ascii' || encoding === 'latin1' || encoding === 'binary') {
          var str = '';
          for (var i = 0; i < sub.length; i++) str += String.fromCharCode(sub[i]);
          return str;
        }
        return _td.decode(sub);
      };

      buf.toJSON = function() {
        return { type: 'Buffer', data: Array.from(buf) };
      };

      buf.equals = function(other) {
        if (buf.length !== other.length) return false;
        for (var i = 0; i < buf.length; i++) if (buf[i] !== other[i]) return false;
        return true;
      };

      buf.compare = function(other) {
        var len = Math.min(buf.length, other.length);
        for (var i = 0; i < len; i++) {
          if (buf[i] < other[i]) return -1;
          if (buf[i] > other[i]) return 1;
        }
        return buf.length < other.length ? -1 : buf.length > other.length ? 1 : 0;
      };

      buf.fill = function(value, start, end) {
        start = start || 0;
        end = end || buf.length;
        var fillVal = typeof value === 'number' ? value : 0;
        Uint8Array.prototype.fill.call(buf, fillVal, start, end);
        return buf;
      };

      buf.indexOf = function(val, byteOffset) {
        byteOffset = byteOffset || 0;
        if (typeof val === 'number') {
          for (var i = byteOffset; i < buf.length; i++) if (buf[i] === val) return i;
          return -1;
        }
        return -1;
      };

      buf.map = function(fn) {
        return _addBufferMethods(Uint8Array.prototype.map.call(buf, fn));
      };

      return buf;
    }

    var _Buffer = {
      from: function(v, encOrOffset, length) {
        if (v instanceof ArrayBuffer) {
          var offset = encOrOffset || 0;
          var len = length !== undefined ? length : v.byteLength - offset;
          return _addBufferMethods(new Uint8Array(v, offset, len));
        }
        if (v instanceof Uint8Array || ArrayBuffer.isView(v)) {
          if (typeof encOrOffset === 'number') {
            return _addBufferMethods(new Uint8Array(v.buffer, (v.byteOffset || 0) + encOrOffset, length));
          }
          return _addBufferMethods(new Uint8Array(v));
        }
        if (typeof v === 'string') {
          var enc = encOrOffset;
          if (enc === 'base64') {
            // Pure-JS base64 decode — atob goes through Go which truncates at null bytes
            var _c = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/';
            var _lk = {};
            for (var ci = 0; ci < _c.length; ci++) _lk[_c[ci]] = ci;
            var bLen = Math.floor(v.length * 3 / 4);
            if (v.length > 1 && v[v.length-1] === '=') bLen--;
            if (v.length > 2 && v[v.length-2] === '=') bLen--;
            var arr = new Uint8Array(bLen);
            var p = 0;
            for (var ci = 0; ci < v.length; ci += 4) {
              var a0 = _lk[v[ci]] || 0, b0 = _lk[v[ci+1]] || 0, c0 = _lk[v[ci+2]] || 0, d0 = _lk[v[ci+3]] || 0;
              arr[p++] = (a0 << 2) | (b0 >> 4);
              if (v[ci+2] !== '=') arr[p++] = ((b0 << 4) | (c0 >> 2)) & 0xff;
              if (v[ci+3] !== '=') arr[p++] = ((c0 << 6) | d0) & 0xff;
            }
            return _addBufferMethods(arr);
          }
          if (enc === 'hex') {
            var arr = new Uint8Array(v.length / 2);
            for (var i = 0; i < v.length; i += 2) arr[i / 2] = parseInt(v.substr(i, 2), 16);
            return _addBufferMethods(arr);
          }
          return _addBufferMethods(_te.encode(v));
        }
        if (Array.isArray(v)) {
          return _addBufferMethods(new Uint8Array(v));
        }
        if (typeof v === 'number') {
          return _addBufferMethods(new Uint8Array(v));
        }
        return _addBufferMethods(new Uint8Array(0));
      },
      alloc: function(n, fill) {
        var b = new Uint8Array(n);
        if (fill !== undefined) b.fill(typeof fill === 'number' ? fill : 0);
        return _addBufferMethods(b);
      },
      allocUnsafe: function(n) { return _addBufferMethods(new Uint8Array(n)); },
      allocUnsafeSlow: function(n) { return _addBufferMethods(new Uint8Array(n)); },
      isBuffer: function(obj) { return !!(obj && obj._isBuffer); },
      isEncoding: function(enc) {
        return ['utf8','utf-8','ascii','latin1','binary','hex','base64','ucs2','ucs-2','utf16le','utf-16le']
          .indexOf((enc || '').toLowerCase()) !== -1;
      },
      byteLength: function(str, enc) {
        if (typeof str === 'string') {
          if (enc === 'base64') return Math.ceil(str.length * 3 / 4);
          return _te.encode(str).length;
        }
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
        return _addBufferMethods(r);
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
    // Support "x instanceof Buffer" — pg uses this check
    Object.defineProperty(_Buffer, Symbol.hasInstance, {
      value: function(obj) { return !!(obj && obj._isBuffer); }
    });
    globalThis.Buffer = _Buffer;
  })();
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
  // setInterval — repeated execution. For QuickJS, we simulate via recursive setTimeout.
  var __intervals = {};
  var __intervalId = 0;
  globalThis.setInterval = function(fn, delay) {
    var args = [];
    for (var i = 2; i < arguments.length; i++) args.push(arguments[i]);
    __intervalId++;
    var id = __intervalId;
    function tick() {
      if (!__intervals[id]) return;
      fn.apply(null, args);
      __intervals[id] = setTimeout(tick, delay || 0);
    }
    __intervals[id] = setTimeout(tick, delay || 0);
    return id;
  };
  globalThis.clearInterval = function(id) {
    if (__intervals[id]) {
      clearTimeout(__intervals[id]);
      delete __intervals[id];
    }
  };
}

// ─── Intl polyfill ──────────────────────────────────────────────────────
// Observational memory uses Intl.DateTimeFormat for timestamp formatting.
// QuickJS doesn't have Intl. Minimal stub that formats dates as ISO strings.
if (typeof Intl === "undefined") {
  // Observational memory calls Intl.DateTimeFormat() WITHOUT new — must return an object either way.
  function _DateTimeFormat(locale, opts) {
    if (!(this instanceof _DateTimeFormat)) return new _DateTimeFormat(locale, opts);
    this._opts = opts || {};
  }
  _DateTimeFormat.prototype.format = function(date) {
    var d = date || new Date();
    if (!(d instanceof Date)) d = new Date(d);
    var Y = d.getFullYear();
    var M = String(d.getMonth() + 1).padStart(2, "0");
    var D = String(d.getDate()).padStart(2, "0");
    var h = String(d.getHours()).padStart(2, "0");
    var m = String(d.getMinutes()).padStart(2, "0");
    return Y + "-" + M + "-" + D + " " + h + ":" + m;
  };
  _DateTimeFormat.prototype.resolvedOptions = function() {
    return { locale: "en-US", timeZone: "UTC" };
  };
  _DateTimeFormat.supportedLocalesOf = function() { return ["en-US"]; };
  globalThis.Intl = { DateTimeFormat: _DateTimeFormat };
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
