import * as esbuild from "esbuild";
import { builtinModules } from "node:module";
import { readFileSync, statSync, writeFileSync } from "node:fs";

// Node.js built-ins that Mastra imports.
// We stub them at build time so esbuild can resolve named imports.
// At runtime in QuickJS, jsbridge polyfills provide the real implementations
// on globalThis (stream, crypto, net, os, Buffer, etc.).
// These stubs are THIN RE-EXPORTS — no logic, just wiring for esbuild resolution.

const compatManifest = JSON.parse(
  readFileSync(new URL("./compat/manifest.json", import.meta.url), "utf8"),
);
const compatEntries = compatManifest.entries || [];
const nodeCompatEntries = new Map(
  compatEntries
    .filter((entry) => entry.kind === "node-module" || entry.kind === "node-subpath")
    .map((entry) => [entry.id, entry]),
);
const nodeCompatAliases = new Map();
for (const entry of compatEntries) {
  for (const alias of entry.aliases || []) {
    nodeCompatAliases.set(alias, entry.id);
  }
}
const packagePatchEntries = new Map(
  compatEntries
    .filter((entry) => entry.kind === "package-patch")
    .map((entry) => [entry.id, entry]),
);
const nativeNodeBuiltins = new Set(
  builtinModules.map((id) => id.startsWith("node:") ? id.slice(5) : id),
);

function isNodeBuiltin(id) {
  if (id.startsWith("node:")) return true;
  const normalized = normalizeId(id);
  if (id.endsWith("/") && normalized === id) {
    return false;
  }
  if (nativeNodeBuiltins.has(normalized)) return true;
  const base = normalized.split("/")[0];
  return nativeNodeBuiltins.has(base);
}

function normalizeId(id) {
  let normalized = id.startsWith("node:") ? id.slice(5) : id;
  if (nodeCompatAliases.has(normalized)) {
    return nodeCompatAliases.get(normalized);
  }
  if (normalized.endsWith("/") && nodeCompatEntries.has(normalized.slice(0, -1))) {
    return normalized.slice(0, -1);
  }
  return normalized;
}

function requirePackagePatch(id) {
  const entry = packagePatchEntries.get(id);
  if (!entry) {
    throw new Error(`package patch ${id} is missing from compat/manifest.json`);
  }
  if (typeof entry.required !== "boolean") {
    throw new Error(`package patch ${id} is missing required=true/false in compat/manifest.json`);
  }
  if (!entry.package || !entry.versionRange || !entry.patchType || !entry.expected || !entry.removalCondition) {
    throw new Error(`package patch ${id} is missing package, versionRange, patchType, expected, or removalCondition metadata in compat/manifest.json`);
  }
  return id;
}

function getPackagePatch(id) {
  requirePackagePatch(id);
  return packagePatchEntries.get(id);
}

function packagePatchApplied(id, detail) {
  getPackagePatch(id);
  console.log(`Package patch ${id} applied: ${detail}`);
}

function packagePatchInstalledVersion(entry) {
  try {
    const pkg = JSON.parse(readFileSync(new URL(`./node_modules/${entry.package}/package.json`, import.meta.url), "utf8"));
    return pkg.version || "unknown";
  } catch (_) {
    return "unknown";
  }
}

function packagePatchMissed(id, detail) {
  const entry = getPackagePatch(id);
  const installed = packagePatchInstalledVersion(entry);
  const message = [
    `Package patch ${id} not applied: ${detail}.`,
    `Package: ${entry.package}@${installed}.`,
    `Version range: ${entry.versionRange}.`,
    `Patch type: ${entry.patchType}.`,
    `Expected: ${entry.expected}.`,
    `Removal condition: ${entry.removalCondition}.`,
  ].join(" ");
  if (entry.required) {
    throw new Error(message);
  }
  console.warn(`Optional ${message}`);
}

const throwFn = (name) => `function() { throw new Error("${name}: not available in QuickJS"); }`;

