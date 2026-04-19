import * as esbuild from "esbuild";
import { statSync, writeFileSync } from "node:fs";

// Node.js built-ins that Mastra imports.
// We stub them at build time so esbuild can resolve named imports.
// At runtime in QuickJS, jsbridge polyfills provide the real implementations
// on globalThis (stream, crypto, net, os, Buffer, etc.).
// These stubs are THIN RE-EXPORTS — no logic, just wiring for esbuild resolution.

const nodeBuiltins = new Set([
  "assert", "async_hooks", "buffer", "child_process", "crypto",
  "diagnostics_channel", "dns", "events", "fs", "http", "https", "module", "timers",
  "net", "os", "path", "perf_hooks", "process", "querystring",
  "stream", "string_decoder", "tls", "url", "util", "worker_threads", "zlib",
]);

const nodeSubpaths = new Set([
  "fs/promises", "stream/web", "path/posix", "stream/promises", "util/types", "timers/promises",
]);

function isNodeBuiltin(id) {
  if (id.startsWith("node:")) return true;
  if (nodeSubpaths.has(id)) return true;
  const base = id.split("/")[0];
  return nodeBuiltins.has(base);
}

function normalizeId(id) {
  return id.startsWith("node:") ? id.slice(5) : id;
}

const throwFn = (name) => `function() { throw new Error("${name}: not available in QuickJS"); }`;

