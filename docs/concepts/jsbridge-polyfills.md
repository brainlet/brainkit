# jsbridge Polyfills

brainkit is a JavaScript runtime where Go fills the blanks. Where Node.js uses C++ (libuv, openssl, V8 builtins) to implement `net.Socket`, `crypto.createHash`, `fs.readFile`, and `stream.Readable`, brainkit uses Go (`net`, `crypto`, `os`, `compress/flate`). The JavaScript libraries that run inside brainkit — Mastra, AI SDK, the pg driver, the MongoDB driver — don't know or care. They call Node.js APIs and get Go-backed implementations.

This principle — **jsbridge-first** — is the most important architectural decision in brainkit. When a bundled library fails because a Node.js API is missing, the fix is always in `internal/jsbridge/*.go`, never in `build.mjs`.

## How It Works

Each polyfill is a Go struct implementing the `Polyfill` interface:

```go
// internal/jsbridge/polyfill.go
type Polyfill interface {
    Name() string
    Setup(ctx *quickjs.Context) error
}
```

Some polyfills need to start goroutines (fetch, net, fs, exec, timers). These implement `BridgeAware`:

```go
type BridgeAware interface {
    SetBridge(b *Bridge)
}
```

The Bridge provides `Bridge.Go(fn)` for tracked goroutines and `Bridge.GoContext()` for cancellation. Every goroutine started by a polyfill is tracked by the Bridge's WaitGroup and cancelled on Close. No orphaned goroutines.

## The 24 Polyfills

Loaded in `internal/embed/agent/sandbox.go` in dependency order:

```go
b, err := jsbridge.New(bridgeCfg,
    jsbridge.Console(),         // console.log/warn/error/info/debug
    jsbridge.Process(),         // process.env, process.cwd, nextTick, stdout
    jsbridge.Encoding(),        // TextEncoder, TextDecoder, btoa, atob
    jsbridge.Streams(),         // Web Streams (ReadableStream, WritableStream, TransformStream)
    jsbridge.Crypto(),          // crypto.subtle + createHash, pbkdf2Sync (merged)
    jsbridge.URL(),             // URL, URLSearchParams
    jsbridge.Timers(),          // setTimeout, clearTimeout
    jsbridge.Scheduling(),      // setImmediate, setInterval, clearInterval
    jsbridge.Abort(),           // AbortController, AbortSignal, DOMException
    jsbridge.Events(),          // EventEmitter (Node.js)
    jsbridge.DOMEvents(),       // EventTarget, Event, CustomEvent (DOM)
    jsbridge.StructuredClone(), // structuredClone
    jsbridge.Navigator(),       // navigator.userAgent, etc.
    jsbridge.Performance(),     // performance.now(), timeOrigin
    jsbridge.Intl(),            // Intl.DateTimeFormat (minimal)
    jsbridge.ErrorCompat(),     // Error.captureStackTrace, Response.json
    jsbridge.NodeStreams(),      // Readable, Writable, Duplex, Transform ← AFTER Events
    jsbridge.Buffer(),          // Buffer.from, alloc, concat ← AFTER Encoding
    jsbridge.OS(),              // os.platform, arch, tmpdir, homedir
    jsbridge.Net(),             // Socket extends Duplex ← AFTER NodeStreams + Buffer
    jsbridge.DNS(),             // dns.lookup, dns.promises ← AFTER Net
    jsbridge.Zlib(),            // zlib.inflate/deflate/gzip ← AFTER Buffer
    jsbridge.WebAssembly(),     // WebAssembly.instantiate (wazero-backed)
    jsbridge.FS(),              // fs.readFile, writeFile, etc.
    jsbridge.Exec(),            // child_process.exec, spawn
    jsbridge.Fetch(fetchOpts...), // fetch, Headers, Request, Response
)
```

### Load Order Matters

The dependency chain is critical:

- **Events** must load before **NodeStreams** — `Readable` extends `EventEmitter`
- **NodeStreams** must load before **Net** — `Socket` extends `Duplex` (which extends `EventEmitter`)
- **Encoding** must load before **Buffer** — `Buffer.from(string, encoding)` uses `TextEncoder`
- **Buffer** must load before **Net** and **Zlib** — they return `Buffer` instances
- **NodeStreams** must load before **Zlib** — `createGzip()` returns a `Transform` stream

Getting this wrong produces cryptic errors: "Duplex is not a constructor", "Buffer is not defined", "EventEmitter is not a function".

## Polyfill Naming

All polyfills set clean names directly on `globalThis`:

| Polyfill | globalThis name |
|----------|----------------|
| NodeStreams | `globalThis.stream` |
| Crypto | merged onto `globalThis.crypto` |
| Net | `globalThis.net` |
| OS | `globalThis.os` |
| DNS | `globalThis.dns` |
| Zlib | `globalThis.zlib` |

No `__node_*` prefix. This was cleaned up — polyfills previously used `globalThis.__node_stream`, etc., requiring a mapping layer in kit_runtime.js endowments.

### The Crypto Merge

`globalThis.crypto` is special. QuickJS provides a basic WebCrypto object with `crypto.subtle` (SubtleCrypto), `crypto.randomUUID()`, and `crypto.getRandomValues()`. Node.js adds `createHash`, `createHmac`, `pbkdf2Sync`, `randomBytes`, `timingSafeEqual`, etc.

brainkit merges both onto the same object:

