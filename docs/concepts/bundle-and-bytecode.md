# Bundle and Bytecode

Every Kit embeds a single pre-built JavaScript bundle that carries the
entire Mastra framework, the Vercel AI SDK, 12 provider factories, and
Zod. It is built once per release with esbuild, compiled to QuickJS
bytecode, and embedded in the Go binary via `//go:embed`. A fresh Kit
loads that bytecode in ~200 ms — versus ~500 ms to parse the raw JS —
and that is what makes `brainkit.New` a library call instead of a
subprocess bring-up.

## What the Bundle Contains

`internal/embed/agent/agent_embed_bundle.js` (~16 MB) and the
accompanying `.bc` file (~15 MB) cover:

- **Mastra core** — `Agent`, `createTool`, `createWorkflow`,
  `createStep`, `Memory`, `RequestContext`, `Observability`,
  `DefaultExporter`.
- **Mastra stores** — `InMemoryStore`, `LibSQLStore`, `PostgresStore`,
  `MongoDBStore`, `UpstashStore`.
- **Mastra vectors** — `LibSQLVector`, `PgVector`, `MongoDBVector`.
- **Mastra workspace** — `Workspace`, `LocalFilesystem`,
  `LocalSandbox`.
- **Mastra RAG** — `MDocument`, `GraphRAG`, `createVectorQueryTool`,
  `createDocumentChunkerTool`, `rerank`, `rerankWithScorer`.
- **Mastra evals** — `createScorer`, `runEvals`,
  `ModelRouterEmbeddingModel`.
- **AI SDK** — `generateText`, `streamText`, `generateObject`,
  `streamObject`, `embed`, `embedMany`.
- **12 AI-SDK provider factories** — `createOpenAI`,
  `createAnthropic`, `createGoogleGenerativeAI`, `createMistral`,
  `createXai`, `createGroq`, `createDeepSeek`, `createCerebras`,
  `createPerplexity`, `createTogetherAI`, `createFireworks`,
  `createCohere`.
- **Zod v4** — `z`.
- **SES** is loaded separately from `ses.umd.js`.

Everything lands on `globalThis.__agent_embed` after the bundle
finishes evaluating.

Source of truth: `internal/embed/agent/bundle/entry.mjs`.

## The Build Pipeline

Bundle production lives in `internal/embed/agent/bundle/`:

```
bundle/
├── build.mjs      ← esbuild driver
├── entry.mjs      ← re-exports every public symbol
├── meta.json      ← esbuild metadata for size reports
├── node_modules/  ← npm install output used by build.mjs
└── package.json
```

To rebuild:

```bash
cd internal/embed/agent/bundle && node build.mjs          # 1. Rebuild JS
go run internal/embed/agent/cmd/compile-bundle/main.go     # 2. Recompile bytecode
go build ./...                                             # 3. Re-embed both
```

esbuild settings (`format: "iife"`, `platform: "browser"`, `minify:
true`, `treeShaking: true`) produce a single IIFE that attaches its
exports to `globalThis.__agent_embed`. A custom plugin —
`nodeStubPlugin` — intercepts every bare `import ... from "stream"`,
`"crypto"`, `"net"`, etc. and replaces it with a thin re-export from
`globalThis`:

```javascript
// build.mjs — stream stub excerpt
"stream": `
  var s = globalThis.stream || {};
  export var Readable = s.Readable;
  export var Writable = s.Writable;
  export var Duplex = s.Duplex;
  export var Transform = s.Transform;
  export default s;
`,
```

The stubs never contain logic — the actual implementations are the Go
polyfills in `internal/jsbridge/*.go` (loaded into the same
`globalThis` before the bundle evaluates). That invariant —
"jsbridge-first, bundle stubs are re-exports" — is enforced by CLAUDE.md
and spot-checked in `jsbridge/*_test.go`.

## The Load Order

`LoadBundle` in `internal/embed/agent/embed.go` runs five phases in
order:

```
1. runtimeGlobalsJS  (pre-lockdown captures + require() shim)
2. sesPolyfillsSource (ses_polyfills.js — console/Iterator fixes)
3. sesSource          (ses.umd.js — Compartment/harden/lockdown)
4. sesLockdownJS      (calls lockdown() with tame-friendly options)
5. bundleBytecode     (agent_embed_bundle.bc — preferred)
    OR bundleSource   (agent_embed_bundle.js — fallback)
```

After phase 5, `globalThis.__agent_embed` is populated and the Kit's
Compartment factory can use it to build per-deployment endowments.

### Pre-lockdown captures

SES's `lockdown()` tames `Math.random`, `Date.now`, and the `Date`
constructor as ambient authority — code inside Compartments cannot
call them. The bundle stores the real implementations before lockdown
runs:

