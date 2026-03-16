import * as esbuild from "esbuild";
import { statSync, writeFileSync } from "node:fs";

// Node.js built-ins that Mastra imports.
// We stub them at build time so esbuild can resolve named imports.
// At runtime in QuickJS, jsbridge polyfills provide the real implementations
// on globalThis (fetch, crypto, TextEncoder, URL, streams, etc.).
// File-system and child-process APIs will throw — they need Go-side bridges.

const nodeBuiltins = new Set([
  "assert", "async_hooks", "buffer", "child_process", "crypto",
  "diagnostics_channel", "dns", "events", "fs", "http", "https", "module", "timers",
  "net", "os", "path", "perf_hooks", "process", "querystring",
  "stream", "string_decoder", "tls", "url", "util", "worker_threads", "zlib",
]);

// Subpath modules that need their own stubs (not just the parent module)
const nodeSubpaths = new Set([
  "fs/promises", "stream/web", "path/posix", "stream/promises", "util/types", "timers/promises",
]);

function isNodeBuiltin(id) {
  if (id.startsWith("node:")) return true;
  if (nodeSubpaths.has(id)) return true;
  const base = id.split("/")[0];
  return nodeBuiltins.has(base);
}

// Normalize "node:crypto" -> "crypto", "stream/web" -> "stream/web"
function normalizeId(id) {
  return id.startsWith("node:") ? id.slice(5) : id;
}

// Per-module stub contents. Each export is a no-op function or empty object.
// These stubs exist ONLY to satisfy esbuild's named-import resolution.
// At runtime, jsbridge polyfills override the ones that matter (crypto, streams, events).
// The fs/path/child_process stubs will throw if actually called — that's intentional,
// those need Go-side bridges before they can work.
const noop = "() => {}";
const noopTrue = "() => true";
const noopFalse = "() => false";
const throwFn = (name) => `() => { throw new Error("${name}: not available in QuickJS, needs Go bridge"); }`;