```javascript
// internal/jsbridge/crypto.go
var _cryptoTarget = globalThis.crypto || {};
Object.assign(_cryptoTarget, {
    createHash: function(alg) { ... },
    createHmac: function(alg, key) { ... },
    pbkdf2Sync: function(password, salt, iterations, keylen, hash) { ... },
    pbkdf2: function(password, salt, iterations, keylen, hash, callback) { ... },
    randomBytes: function(n, cb) { ... },
    timingSafeEqual: function(a, b) { ... },
    getHashes: function() { return ["md5", "sha1", "sha256", "sha512"]; },
    getFips: function() { return 0; },
});
```

After this, `crypto.subtle.digest("SHA-256", data)` (WebCrypto, used by pg SCRAM-SHA-256) and `crypto.createHash("sha256").update(data).digest("hex")` (Node.js, used by MongoDB SCRAM) both work on the same object. This matches Node.js behavior where `require('crypto')` returns one object with everything.

## How esbuild Stubs Work

The Mastra bundle is built with esbuild. When esbuild sees `import { Readable } from 'stream'`, it needs to resolve that import. The `build.mjs` node-stub plugin provides module stubs that re-export from globalThis:

```javascript
// internal/embed/agent/bundle/build.mjs — stream stub
"stream": `
    var S = globalThis.stream || {};
    export var Readable = S.Readable || class Readable {};
    export var Writable = S.Writable || class Writable {};
    export var Duplex = S.Duplex || class Duplex {};
    export var Transform = S.Transform || class Transform {};
    // ...
    export default { Readable, Writable, Duplex, Transform, ... };
`,
```

These stubs are thin re-exports — no logic, no classes, no implementations. At runtime, `globalThis.stream.Readable` is the real class from `jsbridge/nodestreams.go`. The stub just wires esbuild's module resolution to the polyfill.

**Rule:** Never put implementations in build.mjs. If a library needs `crypto.pbkdf2Sync`, implement it in `jsbridge/crypto.go` with a Go test, then re-export it from the build.mjs stub. The MongoDB SCRAM debugging lesson proved this: when build.mjs grew to 1155 lines of inline JS implementations, everything broke. Moving to jsbridge Go polyfills with proper tests fixed it.

## Key Polyfill Implementations

### Net (jsbridge/net.go)

TCP and TLS connections backed by Go `net.Conn`. Each JS `Socket` gets a `goConn` with a unique ID. The Go side manages the connection lifecycle:

- `__go_net_connect(host, port)` → opens TCP connection, starts read loop in `Bridge.Go` goroutine
- Read loop pushes data chunks into JS via `ctx.Schedule` → `socket.push(chunk)`
- `__go_net_write(connID, data)` → writes to the Go conn
- `__go_net_tls_upgrade(connID, servername)` → upgrades the TCP conn to TLS via `crypto/tls`

The JS `Socket` class extends `Duplex` (from NodeStreams) and wraps a `GoSocket`:

```javascript
class Socket extends Duplex {
    constructor() {
        super();
        this._gs = new GoSocket();
    }
    connect(portOrOpts, host) {
        this._gs.connect(portOrOpts, host);
        this._gs.on("data", (chunk) => this.push(chunk));
        this._gs.on("end", () => this.push(null));
        // ...
    }
}
```

### NodeStreams (jsbridge/nodestreams.go)

Full Node.js stream implementation: Readable, Writable, Duplex, Transform, PassThrough. The critical feature for MongoDB compatibility is the async iterator with data transfer on `return()`:

When a `for await` loop exits early (e.g., after reading the SCRAM hello response), the iterator's `return()` method transfers unconsumed buffered data BACK to the Readable's `_buffer`. The next `for await` loop (for the saslStart response) sees that data. Without this, MongoDB's `conn.command()` pattern loses data between consecutive calls.

### Crypto (jsbridge/crypto.go)

Two layers:
1. **Go bridge functions** — `__go_crypto_hash`, `__go_crypto_subtle_digest`, `__go_crypto_subtle_sign`, `__go_crypto_subtle_deriveBits` handle the actual cryptography in Go using `crypto/sha256`, `crypto/hmac`, `golang.org/x/crypto/pbkdf2`
2. **JS wrapper** — `createHash(alg)` returns an object with `update(data)` and `digest(enc)` that accumulates data in JS and calls the Go bridge for the final hash

### Fetch (jsbridge/fetch.go)

HTTP client backed by Go `net/http`. Two modes:
- **Non-streaming**: Go fetches the full response body, returns it to JS via `ctx.Schedule`
- **Streaming (SSE)**: Go reads chunks in a `Bridge.Go` goroutine, pushes each chunk into a JS `ReadableStream` controller via `ctx.Schedule`

The fetch polyfill is the last one loaded because it depends on everything else (Headers, Request, Response, ReadableStream, AbortSignal, TextEncoder).

## Testing

Every polyfill has Go unit tests in `internal/jsbridge/*_test.go`:

```go
// internal/jsbridge/crypto_test.go
func TestCrypto_Pbkdf2Sync(t *testing.T) {
    b := newTestBridge(t)
    result := evalString(t, b, `
        var derived = globalThis.crypto.pbkdf2Sync("pencil", Buffer.from("salt"), 1000, 32, "sha256");
        Buffer.from(derived).toString("hex")
    `)
    assert.Equal(t, expectedHex, result)
}
```

70+ jsbridge unit tests cover: dns lookup, zlib inflate/deflate, exec/spawn, events, crypto hash/hmac/pbkdf2, os platform/arch, buffer encoding, streams pipe/asyncIterator, net socket, webassembly instantiate.

The auth test matrix (`test/auth/auth_test.go`) validates the full polyfill stack end-to-end: pg SCRAM-SHA-256 (WebCrypto path), pg md5 (Node.js crypto path), MongoDB SCRAM-SHA-256 (pbkdf2Sync path), MongoDB SCRAM-SHA-1, all with real containers.