```javascript
// runtimeGlobalsJS
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

The deployment pipeline reads `__brainkit_pre_lockdown` when building
per-package Compartment globals, so deployed `.ts` code sees a
working `Date.now()` and `Math.random()` even though the intrinsics
are tamed. See [deployment-pipeline.md](deployment-pipeline.md).

### The `require()` shim

A handful of bundle dependencies (`@opentelemetry/api`, `zod/v4`,
`vscode-jsonrpc/node`, `vscode-languageserver-protocol`, `execa`)
perform dynamic `require()` calls that esbuild cannot resolve at
build time. The runtimeGlobalsJS installs a `globalThis.require`
function that serves no-op stubs for those cases — OTel becomes a
tracer/span pair of shapes that record nothing; missing LSP deps
become empty objects.

## Bytecode Caching

QuickJS supports compiling JavaScript to a portable bytecode that
loads without parsing. `internal/embed/agent/cmd/compile-bundle/main.go`
runs the bundle through `Bridge.CompileBytecode()` and writes
`agent_embed_bundle.bc` (~15 MB) next to the source.

Both files are embedded in the Go binary:

```go
//go:embed agent_embed_bundle.js
var bundleSource string

//go:embed agent_embed_bundle.bc
var bundleBytecode []byte
```

`LoadBundle` prefers bytecode — the JS source is only loaded when the
`.bc` is empty (which would indicate an out-of-band build):

```go
if len(bundleBytecode) > 0 {
    val, err := b.EvalBytecode(bundleBytecode)
    if err != nil { return err }
    val.Free()
    return nil
}
val, err := b.EvalAsync("agent-embed-bundle.js", bundleSource)
```

### The stale bytecode trap

This has burned real bugs. If you change `build.mjs`, you MUST
rebuild both the JS bundle and the bytecode. The three-step sequence
above is non-optional; the rule lives in `CLAUDE.md`:

> After modifying `internal/embed/agent/bundle/build.mjs` (esbuild
> stubs for Node.js modules), you MUST rebuild THREE things in order:
> JS bundle → bytecode cache (.bc) → `go build ./...`.

Skipping the bytecode step leaves the old code live — the new `.js`
is ignored because the Kit preferentially loads the `.bc`. Symptoms
are usually "not a function" errors in seemingly unrelated code.
Renames like `__node_crypto` → `crypto` have historically tripped
this exact trap in the PgVector probe path.

## Scoped Console

`lockdown()` is called with `consoleTaming: "unsafe"` so bundle code
can `console.log` normally, but SES still emits warnings during
lockdown about non-standard QuickJS intrinsics ("Removing unpermitted
intrinsics …"). The runtime mutes every `console.*` method during
`lockdown()` and restores them afterwards, emitting a single
`[brainkit] SES lockdown complete (<n> non-standard intrinsics
removed)` line at debug level.

## Bundle Size (current build)

| File                          | Size   |
| ----------------------------- | ------ |
| `agent_embed_bundle.js`       | ~16.6 MB |
| `agent_embed_bundle.bc`       | ~15.0 MB |

Rough breakdown of the JS bundle:

| Component                                  | Approx. size |
| ------------------------------------------ | ------------ |
| Mastra core + workflows + agents           | ~6 MB        |
| AI SDK + 12 provider factories             | ~4 MB        |
| `tiktoken` (tokenizer for RAG)             | ~2 MB        |
| MongoDB driver                             | ~600 KB      |
| PostgreSQL driver                          | ~400 KB      |
| Zod v4                                     | ~500 KB      |
| `@libsql/client` (HTTP mode)               | ~200 KB      |
| sentiment, xxhash, misc helpers            | ~2 MB        |
| IIFE wrappers, minification overhead        | ~800 KB      |

Numbers are approximate and drift with releases — read `meta.json`
after a build for the authoritative report.

## One Bundle, Many Compartments

The bundle is loaded exactly once per Kit process. Each deployed
`.ts` package gets its own SES Compartment whose globals reference
the same frozen bundle exports. This is the mechanism that keeps the
memory footprint constant per Kit even when dozens of packages are
deployed — you do not pay for Mastra twice. See
[deployment-pipeline.md](deployment-pipeline.md) for the Compartment
construction step.

## See Also

- `internal/embed/agent/embed.go` — `LoadBundle` / `LoadPrelude`
  entry points.
- `internal/embed/agent/bundle/build.mjs` — esbuild driver and the
  full stub table.
- `internal/embed/agent/bundle/entry.mjs` — canonical export list.
- `internal/embed/agent/cmd/compile-bundle/main.go` — bytecode
  compile step.
- [jsbridge-polyfills.md](jsbridge-polyfills.md) — the Go polyfills
  the bundle stubs re-export from.
- [deployment-pipeline.md](deployment-pipeline.md) — how a
  Compartment consumes the bundle's frozen exports at deploy time.
