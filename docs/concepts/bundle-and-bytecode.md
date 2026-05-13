# Bundle and Bytecode

Every Kit embeds a single pre-built JavaScript bundle that carries the
entire Mastra framework, the Vercel AI SDK, 12 provider factories, and
Zod. It is built once per release with esbuild, compiled to QuickJS
bytecode, and embedded in the Go binary via `//go:embed`. A fresh Kit
loads that bytecode in ~200 ms — versus ~500 ms to parse the raw JS —
and that is what makes `brainkit.New` a library call instead of a
subprocess bring-up.

## What the Bundle Contains

`internal/embed/agent/agent_embed_bundle.js` (~19.2 MiB) and the
accompanying `.bc` file (~21.7 MiB) cover:

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
├── compat/
│   ├── manifest.json ← declared Node/Web/package compatibility surface
│   ├── report.json   ← checked report generated from manifest + meta.json
│   └── inventory.json ← package/version to platform-surface inventory
├── build.mjs      ← esbuild driver
├── entry.mjs      ← re-exports every public symbol
├── meta.json      ← esbuild metadata for size reports
├── node_modules/  ← pnpm install output used by build.mjs
├── pnpm-lock.yaml ← authoritative lockfile for the agent bundle
└── package.json
```

To rebuild:

```bash
make agent-embed-rebuild
```

For a Mastra or AI SDK dependency upgrade, use the full compatibility gate:

```bash
make agent-embed-rebuild-check
```

That target rebuilds the JS bundle, recompiles QuickJS bytecode, checks the
compatibility report against `bundle/compat/report.json`, runs the manifest and
package-patch guard tests, and then runs the focused jsbridge/embed/fixture
runtime checks. Use `make jsbridge-compat-report` to print the current
compatibility surface and `make jsbridge-compat-report-save` only when the
changed report is expected and reviewed. Use `make jsbridge-compat-inventory`
to inspect which packages and versions are responsible for each platform
surface; `make jsbridge-compat-inventory-save` updates the checked
`inventory.json`.

Use the maintenance gates by scope:

- `make jsbridge-compat-check` checks compatibility metadata, the package
  patch registry, bundle stubs, inventory, and the Mastra capability matrix
  without rebuilding artifacts.
- `make jsbridge-lifecycle-check` checks runtime lifecycle and scale behavior:
  bridge resource cancellation, active async handler unmount, shared
  `bus.call` / `bus.callStream` concurrency, and gateway stream shutdown.
- `make agent-embed-check` checks the current checked-in artifacts against
  jsbridge, jsruntime, agent embed, and the focused fixture set.
- `make agent-embed-rebuild-check` is the upgrade gate after dependency,
  package patch, stub, bundle, or bytecode changes.
- `make examples-smoke-live` runs provider-backed examples when
  `OPENAI_API_KEY` is available in the repo root environment; use
  `make examples-smoke-all` before release or broad runtime/profile changes.

Raw TypeScript deployment is separate from the agent embed bundle. Default
runtime and package profiles still prepare `.ts` source through the package
source pipeline and `vendor_typescript`; artifact-only runtime profiles reject
raw TypeScript and accept normalized JavaScript artifacts. Do not use the agent
bundle or bytecode path as a reason to remove raw `.ts` support.

esbuild settings (`format: "iife"`, `platform: "browser"`, `minify:
true`, `treeShaking: true`) produce a single IIFE that attaches its
exports to `globalThis.__agent_embed`. A custom plugin —
`nodeStubPlugin` — intercepts every bare `import ... from "stream"`,
`"crypto"`, `"net"`, etc. and replaces it with a thin re-export from
`globalThis`:

```javascript
// build.mjs — stream stub excerpt
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

The stubs never contain logic — the actual implementations are the Go
polyfills in `internal/jsbridge/*.go` (loaded into the same
`globalThis` before the bundle evaluates). That invariant —
"jsbridge-first, bundle stubs are re-exports" — is enforced by the
compatibility manifest tests and by `make jsbridge-compat-check`.

## Compatibility Artifacts

The bundle has a checked compatibility inventory under
`internal/embed/agent/bundle/compat/`:

- `manifest.json` declares every Node module/subpath, web global,
  dynamic `require()`, external package, and package patch.
- `report.json` is generated from the manifest plus esbuild
  `meta.json`.
- `inventory.json` is generated from the manifest, `meta.json`, and
  bundle `package.json`. It links package names/versions to Node APIs,
  external imports, package patches, dependency class, resources/tests,
  and external-service flags.

Unknown Node builtins and subpaths fail the bundle build. Known
unsupported APIs are owned by jsbridge polyfills that throw typed
errors. Package-specific build quirks are registered as
`package-patch` rows with package, version range, patch type,
required/optional miss policy, expected pattern, tests, and removal
condition.

Use the report target while upgrading dependencies:

```bash
make jsbridge-compat-report
make jsbridge-compat-inventory
make jsbridge-compat-check
make jsbridge-lifecycle-check
make agent-embed-check
make agent-embed-rebuild-check
```

## The Load Order

`LoadBundle` in `internal/embed/agent/embed.go` runs five phases in
order:

```
1. runtimeGlobalsJS  (pre-lockdown captures + require() shim)
2. sesPolyfillsSource (ses_polyfills.js — console/Iterator fixes)
3. sesSource          (ses.umd.js — Compartment/harden/lockdown)
4. bundleBytecode     (agent_embed_bundle.bc — preferred)
    OR bundleSource   (agent_embed_bundle.js — fallback)
5. sesLockdownJS      (calls lockdown() with tame-friendly options)
```

After phase 5, `globalThis.__agent_embed` is populated and the Kit's
Compartment factory can use it to build per-deployment endowments.

Bundle load errors are wrapped with JS diagnostics. A failure during
globals setup, SES load, bytecode load, bundle source load, or lockdown
reports `owner=agent-embed`, the load phase, the source artifact, a
safe bridge resource snapshot, and the JavaScript stack/cause when
QuickJS provides one. Use that context before changing the bundle: a
`not a function` error usually means either stale bytecode, a missing
jsbridge-owned platform surface, or a declared package patch that no
longer matches the dependency version.

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
`agent_embed_bundle.bc` (~21.7 MiB in the current build) next to the
source.

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

This has burned real bugs. If you change `build.mjs`, `entry.mjs`, or any
agent embed package dependency, you MUST rebuild both the JS bundle and the
bytecode. The checked command is:

```bash
make agent-embed-rebuild-check
```

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
| `agent_embed_bundle.js`       | ~19.2 MiB |
| `agent_embed_bundle.bc`       | ~21.7 MiB |

Rough breakdown of the JS bundle:

| Component                                  | Approx. size |
| ------------------------------------------ | ------------ |
| `js-tiktoken` tokenizer data/code          | ~5.3 MiB     |
| PDF.js runtime + worker                    | ~3.1 MiB     |
| Mastra core + workflows + agents           | ~3 MiB+      |
| Mammoth document parser                    | ~860 KiB     |
| Mastra observability                       | ~790 KiB     |
| Mastra memory                              | ~950 KiB     |
| Mastra storage adapters                    | ~1.6 MiB     |
| AI SDK + provider factories                | ~1 MiB+      |

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