const moduleStubs = {
  "crypto": `
    export const randomUUID = () => globalThis.crypto?.randomUUID?.() ?? "00000000-0000-0000-0000-000000000000";
    export const randomBytes = (n) => {
      const b = new Uint8Array(n);
      if (globalThis.crypto?.getRandomValues) globalThis.crypto.getRandomValues(b);
      return b;
    };
    export const randomFillSync = (buf) => {
      if (globalThis.crypto?.getRandomValues) globalThis.crypto.getRandomValues(buf);
      return buf;
    };
    export const randomInt = (min, max) => {
      if (max === undefined) { max = min; min = 0; }
      return min + Math.floor(Math.random() * (max - min));
    };
    export const createHash = (alg) => {
      if (globalThis.__node_crypto?.createHash) return globalThis.__node_crypto.createHash(alg);
      return { update(d) { return this; }, digest(enc) { return "stub"; }, copy() { return this; } };
    };
    export const createHmac = (alg, key) => {
      if (globalThis.__node_crypto?.createHmac) return globalThis.__node_crypto.createHmac(alg, key);
      return { update(d) { return this; }, digest(enc) { return "stub"; } };
    };
    export const createCipheriv = ${throwFn("createCipheriv")};
    export const createDecipheriv = ${throwFn("createDecipheriv")};
    export const createSign = ${throwFn("createSign")};
    export const createVerify = ${throwFn("createVerify")};
    export const pbkdf2 = ${throwFn("pbkdf2")};
    export const pbkdf2Sync = ${throwFn("pbkdf2Sync")};
    export const scrypt = ${throwFn("scrypt")};
    export const scryptSync = ${throwFn("scryptSync")};
    export const timingSafeEqual = (a, b) => {
      if (a.length !== b.length) return false;
      let r = 0;
      for (let i = 0; i < a.length; i++) r |= a[i] ^ b[i];
      return r === 0;
    };
    export const constants = {};
    export const webcrypto = globalThis.crypto;
    export const getHashes = () => ["sha256", "sha512"];
    export const getCiphers = () => [];
    export default { randomUUID, randomBytes, randomFillSync, randomInt, createHash, createHmac,
      createCipheriv, createDecipheriv, createSign, createVerify, pbkdf2, pbkdf2Sync,
      scrypt, scryptSync, timingSafeEqual, constants, webcrypto, getHashes, getCiphers };
  `,
  "stream": `
    // Minimal but functional Node.js stream stubs for MongoDB driver compatibility.
    // Key requirement: socket.pipe(transform) must work end-to-end:
    //   socket data events → transform.write() → _transform() → push() → data events
    class EventEmitterBase {
      constructor() { this._events = {}; }
      on(ev, fn) { (this._events[ev] = this._events[ev] || []).push(fn); return this; }
      addListener(ev, fn) { return this.on(ev, fn); }
      once(ev, fn) { const self = this; const w = function(...a) { self.removeListener(ev, w); fn.call(self, ...a); }; return this.on(ev, w); }
      removeListener(ev, fn) { const a = this._events[ev]; if (a) this._events[ev] = a.filter(f => f !== fn); return this; }
      off(ev, fn) { return this.removeListener(ev, fn); }
      removeAllListeners(ev) { if (ev) delete this._events[ev]; else this._events = {}; return this; }
      emit(ev, ...args) { const fns = this._events[ev]; if (!fns || !fns.length) return false; const self = this; fns.slice().forEach(fn => fn.call(self, ...args)); return true; }
      listeners(ev) { return (this._events[ev] || []).slice(); }
      listenerCount(ev) { return (this._events[ev] || []).length; }
      eventNames() { return Object.keys(this._events); }
      setMaxListeners() { return this; }
      getMaxListeners() { return 10; }
      prependListener(ev, fn) { (this._events[ev] = this._events[ev] || []).unshift(fn); return this; }
      prependOnceListener(ev, fn) { const w = (...a) => { this.removeListener(ev, w); fn(...a); }; return this.prependListener(ev, w); }
      rawListeners(ev) { return this.listeners(ev); }
    }

    export class Readable extends EventEmitterBase {
      constructor(opts) {
        super();
        this.readable = true;
        this.destroyed = false;
        this._paused = true;
        this._readableObjectMode = opts && opts.readableObjectMode;
        this._buffer = [];
      }
      pipe(dest, opts) {
        var self = this;
        this.on("data", function(chunk) {
          if (dest.write) {
            var ok = dest.write(chunk);
            if (ok === false && self.pause) self.pause();
          }
        });
        if (!opts || opts.end !== false) {
          this.on("end", function() { if (dest.end) dest.end(); });
        }
        if (dest.on) {
          dest.on("drain", function() { if (self.resume) self.resume(); });
        }
        if (dest.emit) dest.emit("pipe", this);
        return dest;
      }
      push(chunk) {
        if (chunk === null) { this.emit("end"); return false; }
        if (this._paused) { this._buffer.push(chunk); return true; }
        this.emit("data", chunk);
        return true;
      }
      read() { return this._buffer.length ? this._buffer.shift() : null; }
      unshift(chunk) { this._buffer.unshift(chunk); }
      resume() { this._paused = false; while (this._buffer.length) this.emit("data", this._buffer.shift()); return this; }
      pause() { this._paused = true; return this; }
      isPaused() { return this._paused; }
      destroy(err) { if (this.destroyed) return this; this.destroyed = true; if (err) this.emit("error", err); this.emit("close"); return this; }
      [Symbol.asyncIterator]() {
        var self = this; var done = false; var queue = []; var waiting = null;
        self.on("data", function(chunk) { if (waiting) { var r = waiting; waiting = null; r({value:chunk,done:false}); } else queue.push({value:chunk,done:false}); });
        self.on("end", function() { done = true; if (waiting) { var r = waiting; waiting = null; r({value:undefined,done:true}); } });
        self.on("error", function(err) { done = true; if (waiting) { var r = waiting; waiting = null; r(Promise.reject(err)); } });
        self.resume();
        return { next: function() { if (queue.length) return Promise.resolve(queue.shift()); if (done) return Promise.resolve({value:undefined,done:true}); return new Promise(function(r) { waiting = r; }); },
          return: function() { done = true; self.pause(); return Promise.resolve({value:undefined,done:true}); },
          [Symbol.asyncIterator]: function() { return this; } };
      }
      static from(iterable) { var r = new Readable(); if (iterable && iterable[Symbol.iterator]) { for (var v of iterable) r.push(v); r.push(null); } return r; }
    }

    export class Writable extends EventEmitterBase {
      constructor(opts) {
        super();
        this.writable = true;
        this.destroyed = false;
        this._writableObjectMode = opts && opts.writableObjectMode;
      }
      write(chunk, enc, cb) {
        if (typeof enc === "function") { cb = enc; enc = undefined; }
        if (this._write) { this._write(chunk, enc || "utf8", cb || function(){}); }
        else { if (cb) cb(); }
        return true;
      }
      end(chunk, enc, cb) {
        if (typeof chunk === "function") { cb = chunk; chunk = undefined; }
        if (typeof enc === "function") { cb = enc; enc = undefined; }
        if (chunk !== undefined && chunk !== null) this.write(chunk, enc);
        this.writable = false;
        if (this._final) this._final(function() { if (cb) cb(); });
        else if (cb) cb();
        this.emit("finish");
      }
      destroy(err) { if (this.destroyed) return this; this.destroyed = true; if (err) this.emit("error", err); this.emit("close"); return this; }
      cork() {}
      uncork() {}
    }

    export class Duplex extends Readable {
      constructor(opts) {
        super(opts);
        this.writable = true;
        this._writableObjectMode = opts && opts.writableObjectMode;
      }
      write(chunk, enc, cb) {
        if (typeof enc === "function") { cb = enc; enc = undefined; }
        if (this._write) { this._write(chunk, enc || "utf8", cb || function(){}); }
        else { if (cb) cb(); }
        return true;
      }
      end(chunk, enc, cb) {
        if (typeof chunk === "function") { cb = chunk; chunk = undefined; }
        if (typeof enc === "function") { cb = enc; enc = undefined; }
        if (chunk !== undefined && chunk !== null) this.write(chunk, enc);
        this.writable = false;
        if (this._final) this._final(function() { if (cb) cb(); });
        else if (cb) cb();
        this.emit("finish");
      }
      destroy(err) { if (this.destroyed) return this; this.destroyed = true; if (err) this.emit("error", err); this.emit("close"); return this; }
      cork() {}
      uncork() {}
    }

    export class Transform extends Duplex {
      constructor(opts) {
        super(opts);
        this._readableObjectMode = opts && opts.readableObjectMode;
        this._writableObjectMode = opts && (opts.writableObjectMode !== undefined ? opts.writableObjectMode : false);
      }
      // write() → _transform() → push() → data events
      _write(chunk, enc, cb) {
        this._transform(chunk, enc, cb);
      }
      _transform(chunk, enc, cb) { this.push(chunk); cb(); }
      _flush(cb) { cb(); }
    }

    export class PassThrough extends Transform {}
    export const pipeline = (...args) => { const cb = args.pop(); if (typeof cb === "function") cb(); };
    export const finished = (stream, cb) => { if (cb) cb(); };
    export default { Readable, Writable, Duplex, Transform, PassThrough, pipeline, finished };
  `,
  "stream/web": `
    export const ReadableStream = globalThis.ReadableStream || class ReadableStream {};
    export const WritableStream = globalThis.WritableStream || class WritableStream {};
    export const TransformStream = globalThis.TransformStream || class TransformStream {};
    export default { ReadableStream, WritableStream, TransformStream };
  `,
  "stream/promises": `
    export const pipeline = (...args) => Promise.resolve();
    export const finished = (stream) => Promise.resolve();
    export default { pipeline, finished };
  `,
  "timers": `
    export const setTimeout = globalThis.setTimeout;
    export const clearTimeout = globalThis.clearTimeout;
    export const setInterval = globalThis.setInterval;
    export const clearInterval = globalThis.clearInterval;
    export const setImmediate = globalThis.setImmediate || ((fn) => globalThis.setTimeout(fn, 0));
    export const clearImmediate = globalThis.clearImmediate || (() => {});
    export default { setTimeout, clearTimeout, setInterval, clearInterval, setImmediate, clearImmediate };
  `,
  "timers/promises": `
    export const setTimeout = (delay, value) => new Promise(resolve => globalThis.setTimeout(() => resolve(value), delay));
    export const setInterval = () => { throw new Error("timers/promises.setInterval: not supported in QuickJS"); };
    export const setImmediate = (value) => new Promise(resolve => globalThis.setTimeout(() => resolve(value), 0));
    export const scheduler = {
      wait: (delay) => new Promise(resolve => globalThis.setTimeout(resolve, delay)),
      yield: () => new Promise(resolve => globalThis.setTimeout(resolve, 0)),
    };
    export default { setTimeout, setInterval, setImmediate, scheduler };
  `,
  "dns": `
    export const lookup = (hostname, opts, cb) => {
      if (typeof opts === "function") { cb = opts; opts = {}; }
      // Resolve to the hostname itself — Go's net.Dial handles DNS
      if (cb) cb(null, hostname, 4);
    };
    export const resolve = (hostname, rrtype, cb) => {
      if (typeof rrtype === "function") { cb = rrtype; rrtype = "A"; }
      if (cb) cb(null, [hostname]);
    };
    export const resolve4 = (hostname, cb) => { if (cb) cb(null, [hostname]); };
    export const resolve6 = (hostname, cb) => { if (cb) cb(null, [hostname]); };
    export const promises = {
      lookup: async (hostname) => ({ address: hostname, family: 4 }),
      resolve: async (hostname) => [hostname],
      resolve4: async (hostname) => [hostname],
      resolve6: async (hostname) => [hostname],
    };
    export default { lookup, resolve, resolve4, resolve6, promises };
  `,
  "events": `
    export class EventEmitter {
      constructor() { this._events = {}; this._maxListeners = 10; }
      on(e, fn) { (this._events[e] = this._events[e] || []).push(fn); return this; }
      addListener(e, fn) { return this.on(e, fn); }
      prependListener(e, fn) { (this._events[e] = this._events[e] || []).unshift(fn); return this; }
      off(e, fn) { const a = this._events[e]; if (a) this._events[e] = a.filter(f => f !== fn); return this; }
      removeListener(e, fn) { return this.off(e, fn); }
      emit(e, ...args) { const self = this; for (const fn of (this._events[e] || []).slice()) fn.call(self, ...args); return true; }
      once(e, fn) { const self = this; const w = function(...a) { self.off(e, w); fn.call(self, ...a); }; return this.on(e, w); }
      prependOnceListener(e, fn) { const self = this; const w = function(...a) { self.off(e, w); fn.call(self, ...a); }; return this.prependListener(e, w); }
      removeAllListeners(e) { if (e) delete this._events[e]; else this._events = {}; return this; }
      listeners(e) { return (this._events[e] || []).slice(); }
      rawListeners(e) { return (this._events[e] || []).slice(); }
      listenerCount(e) { return (this._events[e] || []).length; }
      eventNames() { return Object.keys(this._events); }
      setMaxListeners(n) { this._maxListeners = n; return this; }
      getMaxListeners() { return this._maxListeners; }
    }
    EventEmitter.defaultMaxListeners = 10;
    EventEmitter.EventEmitter = EventEmitter;
    export const once = (emitter, event) => new Promise((resolve) => emitter.once(event, (...args) => resolve(args)));
    export const on = (emitter, event) => { throw new Error("events.on: not supported in QuickJS"); };
    export const getEventListeners = (emitter, event) => emitter.listeners?.(event) || [];
    export default EventEmitter;
  `,
  "path": `
    export const sep = "/";
    export const delimiter = ":";
    export const join = (...parts) => parts.filter(Boolean).join("/").replace(/\\/\\/+/g, "/");
    export const resolve = (...parts) => {
      var result = "";
      for (var i = parts.length - 1; i >= 0; i--) {
        var p = String(parts[i] || "");
        if (!p) continue;
        result = result ? p + "/" + result : p;
        if (p.charAt(0) === "/") break;
      }
      if (result.charAt(0) !== "/") {
        var cwd = (typeof process !== "undefined" && typeof process.cwd === "function") ? process.cwd() : "/";
        result = cwd + "/" + result;
      }
      var segs = result.split("/").filter(Boolean);
      var stack = [];
      for (var j = 0; j < segs.length; j++) {
        if (segs[j] === "..") { stack.pop(); }
        else if (segs[j] !== ".") { stack.push(segs[j]); }
      }
      return "/" + stack.join("/");
    };
    export const dirname = (p) => { const i = (p||"").lastIndexOf("/"); return i > 0 ? p.slice(0, i) : i === 0 ? "/" : "."; };
    export const basename = (p, ext) => { const b = (p||"").split("/").pop() || ""; return ext && b.endsWith(ext) ? b.slice(0, -ext.length) : b; };
    export const extname = (p) => { const b = basename(p); const i = b.lastIndexOf("."); return i > 0 ? b.slice(i) : ""; };
    export const normalize = (p) => {
      if (!p) return ".";
      var isAbs = p.charAt(0) === "/";
      var segs = p.split("/").filter(Boolean);
      var stack = [];
      for (var i = 0; i < segs.length; i++) {
        if (segs[i] === "..") { if (stack.length && stack[stack.length-1] !== "..") stack.pop(); else if (!isAbs) stack.push(".."); }
        else if (segs[i] !== ".") stack.push(segs[i]);
      }
      var result = stack.join("/") || (isAbs ? "" : ".");
      return isAbs ? "/" + result : result;
    };
    export const parse = (p) => { const b = basename(p); const e = extname(p); return { root: isAbsolute(p) ? "/" : "", dir: dirname(p), base: b, ext: e, name: e ? b.slice(0, -e.length) : b }; };
    export const relative = (from, to) => {
      var fromParts = resolve(from).split("/").filter(Boolean);
      var toParts = resolve(to).split("/").filter(Boolean);
      var common = 0;
      while (common < fromParts.length && common < toParts.length && fromParts[common] === toParts[common]) common++;
      var result = [];
      for (var i = common; i < fromParts.length; i++) result.push("..");
      for (var j = common; j < toParts.length; j++) result.push(toParts[j]);
      return result.join("/") || ".";
    };
    export const isAbsolute = (p) => (p||"").startsWith("/");
    export const format = (o) => (o.dir || o.root || "") + (o.dir && !o.dir.endsWith("/") ? "/" : "") + (o.base || o.name + (o.ext || ""));
    export const toNamespacedPath = (p) => p;
    const posix = { sep, delimiter, join, resolve, dirname, basename, extname, normalize, parse, relative, isAbsolute, format, toNamespacedPath };
    export { posix };
    export const win32 = posix;
    export default posix;
  `,
  "path/posix": `
    export const sep = "/";
    export const delimiter = ":";
    export const join = (...parts) => parts.filter(Boolean).join("/").replace(/\\/\\/+/g, "/");
    export const resolve = (...parts) => "/" + join(...parts).replace(/^\\/+/, "");
    export const dirname = (p) => { const i = (p||"").lastIndexOf("/"); return i > 0 ? p.slice(0, i) : "."; };
    export const basename = (p, ext) => { const b = (p||"").split("/").pop() || ""; return ext && b.endsWith(ext) ? b.slice(0, -ext.length) : b; };
    export const extname = (p) => { const b = basename(p); const i = b.lastIndexOf("."); return i > 0 ? b.slice(i) : ""; };
    export const normalize = (p) => (p||"").replace(/\\/\\/+/g, "/").replace(/\\/$/, "") || ".";
    export const parse = (p) => { const b = basename(p); const e = extname(p); return { root: "", dir: dirname(p), base: b, ext: e, name: e ? b.slice(0, -e.length) : b }; };
    export const relative = (from, to) => to;
    export const isAbsolute = (p) => (p||"").startsWith("/");
    export const format = (o) => (o.dir || o.root || "") + "/" + (o.base || o.name + (o.ext || ""));
    export const toNamespacedPath = (p) => p;
    export default { sep, delimiter, join, resolve, dirname, basename, extname, normalize, parse, relative, isAbsolute, format, toNamespacedPath };
  `,
  "fs": `
    const notAvailable = (name) => (...args) => {
      const cb = args[args.length - 1];
      if (typeof cb === "function") cb(new Error(name + ": not available in QuickJS"));
      else throw new Error(name + ": not available in QuickJS");
    };
    const notAvailableSync = (name) => () => { throw new Error(name + ": not available in QuickJS"); };
    export const constants = {
      F_OK: 0, R_OK: 4, W_OK: 2, X_OK: 1,
      O_RDONLY: 0, O_WRONLY: 1, O_RDWR: 2, O_CREAT: 64, O_EXCL: 128, O_TRUNC: 512, O_APPEND: 1024,
      S_IFMT: 61440, S_IFREG: 32768, S_IFDIR: 16384, S_IFLNK: 40960,
    };
    // Sync fs stubs — not all can work synchronously with async Go bridges.
    // The critical ones for LocalFilesystem's fs-utils.ts are existsSync and realpathSync.
    export const existsSync = (p) => { try { __go_fs_access(String(p)); return true; } catch { return false; } };
    export const readFileSync = notAvailableSync("readFileSync");
    export const writeFileSync = notAvailableSync("writeFileSync");
    export const appendFileSync = notAvailableSync("appendFileSync");
    export const mkdirSync = notAvailableSync("mkdirSync");
    export const rmdirSync = notAvailableSync("rmdirSync");
    export const realpathSync = (p) => p;
    export const readdirSync = notAvailableSync("readdirSync");
    export const renameSync = notAvailableSync("renameSync");
    export const rmSync = notAvailableSync("rmSync");
    export const statSync = notAvailableSync("statSync");
    export const lstatSync = notAvailableSync("lstatSync");
    export const unlinkSync = notAvailableSync("unlinkSync");
    export const chmodSync = notAvailableSync("chmodSync");
    export const accessSync = notAvailableSync("accessSync");
    export const copyFileSync = notAvailableSync("copyFileSync");
    export const openSync = notAvailableSync("openSync");
    export const closeSync = notAvailableSync("closeSync");
    export const readSync = notAvailableSync("readSync");
    export const writeSync = notAvailableSync("writeSync");
    export const createReadStream = notAvailableSync("createReadStream");
    export const createWriteStream = notAvailableSync("createWriteStream");
    export const watch = notAvailable("watch");
    export const watchFile = notAvailable("watchFile");
    export const unwatchFile = notAvailable("unwatchFile");
    export const readFile = notAvailable("readFile");
    export const writeFile = notAvailable("writeFile");
    export const access = notAvailable("access");
    export const stat = notAvailable("stat");
    export const lstat = notAvailable("lstat");
    export const mkdir = notAvailable("mkdir");
    export const readdir = notAvailable("readdir");
    export const unlink = notAvailable("unlink");
    export const rename = notAvailable("rename");
    export const copyFile = notAvailable("copyFile");
    export const promises = {
      readFile: async () => { throw new Error("fs.promises.readFile: not available in QuickJS"); },
      writeFile: async () => { throw new Error("fs.promises.writeFile: not available in QuickJS"); },
      mkdir: async () => { throw new Error("fs.promises.mkdir: not available in QuickJS"); },
      readdir: async () => { throw new Error("fs.promises.readdir: not available in QuickJS"); },
      stat: async () => { throw new Error("fs.promises.stat: not available in QuickJS"); },
      lstat: async () => { throw new Error("fs.promises.lstat: not available in QuickJS"); },
      access: async () => { throw new Error("fs.promises.access: not available in QuickJS"); },
      unlink: async () => { throw new Error("fs.promises.unlink: not available in QuickJS"); },
      rename: async () => { throw new Error("fs.promises.rename: not available in QuickJS"); },
      copyFile: async () => { throw new Error("fs.promises.copyFile: not available in QuickJS"); },
      rm: async () => { throw new Error("fs.promises.rm: not available in QuickJS"); },
    };
    export default { constants, existsSync, readFileSync, writeFileSync, appendFileSync,
      mkdirSync, rmdirSync, realpathSync, readdirSync, renameSync, rmSync, statSync, lstatSync,
      unlinkSync, chmodSync, accessSync, copyFileSync, openSync, closeSync, readSync, writeSync,
      createReadStream, createWriteStream, watch, watchFile, unwatchFile,
      readFile, writeFile, access, stat, lstat, mkdir, readdir, unlink, rename, copyFile, promises };
  `,
  "fs/promises": `
    // fs/promises — backed by Go bridges (__go_fs_*) registered by jsbridge/fs.go.
    // All bridges return Promises via ctx.NewPromise + Bridge.Go().
    // Go bridges encode errno codes as "ERRNO:CODE:message" prefix in error messages.
    // This helper strips the prefix and sets .code on the Error for Node.js compat.
    function _wrapFsCall(promise) {
      return promise.catch(function(e) {
        if (e && e.message && e.message.startsWith("ERRNO:")) {
          var parts = e.message.split(":");
          e.code = parts[1];
          e.message = parts.slice(2).join(":");
        }
        throw e;
      });
    }
    function _parseStat(json) {
      var raw = JSON.parse(json);
      return {
        size: raw.size, mode: raw.mode,
        mtimeMs: raw.mtimeMs, atimeMs: raw.mtimeMs, birthtimeMs: raw.mtimeMs,
        mtime: new Date(raw.mtimeMs), atime: new Date(raw.mtimeMs), birthtime: new Date(raw.mtimeMs),
        isFile: function() { return raw.isFile; },
        isDirectory: function() { return raw.isDirectory; },
        isSymbolicLink: function() { return !!raw.isSymbolicLink; },
      };
    }
    export function readFile(path, options) {
      return _wrapFsCall(__go_fs_readFile(String(path), typeof options === "string" ? options : "utf8"));
    }
    export function writeFile(path, data) {
      return _wrapFsCall(__go_fs_writeFile(String(path), typeof data === "string" ? data : String(data)));
    }
    export function appendFile(path, data) {
      return _wrapFsCall(__go_fs_appendFile(String(path), String(data)));
    }
    export async function stat(path) {
      var json = await _wrapFsCall(__go_fs_stat(String(path)));
      return _parseStat(json);
    }
    export async function lstat(path) {
      var json = await _wrapFsCall(__go_fs_lstat(String(path)));
      return _parseStat(json);
    }
    export async function readdir(path, options) {
      var withFileTypes = options && options.withFileTypes;
      var json = await _wrapFsCall(__go_fs_readdir(String(path)));
      var entries = JSON.parse(json);
      if (withFileTypes) {
        return entries.map(function(e) {
          return {
            name: e.name,
            isFile: function() { return !e.isDirectory; },
            isDirectory: function() { return e.isDirectory; },
            isSymbolicLink: function() { return false; },
          };
        });
      }
      return entries.map(function(e) { return e.name; });
    }
    export function mkdir(path, options) {
      return _wrapFsCall(__go_fs_mkdir(String(path), !!(options && options.recursive)));
    }
    export async function rm(path, options) {
      if (options && options.recursive) {
        return _wrapFsCall(__go_fs_rm(String(path)));
      } else {
        return _wrapFsCall(__go_fs_unlink(String(path)));
      }
    }
    export function unlink(path) {
      return _wrapFsCall(__go_fs_unlink(String(path)));
    }
    export function rename(oldPath, newPath) {
      return _wrapFsCall(__go_fs_rename(String(oldPath), String(newPath)));
    }
    export function copyFile(src, dest) {
      return _wrapFsCall(__go_fs_copyFile(String(src), String(dest)));
    }
    export function realpath(path) {
      return _wrapFsCall(__go_fs_realpath(String(path)));
    }
    export function access(path) {
      return _wrapFsCall(__go_fs_access(String(path)));
    }
    export function symlink(target, path) {
      return _wrapFsCall(__go_fs_symlink(String(target), String(path)));
    }
    export function readlink(path) {
      return _wrapFsCall(__go_fs_readlink(String(path)));
    }
    export async function open() { throw new Error("fs.open: use readFile/writeFile instead"); }
    export async function mkdtemp() { throw new Error("fs.mkdtemp: not implemented"); }
    export async function chmod() { /* no-op */ }
    export async function chown() { /* no-op */ }
    export async function truncate(path, len) {
      var data = await readFile(path);
      await writeFile(path, data.substring(0, len || 0));
    }
    export async function watch() { throw new Error("fs.watch: not implemented"); }
    export default { readFile, writeFile, appendFile, stat, lstat, readdir, mkdir, rm, unlink,
      rename, copyFile, realpath, access, symlink, readlink, open, mkdtemp, chmod, chown, truncate, watch };
  `,
  "child_process": `
    const notAvailable = (name) => (...args) => {
      const cb = args[args.length - 1];
      if (typeof cb === "function") cb(new Error(name + ": not available in QuickJS"));
      else throw new Error(name + ": not available in QuickJS");
    };
    const notAvailableSync = (name) => () => { throw new Error(name + ": not available in QuickJS"); };
    export const exec = notAvailable("exec");
    export const execFile = notAvailable("execFile");
    export const execSync = notAvailableSync("execSync");
    export const execFileSync = notAvailableSync("execFileSync");
    export const spawn = notAvailableSync("spawn");
    export const spawnSync = notAvailableSync("spawnSync");
    export const fork = notAvailableSync("fork");
    export default { exec, execFile, execSync, execFileSync, spawn, spawnSync, fork };
  `,
  "url": `
    export const URL = globalThis.URL || class URL { constructor(u) { this.href = u; } toString() { return this.href; } };
    export const URLSearchParams = globalThis.URLSearchParams || class URLSearchParams { constructor() { this._p = []; } };
    export const pathToFileURL = (p) => new (globalThis.URL)(\"file://\" + p);
    export const fileURLToPath = (u) => {
      const s = typeof u === "string" ? u : u.href || u.toString();
      return s.startsWith("file://") ? s.slice(7) : s;
    };
    export const format = (u) => typeof u === "string" ? u : u.href || u.toString();
    export const parse = (u) => { try { return new (globalThis.URL)(u); } catch(e) { return { href: u }; } };
    export const resolve = (from, to) => new (globalThis.URL)(to, from).href;
    export default { URL, URLSearchParams, pathToFileURL, fileURLToPath, format, parse, resolve };
  `,
  "module": `
    export const createRequire = () => {
      const r = (mod) => {
        // Delegate to globalThis.require which handles zod/v4, @opentelemetry/api, etc.
        if (typeof globalThis.require === "function") return globalThis.require(mod);
        return {};
      };
      r.resolve = () => "";
      return r;
    };
    export const builtinModules = [];
    export const isBuiltin = () => false;
    export default { createRequire, builtinModules, isBuiltin };
  `,
  "os": `
    export const homedir = () => "/home/user";
    export const tmpdir = () => "/tmp";
    export const platform = () => "linux";
    export const arch = () => "x64";
    export const cpus = () => [{ model: "stub", speed: 0, times: {} }];
    export const hostname = () => "quickjs";
    export const type = () => "Linux";
    export const release = () => "0.0.0";
    export const EOL = "\\n";
    export const endianness = () => "LE";
    export const totalmem = () => 4294967296;
    export const freemem = () => 2147483648;
    export const uptime = () => 0;
    export const loadavg = () => [0, 0, 0];
    export const networkInterfaces = () => ({});
    export const userInfo = () => ({ username: "user", uid: 1000, gid: 1000, shell: "/bin/sh", homedir: "/home/user" });
    export const constants = { signals: {}, errno: {} };
    export default { homedir, tmpdir, platform, arch, cpus, hostname, type, release, EOL, endianness,
      totalmem, freemem, uptime, loadavg, networkInterfaces, userInfo, constants };
  `,
  "util": `
    export const promisify = (fn) => (...args) => new Promise((res, rej) => fn(...args, (err, val) => err ? rej(err) : res(val)));
    export const callbackify = (fn) => (...args) => { const cb = args.pop(); fn(...args).then(v => cb(null, v), e => cb(e)); };
    export const inspect = (obj, opts) => {
      try { return JSON.stringify(obj, null, 2); }
      catch(e) { return String(obj); }
    };
    inspect.custom = Symbol.for("nodejs.util.inspect.custom");
    export const deprecate = (fn, msg) => fn;
    export const inherits = (ctor, superCtor) => { ctor.super_ = superCtor; Object.setPrototypeOf(ctor.prototype, superCtor.prototype); };
    export const format = (fmt, ...args) => {
      let i = 0;
      return String(fmt).replace(/%[sdifjoO%]/g, (m) => {
        if (m === "%%") return "%";
        if (i >= args.length) return m;
        const v = args[i++];
        switch (m) { case "%s": return String(v); case "%d": case "%i": return parseInt(v, 10); case "%f": return parseFloat(v);
          case "%j": try { return JSON.stringify(v); } catch(e) { return "[Circular]"; }
          case "%o": case "%O": return inspect(v); default: return m; }
      });
    };
    export const debuglog = (section) => () => {};
    export const types = {
      isPromise: (v) => v instanceof Promise,
      isDate: (v) => v instanceof Date,
      isRegExp: (v) => v instanceof RegExp,
      isNativeError: (v) => v instanceof Error,
      isMap: (v) => v instanceof Map,
      isSet: (v) => v instanceof Set,
      isTypedArray: (v) => ArrayBuffer.isView(v) && !(v instanceof DataView),
      isArrayBuffer: (v) => v instanceof ArrayBuffer,
      isDataView: (v) => v instanceof DataView,
      isWeakMap: (v) => v instanceof WeakMap,
      isWeakSet: (v) => v instanceof WeakSet,
      isSymbolObject: (v) => typeof v === "object" && typeof v.valueOf() === "symbol",
    };
    export const TextEncoder = globalThis.TextEncoder;
    export const TextDecoder = globalThis.TextDecoder;
    export const isDeepStrictEqual = (a, b) => JSON.stringify(a) === JSON.stringify(b);
    export default { promisify, callbackify, inspect, deprecate, inherits, format, debuglog, types,
      TextEncoder, TextDecoder, isDeepStrictEqual };
  `,
  "util/types": `
    export const isPromise = (v) => v instanceof Promise;
    export const isDate = (v) => v instanceof Date;
    export const isRegExp = (v) => v instanceof RegExp;
    export const isNativeError = (v) => v instanceof Error;
    export const isMap = (v) => v instanceof Map;
    export const isSet = (v) => v instanceof Set;
    export const isTypedArray = (v) => ArrayBuffer.isView(v) && !(v instanceof DataView);
    export const isArrayBuffer = (v) => v instanceof ArrayBuffer;
    export default { isPromise, isDate, isRegExp, isNativeError, isMap, isSet, isTypedArray, isArrayBuffer };
  `,
  "process": `
    const p = globalThis.process || {
      env: {}, cwd: () => "/", platform: "linux",
      version: "v20.0.0", versions: { node: "20.0.0" },
      argv: [], pid: 1, exit: () => {},
    };
    export default p;
    export const env = p.env;
    export const cwd = p.cwd || (() => "/");
    export const platform = p.platform || "linux";
    export const arch = p.arch || "x64";
    export const version = p.version || "v20.0.0";
    export const versions = p.versions || { node: "20.0.0" };
    export const argv = p.argv || [];
    export const execArgv = p.execArgv || [];
    export const pid = p.pid || 1;
    export const exit = p.exit || (() => {});
    export const kill = p.kill || (() => {});
    export const abort = p.abort || (() => {});
    export const umask = p.umask || (() => 0o22);
    export const uptime = p.uptime || (() => 0);
    export const hrtime = p.hrtime || Object.assign(
      (prev) => { const now = Date.now(); const s = Math.floor(now/1000); const ns = (now%1000)*1e6;
        if (prev) { return [s-prev[0], ns-prev[1]<0?(s-prev[0]-1,ns-prev[1]+1e9):(ns-prev[1])]; }
        return [s, ns]; },
      { bigint: () => BigInt(Date.now()) * BigInt(1e6) }
    );
    export const nextTick = (fn, ...args) => queueMicrotask(() => fn(...args));
    export const stdout = p.stdout || { write: () => true, isTTY: false };
    export const stderr = p.stderr || { write: () => true, isTTY: false };
    export const stdin = p.stdin || { isTTY: false, on: () => stdin, resume: () => stdin };
    export const title = p.title || "node";
    export const release = p.release || { name: "node" };
    export const config = p.config || {};
    export const features = p.features || {};
    export const memoryUsage = p.memoryUsage || (() => ({ rss:0, heapTotal:0, heapUsed:0, external:0, arrayBuffers:0 }));
    export const cpuUsage = p.cpuUsage || (() => ({ user:0, system:0 }));
    export const on = p.on || (() => p);
    export const off = p.off || (() => p);
    export const once = p.once || (() => p);
    export const addListener = p.addListener || (() => p);
    export const removeListener = p.removeListener || (() => p);
    export const removeAllListeners = p.removeAllListeners || (() => p);
    export const emit = p.emit || (() => false);
    export const listeners = p.listeners || (() => []);
    export const listenerCount = p.listenerCount || (() => 0);
  `,
  "buffer": `
    export const Buffer = globalThis.Buffer || {
      from: (v, enc) => typeof v === "string" ? new TextEncoder().encode(v) : new Uint8Array(v),
      alloc: (n, fill) => { const b = new Uint8Array(n); if (fill) b.fill(typeof fill === "number" ? fill : 0); return b; },
      allocUnsafe: (n) => new Uint8Array(n),
      allocUnsafeSlow: (n) => new Uint8Array(n),
      isBuffer: () => false,
      isEncoding: (enc) => ["utf8","utf-8","ascii","latin1","binary","hex","base64"].indexOf((enc||"").toLowerCase()) !== -1,
      byteLength: (str, enc) => typeof str === "string" ? new TextEncoder().encode(str).length : (str.byteLength || str.length || 0),
      concat: (bufs, totalLength) => {
        if (!totalLength) { totalLength = 0; for (const b of bufs) totalLength += b.length; }
        const r = new Uint8Array(totalLength); let off = 0;
        for (const b of bufs) { r.set(b, off); off += b.length; }
        return r;
      },
      compare: (a, b) => {
        const len = Math.min(a.length, b.length);
        for (let i = 0; i < len; i++) { if (a[i] < b[i]) return -1; if (a[i] > b[i]) return 1; }
        return a.length < b.length ? -1 : a.length > b.length ? 1 : 0;
      },
    };
    export const SlowBuffer = Buffer;
    export const INSPECT_MAX_BYTES = 50;
    export const kMaxLength = 2147483647;
    export const constants = { MAX_LENGTH: 2147483647, MAX_STRING_LENGTH: 536870888 };
    export default Buffer;
  `,
  "assert": `
    export default function assert(val, msg) { if (!val) throw new Error(msg || "Assertion failed"); };
    export const ok = (val, msg) => { if (!val) throw new Error(msg || "Assertion failed"); };
    export const strictEqual = (a, b, msg) => { if (a !== b) throw new Error(msg || a + " !== " + b); };
    export const notStrictEqual = (a, b, msg) => { if (a === b) throw new Error(msg || a + " === " + b); };
    export const deepStrictEqual = (a, b, msg) => { if (JSON.stringify(a) !== JSON.stringify(b)) throw new Error(msg || "Not deeply equal"); };
    export const throws = (fn, msg) => { try { fn(); throw new Error(msg || "Expected to throw"); } catch(e) {} };
    export const doesNotThrow = (fn, msg) => { try { fn(); } catch(e) { throw new Error(msg || "Did not expect to throw: " + e.message); } };
    export const fail = (msg) => { throw new Error(msg || "Assertion failed"); };
    export const AssertionError = class AssertionError extends Error { constructor(opts) { super(opts?.message || "Assertion failed"); } };
    export const strict = Object.assign(assert, { ok, strictEqual, notStrictEqual, deepStrictEqual, throws, doesNotThrow, fail });
  `,
  "http": `
    const notAvailable = (name) => () => { throw new Error(name + ": not available in QuickJS, use fetch"); };
    export const createServer = notAvailable("http.createServer");
    export const request = notAvailable("http.request");
    export const get = notAvailable("http.get");
    export const Agent = class Agent { constructor() {} };
    export const globalAgent = new Agent();
    export const METHODS = ["GET","HEAD","POST","PUT","DELETE","CONNECT","OPTIONS","TRACE","PATCH"];
    export const STATUS_CODES = { 200:"OK", 201:"Created", 204:"No Content", 301:"Moved Permanently",
      302:"Found", 304:"Not Modified", 400:"Bad Request", 401:"Unauthorized", 403:"Forbidden",
      404:"Not Found", 405:"Method Not Allowed", 500:"Internal Server Error", 502:"Bad Gateway",
      503:"Service Unavailable" };
    export default { createServer, request, get, Agent, globalAgent, METHODS, STATUS_CODES };
  `,
  "https": `
    const notAvailable = (name) => () => { throw new Error(name + ": not available in QuickJS, use fetch"); };
    export const createServer = notAvailable("https.createServer");
    export const request = notAvailable("https.request");
    export const get = notAvailable("https.get");
    export const Agent = class Agent { constructor() {} };
    export const globalAgent = new Agent();
    export default { createServer, request, get, Agent, globalAgent };
  `,
  "net": `
    const notAvailable = (name) => () => { throw new Error(name + ": not available in QuickJS"); };
    export const createServer = notAvailable("net.createServer");
    export class Socket {
      constructor() {
        if (globalThis.GoSocket) {
          this._gs = new globalThis.GoSocket();
        } else {
          this._events = {};
        }
      }
      connect() { if (this._gs) return this._gs.connect.apply(this._gs, arguments); return this; }
      write() { if (this._gs) return this._gs.write.apply(this._gs, arguments); return false; }
      end() { if (this._gs) return this._gs.end.apply(this._gs, arguments); }
      destroy() { if (this._gs) return this._gs.destroy.apply(this._gs, arguments); return this; }
      pipe() { if (this._gs) return this._gs.pipe.apply(this._gs, arguments); return arguments[0]; }
      on(e, fn) { if (this._gs) { this._gs.on(e, fn); return this; } (this._events[e] = this._events[e] || []).push(fn); return this; }
      once(e, fn) { if (this._gs) { this._gs.once(e, fn); return this; } return this.on(e, fn); }
      removeListener(e, fn) { if (this._gs) { this._gs.removeListener(e, fn); return this; } return this; }
      removeAllListeners(e) { if (this._gs) { this._gs.removeAllListeners && this._gs.removeAllListeners(e); return this; } return this; }
      off(e, fn) { return this.removeListener(e, fn); }
      emit() { if (this._gs) return this._gs.emit.apply(this._gs, arguments); return false; }
      pause() { if (this._gs && this._gs.pause) this._gs.pause(); return this; }
      resume() { if (this._gs && this._gs.resume) this._gs.resume(); return this; }
      setNoDelay() { return this; }
      setKeepAlive() { return this; }
      setTimeout() { if (this._gs) return this._gs.setTimeout.apply(this._gs, arguments); return this; }
      ref() { return this; }
      unref() { return this; }
      cork() {}
      uncork() {}
      get remoteAddress() { return this._gs ? this._gs.remoteAddress : undefined; }
      get remotePort() { return this._gs ? this._gs.remotePort : undefined; }
      get writable() { return this._gs ? this._gs.writable : false; }
    }
    export const createConnection = (...args) => { const s = new Socket(); s.connect(args[0], args[1]); return s; };
    export const connect = createConnection;
    export const Server = class Server {};
    export const isIP = (input) => { try { return input.includes(":") ? 6 : input.match(/^\\d+\\.\\d+\\.\\d+\\.\\d+$/) ? 4 : 0; } catch(e) { return 0; } };
    export const isIPv4 = (input) => isIP(input) === 4;
    export const isIPv6 = (input) => isIP(input) === 6;
    export default { createServer, createConnection, connect, Socket, Server, isIP, isIPv4, isIPv6 };
  `,
  "tls": `
    const notAvailable = (name) => () => { throw new Error(name + ": not available in QuickJS"); };
    export const createServer = notAvailable("tls.createServer");
    export const connect = notAvailable("tls.connect");
    export const TLSSocket = class TLSSocket {};
    export const DEFAULT_ECDH_CURVE = "auto";
    export const DEFAULT_MIN_VERSION = "TLSv1.2";
    export const DEFAULT_MAX_VERSION = "TLSv1.3";
    export default { createServer, connect, TLSSocket, DEFAULT_ECDH_CURVE, DEFAULT_MIN_VERSION, DEFAULT_MAX_VERSION };
  `,
  "querystring": `
    export const parse = (str) => {
      const obj = {};
      (str || "").split("&").forEach(pair => {
        if (!pair) return;
        const [k, ...v] = pair.split("=");
        obj[decodeURIComponent(k)] = decodeURIComponent(v.join("="));
      });
      return obj;
    };
    export const stringify = (obj) => Object.entries(obj || {}).map(([k,v]) => encodeURIComponent(k) + "=" + encodeURIComponent(v)).join("&");
    export const encode = stringify;
    export const decode = parse;
    export const escape = encodeURIComponent;
    export const unescape = decodeURIComponent;
    export default { parse, stringify, encode, decode, escape, unescape };
  `,
  "string_decoder": `
    export class StringDecoder {
      constructor(encoding) { this.encoding = encoding || "utf-8"; this._decoder = new TextDecoder(this.encoding); }
      write(buf) { return this._decoder.decode(buf instanceof Uint8Array ? buf : new Uint8Array(buf), { stream: true }); }
      end(buf) { if (buf) return this._decoder.decode(buf instanceof Uint8Array ? buf : new Uint8Array(buf)); return this._decoder.decode(); }
    }
    export default { StringDecoder };
  `,
  "perf_hooks": `
    export const performance = globalThis.performance || { now: () => Date.now(), timeOrigin: Date.now() };
    export const PerformanceObserver = class PerformanceObserver { constructor() {} observe() {} disconnect() {} };
    export const monitorEventLoopDelay = () => ({ enable: () => {}, disable: () => {}, percentile: () => 0, min: 0, max: 0, mean: 0, stddev: 0 });
    export default { performance, PerformanceObserver, monitorEventLoopDelay };
  `,
  "async_hooks": `
    export const createHook = () => ({ enable: () => {}, disable: () => {} });
    export const executionAsyncId = () => 0;
    export const triggerAsyncId = () => 0;
    export const executionAsyncResource = () => ({});
    export class AsyncLocalStorage {
      constructor() { this._store = undefined; }
      getStore() { return this._store; }
      run(store, fn, ...args) { const prev = this._store; this._store = store; try { return fn(...args); } finally { this._store = prev; } }
      enterWith(store) { this._store = store; }
      disable() { this._store = undefined; }
    }
    export class AsyncResource {
      constructor(type) { this.type = type; }
      runInAsyncScope(fn, thisArg, ...args) { return fn.apply(thisArg, args); }
      emitDestroy() { return this; }
      asyncId() { return 0; }
      triggerAsyncId() { return 0; }
    }
    export default { createHook, executionAsyncId, triggerAsyncId, executionAsyncResource, AsyncLocalStorage, AsyncResource };
  `,
  "diagnostics_channel": `
    export const channel = (name) => ({
      subscribe: () => {},
      unsubscribe: () => {},
      publish: () => {},
      hasSubscribers: false,
    });
    export const hasSubscribers = () => false;
    export const subscribe = () => {};
    export const unsubscribe = () => {};
    export class Channel { constructor() { this.hasSubscribers = false; } subscribe() {} unsubscribe() {} publish() {} }
    export default { channel, hasSubscribers, subscribe, unsubscribe, Channel };
  `,
  "worker_threads": `
    export const isMainThread = true;
    export const parentPort = null;
    export const workerData = undefined;
    export const threadId = 0;
    export class Worker { constructor() { throw new Error("Worker threads not available in QuickJS"); } }
    export class MessageChannel { constructor() { this.port1 = {}; this.port2 = {}; } }
    export class MessagePort {}
    export default { isMainThread, parentPort, workerData, threadId, Worker, MessageChannel, MessagePort };
  `,
  "zlib": `
    const notAvailable = (name) => () => { throw new Error(name + ": not available in QuickJS"); };
    export const createGzip = notAvailable("createGzip");
    export const createGunzip = notAvailable("createGunzip");
    export const createDeflate = notAvailable("createDeflate");
    export const createInflate = notAvailable("createInflate");
    export const gzip = notAvailable("gzip");
    export const gunzip = notAvailable("gunzip");
    export const deflate = notAvailable("deflate");
    export const inflate = notAvailable("inflate");
    export const gzipSync = notAvailable("gzipSync");
    export const gunzipSync = notAvailable("gunzipSync");
    export const deflateSync = notAvailable("deflateSync");
    export const inflateSync = notAvailable("inflateSync");
    export const brotliCompressSync = notAvailable("brotliCompressSync");
    export const brotliDecompressSync = notAvailable("brotliDecompressSync");
    export const constants = {};
    export default { createGzip, createGunzip, createDeflate, createInflate, gzip, gunzip, deflate, inflate,
      gzipSync, gunzipSync, deflateSync, inflateSync, brotliCompressSync, brotliDecompressSync, constants };
  `,
};

