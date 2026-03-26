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

### 1. `crypto.getFips()` — DONE (2026-03-26)
- **Fixed in:** jsbridge/crypto.go + build.mjs crypto stub
- Returns 0 (not FIPS mode)

### 2. `dns.lookup(host, callback)` — DONE (2026-03-26)
- **Fixed in:** New jsbridge/dns.go + build.mjs dns stub
- Go `net.LookupHost()` backed, sync + async (Promises)

### 3. `process.emitWarning(msg)` — DONE (2026-03-26)
- **Fixed in:** jsbridge/process.go
- No-op stub

### 4. `zlib.inflate/deflate` — DONE (2026-03-26)
- **Fixed in:** New jsbridge/zlib.go + build.mjs zlib stub
- Go `compress/zlib`, `compress/flate`, `compress/gzip` backed
- Sync + async callback + Transform stream variants

---

## Gaps — MODERATE

Will cause failures in specific code paths (auth methods, discovery, logging).

### 5. `dns.promises.lookup/resolveSrv/resolveCname/resolvePtr` — DONE (2026-03-26)
- **Fixed in:** jsbridge/dns.go — promises.lookup is Go-backed, SRV/CNAME/PTR return empty (stubs)

### 6. `util.inspect(obj, options)` — DONE (2026-03-26)
- **Fixed in:** build.mjs util stub — accepts opts param, handles compact flag

### 7. `os.release()` — DONE (2026-03-26)
- **Fixed in:** jsbridge/os.go — uses `uname -r` for real kernel version

### 8. `tls.connect()` — DONE (2026-03-26)
- **Fixed in:** net.go (__go_net_tls_upgrade) + build.mjs tls stub

### 9. `stream.Readable.toWeb/fromWeb` — DONE (2026-03-26)
- **Fixed in:** build.mjs stream stub — Readable.from, Readable.toWeb, Readable.fromWeb

---

## Gaps — LOW — ALL DONE (2026-03-26)

### 10. `util.types.isRegExp/isMap/isSet/isTypedArray` — DONE
- **Fixed in:** build.mjs util + util/types stubs

### 11. `EventEmitter.captureRejections` — DONE
- **Fixed in:** jsbridge/events.go — static props: captureRejections, defaultMaxListeners, setMaxListeners, listenerCount

### 12. `process.getuid()/getgid()` — DONE
- **Fixed in:** jsbridge/process.go — getuid, getgid, geteuid, getegid, setuid, setgid

### 13. `Buffer.poolSize` — DONE
- **Fixed in:** jsbridge/buffer.go — `Buffer.poolSize = 8192`

### 14. `child_process.execFileSync`
- **Status:** Still throws. Not yet needed at runtime — Mastra workspace is optional.
- **Fix when needed:** Go `os/exec` backed, synchronous

---

## Lessons Learned

1. **"not a function" errors deep in drivers** are almost always missing polyfill stubs. The error message is misleading — it's not about `this` binding, Proxies, or SES. It's a function that doesn't exist.

2. **Diagnostic fixtures** are the fastest way to find root cause. Create a fixture that tests each operation individually and traces the error chain.

3. **The two most impactful fixes so far** were trivial:
   - `util.types.isDate` — one line in build.mjs, unblocked all pg fixtures
   - `EventEmitter.setMaxListeners` — one method in events.go, unblocked all MongoDB fixtures

4. **esbuild tree-shaking** means many stubs are never called. Only add implementations for APIs that are actually reached at runtime, not everything a module exports.

5. **Test ALL combinations** after fixing a polyfill — the fix might unblock one path but reveal the next missing API in the chain.
