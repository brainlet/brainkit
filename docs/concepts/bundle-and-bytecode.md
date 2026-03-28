# Bundle and Bytecode

brainkit bundles the entire Mastra framework + AI SDK + 12 provider factories + Zod into a single JavaScript file that loads into QuickJS at startup. This bundle is 16.5MB of minified JS. Loading it from source takes ~500ms. Loading precompiled bytecode takes ~200ms. Both are embedded in the Go binary via `//go:embed`.

## What's in the Bundle

The bundle (`internal/embed/agent/agent_embed_bundle.js`) is built by esbuild from `internal/embed/agent/bundle/entry.mjs`. It contains:

- **Mastra core**: Agent, createTool, createWorkflow, createStep, Memory, RequestContext
- **Mastra stores**: InMemoryStore, LibSQLStore, PostgresStore, MongoDBStore, UpstashStore
- **Mastra vectors**: LibSQLVector, PgVector, MongoDBVector
- **Mastra workspace**: Workspace, LocalFilesystem, LocalSandbox
- **Mastra RAG**: MDocument, GraphRAG, createVectorQueryTool, createDocumentChunkerTool, rerank
- **Mastra evals**: createScorer, runEvals, Observability, DefaultExporter
- **AI SDK**: generateText, streamText, generateObject, streamObject, embed, embedMany
- **AI SDK providers**: createOpenAI, createAnthropic, createGoogleGenerativeAI, createMistral, createXai, createGroq, createDeepSeek, createCerebras, createPerplexity, createTogetherAI, createFireworks, createCohere
- **Zod v4**: z (schema builder + validation)
- **SES**: Compartment, lockdown, harden (loaded separately as ses.umd.js)

Everything is exposed on `globalThis.__agent_embed` after the bundle loads.

## The Build Process

`internal/embed/agent/bundle/build.mjs` runs esbuild:

```bash
cd internal/embed/agent/bundle && node build.mjs
```

esbuild bundles `entry.mjs` with:
- `format: "iife"` — wraps in an immediately-invoked function
- `platform: "browser"` — no Node.js built-in resolution
- `minify: true` + `treeShaking: true`
- `nodeStubPlugin` — replaces `import { X } from 'stream'` etc. with thin re-exports from globalThis

### Node.js Module Stubs

The Mastra code and its npm dependencies do `import { Readable } from 'stream'`, `import crypto from 'crypto'`, etc. Since esbuild runs in Node.js at build time, it needs to resolve these imports. The `nodeStubPlugin` intercepts them and provides stub modules:

```javascript
// build.mjs — crypto stub
"crypto": `
    var C = globalThis.crypto || {};
    export var createHash = C.createHash || function() { ... };
    export var pbkdf2Sync = C.pbkdf2Sync || function() { ... };
    export var webcrypto = globalThis.crypto;
    export default { createHash, pbkdf2Sync, webcrypto, ... };
`,
```

At runtime, `globalThis.crypto` has the real Go-backed implementations from jsbridge polyfills. The stub just wires the import resolution. See [jsbridge-polyfills.md](jsbridge-polyfills.md).

**Critical rule:** Never put implementations in build.mjs stubs. They are thin re-exports ONLY. Implementations go in `internal/jsbridge/*.go` with Go test coverage.

## The Loading Sequence

`LoadBundle` in `internal/embed/agent/embed.go` loads SES and the Mastra bundle in order:

```
1. runtimeGlobalsJS (agent-embed-setup.js)
   └─ Pre-lockdown captures: saves Math.random, Date.now, Date constructor
      before SES freezes them as "ambient authority"
   └─ require() shim: handles dynamic require() for otel, zod, vscode-jsonrpc

2. sesPolyfillsSource (ses-polyfills.js)
   └─ Console stubs, Iterator prototype fix for QuickJS compatibility

3. sesSource (ses.umd.js)
   └─ SES library: provides Compartment, harden, lockdown

4. sesLockdownJS
   └─ lockdown({ errorTaming: "unsafe", overrideTaming: "moderate",
                 consoleTaming: "unsafe", evalTaming: "unsafe-eval" })
   └─ After this: Math.random(), Date.now(), new Date() throw in Compartments
      (restored via pre-lockdown captures in kit_runtime.js endowments)

5. bundleBytecode (agent_embed_bundle.bc) — preferred
   OR bundleSource (agent_embed_bundle.js) — fallback
   └─ After this: globalThis.__agent_embed has everything
```

### Pre-lockdown Captures

SES lockdown tames `Math.random`, `Date.now`, and `Date()` as "ambient authority" — code in Compartments can't call them. But AI SDK, Mastra, and user code need dates and random numbers.