// Fallback: empty module for any Node.js built-in not explicitly stubbed
const fallbackStub = "export default {};";

const nodeStubPlugin = {
  name: "node-stub",
  setup(build) {
    build.onResolve({ filter: /.*/ }, (args) => {
      if (isNodeBuiltin(args.path)) {
        return { path: args.path, namespace: "node-stub" };
      }
    });
    build.onLoad({ filter: /.*/, namespace: "node-stub" }, (args) => {
      const id = normalizeId(args.path);
      return {
        contents: moduleStubs[id] || fallbackStub,
        loader: "js",
      };
    });
  },
};

const result = await esbuild.build({
  entryPoints: ["entry.mjs"],
  bundle: true,
  format: "iife",
  platform: "browser",
  target: "es2020",
  minify: true,
  treeShaking: true,
  plugins: [
    nodeStubPlugin,
    // Redirect EXACT 'zod' imports to 'zod/v4' so all code uses ONE Zod version.
    // Without this, `import { z } from 'zod'` gives v3 (no toJSONSchema)
    // while `import { z } from 'zod/v4'` gives v4 (has toJSONSchema).
    // Mixing them causes "cannot read property 'def' of undefined" when passing
    // v3 schemas to v4's toJSONSchema.
    {
      name: "zod-unify",
      setup(build) {
        // Redirect EXACT 'zod' to 'zod/v4', always resolving from the bundle root.
        // This ensures ALL packages use the same physical Zod v4 module.
        // Without the fixed resolveDir, nested node_modules can resolve to different copies.
        build.onResolve({ filter: /^zod$/ }, (args) => {
          return build.resolve("zod/v4", {
            resolveDir: args.resolveDir,
            kind: args.kind,
            importer: args.importer,
          });
        });
      },
    },
    // Note: js-tiktoken/lite and js-tiktoken/ranks/* are bundled normally.
    // The getTiktoken() function that uses them is patched post-build to use
    // encodingForModel() instead (see post-process section below).
  ],
  external: [
    // Database drivers — bundled with TCP socket polyfill via jsbridge/net.go
    // "pg" — bundled
    // "mongodb" — bundled
    // "@libsql/client", — bundled: HTTP-based, works through fetch
    "better-sqlite3",
    // Native modules that can't run in QuickJS
    "@ast-grep/napi",
    "fastembed",
    "@opentelemetry/api",
    // Server framework — not needed
    "hono",
  ],
  banner: {
    // Runs BEFORE any bundled module code — needed for dynamic require("zod/v4")
    // in the AI SDK's toJSONSchema resolution (S1A function).
    js: `var __zod_v4_deferred = null;`,
  },
  define: {
    "process.env.NODE_ENV": '"production"',
  },
  logLevel: "info",
  metafile: true,
  outfile: "../agent_embed_bundle.js",
});

