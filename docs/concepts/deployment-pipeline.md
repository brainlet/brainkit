# Deployment Pipeline

A deployment is a `.ts` package registered with a live Kit. Deploying
runs the package entry inside a fresh SES Compartment, exposes any
`bus.on(topic, …)` handler at `ts.<pkg>.<topic>`, and binds every
resource the package registers (agents, tools, workflows, memories,
subscriptions) to the package name for teardown.

All three build surfaces — Go library, CLI, in-JS `bus.call` —
converge on the same typed topic: `package.deploy`.

## Building a Package

`brainkit.Package` is a value type with three producers:

```go
// Inline: single file of source as a string.
brainkit.PackageInline("greeter", "greeter.ts",
    `bus.on("hello", (m) => m.reply({ greeting: "hi " + m.payload.name }));`)

// File on disk: single `.ts`. Imports are bundled by esbuild at
// deploy time on the handler side.
brainkit.PackageFromFile("./services/greeter.ts")

// Directory with manifest.json: multi-file package, version,
// additional files. The handler reads the manifest and bundles
// the entry.
brainkit.PackageFromDir("./services/greeter")
```

Each producer returns a `Package{Name, Version, Entry, Files, path}`
value. `path` is set only by the `FromDir`/`FromFile` producers and
tells the handler to bundle from disk; `Files` is set by `PackageInline`
and carries the source verbatim.

`Deploy` sends the package as a `sdk.PackageDeployMsg`:

```go
resp, err := kit.Deploy(ctx, pkg)  // Call[PackageDeployMsg, PackageDeployResp]
// → DeployResult{Name, Version, Source, Resources}
```

`PackageDeployMsg` carries either `Path` (filesystem-backed) or
`Manifest + Files` (inline). The handler owns all bundling logic —
the Go caller never runs esbuild. See
`sdk/package_deploy_messages.go`.

## Entry Points: Go, CLI, In-JS

The same bus topic powers three very different callers:

### Go library

```go
_, err := kit.Deploy(ctx, brainkit.PackageFromDir("./services/greeter"))
```

### CLI

```sh
brainkit deploy ./services/greeter
```

The CLI walks the path, packs it into `PackageDeployMsg`, and POSTs it
to the running Kit's gateway at `/api/bus` (or `/api/stream` when
chunked output is requested).

### Inside a running `.ts`

```typescript
const resp = await bus.call("package.deploy",
    { manifest: { name: "child", entry: "child.ts" },
      files: { "child.ts": source } },
    { timeoutMs: 30000 });
```

`examples/agent-spawner/main.go` uses this form to let an architect
agent design and deploy new agents at runtime. The deployed agent is a
first-class bus citizen immediately — no orchestration of a parent
process required.

## Pipeline Stages

The deploy handler runs the package through six stages:

### 1. Bundle / load

When `PackageDeployMsg.Path` is set, the handler reads
`manifest.json` (if any), resolves the entry, and runs esbuild inline
(pure-Go port) to produce a single JS blob with dependencies inlined.
When `PackageDeployMsg.Files` is set, the files are loaded verbatim —
no bundler is run, because inline packages are assumed single-file.

### 2. TypeScript transpile

If the entry ends in `.ts`, the source is fed through the vendored
microsoft/typescript-go transpiler. Types, interfaces, generics, and
`import type` lines are stripped; every runtime construct (imports,
async/await, classes, top-level await) is preserved.

### 3. ES import strip

Runtime `import` lines such as `import { Agent, createTool, z } from
"agent"` are removed before evaluation. The symbols they refer to are
injected as Compartment endowments instead — the deployed code sees
them as globals.

### 4. Compartment + endowments

The handler creates a fresh `Compartment` per deployment. The
Compartment's globals include:

- **Bus surface** — `bus` (with `bus.on` auto-prefixed to
  `ts.<pkg>.<topic>`), `kit` (with `kit.register` attributing resources
  to this package).
- **AI SDK** — `generateText`, `streamText`, `generateObject`,
  `streamObject`, `embed`, `embedMany`, `z`.
- **Mastra exports** — `Agent`, `createTool`, `createWorkflow`,
  `createStep`, `Memory`, `LibSQLStore`, `PgVector`, `MDocument`,
  `createVectorQueryTool`, `createScorer`, `Observability`, etc.
- **Web APIs** — `fetch`, `Headers`, `Request`, `Response`, `URL`,
  `URLSearchParams`, `AbortController`, `AbortSignal`, `TextEncoder`,
  `TextDecoder`, `ReadableStream`, `WritableStream`, `TransformStream`,
  `atob`, `btoa`, `crypto`, `structuredClone`.
- **Node.js compat** — `Buffer`, `process`, `EventEmitter`, `stream`,
  `net`, `os`, `dns`, `zlib`, `child_process`, `fs`.
- **Tamed intrinsics** — `Date` and `Math` are restored via SES
  endowments that capture the real implementations before
  `lockdown()`. Without this, `Date.now()` and `Math.random()` would
  throw inside a Compartment.