// ─── Module stubs: thin re-exports from jsbridge polyfills on globalThis ──
const moduleStubs = {
  "crypto": `
    var C = globalThis.crypto;
    export var randomUUID = C.randomUUID;
    export var randomBytes = C.randomBytes;
    export var randomFillSync = C.randomFillSync;
    export var randomInt = C.randomInt;
    export var createHash = C.createHash;
    export var createHmac = C.createHmac;
    export var pbkdf2 = C.pbkdf2;
    export var pbkdf2Sync = C.pbkdf2Sync;
    export var timingSafeEqual = C.timingSafeEqual;
    export var getHashes = C.getHashes;
    export var getCiphers = C.getCiphers;
    export var getFips = C.getFips;
    export var createCipheriv = C.createCipheriv;
    export var createDecipheriv = C.createDecipheriv;
    export var createSign = C.createSign;
    export var createVerify = C.createVerify;
    export var scrypt = C.scrypt;
    export var scryptSync = C.scryptSync;
    export var constants = {};
    export var webcrypto = globalThis.crypto;
    export default { randomUUID, randomBytes, randomFillSync, randomInt, createHash, createHmac,
      createCipheriv, createDecipheriv, createSign, createVerify, pbkdf2, pbkdf2Sync,
      scrypt, scryptSync, timingSafeEqual, constants, webcrypto, getHashes, getCiphers, getFips };
  `,
  "stream": `
    var S = globalThis.stream;
    export var EventEmitter = S.EventEmitter;
    export var Readable = S.Readable;
    export var Writable = S.Writable;
    export var Duplex = S.Duplex;
    export var Transform = S.Transform;
    export var PassThrough = S.PassThrough;
    export var pipeline = S.pipeline;
    export var finished = S.finished;
    export var Stream = S.Stream;
    Stream.EventEmitter = EventEmitter;
    Stream.Readable = Readable;
    Stream.Writable = Writable;
    Stream.Duplex = Duplex;
    Stream.Transform = Transform;
    Stream.PassThrough = PassThrough;
    Stream.pipeline = pipeline;
    Stream.finished = finished;
    Stream.Stream = Stream;
    export default Stream;
  `,
  "stream/web": `
    export var ReadableStream = globalThis.ReadableStream;
    export var WritableStream = globalThis.WritableStream;
    export var TransformStream = globalThis.TransformStream;
    export default { ReadableStream, WritableStream, TransformStream };
  `,
  "stream/promises": `
    var P = globalThis.stream.promises;
    export var pipeline = P.pipeline;
    export var finished = P.finished;
    export default P;
  `,
  "net": `
    var N = globalThis.net;
    export var Socket = N.Socket;
    export var createConnection = N.createConnection;
    export var connect = N.connect;
    export var createServer = N.createServer;
    export var Server = N.Server;
    export var isIP = N.isIP;
    export var isIPv4 = N.isIPv4;
    export var isIPv6 = N.isIPv6;
    export default { Socket, createConnection, connect, createServer, Server, isIP, isIPv4, isIPv6 };
  `,
  "tls": `
    var T = globalThis.tls;
    export var createServer = T.createServer;
    export var connect = T.connect;
    export var TLSSocket = T.TLSSocket;
    export var DEFAULT_ECDH_CURVE = T.DEFAULT_ECDH_CURVE;
    export var DEFAULT_MIN_VERSION = T.DEFAULT_MIN_VERSION;
    export var DEFAULT_MAX_VERSION = T.DEFAULT_MAX_VERSION;
    export default T;
  `,
  "buffer": `
    export var Buffer = globalThis.Buffer;
    export default { Buffer };
  `,
  "events": `
    export var EventEmitter = globalThis.EventEmitter;
    export default EventEmitter;
  `,
  "path": `
    var P = globalThis.path;
    export var join = P.join;
    export var resolve = P.resolve;
    export var dirname = P.dirname;
    export var basename = P.basename;
    export var extname = P.extname;
    export var normalize = P.normalize;
    export var isAbsolute = P.isAbsolute;
    export var parse = P.parse;
    export var relative = P.relative;
    export var sep = P.sep;
    export var delimiter = P.delimiter;
    export var posix = P;
    export default { join, resolve, dirname, basename, extname, normalize, isAbsolute, parse, relative, sep, delimiter, posix };
  `,
  "path/posix": `
    var P = globalThis.path.posix;
    export var join = P.join;
    export var resolve = P.resolve;
    export var dirname = P.dirname;
    export var basename = P.basename;
    export var extname = P.extname;
    export var sep = P.sep;
    export default { join, resolve, dirname, basename, extname, sep };
  `,
  "os": `
    var O = globalThis.os;
    export var platform = O.platform;
    export var arch = O.arch;
    export var tmpdir = O.tmpdir;
    export var homedir = O.homedir;
    export var hostname = O.hostname;
    export var type = O.type;
    export var EOL = O.EOL;
    export var cpus = O.cpus;
    export var release = O.release;
    export var totalmem = O.totalmem;
    export var freemem = O.freemem;
    export var endianness = O.endianness;
    export default { platform, arch, tmpdir, homedir, hostname, type, EOL, cpus, release, totalmem, freemem, endianness };
  `,
  "fs": `
    var F = globalThis.fs;
    export var readFile = F.readFile;
    export var writeFile = F.writeFile;
    export var appendFile = F.appendFile;
    export var readdir = F.readdir;
    export var stat = F.stat;
    export var lstat = F.lstat;
    export var access = F.access;
    export var mkdir = F.mkdir;
    export var unlink = F.unlink;
    export var rm = F.rm;
    export var rename = F.rename;
    export var copyFile = F.copyFile;
    export var realpath = F.realpath;
    export var readFileSync = F.readFileSync.bind(F);
    export var writeFileSync = F.writeFileSync.bind(F);
    export var existsSync = F.existsSync.bind(F);
    export var realpathSync = F.realpathSync.bind(F);
    export var mkdirSync = F.mkdirSync.bind(F);
    export var renameSync = F.renameSync.bind(F);
    export var rmSync = F.rmSync.bind(F);
    export var readdirSync = F.readdirSync.bind(F);
    export var statSync = F.statSync.bind(F);
    export var lstatSync = F.lstatSync.bind(F);
    export var unlinkSync = F.unlinkSync.bind(F);
    export var appendFileSync = F.appendFileSync.bind(F);
    export var createReadStream = F.createReadStream.bind(F);
    export var createWriteStream = F.createWriteStream.bind(F);
    export var promises = F.promises;
    export var constants = F.constants;
    export default { readFile, writeFile, appendFile, readdir, stat, lstat, access, mkdir, unlink, rm, rename, copyFile, realpath, readFileSync, writeFileSync, existsSync, realpathSync, mkdirSync, renameSync, rmSync, readdirSync, statSync, lstatSync, unlinkSync, appendFileSync, createReadStream, createWriteStream, promises, constants };
  `,
  "fs/promises": `
    var F = globalThis.fs.promises;
    export var readFile = F.readFile;
    export var writeFile = F.writeFile;
    export var readdir = F.readdir;
    export var stat = F.stat;
    export var lstat = F.lstat;
    export var mkdir = F.mkdir;
    export var mkdtemp = F.mkdtemp;
    export var rmdir = F.rmdir;
    export var rm = F.rm;
    export var unlink = F.unlink;
    export var access = F.access;
    export var copyFile = F.copyFile;
    export var rename = F.rename;
    export var realpath = F.realpath;
    export var appendFile = F.appendFile;
    export var symlink = F.symlink;
    export var readlink = F.readlink;
    export var chmod = F.chmod;
    export var chown = F.chown;
    export var truncate = F.truncate;
    export var utimes = F.utimes;
    export default { readFile, writeFile, readdir, stat, lstat, mkdir, mkdtemp, rmdir, rm, unlink, access, copyFile, rename, realpath, appendFile, symlink, readlink, chmod, chown, truncate, utimes };
  `,
  "url": `
    var U = globalThis.node_url;
    export var URL = U.URL;
    export var URLSearchParams = U.URLSearchParams;
    export var fileURLToPath = U.fileURLToPath;
    export var pathToFileURL = U.pathToFileURL;
    export var format = U.format;
    export var parse = U.parse;
    export var resolve = U.resolve;
    export default U;
  `,
  "process": `
    var _p = globalThis.process;
    export var env = _p.env;
    export var version = _p.version;
    export var versions = _p.versions;
    export var platform = _p.platform;
    export var arch = _p.arch;
    export var pid = _p.pid;
    export var argv = _p.argv;
    export var cwd = _p.cwd;
    export var nextTick = _p.nextTick;
    export var stdout = _p.stdout;
    export var stderr = _p.stderr;
    export default _p;
  `,
  "util": `
    var U = globalThis.util;
    export var promisify = U.promisify;
    export var inherits = U.inherits;
    export var deprecate = U.deprecate;
    export var types = U.types;
    export var inspect = U.inspect;
    export var format = U.format;
    export var TextEncoder = U.TextEncoder;
    export var TextDecoder = U.TextDecoder;
    export default U;
  `,
  "util/types": `
    var T = globalThis.utilTypes;
    export var isUint8Array = T.isUint8Array;
    export var isArrayBuffer = T.isArrayBuffer;
    export var isDate = T.isDate;
    export var isRegExp = T.isRegExp;
    export var isMap = T.isMap;
    export var isSet = T.isSet;
    export var isTypedArray = T.isTypedArray;
    export default T;
  `,
  "child_process": `
    var CP = globalThis.child_process;
    export var exec = CP.exec;
    export var spawn = CP.spawn;
    export var execSync = CP.execSync;
    export var execFile = CP.execFile;
    export var execFileSync = CP.execFileSync;
    export var spawnSync = CP.spawnSync;
    export default { exec, spawn, execSync, execFile, execFileSync, spawnSync };
  `,
  "http": `
    var H = globalThis.http;
    export var createServer = H.createServer;
    export var request = H.request;
    export var get = H.get;
    export var Agent = H.Agent;
    export var globalAgent = H.globalAgent;
    export var METHODS = H.METHODS;
    export var STATUS_CODES = H.STATUS_CODES;
    export default H;
  `,
  "https": `
    var H = globalThis.https;
    export var createServer = H.createServer;
    export var request = H.request;
    export var get = H.get;
    export var Agent = H.Agent;
    export var globalAgent = H.globalAgent;
    export default H;
  `,
  "assert": `
    var A = globalThis.assert;
    export var ok = A.ok;
    export var strictEqual = A.strictEqual;
    export var deepStrictEqual = A.deepStrictEqual;
    export var throws = A.throws;
    export var fail = A.fail;
    export default A;
  `,
  "querystring": `
    var Q = globalThis.querystring;
    export var parse = Q.parse;
    export var stringify = Q.stringify;
    export var encode = Q.encode;
    export var decode = Q.decode;
    export default Q;
  `,
  "string_decoder": `
    export var StringDecoder = globalThis.StringDecoder;
    export default { StringDecoder };
  `,
  "perf_hooks": `
    var P = globalThis.perf_hooks;
    export var performance = globalThis.performance;
    export var PerformanceObserver = P.PerformanceObserver;
    export var monitorEventLoopDelay = P.monitorEventLoopDelay;
    export default P;
  `,
  "timers": `
    export var setTimeout = globalThis.setTimeout;
    export var clearTimeout = globalThis.clearTimeout;
    export var setInterval = globalThis.setInterval;
    export var clearInterval = globalThis.clearInterval;
    export var setImmediate = globalThis.setImmediate;
    export var clearImmediate = globalThis.clearImmediate;
    export default { setTimeout, clearTimeout, setInterval, clearInterval, setImmediate, clearImmediate };
  `,
  "timers/promises": `
    var P = globalThis.timersPromises;
    export var setTimeout = P.setTimeout;
    export var setInterval = P.setInterval;
    export default P;
  `,
  "module": `
    var M = globalThis.node_module;
    export var createRequire = M.createRequire;
    export default M;
  `,
  "dns": `
    var D = globalThis.dns;
    export var lookup = D.lookup;
    export var resolve4 = D.resolve4;
    export var Resolver = D.Resolver;
    export var promises = D.promises;
    export var ADDRCONFIG = D.ADDRCONFIG;
    export var V4MAPPED = D.V4MAPPED;
    export var NODATA = D.NODATA;
    export var NOTFOUND = D.NOTFOUND;
    export var TIMEOUT = D.TIMEOUT;
    export default { lookup, resolve4, Resolver, promises, ADDRCONFIG, V4MAPPED, NODATA, NOTFOUND, TIMEOUT };
  `,
  "dns/promises": `
    var P = globalThis.dns.promises;
    export var lookup = P.lookup;
    export var resolve4 = P.resolve4;
    export var resolveSrv = P.resolveSrv;
    export var resolveCname = P.resolveCname;
    export var resolvePtr = P.resolvePtr;
    export default P;
  `,
  "async_hooks": `
    var A = globalThis.async_hooks;
    export var createHook = A.createHook;
    export var executionAsyncId = A.executionAsyncId;
    export var triggerAsyncId = A.triggerAsyncId;
    export var executionAsyncResource = A.executionAsyncResource;
    export var AsyncLocalStorage = A.AsyncLocalStorage;
    export var AsyncResource = A.AsyncResource;
    export default A;
  `,
  "diagnostics_channel": `
    var D = globalThis.diagnostics_channel;
    export var channel = D.channel;
    export var tracingChannel = D.tracingChannel;
    export var hasSubscribers = D.hasSubscribers;
    export var subscribe = D.subscribe;
    export var unsubscribe = D.unsubscribe;
    export var Channel = D.Channel;
    export default D;
  `,
  "worker_threads": `
    var W = globalThis.worker_threads;
    export var isMainThread = W.isMainThread;
    export var parentPort = W.parentPort;
    export var workerData = W.workerData;
    export var threadId = W.threadId;
    export var Worker = W.Worker;
    export var MessageChannel = W.MessageChannel;
    export var MessagePort = W.MessagePort;
    export default W;
  `,
  "zlib": `
    var Z = globalThis.zlib;
    export var createGzip = Z.createGzip;
    export var createGunzip = Z.createGunzip;
    export var createDeflate = Z.createDeflate;
    export var createInflate = Z.createInflate;
    export var gzip = Z.gzip;
    export var gunzip = Z.gunzip;
    export var deflate = Z.deflate;
    export var inflate = Z.inflate;
    export var gzipSync = Z.gzipSync;
    export var gunzipSync = Z.gunzipSync;
    export var deflateSync = Z.deflateSync;
    export var inflateSync = Z.inflateSync;
    export var inflateRaw = Z.inflateRaw;
    export var deflateRaw = Z.deflateRaw;
    export var inflateRawSync = Z.inflateRawSync;
    export var deflateRawSync = Z.deflateRawSync;
    export var brotliCompressSync = Z.brotliCompressSync;
    export var brotliDecompressSync = Z.brotliDecompressSync;
    export var constants = Z.constants;
    export default { createGzip, createGunzip, createDeflate, createInflate, gzip, gunzip, deflate, inflate,
      gzipSync, gunzipSync, deflateSync, inflateSync, inflateRaw, deflateRaw, inflateRawSync, deflateRawSync,
      brotliCompressSync, brotliDecompressSync, constants };
  `,
};

