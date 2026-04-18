# jsbridge Polyfills

brainkit is a JavaScript runtime where Go fills every Node.js gap.
Where Node.js uses C++ (libuv, OpenSSL, V8 built-ins) to implement
`net.Socket`, `crypto.createHash`, `fs.readFile`, `stream.Readable`,
brainkit uses Go (`net`, `crypto`, `os`, `compress/flate`, `os/exec`).
Libraries bundled into the runtime — Mastra, the AI SDK, the pg
driver, the MongoDB driver — do not know they are running on
QuickJS. They call Node.js APIs, and the polyfills return Go-backed
implementations.

This invariant — **jsbridge-first** — is the architectural rule. When
a library fails because a Node.js API is missing, the fix is in
`internal/jsbridge/*.go`, never in `build.mjs`. The bundle stubs are
thin re-exports from `globalThis`; the logic lives in Go where it is
testable.

## How a Polyfill Works

Every polyfill implements one interface:

```go
type Polyfill interface {
    Name() string
    Setup(ctx *quickjs.Context) error
}
```

A polyfill that starts goroutines (fetch, net, fs, exec, timers,
scheduling, zlib) additionally implements:

```go
type BridgeAware interface {
    SetBridge(b *Bridge)
}
```

`Bridge.Go(fn)` starts a tracked goroutine that counts toward the
bridge's `WaitGroup` and receives a context that cancels on `Close`.
No polyfill uses bare `go` — every goroutine ends deterministically
when the Kit shuts down, which is what makes `kit.Close()` safe
against QuickJS being freed mid-call.

## The Polyfill Set

`internal/embed/agent/sandbox.go` wires the polyfills into a new
`jsbridge.Bridge` in strict dependency order:

```go
b, err := jsbridge.New(bridgeCfg,
    // --- Core runtime ---
    jsbridge.Inspect(),        // __util_inspect, __util_format — before Console
    jsbridge.Console(),        // console.log/warn/error/info/debug
    jsbridge.Process(),        // process.env, process.version, nextTick, stdout
    jsbridge.Encoding(),       // TextEncoder, TextDecoder, btoa, atob
    jsbridge.Streams(),        // Web Streams (Readable/Writable/Transform)
    jsbridge.Crypto(),         // crypto.subtle + createHash / pbkdf2Sync
    jsbridge.URL(),            // URL, URLSearchParams
    jsbridge.Timers(),         // setTimeout, clearTimeout
    jsbridge.Scheduling(),     // setImmediate, clearImmediate, setInterval, clearInterval
    jsbridge.Abort(),          // AbortController, AbortSignal, DOMException
    jsbridge.Events(),         // EventEmitter (Node.js)
    jsbridge.DOMEvents(),      // EventTarget, Event, CustomEvent (DOM)
    jsbridge.StructuredClone(),
    jsbridge.Navigator(),      // navigator.userAgent, etc.
    jsbridge.Performance(),    // performance.now(), timeOrigin
    jsbridge.Intl(),           // Intl.DateTimeFormat (minimal)
    jsbridge.ErrorCompat(),    // Error.captureStackTrace, global alias, Response.json
    // --- Node.js module APIs ---
    jsbridge.NodeStreams(),    // Readable, Writable, Duplex, Transform — after Events
    jsbridge.Buffer(),         // Buffer.from/alloc/concat — after Encoding
    jsbridge.OS(),             // os.platform, arch, tmpdir, homedir
    jsbridge.Net(),            // Socket extends Duplex — after NodeStreams + Buffer
    jsbridge.DNS(),            // dns.lookup, dns.promises — after Net
    jsbridge.Zlib(),           // zlib.inflate/deflate/gzip — after Buffer
    jsbridge.WebAssembly(),    // WebAssembly.instantiate (wazero-backed)
    jsbridge.FS(cfg.CWD),      // fs / fs/promises (workspace-scoped)
    jsbridge.Exec(cfg.CWD),    // child_process.exec, spawn — rebased under CWD
    jsbridge.Fetch(fetchOpts...), // fetch, Headers, Request, Response, FormData, Blob, File
    jsbridge.WebSocketPoly(),  // client WebSocket (WHATWG + Node `ws` compat) — after Fetch
    jsbridge.Audio(jsbridge.AudioWithSink(cfg.AudioSink)), // web-standard Audio class
)
```

29 polyfills in total. The exact list is the source of truth —
`sandbox.go` imports them in the order above, and SES lockdown runs
afterwards. Each polyfill has focused Go tests in
`internal/jsbridge/*_test.go`.

## Why Order Matters

The chain is strict:

- **Events → NodeStreams** — `Readable` extends `EventEmitter`.
- **NodeStreams → Net** — `Socket` extends `Duplex`.
- **Encoding → Buffer** — `Buffer.from(str, "utf-8")` uses
  `TextEncoder`.