// ─── Module stubs: thin re-exports from jsbridge polyfills on globalThis ──
const moduleStubs = {
  "crypto": `
    var C = globalThis.crypto || {};
    export var randomUUID = function() { return globalThis.crypto.randomUUID(); };
    export var randomBytes = C.randomBytes || function(n) { return new Uint8Array(n); };
    export var randomFillSync = C.randomFillSync || function(buf) { return buf; };
    export var randomInt = C.randomInt || function(min, max) { if (max === undefined) { max = min; min = 0; } return min + Math.floor(Math.random() * (max - min)); };
    export var createHash = C.createHash || function() { return { update: function() { return this; }, digest: function() { return ""; } }; };
    export var createHmac = C.createHmac || function() { return { update: function() { return this; }, digest: function() { return ""; } }; };
    export var pbkdf2 = C.pbkdf2 || ${throwFn("pbkdf2")};
    export var pbkdf2Sync = C.pbkdf2Sync || ${throwFn("pbkdf2Sync")};
    export var timingSafeEqual = C.timingSafeEqual || function(a, b) { if (a.length !== b.length) return false; var r = 0; for (var i = 0; i < a.length; i++) r |= a[i] ^ b[i]; return r === 0; };
    export var getHashes = C.getHashes || function() { return ["sha256", "sha512"]; };
    export var getCiphers = C.getCiphers || function() { return []; };
    export var getFips = C.getFips || function() { return 0; };
    export var createCipheriv = ${throwFn("createCipheriv")};
    export var createDecipheriv = ${throwFn("createDecipheriv")};
    export var createSign = ${throwFn("createSign")};
    export var createVerify = ${throwFn("createVerify")};
    export var scrypt = ${throwFn("scrypt")};
    export var scryptSync = ${throwFn("scryptSync")};
    export var constants = {};
    export var webcrypto = globalThis.crypto;
    export default { randomUUID, randomBytes, randomFillSync, randomInt, createHash, createHmac,
      createCipheriv, createDecipheriv, createSign, createVerify, pbkdf2, pbkdf2Sync,
      scrypt, scryptSync, timingSafeEqual, constants, webcrypto, getHashes, getCiphers, getFips };
  `,
  "stream": `
    var S = globalThis.stream || {};
    export var Readable = S.Readable || class Readable {};
    export var Writable = S.Writable || class Writable {};
    export var Duplex = S.Duplex || class Duplex {};
    export var Transform = S.Transform || class Transform {};
    export var PassThrough = S.PassThrough || class PassThrough {};
    export var pipeline = S.pipeline || function() { var cb = arguments[arguments.length - 1]; if (typeof cb === "function") cb(); };
    export var finished = S.finished || function(stream, cb) { if (cb) cb(); };
    export var Stream = S.Stream || Readable;
    if (!Readable.from) Readable.from = function(iterable) { var r = new Readable(); if (iterable && iterable[Symbol.iterator]) { for (var v of iterable) r.push(v); r.push(null); } return r; };
    if (!Readable.toWeb) Readable.toWeb = function(nodeStream) { return new ReadableStream({ start(ctrl) { nodeStream.on("data", function(c) { ctrl.enqueue(c); }); nodeStream.on("end", function() { ctrl.close(); }); } }); };
    if (!Readable.fromWeb) Readable.fromWeb = function(webStream) { var r = new Readable(); var reader = webStream.getReader(); (async function pump() { var res = await reader.read(); if (res.done) { r.push(null); return; } r.push(res.value); pump(); })(); return r; };
    export default { Readable, Writable, Duplex, Transform, PassThrough, pipeline, finished, Stream };
  `,
  "stream/web": `
    export var ReadableStream = globalThis.ReadableStream || class ReadableStream {};
    export var WritableStream = globalThis.WritableStream || class WritableStream {};
    export var TransformStream = globalThis.TransformStream || class TransformStream {};
    export default { ReadableStream, WritableStream, TransformStream };
  `,
  "stream/promises": `
    export var pipeline = function() { return Promise.resolve(); };
    export var finished = function() { return Promise.resolve(); };
    export default { pipeline, finished };
  `,
  "net": `
    var N = globalThis.net || {};
    export var Socket = N.Socket || class Socket {};
    export var createConnection = N.createConnection || function() { return new Socket(); };
    export var connect = N.connect || createConnection;
    export var createServer = N.createServer || ${throwFn("net.createServer")};
    export var Server = N.Server || class Server {};
    export var isIP = N.isIP || function() { return 0; };
    export var isIPv4 = N.isIPv4 || function() { return false; };
    export var isIPv6 = N.isIPv6 || function() { return false; };
    export default { Socket, createConnection, connect, createServer, Server, isIP, isIPv4, isIPv6 };
  `,
  "tls": `
    export var createServer = ${throwFn("tls.createServer")};
    export var connect = function(options) {
      // pg SSL upgrade: tls.connect({ socket: existingSocket, servername: host })
      // Upgrades existing TCP connection to TLS via Go crypto/tls
      if (options && options.socket && options.socket._gs && options.socket._gs._id) {
        var servername = options.servername || options.host || "";
        var ok = __go_net_tls_upgrade(options.socket._gs._id, servername);
        if (!ok) throw new Error("tls.connect: TLS upgrade failed");
        // Return the same socket — its underlying Go conn is now TLS
        options.socket.emit("secureConnect");
        return options.socket;
      }
      // Raw GoSocket (not wrapped in Duplex Socket)
      if (options && options.socket && options.socket._id) {
        var servername = options.servername || options.host || "";
        var ok = __go_net_tls_upgrade(options.socket._id, servername);
        if (!ok) throw new Error("tls.connect: TLS upgrade failed");
        options.socket._emit && options.socket._emit("secureConnect");
        return options.socket;
      }
      throw new Error("tls.connect: requires options.socket (TLS upgrade of existing connection)");
    };
    export var TLSSocket = class TLSSocket {};
    export var DEFAULT_ECDH_CURVE = "auto";
    export var DEFAULT_MIN_VERSION = "TLSv1.2";
    export var DEFAULT_MAX_VERSION = "TLSv1.3";
    export default { createServer, connect, TLSSocket, DEFAULT_ECDH_CURVE, DEFAULT_MIN_VERSION, DEFAULT_MAX_VERSION };
  `,
  "buffer": `
    export var Buffer = globalThis.Buffer || { from: function() { return new Uint8Array(0); }, alloc: function(n) { return new Uint8Array(n); }, isBuffer: function() { return false; } };
    export default { Buffer };
  `,
  "events": `
    export var EventEmitter = globalThis.EventEmitter || class EventEmitter {};
    export default EventEmitter;
  `,
  "path": `
    var P = globalThis.path || {};
    export var join = P.join || function() { return Array.prototype.join.call(arguments, "/"); };
    export var resolve = P.resolve || join;
    export var dirname = P.dirname || function(p) { return p.replace(/\\/[^\\/]*$/, ""); };
    export var basename = P.basename || function(p) { return p.replace(/.*\\//, ""); };
    export var extname = P.extname || function(p) { var m = p.match(/\\.[^.]+$/); return m ? m[0] : ""; };
    export var normalize = function(p) { return p.replace(/\\/+/g, "/").replace(/\\/$/,""); };
    export var isAbsolute = function(p) { return p.charAt(0) === "/"; };
    export var parse = function(p) { var b = basename(p); var e = extname(p); return { root: "", dir: dirname(p), base: b, ext: e, name: b.replace(e, "") }; };
    export var relative = function(from, to) { return to; };
    export var sep = "/";
    export var delimiter = ":";
    export var posix = P;
    export default { join, resolve, dirname, basename, extname, normalize, isAbsolute, parse, relative, sep, delimiter, posix };
  `,
  "path/posix": `
    var P = globalThis.path || {};
    export var join = P.join || function() { return ""; };
    export var resolve = P.resolve || join;
    export var dirname = P.dirname || function() { return ""; };
    export var basename = P.basename || function() { return ""; };
    export var extname = P.extname || function() { return ""; };
    export var sep = "/";
    export default { join, resolve, dirname, basename, extname, sep };
  `,
  "os": `
    var O = globalThis.os || {};
    export var platform = O.platform || function() { return "linux"; };
    export var arch = O.arch || function() { return "x64"; };
    export var tmpdir = O.tmpdir || function() { return "/tmp"; };
    export var homedir = O.homedir || function() { return "/"; };
    export var hostname = O.hostname || function() { return "localhost"; };
    export var type = O.type || function() { return "Linux"; };
    export var EOL = O.EOL || "\\n";
    export var cpus = O.cpus || function() { return []; };
    export var release = O.release || function() { return ""; };
    export var totalmem = O.totalmem || function() { return 0; };
    export var freemem = O.freemem || function() { return 0; };
    export var endianness = O.endianness || function() { return "LE"; };
    export default { platform, arch, tmpdir, homedir, hostname, type, EOL, cpus, release, totalmem, freemem, endianness };
  `,
  "fs": `
    var F = globalThis.fs || {};
    export var readFile = F.readFile || ${throwFn("fs.readFile")};
    export var writeFile = F.writeFile || ${throwFn("fs.writeFile")};
    export var appendFile = F.appendFile || ${throwFn("fs.appendFile")};
    export var readdir = F.readdir || ${throwFn("fs.readdir")};
    export var stat = F.stat || ${throwFn("fs.stat")};
    export var lstat = F.lstat || ${throwFn("fs.lstat")};
    export var access = F.access || ${throwFn("fs.access")};
    export var mkdir = F.mkdir || ${throwFn("fs.mkdir")};
    export var unlink = F.unlink || ${throwFn("fs.unlink")};
    export var rm = F.rm || ${throwFn("fs.rm")};
    export var rename = F.rename || ${throwFn("fs.rename")};
    export var copyFile = F.copyFile || ${throwFn("fs.copyFile")};
    export var realpath = F.realpath || ${throwFn("fs.realpath")};
    export var readFileSync = (F.readFileSync && F.readFileSync.bind(F)) || ${throwFn("fs.readFileSync")};
    export var writeFileSync = (F.writeFileSync && F.writeFileSync.bind(F)) || ${throwFn("fs.writeFileSync")};
    export var existsSync = (F.existsSync && F.existsSync.bind(F)) || function() { return false; };
    export var realpathSync = (F.realpathSync && F.realpathSync.bind(F)) || function(p) { return p; };
    export var mkdirSync = (F.mkdirSync && F.mkdirSync.bind(F)) || ${throwFn("fs.mkdirSync")};
    export var renameSync = (F.renameSync && F.renameSync.bind(F)) || ${throwFn("fs.renameSync")};
    export var rmSync = (F.rmSync && F.rmSync.bind(F)) || ${throwFn("fs.rmSync")};
    export var readdirSync = (F.readdirSync && F.readdirSync.bind(F)) || function() { return []; };
    export var statSync = (F.statSync && F.statSync.bind(F)) || function() { return { isFile: function() { return false; }, isDirectory: function() { return false; }, size: 0 }; };
    export var appendFileSync = (F.appendFileSync && F.appendFileSync.bind(F)) || ${throwFn("fs.appendFileSync")};
    export var createReadStream = (F.createReadStream && F.createReadStream.bind(F)) || ${throwFn("fs.createReadStream")};
    export var createWriteStream = (F.createWriteStream && F.createWriteStream.bind(F)) || ${throwFn("fs.createWriteStream")};
    export var promises = F.promises || {};
    export var constants = { F_OK: 0, R_OK: 4, W_OK: 2, X_OK: 1 };
    export default { readFile, writeFile, appendFile, readdir, stat, lstat, access, mkdir, unlink, rm, rename, copyFile, realpath, readFileSync, writeFileSync, existsSync, realpathSync, mkdirSync, renameSync, rmSync, readdirSync, statSync, appendFileSync, createReadStream, createWriteStream, promises, constants };
  `,
  "fs/promises": `
    // Prefer the fully-implemented globalThis.fs.promises surface
    // installed by internal/jsbridge/fs.go. Fall back to
    // top-level globalThis.fs (also populated with promisified
    // methods), then to a throwFn for truly unavailable ops.
    var F = (globalThis.fs && globalThis.fs.promises) || globalThis.fs || {};
    var T = globalThis.fs || {};
    export var readFile = F.readFile || T.readFile || ${throwFn("fs.readFile")};
    export var writeFile = F.writeFile || T.writeFile || ${throwFn("fs.writeFile")};
    export var readdir = F.readdir || T.readdir || ${throwFn("fs.readdir")};
    export var stat = F.stat || T.stat || ${throwFn("fs.stat")};
    export var lstat = F.lstat || T.lstat || ${throwFn("fs.lstat")};
    export var mkdir = F.mkdir || T.mkdir || ${throwFn("fs.mkdir")};
    export var mkdtemp = F.mkdtemp || T.mkdtemp || ${throwFn("fs.mkdtemp")};
    export var rmdir = F.rmdir || T.rmdir || ${throwFn("fs.rmdir")};
    export var rm = F.rm || T.rm || ${throwFn("fs.rm")};
    export var unlink = F.unlink || T.unlink || ${throwFn("fs.unlink")};
    export var access = F.access || T.access || ${throwFn("fs.access")};
    export var copyFile = F.copyFile || T.copyFile || ${throwFn("fs.copyFile")};
    export var rename = F.rename || T.rename || ${throwFn("fs.rename")};
    export var realpath = F.realpath || T.realpath || ${throwFn("fs.realpath")};
    export var appendFile = F.appendFile || T.appendFile || ${throwFn("fs.appendFile")};
    export var symlink = F.symlink || T.symlink || ${throwFn("fs.symlink")};
    export var readlink = F.readlink || T.readlink || ${throwFn("fs.readlink")};
    export var chmod = F.chmod || T.chmod || ${throwFn("fs.chmod")};
    export var chown = F.chown || T.chown || ${throwFn("fs.chown")};
    export var truncate = F.truncate || T.truncate || ${throwFn("fs.truncate")};
    export var utimes = F.utimes || T.utimes || ${throwFn("fs.utimes")};
    export default { readFile, writeFile, readdir, stat, lstat, mkdir, mkdtemp, rmdir, rm, unlink, access, copyFile, rename, realpath, appendFile, symlink, readlink, chmod, chown, truncate, utimes };
  `,
  "url": `
    export var URL = globalThis.URL;
    export var URLSearchParams = globalThis.URLSearchParams;
    export var fileURLToPath = function(u) { return typeof u === "string" ? u.replace("file://", "") : String(u); };
    export var pathToFileURL = function(p) { return new URL("file://" + p); };
    // node:url legacy "format" / "parse" API — only shape-
    // compliant enough to keep node-fetch (pulled in by
    // @google/genai → GeminiLiveVoice) happy. Full API lives
    // on the WHATWG URL constructor; consumers doing anything
    // beyond the basics should use that directly.
    export var format = function(urlObj) {
      if (typeof urlObj === "string") return urlObj;
      if (urlObj instanceof URL) return urlObj.toString();
      var u = String(urlObj.protocol || "http:") + (urlObj.slashes === false ? "" : "//") +
              String(urlObj.host || (urlObj.hostname || "") + (urlObj.port ? ":" + urlObj.port : "")) +
              String(urlObj.pathname || "") + String(urlObj.search || "") + String(urlObj.hash || "");
      return u;
    };
    export var parse = function(s) {
      try { var u = new URL(s); return { protocol: u.protocol, host: u.host, hostname: u.hostname, port: u.port, pathname: u.pathname, search: u.search, hash: u.hash, href: u.href, path: u.pathname + u.search }; }
      catch (_) { return { href: String(s) }; }
    };
    export var resolve = function(base, rel) { try { return new URL(rel, base).toString(); } catch(_) { return rel; } };
    export default { URL: globalThis.URL, URLSearchParams: globalThis.URLSearchParams, fileURLToPath, pathToFileURL, format, parse, resolve };
  `,
  "process": `
    var _p = globalThis.process || {};
    export var env = _p.env || {};
    export var version = _p.version || "v20.0.0";
    export var versions = _p.versions || {};
    export var platform = _p.platform || "linux";
    export var arch = _p.arch || "x64";
    export var pid = _p.pid || 1;
    export var argv = _p.argv || [];
    export var cwd = _p.cwd || function() { return "/"; };
    export var nextTick = _p.nextTick || function(fn) { queueMicrotask(fn); };
    export var stdout = _p.stdout || { write: function() { return true; } };
    export var stderr = _p.stderr || { write: function() { return true; } };
    export default _p;
  `,
  "util": `
    export var promisify = function(fn) { return function() { var args = Array.prototype.slice.call(arguments); return new Promise(function(resolve, reject) { args.push(function(err, result) { if (err) reject(err); else resolve(result); }); fn.apply(null, args); }); }; };
    export var inherits = function(ctor, superCtor) { ctor.prototype = Object.create(superCtor.prototype); ctor.prototype.constructor = ctor; };
    export var deprecate = function(fn) { return fn; };
    export var types = { isUint8Array: function(v) { return v instanceof Uint8Array; }, isDate: function(v) { return v instanceof Date || (v !== null && typeof v === "object" && typeof v.getTime === "function" && typeof v.toISOString === "function"); }, isArrayBuffer: function(v) { return v instanceof ArrayBuffer; }, isRegExp: function(v) { return v instanceof RegExp; }, isMap: function(v) { return v instanceof Map; }, isSet: function(v) { return v instanceof Set; }, isTypedArray: function(v) { return ArrayBuffer.isView(v) && !(v instanceof DataView); } };
    export var inspect = function(v, opts) { try { return JSON.stringify(v, null, opts && opts.compact === false ? 2 : undefined) || String(v); } catch(e) { return String(v); } };
    export var format = function(fmt) { var args = Array.prototype.slice.call(arguments, 1); var i = 0; return String(fmt).replace(/%[sdj%]/g, function(m) { if (m === "%%") return "%"; if (i >= args.length) return m; return String(args[i++]); }); };
    export var TextEncoder = globalThis.TextEncoder;
    export var TextDecoder = globalThis.TextDecoder;
    export default { promisify, inherits, deprecate, types, inspect, format, TextEncoder, TextDecoder };
  `,
  "util/types": `
    export var isUint8Array = function(v) { return v instanceof Uint8Array; };
    export var isArrayBuffer = function(v) { return v instanceof ArrayBuffer; };
    export var isDate = function(v) { return v instanceof Date || (v !== null && typeof v === "object" && typeof v.getTime === "function" && typeof v.toISOString === "function"); };
    export var isRegExp = function(v) { return v instanceof RegExp; };
    export var isMap = function(v) { return v instanceof Map; };
    export var isSet = function(v) { return v instanceof Set; };
    export var isTypedArray = function(v) { return ArrayBuffer.isView(v) && !(v instanceof DataView); };
    export default { isUint8Array, isArrayBuffer, isDate, isRegExp, isMap, isSet, isTypedArray };
  `,
  "child_process": `
    var CP = globalThis.child_process || {};
    export var exec = CP.exec || ${throwFn("child_process.exec")};
    export var spawn = CP.spawn || ${throwFn("child_process.spawn")};
    export var execSync = CP.execSync || ${throwFn("child_process.execSync")};
    export var execFile = CP.exec || ${throwFn("child_process.execFile")};
    export var execFileSync = CP.execFileSync || ${throwFn("child_process.execFileSync")};
    export var spawnSync = CP.spawnSync || ${throwFn("child_process.spawnSync")};
    export default { exec, spawn, execSync, execFile, execFileSync, spawnSync };
  `,
  "http": `
    export var createServer = ${throwFn("http.createServer")};
    export var request = ${throwFn("http.request")};
    export var get = ${throwFn("http.get")};
    export var Agent = class Agent { constructor() {} };
    export var globalAgent = new Agent();
    export var METHODS = ["GET","HEAD","POST","PUT","DELETE","CONNECT","OPTIONS","TRACE","PATCH"];
    export var STATUS_CODES = { 200:"OK", 201:"Created", 204:"No Content", 301:"Moved Permanently", 302:"Found", 304:"Not Modified", 400:"Bad Request", 401:"Unauthorized", 403:"Forbidden", 404:"Not Found", 500:"Internal Server Error" };
    export default { createServer, request, get, Agent, globalAgent, METHODS, STATUS_CODES };
  `,
  "https": `
    export var createServer = ${throwFn("https.createServer")};
    export var request = ${throwFn("https.request")};
    export var get = ${throwFn("https.get")};
    export var Agent = class Agent { constructor() {} };
    export var globalAgent = new Agent();
    export default { createServer, request, get, Agent, globalAgent };
  `,
  "assert": `
    export default function assert(val, msg) { if (!val) throw new Error(msg || "Assertion failed"); };
    export var ok = function(val, msg) { if (!val) throw new Error(msg || "Assertion failed"); };
    export var strictEqual = function(a, b, msg) { if (a !== b) throw new Error(msg || a + " !== " + b); };
    export var deepStrictEqual = function(a, b, msg) { if (JSON.stringify(a) !== JSON.stringify(b)) throw new Error(msg || "Not deeply equal"); };
    export var throws = function(fn, msg) { try { fn(); throw new Error(msg || "Expected to throw"); } catch(e) {} };
    export var fail = function(msg) { throw new Error(msg || "Assertion failed"); };
  `,
  "querystring": `
    export var parse = function(str) { var obj = {}; (str || "").split("&").forEach(function(pair) { if (!pair) return; var kv = pair.split("="); obj[decodeURIComponent(kv[0])] = decodeURIComponent(kv.slice(1).join("=")); }); return obj; };
    export var stringify = function(obj) { return Object.entries(obj || {}).map(function(kv) { return encodeURIComponent(kv[0]) + "=" + encodeURIComponent(kv[1]); }).join("&"); };
    export var encode = stringify;
    export var decode = parse;
    export default { parse, stringify, encode, decode };
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
    export var performance = globalThis.performance || { now: function() { return Date.now(); }, timeOrigin: Date.now() };
    export var PerformanceObserver = class PerformanceObserver { constructor() {} observe() {} disconnect() {} };
    export var monitorEventLoopDelay = function() { return { enable: function() {}, disable: function() {}, percentile: function() { return 0; }, min: 0, max: 0, mean: 0, stddev: 0 }; };
    export default { performance, PerformanceObserver, monitorEventLoopDelay };
  `,
  "timers": `
    export var setTimeout = globalThis.setTimeout;
    export var clearTimeout = globalThis.clearTimeout;
    export var setInterval = globalThis.setInterval;
    export var clearInterval = globalThis.clearInterval;
    export var setImmediate = globalThis.setImmediate || function(fn) { return globalThis.setTimeout(fn, 0); };
    export var clearImmediate = globalThis.clearImmediate || function() {};
    export default { setTimeout, clearTimeout, setInterval, clearInterval, setImmediate, clearImmediate };
  `,
  "timers/promises": `
    export var setTimeout = function(ms) { return new Promise(function(r) { globalThis.setTimeout(r, ms); }); };
    export var setInterval = function() { return { [Symbol.asyncIterator]: function() { return { next: function() { return new Promise(function() {}); } }; } }; };
    export default { setTimeout, setInterval };
  `,
  "module": `
    export var createRequire = function() { return globalThis.require || function() { return {}; }; };
    export default { createRequire };
  `,
  "dns": `
    var D = globalThis.dns || {};
    export var lookup = D.lookup || ${throwFn("dns.lookup")};
    export var resolve4 = D.resolve4 || ${throwFn("dns.resolve4")};
    export var Resolver = D.Resolver || class Resolver {};
    export var promises = D.promises || { lookup: ${throwFn("dns.lookup")}, resolve4: ${throwFn("dns.resolve4")} };
    export var ADDRCONFIG = D.ADDRCONFIG || 0;
    export var V4MAPPED = D.V4MAPPED || 0;
    export var NODATA = D.NODATA || "ENODATA";
    export var NOTFOUND = D.NOTFOUND || "ENOTFOUND";
    export var TIMEOUT = D.TIMEOUT || "ETIMEOUT";
    export default { lookup, resolve4, Resolver, promises, ADDRCONFIG, V4MAPPED, NODATA, NOTFOUND, TIMEOUT };
  `,
  "async_hooks": `
    export var createHook = function() { return { enable: function() {}, disable: function() {} }; };
    export var executionAsyncId = function() { return 0; };
    export var triggerAsyncId = function() { return 0; };
    export var executionAsyncResource = function() { return {}; };
    export class AsyncLocalStorage {
      constructor() { this._store = undefined; }
      getStore() { return this._store; }
      run(store, fn) { var prev = this._store; this._store = store; try { var args = Array.prototype.slice.call(arguments, 2); return fn.apply(null, args); } finally { this._store = prev; } }
      enterWith(store) { this._store = store; }
      disable() { this._store = undefined; }
    }
    export class AsyncResource {
      constructor(type) { this.type = type; }
      runInAsyncScope(fn, thisArg) { var args = Array.prototype.slice.call(arguments, 2); return fn.apply(thisArg, args); }
      emitDestroy() { return this; }
      asyncId() { return 0; }
      triggerAsyncId() { return 0; }
    }
    export default { createHook, executionAsyncId, triggerAsyncId, executionAsyncResource, AsyncLocalStorage, AsyncResource };
  `,
  "diagnostics_channel": `
    var _noop = function() {};
    var _ch = function() { return { subscribe: _noop, unsubscribe: _noop, publish: _noop, hasSubscribers: false, bindStore: _noop, runStores: _noop }; };
    export var channel = _ch;
    export var tracingChannel = function(name) { return { start: _ch(), end: _ch(), asyncStart: _ch(), asyncEnd: _ch(), error: _ch(), subscribe: _noop, unsubscribe: _noop, hasSubscribers: false }; };
    export var hasSubscribers = function() { return false; };
    export var subscribe = _noop;
    export var unsubscribe = _noop;
    export class Channel { constructor() { this.hasSubscribers = false; } subscribe() {} unsubscribe() {} publish() {} bindStore() {} runStores() {} }
    export default { channel, hasSubscribers, subscribe, unsubscribe, Channel };
  `,
  "worker_threads": `
    export var isMainThread = true;
    export var parentPort = null;
    export var workerData = undefined;
    export var threadId = 0;
    export class Worker { constructor() { throw new Error("Worker threads not available in QuickJS"); } }
    export class MessageChannel { constructor() { this.port1 = {}; this.port2 = {}; } }
    export class MessagePort {}
    export default { isMainThread, parentPort, workerData, threadId, Worker, MessageChannel, MessagePort };
  `,
  "zlib": `
    var Z = globalThis.zlib || {};
    export var createGzip = Z.createGzip || ${throwFn("createGzip")};
    export var createGunzip = Z.createGunzip || ${throwFn("createGunzip")};
    export var createDeflate = Z.createDeflate || ${throwFn("createDeflate")};
    export var createInflate = Z.createInflate || ${throwFn("createInflate")};
    export var gzip = Z.gzip || ${throwFn("gzip")};
    export var gunzip = Z.gunzip || ${throwFn("gunzip")};
    export var deflate = Z.deflate || ${throwFn("deflate")};
    export var inflate = Z.inflate || ${throwFn("inflate")};
    export var gzipSync = Z.gzipSync || ${throwFn("gzipSync")};
    export var gunzipSync = Z.gunzipSync || ${throwFn("gunzipSync")};
    export var deflateSync = Z.deflateSync || ${throwFn("deflateSync")};
    export var inflateSync = Z.inflateSync || ${throwFn("inflateSync")};
    export var inflateRaw = Z.inflateRaw || ${throwFn("inflateRaw")};
    export var deflateRaw = Z.deflateRaw || ${throwFn("deflateRaw")};
    export var inflateRawSync = Z.inflateRawSync || ${throwFn("inflateRawSync")};
    export var deflateRawSync = Z.deflateRawSync || ${throwFn("deflateRawSync")};
    export var brotliCompressSync = ${throwFn("brotliCompressSync")};
    export var brotliDecompressSync = ${throwFn("brotliDecompressSync")};
    export var constants = Z.constants || {};
    export default { createGzip, createGunzip, createDeflate, createInflate, gzip, gunzip, deflate, inflate,
      gzipSync, gunzipSync, deflateSync, inflateSync, inflateRaw, deflateRaw, inflateRawSync, deflateRawSync,
      brotliCompressSync, brotliDecompressSync, constants };
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
  target: "esnext",
  minify: true,
  treeShaking: true,
  plugins: [
    nodeStubPlugin,
    // Redirect `ws` (Node WebSocket) to the jsbridge polyfill.
    // @mastra/voice-openai-realtime does `import { WebSocket } from 'ws'`;
    // without this alias esbuild resolves ws's browser field to an
    // empty module and `new WebSocket(...)` throws "not a function".
    {
      name: "ws-alias",
      setup(build) {
        build.onResolve({ filter: /^ws$/ }, () => ({
          path: "ws-polyfill",
          namespace: "ws-polyfill-ns",
        }));
        build.onLoad({ filter: /.*/, namespace: "ws-polyfill-ns" }, () => ({
          contents: `
const WS = globalThis.WebSocket;
export { WS as WebSocket };
export default WS;
`,
          loader: "js",
        }));
      },
    },
    // Force lru-cache to use CJS build — ESM version uses top-level await
    // which esbuild can't bundle in IIFE format. CJS version works fine.
    {
      name: "lru-cache-cjs",
      setup(build) {
        build.onResolve({ filter: /^lru-cache$/ }, (args) => {
          return {
            path: import.meta.dirname + "/node_modules/lru-cache/dist/commonjs/index.js",
          };
        });
      },
    },
    // Replace big.js with a Number-backed shim. The real library
    // triggers a native SIGBUS inside QuickJS when rerank/* call
    // `new Big(0).plus(...)` — see
    // ../../brainkit-maps/knowledge/rerank-sigbus-bigjs.md.
    // Mastra only uses Big for rerank weight validation (3 default
    // weights summing to 1.0), so Number precision is sufficient.
    {
      name: "big-js-shim",
      setup(build) {
        build.onResolve({ filter: /^big\.js$/ }, () => ({
          path: "big-js-shim",
          namespace: "big-js-shim-ns",
        }));
        build.onLoad({ filter: /.*/, namespace: "big-js-shim-ns" }, () => ({
          contents: `
function Big(n) {
  if (!(this instanceof Big)) return new Big(n);
  if (n instanceof Big) { this.v = n.v; return; }
  this.v = typeof n === "number" ? n : parseFloat(String(n));
}
Big.prototype.plus = function(w) {
  var o = w instanceof Big ? w.v : (typeof w === "number" ? w : parseFloat(String(w)));
  return new Big(this.v + o);
};
Big.prototype.minus = function(w) {
  var o = w instanceof Big ? w.v : (typeof w === "number" ? w : parseFloat(String(w)));
  return new Big(this.v - o);
};
Big.prototype.times = function(w) {
  var o = w instanceof Big ? w.v : (typeof w === "number" ? w : parseFloat(String(w)));
  return new Big(this.v * o);
};
Big.prototype.div = function(w) {
  var o = w instanceof Big ? w.v : (typeof w === "number" ? w : parseFloat(String(w)));
  return new Big(this.v / o);
};
Big.prototype.eq = function(w) {
  var o = w instanceof Big ? w.v : (typeof w === "number" ? w : parseFloat(String(w)));
  return Math.abs(this.v - o) < 1e-9;
};
Big.prototype.cmp = function(w) {
  var o = w instanceof Big ? w.v : (typeof w === "number" ? w : parseFloat(String(w)));
  if (Math.abs(this.v - o) < 1e-9) return 0;
  return this.v < o ? -1 : 1;
};
Big.prototype.gt = function(w) { return this.cmp(w) > 0; };
Big.prototype.gte = function(w) { return this.cmp(w) >= 0; };
Big.prototype.lt = function(w) { return this.cmp(w) < 0; };
Big.prototype.lte = function(w) { return this.cmp(w) <= 0; };
Big.prototype.toString = function() { return String(this.v); };
Big.prototype.valueOf = function() { return this.v; };
Big.prototype.toNumber = function() { return this.v; };
Big.prototype.toFixed = function(dp) { return this.v.toFixed(dp); };
Big.DP = 20;
Big.RM = 1;
Big.roundDown = 0;
Big.roundHalfUp = 1;
Big.roundHalfEven = 2;
Big.roundUp = 3;
export { Big };
export default Big;
`,
          loader: "js",
        }));
      },
    },
    // Redirect EXACT 'zod' imports to 'zod/v4' so all code uses ONE Zod version.
    {
      name: "zod-unify",
      setup(build) {
        build.onResolve({ filter: /^zod$/ }, (args) => {
          return build.resolve("zod/v4", {
            resolveDir: args.resolveDir,
            kind: args.kind,
            importer: args.importer,
          });
        });
      },
    },
    // Force vscode-jsonrpc/node to use the Node.js version, not browser.
    {
      name: "vscode-jsonrpc-node",
      setup(build) {
        build.onResolve({ filter: /^vscode-jsonrpc\/node$/ }, (args) => {
          return {
            path: import.meta.dirname + "/node_modules/vscode-jsonrpc/lib/node/main.js",
          };
        });
      },
    },
  ],
  external: [
    "better-sqlite3",
    "@ast-grep/napi",
    "fastembed",
    // `@opentelemetry/api` is now bundled in — the OTel span
    // processors we expose (Batch/Simple/Noop + samplers) need it at
    // runtime, and there's no host-side injection path in brainkit.
    "hono",
  ],
  banner: {
    js: `var __zod_v4_deferred = null;`,
  },
  define: {
    "process.env.NODE_ENV": '"production"',
  },
  logLevel: "info",
  metafile: true,
  outfile: "../agent_embed_bundle.js",
});

// Post-process: patch bundled library code for QuickJS compatibility.
// These are library-specific fixes, NOT polyfill concerns.
import { readFileSync } from "node:fs";
{
  let bundle = readFileSync("../agent_embed_bundle.js", "utf8");

  // Patch @libsql/client value serializer: handle undefined → null
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

  // Patch getExeca() to use __execa_polyfill instead of dynamic import("execa")
  const execaPattern = /try\{let (\w+)=\(await import\("execa"\)\)\.execa;return (\w+)=\1,\1\}/;
  const execaMatch = bundle.match(execaPattern);
  if (execaMatch) {
    const v = execaMatch[1], cached = execaMatch[2];
    const fix = `try{let ${v}=globalThis.__execa_polyfill;if(!${v})throw new Error("no execa");return ${cached}=${v},${v}}`;
    bundle = bundle.replace(execaMatch[0], fix);
    writeFileSync("../agent_embed_bundle.js", bundle);
    console.log(`Patched getExeca: uses __execa_polyfill`);
  }

  // Patch @mastra/rag validation: z.function() → z.any() (Zod v4 compat).
  // The minifier picks a different short identifier per rebuild, so match
  // whatever name it used this pass.
  const funcPattern = /lengthFunction:([a-zA-Z_$][a-zA-Z_$0-9]*)\.optional\(\1\.function\(\)\)/;
  const funcMatch = bundle.match(funcPattern);
  if (funcMatch) {
    const v = funcMatch[1];
    const replacement = `lengthFunction:${v}.optional(${v}.any())`;
    bundle = bundle.replace(funcPattern, replacement);
    writeFileSync("../agent_embed_bundle.js", bundle);
    console.log(`Patched: z.function() → z.any() for RAG validation (Zod v4 compat, minifier used '${v}')`);
  }

  // Patch getTiktoken() to use getEncoding('o200k_base') with fallback
  const tiktokenFnPattern = /async function (\w+)\(\)\{let e=globalThis\[(\w+)\];if\(e\)return e;/;
  const tiktokenFnMatch = bundle.match(tiktokenFnPattern);
  if (tiktokenFnMatch) {
    const fn = tiktokenFnMatch[1];
    const key = tiktokenFnMatch[2];
    const fnStart = tiktokenFnMatch.index;
    let depth = 0, fnEnd = fnStart;
    for (let i = fnStart; i < Math.min(fnStart + 500, bundle.length); i++) {
      if (bundle[i] === '{') depth++;
      else if (bundle[i] === '}') { depth--; if (depth === 0) { fnEnd = i + 1; break; } }
    }
    const oldFn = bundle.substring(fnStart, fnEnd);
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