- **Per-source logger** — `console.log` inside `greeter.ts` logs with
  a `[greeter.ts]` tag so output can be attributed.

Every endowment is hardened (`Object.freeze` + deep freeze). The
package cannot monkey-patch `fetch` for another package.

### 5. Evaluate inside the Compartment

The handler wraps the code in an async IIFE and evaluates it inside
the Compartment:

```javascript
await compartment.evaluate(`(async () => {
    ${code}
})()`);
```

Top-level await works because the entire body is async. Deployed code
can do:

```typescript
const r = await generateText({
    model: model("openai", "gpt-4o-mini"),
    prompt: msg.payload.prompt,
});
msg.reply({ text: r.text });
```

If evaluation throws, the deploy fails and any resources registered up
to that point are rolled back. The Compartment reference is dropped
and `package.deploy` returns an error envelope
(`BridgeError`/`DeployError` depending on the stage).

### 6. Resource tracking

Every `kit.register(type, name, ref)` call made during evaluation
records an entry under the current package. The tracked types are:

| Type           | Created by                                            | Cleanup on teardown                             |
| -------------- | ----------------------------------------------------- | ----------------------------------------------- |
| `tool`         | `kit.register("tool", name, toolRef)`                 | Deregister from shared tool registry.           |
| `agent`        | `kit.register("agent", name, agentRef)`               | Unregister from agent registry.                 |
| `workflow`     | `kit.register("workflow", name, wf)`                  | Remove from JS workflow registry.               |
| `memory`       | `kit.register("memory", name, memRef)`                | Remove from JS memory registry.                 |
| `subscription` | `bus.on(topic, h)` or `bus.subscribe(topic, h)`       | Unsubscribe from transport + drop JS handler.   |

Resources appear in `DeployResult.Resources` so the caller can see what
was registered. `kit.List(ctx)` returns the names and status of every
currently deployed package.

## Addressing a Deployment

`bus.on("hello", …)` inside package `greeter` subscribes to
`ts.greeter.hello`. Callers address that mailbox through any of:

```go
// Go, in-process
reply, _ := brainkit.Call[sdk.CustomMsg, json.RawMessage](
    kit, ctx,
    sdk.CustomMsg{Topic: "ts.greeter.hello",
        Payload: json.RawMessage(`{"name":"world"}`)},
    brainkit.WithCallTimeout(2*time.Second))
```

```typescript
// Another .ts package
const r = await bus.call("ts.greeter.hello", { name: "world" },
    { timeoutMs: 2000 });
// Or the symmetric helper:
await bus.sendTo("greeter", "hello", { name: "world" });
```

```sh
# CLI against a running Kit's gateway
brainkit call ts.greeter.hello --payload '{"name":"world"}'
```

## Lifecycle: Teardown, Redeploy, Get, List

```go
err := kit.Teardown(ctx, "greeter")        // revert every registered resource
info, ok, _ := kit.Get(ctx, "greeter")     // status + version
pkgs, _ := kit.List(ctx)                   // everything currently deployed
```

Redeploy is a `Deploy` on an existing package name — the handler tears
down the old instance and brings up the new one in a single bus call.
`DeployResult.Resources` reflects the newly registered set. Teardown
is idempotent; tearing down a name that does not exist returns
`{Removed: false}` without error.

Every deploy/teardown is a typed `package.*` call, so the same
control-plane surface is available from any subsystem — a schedule can
redeploy packages, a plugin can teardown a package, a workflow can
deploy a package as part of a step.

## Manifest Format

```json
{
  "name": "greeter",
  "version": "0.1.0",
  "entry": "greeter.ts"
}
```

`version` is optional. `entry` is required for inline and dir-based
packages; `PackageFromFile` synthesizes a manifest with the filename
stem as `name` and the basename as `entry`.

## Common Pitfalls

- **Missing deadline.** `Call[PackageDeployMsg, …]` requires a deadline;
  `kit.Deploy` sets 30s by default, but if you call it directly with
  no context timeout it errors out immediately. Pass
  `WithCallTimeout(d)` or a context with a deadline.
- **Circular packages.** A package that deploys another that deploys
  itself will trip the depth middleware (default 16) and return
  `CYCLE_DETECTED`. Split the work across separate bus calls with
  explicit continuation.
- **Missing provider key at deploy time.** Providers are resolved
  lazily at `model(...)` call time, not at deploy time. Deployment
  succeeds even if `OPENAI_API_KEY` is missing; the error surfaces on
  the first agent invocation.

## See Also

- `examples/hello-embedded/main.go` — minimal inline deploy.
- `examples/agent-spawner/main.go` — in-JS deploy via `bus.call`.
- `examples/go-tools/main.go` — deploying a package that consumes
  Go-registered tools.
- `sdk/package_deploy_messages.go` — typed message contracts.
- [bus-and-messaging.md](bus-and-messaging.md) — how `ts.<pkg>.<topic>`
  fits into the larger bus model.
- [bundle-and-bytecode.md](bundle-and-bytecode.md) — how the runtime
  itself (Mastra + polyfills) is assembled once at process start.