- **Buffer → Zlib / Net** — both return `Buffer` instances.
- **Inspect → Console** — `console.log` formats with `util.format`.

Getting ordering wrong yields cryptic errors: "Duplex is not a
constructor", "Buffer is not defined", "EventEmitter is not a
function". The polyfill harness refuses to double-install, but it
cannot detect reordering; keep the wiring in `sandbox.go`
authoritative.

## Clean Names, No `__node_*`

Polyfills set their canonical globals directly:

| Polyfill     | globalThis target                                    |
| ------------ | ---------------------------------------------------- |
| NodeStreams  | `globalThis.stream`                                  |
| Crypto       | merged onto `globalThis.crypto`                      |
| Net          | `globalThis.net`                                     |
| OS           | `globalThis.os`                                      |
| DNS          | `globalThis.dns`                                     |
| Zlib         | `globalThis.zlib`                                    |
| FS           | `globalThis.fs` + `globalThis.fs.promises`          |
| WebAssembly  | `globalThis.WebAssembly`                             |
| Fetch        | `globalThis.fetch`, `Headers`, `Request`, `Response` |

An older generation used `__node_*` prefixes (`__node_stream`,
`__node_crypto`), which required remapping inside the bundle stubs.
That was removed — today the names match what Node.js exposes and
what the bundle expects. The CLAUDE.md rule is explicit: "Polyfills
set clean names directly on globalThis. No `__node_*` prefix."

## Crypto Merge

WebCrypto and Node's `crypto` module collide in Node.js because both
live on `require('crypto')`. brainkit reproduces that:
`globalThis.crypto` starts with QuickJS's WebCrypto (`subtle`,
`randomUUID`, `getRandomValues`) and the Crypto polyfill merges Node
APIs onto the same object:

```javascript
Object.assign(globalThis.crypto, {
    createHash, createHmac, pbkdf2Sync, pbkdf2,
    randomBytes, timingSafeEqual,
    getHashes: () => ["md5","sha1","sha256","sha512"],
    getFips: () => 0,
    webcrypto: globalThis.crypto,
});
```

