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
Long-lived runtime goroutines must be bridge-tracked; short helper
goroutines must be scoped to a bridge-tracked owner. This is what makes
`kit.Close()` safe against QuickJS being freed mid-call.

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
    jsbridge.Path(),           // path and path.posix helpers
    jsbridge.NodeCompat(),     // assert, querystring, util, perf_hooks shapes
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

31 installed polyfills and compatibility layers in total. The exact list
is the source of truth — `sandbox.go` imports them in the order above,
and SES lockdown runs afterwards. Each polyfill has focused Go tests in
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
| Buffer       | `globalThis.Buffer`                                  |
| Crypto       | merged onto `globalThis.crypto`                      |
| Path         | `globalThis.path`                                    |
| NodeCompat   | `globalThis.assert`, `querystring`, `util`, etc.     |
| Process      | `globalThis.process`                                 |
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
what the bundle expects. The compatibility manifest and embed tests
guard that bundle stubs only reference declared globals.

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
    var S = globalThis.stream;
    export var Readable = S.Readable;
    export var Writable = S.Writable;
    export var Duplex = S.Duplex;
    export var Transform = S.Transform;
    export var PassThrough = S.PassThrough;
    export var pipeline = S.pipeline;
    export var finished = S.finished;
    export default S;
`,
```

No classes, no logic, no implementations. Every symbol comes from a
Go polyfill that is loaded before the bundle. Putting implementation
code into a stub is a maintenance trap — the stubs live inside
`build.mjs` JS strings where they are not covered by Go tests and
ship with the bundle at build time.

The checked rule is:

- jsbridge-owned stubs are mechanical re-exports from declared
  `globalThis` symbols;
- unknown Node builtins and subpaths fail the bundle build;
- unsupported surfaces throw typed errors from their jsbridge owner;
- package-specific exceptions are declared as package patches.

Those guardrails are enforced by the compatibility manifest tests under
`internal/embed/agent`.

## Compatibility Manifest

The agent embed compatibility surface is declared in
`internal/embed/agent/bundle/compat/manifest.json`. It records every
Node module/subpath, dynamic `require()`, web global, external package,
and package patch that the Mastra bundle depends on.

Each row has an owner, reason, tests, globals/exports when applicable,
and one status:

| Status | Meaning |
| ------ | ------- |
| `exact` | Intended to match the used Node/Web semantics. |
| `compat` | Partial by design, but enough for the dependency closure. |
| `stub` | Shape-only no-op for optional paths that are known not to need behavior. |
| `unsupported` | Explicitly unavailable and expected to throw a typed error. |

`internal/embed/agent/bundle/compat/report.json` is the checked report
generated from the manifest and esbuild metadata. The Make targets are:

```bash
make jsbridge-compat-report
make jsbridge-compat-inventory
make jsbridge-compat-check
make jsbridge-lifecycle-check
make agent-embed-rebuild-check
```

`internal/embed/agent/bundle/compat/inventory.json` is the richer review
artifact. It is generated from the manifest, esbuild metadata, and bundle
`package.json`; it links package names/versions to Node/Web APIs, external
imports, package patches, dependency class, resource/tests, and
external-service flags. Use it to answer "which dependency needs this
polyfill or patch?" before adding new platform behavior.

`internal/embed/agent/bundle/compat/node-api-target.json` is the semantic
target for the compatibility platform. It groups module loading, process,
async context, streams, encoding, crypto, filesystem, networking,
compression, child processes, WASM, diagnostics/observability, workers,
native/optional dependencies, and package patches into explicit support
statuses:

- `standard`
- `exact-used`
- `compat-partial`
- `shape-only`
- `unsupported-boundary`
- `external-service`
- `package-patched`

Each row records owner, surfaces, packages using the surface, tests, accepted
semantics, known gaps, lifecycle impact, and the next proof required. The file
is checked by `make agent-embed-node-api-target-check` and included in
`make jsbridge-compat-check`.

Unsupported rows are required to be actionable, not only absent. Manifest and
inventory entries for unsupported runtime boundaries carry a `boundaryClass`
and `suggestedOwner`. The current boundary classes are:

- `native-addon`
- `worker`
- `server-listener`
- `external-service`
- `optional-native`
- `unsupported-node-api`

`internal/embed/agent/bundle/compat/capability-matrix.json` is the
capability-level view. It maps Agent, AI SDK, workflow, tools, memory, RAG,
vector/storage, observability, voice, provider, scorer, and eval surfaces to
support levels and proof tiers:

