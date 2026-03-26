# Node.js Polyfill Gap Analysis

> Audit date: 2026-03-26. Based on reading pg, mongodb, mastra, AI SDK source code
> and tracing actual runtime failures in QuickJS + SES Compartments.

## Architecture

brainkit is a JS runtime where Go fills the blanks — like Node.js uses C++.

Two polyfill layers:
1. **jsbridge polyfills** (`internal/jsbridge/*.go`) — Go-backed, loaded at runtime on `globalThis.__node_*`
2. **esbuild module stubs** (`internal/embed/agent/bundle/build.mjs`) — thin re-exports for bundle resolution

Bundled libraries that depend on Node.js APIs:
- **pg** (node-postgres) — PostgreSQL driver
- **mongodb** (node-mongodb-native) — MongoDB driver
- **@libsql/client** — LibSQL/SQLite HTTP client
- **@mastra/core, @mastra/memory, @mastra/pg, @mastra/mongodb** — Mastra framework
- **ai** (Vercel AI SDK) — LLM provider integrations
- **zod/v4** — Schema validation

---

## Already Provided (working, tested)

| Module | What works | Where |
|--------|-----------|-------|
| `net` | Socket (connect/write/end/destroy/pipe/setNoDelay/setKeepAlive/setTimeout), createConnection, isIP | jsbridge/net.go |
| `events` | EventEmitter (on/once/emit/off/addListener/removeListener/removeAllListeners/setMaxListeners/getMaxListeners/prependListener/prependOnceListener/rawListeners/eventNames/listenerCount) | jsbridge/events.go |
| `crypto` | createHash, createHmac (binary-safe), pbkdf2Sync, pbkdf2, timingSafeEqual, randomBytes (sync+cb), randomFillSync, randomInt, getHashes, getCiphers, SubtleCrypto (digest/sign/importKey/deriveBits/exportKey), crypto.randomUUID, crypto.getRandomValues | jsbridge/crypto.go |
| `buffer` | Buffer.from (string/array/base64/hex/ArrayBuffer/Uint8Array), alloc, allocUnsafe, concat, isBuffer, isEncoding, byteLength, compare. Instance: readInt32BE/LE, writeInt32BE/LE, readInt16BE/LE, writeInt16BE/LE, readFloat/DoubleLE, writeFloat/DoubleLE, readUInt8, writeUInt8, toString(utf8/hex/base64/ascii), toJSON, equals, compare, fill, indexOf, copy, write, slice, subarray, map | jsbridge/buffer.go |
| `stream` | Readable (push/read/unshift/resume/pause/pipe/unpipe/destroy/setEncoding/Symbol.asyncIterator with data handoff), Writable (write/end/destroy/cork/uncork), Duplex, Transform, PassThrough, pipeline, finished, Readable.from | jsbridge/nodestreams.go |
| `stream` (Web) | ReadableStream, WritableStream, TransformStream | jsbridge/streams.go |
| `process` | env (Go-backed Proxy), cwd, version, versions, platform, arch, pid, argv, execPath, title, hrtime (+ bigint), nextTick, stdout/stderr/stdin, on/once/off/emit/removeListener/removeAllListeners, chdir, exit, umask, uptime, cpuUsage, memoryUsage, release, config, features | jsbridge/process.go |
| `os` | platform, arch, tmpdir, homedir, hostname, type, cpus, EOL, endianness, release (stub), totalmem, freemem, uptime, loadavg, networkInterfaces, userInfo | jsbridge/os.go |
| `path` | join, resolve, dirname, basename, extname, parse, relative, sep, delimiter, posix | build.mjs stub |
| `url` | URL, URLSearchParams, fileURLToPath, pathToFileURL | build.mjs stub (delegates to globalThis) |
| `util` | promisify, inherits, deprecate, inspect (basic), format, types.isUint8Array, types.isDate, types.isArrayBuffer, TextEncoder, TextDecoder | build.mjs stub |
| `fs` | readFile, writeFile, readdir, stat, mkdir, unlink, rm (all async via kit bridge). Sync stubs: existsSync, realpathSync, statSync, readdirSync | build.mjs stub |
| `child_process` | exec, spawn (via jsbridge/exec.go). execSync/execFile/execFileSync throw | build.mjs stub |
| `timers` | setTimeout, clearTimeout, setInterval, clearInterval, setImmediate, clearImmediate | jsbridge/timers.go + build.mjs |
| `timers/promises` | setTimeout (Promise-based) | build.mjs stub |
| `async_hooks` | AsyncLocalStorage (run/getStore/enterWith/disable), AsyncResource, createHook, executionAsyncId | build.mjs stub |
| `fetch` | fetch, Headers, Request, Response (with SSE streaming via ReadableStream) | jsbridge/fetch.go |
| Other | AbortController/AbortSignal, TextEncoder/TextDecoder, btoa/atob, structuredClone, navigator, performance.now, EventTarget/Event/CustomEvent, Error.captureStackTrace, global alias, Response.json static | jsbridge/*.go |

---

## Gaps — CRITICAL

Will cause immediate runtime failures when the code path is hit.

### 1. `crypto.getFips()`
- **Used by:** MongoDB SCRAM auth (checks FIPS mode before using MD5)
- **Location:** `node-mongodb-native/src/cmap/auth/mongo_credentials.ts:236`
- **Current:** Missing — throws "not a function"
- **Fix:** Add `getFips: function() { return 0; }` to `__node_crypto` in jsbridge/crypto.go
- **Layer:** jsbridge + build.mjs crypto stub

### 2. `dns.lookup(host, callback)`
- **Used by:** pg connection-parameters.js:163 — resolves hostname before TCP connect
- **Current:** build.mjs dns stub throws
- **Fix:** New jsbridge polyfill backed by Go `net.LookupHost()`. Return `{ address, family }`.
- **Layer:** New jsbridge/dns.go + build.mjs dns stub update
- **Note:** pg falls back to direct TCP connect if dns.lookup fails, so may not block basic use. But production configs with hostnames will fail.

### 3. `process.emitWarning(msg)`
- **Used by:** MongoDB driver — deprecation warnings for old options
- **Current:** Missing on process object
- **Fix:** Add `if (!p.emitWarning) p.emitWarning = function() {};` in jsbridge/process.go
- **Layer:** jsbridge/process.go

### 4. `zlib.inflate(buf, callback)` + `zlib.deflate(buf, options, callback)`
- **Used by:** MongoDB wire protocol compression (snappy/zlib/zstd)
- **Location:** `node-mongodb-native/src/cmap/wire_protocol/compression.ts:46-57`
- **Current:** build.mjs zlib stub throws
- **Fix:** New jsbridge polyfill backed by Go `compress/flate` or `compress/zlib`
- **Layer:** New jsbridge/zlib.go + build.mjs zlib stub update
- **Note:** Only needed if MongoDB server has compression enabled. Default: no compression.

---

## Gaps — MODERATE

Will cause failures in specific code paths (auth methods, discovery, logging).

### 5. `dns.promises.lookup/resolveSrv/resolveCname/resolvePtr`
- **Used by:** MongoDB SRV discovery (`mongodb+srv://` URLs), GSSAPI/Kerberos auth
- **Current:** build.mjs dns stub throws
- **Fix:** Same jsbridge/dns.go as #2, add Promise-based wrappers
- **Layer:** jsbridge/dns.go + build.mjs dns stub

### 6. `util.inspect(obj, options)` — options parameter
- **Used by:** MongoDB logger (`mongo_logger.ts`) with `{ compact: true, breakLength: Infinity }`
- **Current:** build.mjs has `function(v) { return JSON.stringify(v); }` — ignores options
- **Fix:** Accept second parameter, still use JSON.stringify but handle edge cases (circular refs, depth)
- **Layer:** build.mjs util stub

### 7. `os.release()` — real value
- **Used by:** MongoDB client metadata (`os.release()` in handshake)
- **Current:** Returns `"0.0.0"` stub
- **Fix:** Back with Go `runtime.GOOS` version or `syscall.Uname`
- **Layer:** jsbridge/os.go

### 8. `tls.connect()` with socket wrapping
- **Used by:** pg SSL connections — upgrades plain TCP to TLS after negotiation
- **Current:** build.mjs tls stub throws
- **Fix:** New jsbridge polyfill backed by Go `crypto/tls`. Accept `{ socket }` option to wrap existing connection.
- **Layer:** New jsbridge/tls.go + build.mjs tls stub update
- **Note:** Only needed for SSL PostgreSQL connections. Non-SSL works fine.

### 9. `stream.Readable.toWeb()` / `stream.Readable.fromWeb()`
- **Used by:** Possible AI SDK or Mastra usage for stream conversion
- **Current:** Missing
- **Fix:** Stubs that convert between Node.js streams and Web Streams
- **Layer:** jsbridge/nodestreams.go or build.mjs stream stub

---

## Gaps — LOW

Unlikely to be hit but good for completeness.

### 10. `util.types.isRegExp/isMap/isSet/isTypedArray`
- **Used by:** Possible runtime type checks
- **Fix:** Simple `instanceof` checks in build.mjs util/types stub

### 11. `EventEmitter.captureRejections`
- **Used by:** Node.js 12+ feature, some libraries check for it
- **Fix:** Static property `EventEmitter.captureRejections = false`

### 12. `process.getuid()/getgid()`
- **Used by:** Possible POSIX permission checks
- **Fix:** `function() { return 0; }` in jsbridge/process.go

### 13. `Buffer.poolSize`
- **Used by:** Node.js internals
- **Fix:** Static property `Buffer.poolSize = 8192`

### 14. `child_process.execFileSync`
- **Used by:** Mastra workspace operations
- **Fix:** Implement with Go `os/exec` (synchronous, blocks QuickJS thread)

---

## Lessons Learned

1. **"not a function" errors deep in drivers** are almost always missing polyfill stubs. The error message is misleading — it's not about `this` binding, Proxies, or SES. It's a function that doesn't exist.

2. **Diagnostic fixtures** are the fastest way to find root cause. Create a fixture that tests each operation individually and traces the error chain.

3. **The two most impactful fixes so far** were trivial:
   - `util.types.isDate` — one line in build.mjs, unblocked all pg fixtures
   - `EventEmitter.setMaxListeners` — one method in events.go, unblocked all MongoDB fixtures

4. **esbuild tree-shaking** means many stubs are never called. Only add implementations for APIs that are actually reached at runtime, not everything a module exports.

5. **Test ALL combinations** after fixing a polyfill — the fix might unblock one path but reveal the next missing API in the chain.