After the merge, `crypto.subtle.digest(...)` (used by pg's
SCRAM-SHA-256 handshake) and `crypto.createHash('sha256').update(…).
digest('hex')` (used by MongoDB's SCRAM implementation) both work
off the same object — matching Node.js semantics exactly.

## Bundle Stubs Are Thin

The Mastra bundle is built with esbuild. Bare `import ... from
"stream"` / `"crypto"` / `"net"` calls are intercepted by a custom
esbuild plugin that emits tiny stubs:

```javascript
// build.mjs — stream stub
"stream": `
    var S = globalThis.stream || {};
    export var Readable = S.Readable;
    export var Writable = S.Writable;
    export var Duplex = S.Duplex;
    export var Transform = S.Transform;
    export default { Readable, Writable, Duplex, Transform };
`,
```

No classes, no logic, no implementations. Every symbol comes from a
Go polyfill that is loaded before the bundle. Putting implementation
code into a stub is a maintenance trap — the stubs live inside
`build.mjs` JS strings where they are not covered by Go tests and
ship with the bundle at build time. The rule, encoded in CLAUDE.md:

> When a bundled library fails because a Node.js API is missing, add
> it to `internal/jsbridge/*.go` with a Go test. `build.mjs` module
> stubs are thin re-exports from globalThis — no logic. Never put
> implementations in build.mjs.

## Key Polyfill Internals

### Net — TCP / TLS

A JS `Socket` wraps a Go `net.Conn`. Each socket has a unique connection
ID registered in a Go-side map. The bridge exposes:

- `__go_net_connect(host, port)` — dials a TCP connection, starts a
  read loop in a tracked goroutine.
- `__go_net_write(connID, data)` — `Conn.Write`.
- `__go_net_tls_upgrade(connID, servername)` — wraps the conn in
  `crypto/tls.Client`.

The read loop calls `ctx.Schedule` to push chunks into JS
(`socket.push(chunk)`), which then flows through the NodeStreams
Duplex backbone.

### NodeStreams — async iterator transfer

Mastra / MongoDB drivers consume Readables with `for await`. When a
loop exits early (after a handshake response), the iterator's
`return()` method transfers unconsumed buffered data back into the
Readable's `_buffer` so the next `for await` sees it. Without this
transfer, consecutive `conn.command()` calls lose bytes between each
other — that behavior was implemented explicitly in `nodestreams.go`.

### FS — workspace-scoped

`FS(cfg.CWD)` is given the Kit's `Config.FSRoot`. Every path is
resolved against that root and checked for escape (`..` traversal,
absolute outside-root paths). Escaping returns a typed
`*sdk.WorkspaceEscapeError`. When `FSRoot` is empty, every `fs.*`
call fails with `NOT_CONFIGURED`.

### Fetch — HTTP + streaming SSE, binary-safe

Two paths:

- **Non-streaming.** Go buffers the full response body, hands it to
  JS through a single `ctx.Schedule` callback.
- **Streaming (SSE / chunked).** A tracked goroutine reads chunks
  from the HTTP response and pushes each into a JS `ReadableStream`
  controller via `ctx.Schedule`.

Non-text bodies (MP3, PNG, `application/octet-stream`, …) are
base64-encoded on both legs with an `x-brainkit-body-encoding:
base64` marker or a `bodyEncoding` field on the response JSON,
so arbitrary bytes survive the Go-string + JSON hop without
UTF-8 replacement-char corruption. Request body coverage
includes `FormData` (serialized to `multipart/form-data` with a
generated boundary), `Blob`, `ArrayBuffer`, and typed arrays.

Also ships alongside `Fetch`: polyfills for `Headers` (with
`append`, `getSetCookie`, case-insensitive keys), `Request`,
`Response`, `FormData`, `Blob`, and `File` (extends `Blob`). It
accepts a `FetchSpanHook` so the tracing module can attach
OTel spans around each outbound request.

### WebSocket — client-side Node + WHATWG combined

`globalThis.WebSocket` wraps `github.com/coder/websocket` in
client mode. Single class covers both surfaces any consumer
expects:

- **WHATWG** — `new WebSocket(url, protocols)`, `onopen /
  onmessage / onerror / onclose`, `addEventListener`, `send`,
  `close`, `readyState`, the four state constants.
- **Node `ws`** — `new WebSocket(url, protocols, {headers})`
  for `Authorization` / custom handshake headers, EventEmitter
  `ws.on("message" | "open" | "error" | "close", fn)`, binary
  frames delivered as `Buffer` / `Uint8Array`.

Both API styles live on the same object — any consumer resolves.
Binary frames cross the JS↔Go boundary via base64 (same pattern
as Fetch and Audio), so non-ASCII byte streams stay intact.

Needed because `@mastra/voice-openai-realtime` does `import
{ WebSocket } from "ws"`; `build.mjs` aliases `ws` to a tiny
shim that re-exports `globalThis.WebSocket` so the Mastra lib
binds to the polyfill unchanged.

### Audio — web-standard `new Audio(src).play()`

Lifts `globalThis.Audio` shaped like `HTMLAudioElement`.
Resolves `src` into bytes (URL via `fetch`, path via `fs`,
`Buffer` / `Uint8Array` / `Blob` / Node Readable / Web
`ReadableStream`), sniffs the container magic for MP3 / WAV /
OGG / FLAC, and hands the payload to a configured
`jsbridge.AudioSink`. With no sink wired, `play()` resolves
silently so portable agent code runs on headless kits.

The public Go side lives at `brainkit/audio` (`Sink`, `Null`,
`Func`, `Composite`) with opt-in desktop playback in
`brainkit/audio/local`. See the
[voice-and-audio guide](../guides/voice-and-audio.md) for the
wiring shape.

### WebAssembly — wazero-backed

`WebAssembly.instantiate` is implemented on top of `tetratelabs/wazero`.
This lets libraries that ship WASM modules (xxhash-wasm, some
cryptography libraries) load transparently in QuickJS.

## Testing Story

Every polyfill ships with Go unit tests under
`internal/jsbridge/*_test.go`. Representative coverage:

- `crypto_test.go` — hash, hmac, pbkdf2Sync, randomBytes,
  timingSafeEqual, subtle digest/sign/deriveBits.
- `nodestreams_test.go` — `for await` iteration, `pipe`, `return()`
  data transfer.
- `net_test.go` — TCP connect/write/close, TLS upgrade.
- `dns_test.go` — `dns.lookup` (sync + promises).
- `zlib_test.go` — inflate/deflate/gzip round trips.
- `fetch_test.go` — status, headers, streaming bodies, `AbortSignal`,
  multipart/form-data + binary body round trips.
- `fs_test.go` — workspace escape, readFile/writeFile, promises API,
  binary-safe `createReadStream` chunks.
- `websocket_test.go` — text + binary round-trip, `Authorization`
  header forwarded through the handshake.
- `audio_test.go` — sink dispatch, mime sniff, pause/cancel,
  Null default.

The tests run under the standard Go toolchain — no Node, no esbuild,
no browser — because the polyfills are Go code. That is the whole
point.

## See Also

- `internal/jsbridge/*.go` — polyfill sources.
- `internal/embed/agent/sandbox.go` — canonical load order.
- `internal/embed/agent/bundle/build.mjs` — stub definitions.
- [bundle-and-bytecode.md](bundle-and-bytecode.md) — how the stubs
  and polyfills meet at bundle load time.
- [deployment-pipeline.md](deployment-pipeline.md) — how deployed
  `.ts` code inherits the polyfills through Compartment endowments.