- `import-only` for constructor/import shape without external credentials;
- `offline-fake` for fake model or provider-independent runtime proof;
- `local-service` for local HTTP/WebSocket/service probes;
- `live-provider` for OpenAI or other provider-backed runs;
- `external-service` for Podman-backed storage/vector/service dependencies.

Schema v2 also records Node APIs, known gaps, and promotion conditions. For
example, jsbridge now provides real `diagnostics_channel` publish/subscriber
semantics, but Mastra observability support still does not imply real OTel
active span propagation. General voice support also does not imply Gemini Live
is runnable. Those paths have explicit rows until they are promoted by focused
proof.

The matrix is checked by `make agent-embed-capability-matrix-check` and is
included in `make jsbridge-compat-check`. Use it before claiming a Mastra
capability is supported; every row must point at real fixtures, examples, or
tests.

Use `make jsbridge-compat-report-save` only when a dependency upgrade
intentionally changes the compatibility surface and the diff has been
reviewed.

## Intentional Compatibility Contracts

Some manifest rows are deliberately partial because the current dependency
closure needs import/runtime shape, not full Node behavior:

- `async_hooks` is partial. `AsyncLocalStorage` supports synchronous
  `run`, `enterWith`, `disable`, independent nested instances, and
  `AsyncResource.runInAsyncScope` with captured stores. It also propagates
  stores across Brainkit-owned callback boundaries: timers, timer promise
  callback chains, fetch `.then()` callbacks returned by jsbridge `fetch`, and
  `EventEmitter` listeners, including stream/http callbacks registered through
  that emitter. The JS runtime also binds `bus.subscribe` handlers and
  `bus.callStream` `onChunk` / final promise callbacks. It does not provide
  general Promise-hook propagation across
  arbitrary `await` chains, and `executionAsyncId` / `triggerAsyncId` return
  `0`.
- `diagnostics_channel` is partial runtime compatibility. Channels retain
  subscribers, `publish()` invokes them synchronously in subscription order,
  subscriber errors propagate to the publisher, `hasSubscribers` tracks
  subscribe/unsubscribe transitions, module-level
  `subscribe`/`unsubscribe`/`hasSubscribers` delegate to named channels, and
  `tracingChannel()` exposes real child channels. OTel active span/context
  propagation is still not real until a tracing module owns it.
- `module.createRequire` is bundle-local. It returns the same dynamic
  `require()` shim used by the agent embed, not a filesystem-aware Node
  module resolver.
- Dynamic `require("zod")` and `require("zod/v4")` resolve to the same
  bundled Zod v4 singleton. Dynamic `require("@opentelemetry/api")` returns a
  no-op tracer shape. Dynamic LSP/jsonrpc requires return optional
  shape-only objects. Dynamic `require("execa")` returns the explicit execa
  hook and otherwise throws from that hook when process execution is not
  available.
- Unknown dynamic require requests throw
  `BrainkitUnsupportedDynamicRequireError` with
  `code=BRAINKIT_UNSUPPORTED_DYNAMIC_REQUIRE`; the error records the requested
  specifier, `boundaryCode=BRAINKIT_UNSUPPORTED_BOUNDARY`, `boundaryClass`,
  `suggestedOwner`, package metadata when known, and
  `internal/embed/agent.runtimeGlobalsJS` as owner. Adding a new dynamic
  require path requires a manifest row plus an explicit shim, package patch, or
  unsupported boundary.
- Worker construction and Node server listeners throw
  `BrainkitUnsupportedBoundaryError` with
  `code=BRAINKIT_UNSUPPORTED_BOUNDARY`. Current server listener boundaries are
  `http.createServer`, `https.createServer`, `net.createServer`, and
  `tls.createServer`; listener lifecycle belongs to `modules/gateway` or a
  future server-listener runtime profile, not to the generic client-side
  jsbridge polyfills.

These are not hidden fallbacks. They are manifest rows with focused tests and
must stay documented until a real owner upgrades the behavior.

## Package Patch Registry

Some compatibility work is package-specific rather than a Node/Web API.
Those patches are manifest rows with `kind: "package-patch"` and must
declare:

- `package`
- `versionRange`
- `patchType`
- `required`
- `expected`
- `removalCondition`
- `reason`
- `tests`

Required patch misses fail the bundle build. Optional patch misses print
an explicit optional-not-applied message, so a package upgrade does not
silently invalidate a regex rewrite.