// Post-process: patch @libsql/client value serializer to handle `undefined`.
// The serializer function checks null, string, number, bigint, boolean, ArrayBuffer,
// Uint8Array, Date, object — but NOT undefined. The OM processor stores message metadata
// with undefined fields that reach the serializer. Fix: coerce undefined to null.
import { readFileSync } from "node:fs";
{
  let bundle = readFileSync("../agent_embed_bundle.js", "utf8");
  const oldPattern = /function (\w+)\(e\)\{if\(e===null\)return null;if\(typeof e==.?"string"\)return e;/;
  const match = bundle.match(oldPattern);
  if (match) {
    const fname = match[1];
    const old = `function ${fname}(e){if(e===null)return null;`;
    const fix = `function ${fname}(e){if(e===void 0||e===null)return null;`;
    bundle = bundle.replace(old, fix);
    writeFileSync("../agent_embed_bundle.js", bundle);
    console.log(`Patched ${fname}: undefined → null in @libsql/client value serializer`);
  } else {
    console.warn("WARNING: Could not find @libsql/client value serializer to patch");
  }

  // Patch @mastra/rag validation: z.optional(z.function()) → z.optional(z.any()).
  // Zod v4's $ZodFunction doesn't extend $ZodType — it lacks _zod, causing
  // safeParse to crash with "cannot read property 'optin' of undefined".
  const funcPattern = 'lengthFunction:s.optional(s.function())';
  if (bundle.includes(funcPattern)) {
    bundle = bundle.replace(funcPattern, 'lengthFunction:s.optional(s.any())');
    writeFileSync("../agent_embed_bundle.js", bundle);
    console.log('Patched: z.function() → z.any() for RAG validation (Zod v4 compat)');
  }

  // Patch getTiktoken() to use getEncoding('o200k_base') with fallback.
  // The original function uses dynamic import('js-tiktoken/lite') which esbuild
  // converts to Promise.resolve().then(...). The Tiktoken constructor from /lite
  // fails in QuickJS. We replace the function body with getEncoding() which works.
  const tiktokenFnPattern = /async function (\w+)\(\)\{let e=globalThis\[(\w+)\];if\(e\)return e;/;
  const tiktokenFnMatch = bundle.match(tiktokenFnPattern);
  if (tiktokenFnMatch) {
    const fn = tiktokenFnMatch[1];
    const key = tiktokenFnMatch[2];
    // Find the full function
    const fnStart = tiktokenFnMatch.index;
    let depth = 0, fnEnd = fnStart;
    for (let i = fnStart; i < Math.min(fnStart + 500, bundle.length); i++) {
      if (bundle[i] === '{') depth++;
      else if (bundle[i] === '}') { depth--; if (depth === 0) { fnEnd = i + 1; break; } }
    }
    const oldFn = bundle.substring(fnStart, fnEnd);
    // Find getEncoding function (switch on encoding names including o200k_base)
    const encIdx = bundle.indexOf('"o200k_base":return new');
    const encChunk = bundle.substring(Math.max(0, encIdx - 300), encIdx + 50);
    const encMatch = encChunk.match(/function (\w+)\(e(?:,\w+)?\)\{switch\(e\)/);
    if (encMatch) {
      const encFn = encMatch[1];
      const replacement = `async function ${fn}(){let e=globalThis[${key}];if(e)return e;try{let I=${encFn}("o200k_base");return globalThis[${key}]=I,I}catch(_){var F={encode:function(s){return Array.from({length:Math.ceil((s||"").length/4)},function(_,i){return i})},decode:function(t){return"[decoded]"}};return globalThis[${key}]=F,F}}`;
      bundle = bundle.replace(oldFn, replacement);
      writeFileSync("../agent_embed_bundle.js", bundle);
      console.log(`Patched ${fn}: getTiktoken uses ${encFn}('o200k_base') with fallback`);
    }
  }
}

// Report size
const stats = statSync("../agent_embed_bundle.js");
console.log(`Bundle size: ${(stats.size / 1024).toFixed(1)} KB`);

// Write metafile for analysis
writeFileSync("meta.json", JSON.stringify(result.metafile));
console.log("Metafile written to meta.json (use https://esbuild.github.io/analyze/ to inspect)");
