package esbuild

import "strings"

const npmPreviewNodeStubNamespace = "brainkit-node-stub"

var nodeBuiltinNames = map[string]struct{}{
	"assert":              {},
	"async_hooks":         {},
	"buffer":              {},
	"child_process":       {},
	"cluster":             {},
	"constants":           {},
	"crypto":              {},
	"dgram":               {},
	"diagnostics_channel": {},
	"dns":                 {},
	"domain":              {},
	"events":              {},
	"fs":                  {},
	"http":                {},
	"http2":               {},
	"https":               {},
	"inspector":           {},
	"module":              {},
	"net":                 {},
	"os":                  {},
	"path":                {},
	"perf_hooks":          {},
	"process":             {},
	"querystring":         {},
	"readline":            {},
	"repl":                {},
	"stream":              {},
	"string_decoder":      {},
	"timers":              {},
	"tls":                 {},
	"tty":                 {},
	"url":                 {},
	"util":                {},
	"v8":                  {},
	"vm":                  {},
	"worker_threads":      {},
	"zlib":                {},
}

var npmPreviewNodeStubs = map[string]string{
	"assert": `
var A = globalThis.assert;
export var ok = A.ok;
export var strictEqual = A.strictEqual;
export var deepStrictEqual = A.deepStrictEqual;
export var throws = A.throws;
export var fail = A.fail;
export default A;
`,
	"assert/strict": `
var A = globalThis.assert;
export var ok = A.ok;
export var strictEqual = A.strictEqual;
export var deepStrictEqual = A.deepStrictEqual;
export var throws = A.throws;
export var fail = A.fail;
export default A;
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
	"buffer": `
var B = globalThis.Buffer;
function NodeBuffer(arg, encodingOrOffset, length) {
	if (typeof arg === "number") return B.alloc(arg);
	return B.from(arg, encodingOrOffset, length);
}
for (var k in B) {
	try { NodeBuffer[k] = B[k]; } catch (_) {}
}
NodeBuffer.prototype = Uint8Array.prototype;
export var Buffer = NodeBuffer;
export default { Buffer };
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
	"events": `
export var EventEmitter = globalThis.EventEmitter;
export default EventEmitter;
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
	"module": `
var M = globalThis.node_module;
export var createRequire = M.createRequire;
export var builtinModules = M.builtinModules;
export var isBuiltin = M.isBuiltin;
export var syncBuiltinESMExports = M.syncBuiltinESMExports;
export var Module = M.Module;
export default M;
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
	"perf_hooks": `
var P = globalThis.perf_hooks;
export var performance = globalThis.performance;
export var PerformanceObserver = P.PerformanceObserver;
export var monitorEventLoopDelay = P.monitorEventLoopDelay;
export default P;
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
export var getBuiltinModule = _p.getBuiltinModule;
export var nextTick = _p.nextTick;
export var stdout = _p.stdout;
export var stderr = _p.stderr;
export default _p;
`,
	"querystring": `
var Q = globalThis.querystring;
export var parse = Q.parse;
export var stringify = Q.stringify;
export var encode = Q.encode;
export var decode = Q.decode;
export default Q;
`,
	"stream": `
var S = globalThis.stream;
var EventEmitter = S.EventEmitter;
function Stream() {
	if (typeof EventEmitter === "function") EventEmitter.call(this);
}
Stream.prototype = (S.Stream && S.Stream.prototype) || (EventEmitter && EventEmitter.prototype) || {};
Stream.EventEmitter = EventEmitter;
Stream.Readable = S.Readable;
Stream.Writable = S.Writable;
Stream.Duplex = S.Duplex;
Stream.Transform = S.Transform;
Stream.PassThrough = S.PassThrough;
Stream.pipeline = S.pipeline;
Stream.finished = S.finished;
Stream.Stream = Stream;
module.exports = Stream;
`,
	"stream/promises": `
var P = globalThis.stream.promises;
export var pipeline = P.pipeline;
export var finished = P.finished;
export default P;
`,
	"stream/web": `
export var ReadableStream = globalThis.ReadableStream;
export var WritableStream = globalThis.WritableStream;
export var TransformStream = globalThis.TransformStream;
export default { ReadableStream, WritableStream, TransformStream };
`,
	"string_decoder": `
export var StringDecoder = globalThis.StringDecoder;
export default { StringDecoder };
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
}

func normalizeNodeBuiltinImport(specifier string) string {
	return strings.TrimPrefix(specifier, "node:")
}

func npmPreviewNodeStub(specifier string) (string, string, bool) {
	id := normalizeNodeBuiltinImport(specifier)
	contents, ok := npmPreviewNodeStubs[id]
	return id, contents, ok
}

func isNodeBuiltinImport(specifier string) bool {
	if strings.HasPrefix(specifier, "node:") {
		return true
	}
	id := normalizeNodeBuiltinImport(specifier)
	if _, ok := nodeBuiltinNames[id]; ok {
		return true
	}
	if slash := strings.Index(id, "/"); slash > 0 {
		_, ok := nodeBuiltinNames[id[:slash]]
		return ok
	}
	return false
}