The current registry covers the known exceptional cases: `ws` aliasing,
`lru-cache` CJS resolution, `big.js`, Zod unification,
`vscode-jsonrpc/node`, optional libsql serialization, execa dynamic
import replacement, Mastra RAG Zod function validation, Mastra OpenAI
structured-output schema wrapping, optional tiktoken fallback, and the
disabled Gemini live voice entry.

## Resource Accounting

`Bridge.DebugSnapshot()` exposes a `Resources` map with aggregate counts
for active bridge-owned resources. Polyfills either call
`Bridge.TrackResource(kind)` for scoped lifetimes or register a provider
that reports the size of an owner map.

Current counters include:

- `timers.timeouts`
- `fetch.requests`
- `fetch.responseBodies`
- `fs.fileHandles`
- `fs.watchers`
- `net.tcpSockets`
- `net.tlsSockets`
- `websocket.connections`
- `websocket.pendingDials`
- `exec.spawnedProcesses`
- `exec.commands`
- `wasm.runtimes`
- `wasm.modules`
- `audio.playing`
- `audio.waiters`

Resource counters are intentionally aggregate only: they are useful for
teardown proof and diagnostics without exposing QuickJS values or raw Go
handles.

## Lifecycle And Scale Gate

`make jsbridge-lifecycle-check` is the focused runtime maintenance gate. It
does not rebuild the agent bundle; it proves that the current runtime surfaces
shut down and scale correctly:

- bridge close cancellation for pending fetches, timers, fs handles, spawned
  processes, WebSocket handshakes, and other tracked resources;
- concurrent Mastra Agent fake-provider generation;
- sandbox close while provider-backed generation is pending;
- `modules/jsruntime` hot-unmount while an async JS handler is active, followed
  by clean remount;
- concurrent JavaScript `bus.call` and `bus.callStream` through the shared
  caller path;
- gateway stream concurrency and shutdown behavior;
- plugin and bridge lifecycle regressions that match the focused test names.

When a new polyfill or embed feature owns a long-lived resource, add three
things together: a `Bridge.DebugSnapshot()` resource counter, a direct close or
cancel regression, and coverage in `make jsbridge-lifecycle-check`. A feature
that only works while the Kit lives forever is not complete enough for
hot-mountable Brainkit modules.

## Reading JS Diagnostics

JavaScript failures that cross into Go are wrapped with
`jsbridge.DiagnosticError`. The wrapper preserves the original error for
`errors.As` / `errors.Is`, including `*quickjs.Error`, and adds safe
runtime context:

```text
brainkit js error (owner=jsruntime, phase=eval, source=diagnostic-runtime.ts):
TypeError: missing runtime surface: runtime.getVersionOverrides is not a function
Bridge snapshot: closing=false closed=false activeGoroutines=5 resources={wasm.modules=1, wasm.runtimes=1}
JavaScript stack:
...
```

Fields mean:

- `owner` is the runtime boundary that observed the failure, usually
  `agent-embed` or `jsruntime`.
- `phase` names the step: bundle load, runtime eval, call, stream,
  scorer run, workflow step, or provider request when known.
- `source` is the bundle file, deployment source, module path, or
  synthetic dispatch file.
- `function`, `provider`, and `model` appear only when that context is
  known and safe to expose.
- `Bridge snapshot` is aggregate lifecycle/resource state, not raw
  QuickJS values.
- `JavaScript cause` and `JavaScript stack` preserve the underlying
  dependency failure. For scorer/Agent failures, the useful line is
  often the cause below an outer Mastra message such as
  `Scorer Run Failed`.

Diagnostics intentionally avoid prompt bodies, API keys, request bodies,
and provider payloads by default. If a failure only says
`not a function`, add the missing surface name at the owner boundary or
in the package patch so the final diagnostic names the actionable API,
for example `runtime.getVersionOverrides is not a function`.

## Conformance Packs

`internal/jsbridge/conformance_test.go` runs table-driven snippets in a
real QuickJS bridge. The packs cover:

- Web APIs: URL, URLSearchParams, AbortController, EventTarget, Fetch
  classes, FormData, Blob/File, Web Streams, and Audio shape.
- Node core: process, Buffer, crypto, os, path, EventEmitter, streams,
  timers/promises, fs, child_process, assert, querystring, StringDecoder,
  util.types, zlib, dns, and client-side `http`/`https` request/get over
  fetch.
- Unsupported typed failures: http/https server creation, worker_threads,
  unsupported crypto cipher creation, and zlib brotli.

