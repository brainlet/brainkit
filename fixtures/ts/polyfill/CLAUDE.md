# polyfill/ Fixtures

Tests Node.js API polyfills implemented in Go and exposed to QuickJS: buffer, crypto, dns, events, child_process, os, process, stream, util, and zlib.

## Fixtures

### buffer/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| pool-size | no | none | `Buffer.poolSize` (8192), `Buffer.isEncoding` for utf8/hex/fake, `Buffer.byteLength` for string and base64, `Buffer.compare` equality and ordering |

### crypto/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| getfips | no | none | `crypto.getFips()` returns 0, `crypto.getHashes()` returns array, `crypto.getCiphers()` is empty, `crypto.timingSafeEqual` correct for equal and different buffers |

### dns/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| lookup | no | none | `dns.lookup("localhost")` callback returns addr+family; `dns.promises.lookup("localhost")` returns address+family via Go `net.LookupHost` |

### events/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| max-listeners | no | none | `EventEmitter.setMaxListeners`/`getMaxListeners`, `prependListener` fires before normal, `eventNames()`, `off()` removes listener, static `captureRejections` and `defaultMaxListeners` |

### exec/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| sync | no | none | `child_process.execSync`, `execFileSync`, `spawnSync` all execute shell commands via Go `os/exec`; non-zero exit throws catchable error |

### os/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| release | no | none | `os.release()` returns real kernel version (not stub "0.0.0"), `os.platform()`, `os.arch()`, `os.type()`, `os.hostname()`, `os.cpus()` has entries, `os.EOL` is string |

### process/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| extras | no | none | `process.emitWarning` does not throw, `process.getuid()`/`getgid()` return numbers, `process.hrtime()` returns [sec, nsec] array, `process.nextTick` fires via `queueMicrotask` |

### stream/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| readable-from | no | none | `stream.Readable.from()` creates readable from iterable (3 chunks), `pipe()` to Writable collects output |

### util/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| types | no | none | Verifies `Date`, `RegExp`, `Map`, `Set`, `Uint8Array` instanceof checks work; `Buffer.isBuffer` true for Buffer, false for string |

### zlib/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| deflate-inflate | no | none | Sync `zlib.deflateSync`/`inflateSync` roundtrip, `gzipSync`/`gunzipSync` roundtrip, async callback `deflate`/`inflate`, `zlib.constants.Z_DEFAULT_COMPRESSION` exists |