for (const id of Object.keys(moduleStubs)) {
  if (!nodeCompatEntries.has(id)) {
    throw new Error(`module stub ${id} is missing from compat/manifest.json`);
  }
}
for (const id of nodeCompatEntries.keys()) {
  if (!moduleStubs[id]) {
    throw new Error(`compat manifest declares ${id} but build.mjs has no module stub`);
  }
}

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
      const entry = nodeCompatEntries.get(id);
      if (!entry) {
        return {
          errors: [{
            text: `Node builtin ${args.path} normalized to ${id} is not declared in compat/manifest.json`,
          }],
        };
      }
      const contents = moduleStubs[id];
      if (!contents) {
        return {
          errors: [{
            text: `Node builtin ${args.path} normalized to ${id} has no build stub`,
          }],
        };
      }
      return {
        contents,
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
      name: requirePackagePatch("ws-alias"),
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
      name: requirePackagePatch("lru-cache-cjs"),
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
      name: requirePackagePatch("big-js-shim"),
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
      name: requirePackagePatch("zod-unify"),
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
      name: requirePackagePatch("vscode-jsonrpc-node"),
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
{
  let bundle = readFileSync("../agent_embed_bundle.js", "utf8");

  // Patch @libsql/client value serializer: handle undefined → null
  requirePackagePatch("libsql-undefined-null");
  const oldPattern = /function (\w+)\(e\)\{if\(e===null\)return null;if\(typeof e==.?"string"\)return e;/;
  const match = bundle.match(oldPattern);
  if (match) {
    const fname = match[1];
    const old = `function ${fname}(e){if(e===null)return null;`;
    const fix = `function ${fname}(e){if(e===void 0||e===null)return null;`;
    bundle = bundle.replace(old, fix);
    writeFileSync("../agent_embed_bundle.js", bundle);
    packagePatchApplied("libsql-undefined-null", `${fname}: undefined → null in @libsql/client value serializer`);
  } else {
    packagePatchMissed("libsql-undefined-null", "could not find @libsql/client value serializer");
  }

  // Patch getExeca() to use __execa_polyfill instead of dynamic import("execa")
  requirePackagePatch("execa-dynamic-import");
  const execaPattern = /try\{let ([A-Za-z_$][A-Za-z0-9_$]*)=\(await import\("execa"\)\)\.execa;return ([A-Za-z_$][A-Za-z0-9_$]*)=\1,\1\}/;
  const execaMatch = bundle.match(execaPattern);
  if (execaMatch) {
    const v = execaMatch[1], cached = execaMatch[2];
    const fix = `try{let ${v}=globalThis.__execa_polyfill;if(!${v})throw new Error("no execa");return ${cached}=${v},${v}}`;
    bundle = bundle.replace(execaMatch[0], fix);
    writeFileSync("../agent_embed_bundle.js", bundle);
    packagePatchApplied("execa-dynamic-import", "getExeca uses __execa_polyfill");
  } else {
    packagePatchMissed("execa-dynamic-import", "could not find dynamic import(\"execa\") helper");
  }

  // Patch @mastra/rag validation: z.function() → z.any() (Zod v4 compat).
  // The minifier picks a different short identifier per rebuild, so match
  // whatever name it used this pass.
  requirePackagePatch("mastra-rag-zod-function");
  const funcPattern = /lengthFunction:([a-zA-Z_$][a-zA-Z_$0-9]*)\.optional\(\1\.function\(\)\)/;
  const funcMatch = bundle.match(funcPattern);
  if (funcMatch) {
    const v = funcMatch[1];
    const replacement = `lengthFunction:${v}.optional(${v}.any())`;
    bundle = bundle.replace(funcPattern, replacement);
    writeFileSync("../agent_embed_bundle.js", bundle);
    packagePatchApplied("mastra-rag-zod-function", `z.function() → z.any() for RAG validation (minifier used '${v}')`);
  } else {
    packagePatchMissed("mastra-rag-zod-function", "could not find RAG lengthFunction z.function validator");
  }

  // Patch @mastra/schema-compat OpenAI null-transform wrapper. Mastra's
  // Agent.generate path first converts structuredOutput.schema to a Standard
  // Schema, then wraps OpenAI schemas to convert nulls back to undefined.
  // The upstream wrapper returns a plain object, but later Agent code still
  // expects the original schema's Zod methods/prototype. Preserve that
  // prototype and override only the Standard Schema metadata.
  requirePackagePatch("mastra-schema-null-transform-prototype");
  const nullTransformPattern = /function ([A-Za-z_$][A-Za-z0-9_$]*)\(([A-Za-z_$][A-Za-z0-9_$]*)\)\{let ([A-Za-z_$][A-Za-z0-9_$]*);try\{\3=\2\["~standard"\]\.jsonSchema\.input\(\{target:"draft-07"\}\)\}catch\{\}if\(!\3\)return \2;let ([A-Za-z_$][A-Za-z0-9_$]*)=\2\["~standard"\];return\{"~standard":\{version:\4\.version,vendor:\4\.vendor,types:\4\.types,validate:\(([A-Za-z_$][A-Za-z0-9_$]*),([A-Za-z_$][A-Za-z0-9_$]*)\)=>\{let ([A-Za-z_$][A-Za-z0-9_$]*)=([A-Za-z_$][A-Za-z0-9_$]*)\(\5,\3\);return \4\.validate\(\7,\6\)\},jsonSchema:\4\.jsonSchema\}\}\}/;
  const nullTransformMatch = bundle.match(nullTransformPattern);
  if (nullTransformMatch) {
    const fn = nullTransformMatch[1];
    const schema = nullTransformMatch[2];
    const jsonSchema = nullTransformMatch[3];
    const standard = nullTransformMatch[4];
    const value = nullTransformMatch[5];
    const options = nullTransformMatch[6];
    const transformed = nullTransformMatch[7];
    const transformFn = nullTransformMatch[8];
    const replacement = `function ${fn}(${schema}){let ${jsonSchema};try{${jsonSchema}=${schema}["~standard"].jsonSchema.input({target:"draft-07"})}catch{}if(!${jsonSchema})return ${schema};let ${standard}=${schema}["~standard"],wrapper=Object.create(${schema});return Object.defineProperty(wrapper,"~standard",{value:{version:${standard}.version,vendor:${standard}.vendor,types:${standard}.types,validate:(${value},${options})=>{let ${transformed}=${transformFn}(${value},${jsonSchema});return ${standard}.validate(${transformed},${options})},jsonSchema:${standard}.jsonSchema},writable:!1,enumerable:!0,configurable:!1}),wrapper}`;
    bundle = bundle.replace(nullTransformMatch[0], replacement);
    writeFileSync("../agent_embed_bundle.js", bundle);
    packagePatchApplied("mastra-schema-null-transform-prototype", `${fn}: OpenAI structured-output null wrapper preserves schema prototype`);
  } else {
    packagePatchMissed("mastra-schema-null-transform-prototype", "could not find @mastra/schema-compat wrapSchemaWithNullTransform helper");
  }

  // Patch getTiktoken() to use getEncoding('o200k_base') with fallback
  requirePackagePatch("tiktoken-fallback");
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
      packagePatchApplied("tiktoken-fallback", `${fn}: getTiktoken uses ${encFn}('o200k_base') with fallback`);
    } else {
      packagePatchMissed("tiktoken-fallback", "found getTiktoken helper but could not find o200k_base encoder function");
    }
  } else {
    packagePatchMissed("tiktoken-fallback", "could not find getTiktoken helper");
  }
}

// Report size
const stats = statSync("../agent_embed_bundle.js");
console.log(`Bundle size: ${(stats.size / 1024).toFixed(1)} KB`);

// Write metafile for analysis
writeFileSync("meta.json", JSON.stringify(result.metafile));
console.log("Metafile written to meta.json (use https://esbuild.github.io/analyze/ to inspect)");