`internal/embed/agent/conformance_test.go` covers the SES and bundle load
contract: bundle exports survive lockdown, Compartment/harden/lockdown
exist, intrinsics are frozen, pre-lockdown Math/Date captures exist, and
Compartment endowments evaluate correctly.

## Dependency Failure Playbook

When a Mastra, AI SDK, provider, or storage dependency fails in QuickJS:

1. Reproduce with `make agent-embed-rebuild-check` or a focused fixture.
2. If the error is `PACKAGE_RESOLVER_UNSUPPORTED_IMPORT`, it happened before
   jsbridge runtime execution. The default package deploy profile is
   `source-relative`; only relative package files and the bare endowments
   `kit`, `ai`, `agent`, and `compiler` are accepted. Broad npm dependency
   resolution belongs to a future opt-in npm ecosystem resolver profile.
3. Read the `brainkit js error (...)` context first. Confirm the owner,
   phase, source, provider/model identifiers when present, bridge
   snapshot, JavaScript cause, and JavaScript stack.
4. Inspect `make jsbridge-compat-report` for new builtins, subpaths,
   dynamic requires, externals, or package patch drift.
5. Add or update the manifest row with owner, status, reason, tests, and
   resources.
6. Implement runtime behavior in `internal/jsbridge` when the gap is a
   Node/Web API. Keep `build.mjs` as a mechanical re-export.
7. Use a package-patch row only for dependency-specific build or source
   quirks. Mark misses required unless the package version legitimately
   may not contain the pattern.
8. Add direct conformance/unit coverage and, where needed, a fixture under
   `test/fixtures`.
9. If the failure involves cancellation, streams, goroutines, timers, sockets,
   subprocesses, or mount/unmount behavior, add a lifecycle regression and run
   `make jsbridge-lifecycle-check`.
10. Rebuild bytecode and rerun `make agent-embed-rebuild-check`.

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

Every polyfill ships with Go unit tests under `internal/jsbridge/*_test.go`,
with conformance packs for cross-surface behavior. Representative coverage:

- `crypto_test.go` — hash, hmac, pbkdf2Sync, randomBytes,
  timingSafeEqual, subtle digest/sign/deriveBits.
- `nodestreams_test.go` — `for await` iteration, `pipe`, `return()`
  data transfer.
- `nodecompat_test.go` — assert, querystring, StringDecoder, util,
  perf_hooks, client-side http/https, unsupported http/https server APIs,
  worker_threads, partial async_hooks, and diagnostics_channel runtime
  semantics.
- `net_test.go` — TCP connect/write/close, TLS upgrade.
- `dns_test.go` — `dns.lookup` (sync + promises).
- `zlib_test.go` — inflate/deflate/gzip round trips.
- `bridge_test.go` — fetch, timers, fs, exec, resource snapshots, close
  behavior, and cross-polyfill bridge behavior.
- `websocket_test.go` — text + binary round-trip, `Authorization`
  header forwarded through the handshake.
- `audio_test.go` — sink dispatch, mime sniff, pause/cancel,
  Null default.
- `conformance_test.go` — Web API, Node core, unsupported typed-failure,
  and resource-drain packs.
- `internal/embed/agent/conformance_test.go` — SES/load-order, bundle
  export contract, dynamic `require()`, and `module.createRequire` contract.

The jsbridge tests run under the standard Go toolchain — no Node, no
esbuild, no browser — because the polyfills are Go code. Bundle-level
checks are intentionally separate and run through `make jsbridge-compat-check`
or the full `make agent-embed-rebuild-check` upgrade gate.

Use this gate split during development:

- `make jsbridge-compat-check` for manifest, inventory, Node API target,
  capability-matrix, package-patch, and bundle-stub drift.
- `make jsbridge-lifecycle-check` for cancellation, streaming, runtime
  unmount, resource-drain, and shared-caller scale proof.
- `make agent-embed-check` for current artifacts plus the focused fixture set.
- `make agent-embed-rebuild-check` after dependency, patch, stub, or bytecode
  changes.
- `make examples-smoke-live` when provider-backed examples should be exercised
  with `OPENAI_API_KEY` loaded from the repo root environment.

## See Also

- `internal/jsbridge/*.go` — polyfill sources.
- `internal/embed/agent/sandbox.go` — canonical load order.
- `internal/embed/agent/bundle/build.mjs` — stub definitions.
- [bundle-and-bytecode.md](bundle-and-bytecode.md) — how the stubs
  and polyfills meet at bundle load time.
- [deployment-pipeline.md](deployment-pipeline.md) — how deployed
  `.ts` code inherits the polyfills through Compartment endowments.