The solution: capture the real implementations BEFORE lockdown runs:

```javascript
// runtimeGlobalsJS — runs BEFORE lockdown
(function() {
    var _origMathRandom = Math.random.bind(Math);
    var _origDateNow = Date.now.bind(Date);
    var _origDate = Date;
    globalThis.__brainkit_pre_lockdown = {
        mathRandom: _origMathRandom,
        dateNow: _origDateNow,
        Date: _origDate,
    };
})();
```

Then kit_runtime.js builds `BrainkitDate` and a patched `Math` from these captures and includes them in the Compartment endowments. Inside a Compartment, `Date.now()` calls the captured `_origDateNow`, not the tamed one.

## Bytecode Caching

QuickJS can compile JavaScript to bytecode — a binary format that loads faster than parsing and evaluating source text. brainkit pre-compiles the bundle:

```bash
go run internal/embed/agent/cmd/compile-bundle/main.go
# Output: agent_embed_bundle.js (16.2 MB) → agent_embed_bundle.bc (14.8 MB)
```

Both files are embedded in the Go binary:

```go
//go:embed agent_embed_bundle.js
var bundleSource string

//go:embed agent_embed_bundle.bc
var bundleBytecode []byte
```

`LoadBundle` prefers bytecode:

```go
if len(bundleBytecode) > 0 {
    val, err := b.EvalBytecode(bundleBytecode)
    // ...
    return nil
}
// Fallback to source
val, err := b.EvalAsync("agent-embed-bundle.js", bundleSource)
```

### The Stale Bytecode Trap

**This has caused real bugs.** After modifying `build.mjs` (e.g., renaming `__node_crypto` to `crypto`), you MUST rebuild BOTH the JS bundle AND the bytecode:

```bash
cd internal/embed/agent/bundle && node build.mjs          # 1. Rebuild JS bundle
go run internal/embed/agent/cmd/compile-bundle/main.go     # 2. Recompile bytecode
go build ./...                                             # 3. Re-embed both
```

If you skip step 2, the `.bc` file still contains the OLD code. Since `LoadBundle` loads bytecode preferentially, the new `.js` is ignored. The code looks correct in `agent_embed_bundle.js` but the runtime uses the stale `.bc`. This manifests as "not a function" errors in seemingly unrelated code.

Example: after the `__node_*` → clean name rename, the PgVector probe test failed with "not a function" because the bytecode still referenced `globalThis.__node_crypto` which no longer existed. Rebuilding the bytecode fixed it.

## The require() Shim

The Mastra bundle has dynamic `require()` calls for packages that can't be resolved at esbuild time:

```javascript
globalThis.require = function(mod) {
    if (mod === "@opentelemetry/api") return _otelStub;
    if (mod === "zod/v4" || mod === "zod") return globalThis.__zod_v4_module || _zodV4Wrapper;
    if (mod === "vscode-jsonrpc/node") return globalThis.__vscode_jsonrpc_node || {};
    if (mod === "execa") return { execa: globalThis.__execa_polyfill || throwFn };
    return {};
};
```

The OpenTelemetry stub is a full no-op implementation: `_noopTracer`, `_noopSpan`, `SpanStatusCode`, `SpanKind`, `context`, `diag`, `propagation`. This prevents Mastra's OTel instrumentation from crashing in QuickJS while being effectively disabled.

## Bundle Size Breakdown

The 16.5MB bundle breaks down roughly as:

| Component | Size |
|-----------|------|
| Mastra core + workflows + agents | ~6MB |
| AI SDK + 12 provider factories | ~4MB |
| pg driver (node-postgres) | ~400KB |
| MongoDB driver (node-mongodb-native) | ~600KB |
| @libsql/client (HTTP mode) | ~200KB |
| tiktoken (tokenizer for RAG) | ~2MB |
| Zod v4 | ~500KB |
| Other deps (sentiment, xxhash, etc.) | ~2MB |
| esbuild overhead (IIFE wrappers, etc.) | ~800KB |

## The AS Compiler's Separate Bundle

The AssemblyScript compiler (`internal/embed/compiler/`) has its own bundle and its own QuickJS runtime. It doesn't share the Mastra bundle's QuickJS instance because:

1. AS compilation is CPU-bound — it would block the agent runtime
2. The AS compiler needs Binaryen (C bridge), which has its own memory model
3. The compiler is lazy — `ensureCompiler()` only creates it when `wasm.compile` is first called

The compiler bundle has a separate, simpler `build.mjs` with only `fs` and `crypto` stubs (no Mastra, no AI SDK).
