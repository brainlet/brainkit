# Deployment Pipeline

When you call `kit.Deploy("my-service.ts", code)`, the code goes through a 5-stage pipeline before it runs. Each stage exists for a specific reason, and the order is load-bearing.

## The Pipeline

```
.ts source code
    │
    ├─ 1. TypeScript transpilation (typescript-go)
    │     Strips: type annotations, interfaces, generics, type aliases, `import type`
    │     Keeps: all runtime code, imports, exports, async/await, comments
    │
    ├─ 2. ES import stripping (regex)
    │     Removes: `import { X } from "module";` lines
    │     Why: Compartments inject symbols as globals (endowments), not ES modules
    │
    ├─ 3. Compartment creation
    │     Creates a new SES Compartment with per-source endowments
    │     Sets deployment namespace: "my-service.ts" → "ts.my-service"
    │
    ├─ 4. Code evaluation
    │     Wraps code in `(async () => { ... })()`
    │     Evaluates inside the Compartment via EvalTS
    │     Top-level await works — the entire body is async
    │
    └─ 5. Resource tracking
          Collects all resources created during evaluation
          Stores deployment info (source, createdAt, resources)
```

## Stage 1: TypeScript Transpilation

If the source file ends in `.ts`, brainkit transpiles it to JavaScript using a vendored copy of microsoft/typescript-go. This is a native Go implementation — no Node.js, no esbuild, no subprocess.

```go
// kit/handlers_lifecycle.go
if strings.HasSuffix(source, ".ts") {
    js, transpileErr := typescript.Transpile(code, typescript.TranspileOptions{FileName: source})
    if transpileErr != nil {
        return nil, fmt.Errorf("deploy %s: transpile: %w", source, transpileErr)
    }
    code = stripESImports(js)
}
```

The transpiler strips everything that's type-only:
- `type Foo = { ... }` → removed
- `interface Bar { ... }` → removed
- `function greet(name: string): void` → `function greet(name) { }`
- `const x: number = 5` → `const x = 5`
- `import type { Foo } from "module"` → removed
- `import { Agent } from "agent"` → kept (runtime import, stripped in stage 2)

If the source is `.js`, stages 1 and 2 are skipped entirely.

## Stage 2: ES Import Stripping

After transpilation, ES import statements are removed:

```go
// kit/handlers_lifecycle.go
var esImportRe = regexp.MustCompile(`(?m)^import\s+(type\s+)?(\{[^}]*\}|[^\s]+)\s+from\s+"[^"]+";\s*\n?`)

func stripESImports(js string) string {
    return esImportRe.ReplaceAllString(js, "")
}
```

This turns:
```typescript
import { Agent, createTool, z } from "agent";
import { bus, model, output } from "kit";
```
Into nothing. The symbols `Agent`, `createTool`, `z`, `bus`, `model`, `output` are not resolved via ES module system — they're injected as Compartment endowments (globals).

**Why not use ES modules?** SES Compartments don't support ES module resolution. They provide a flat global scope populated by endowments. The four modules (`"kit"`, `"ai"`, `"agent"`, `"compiler"`) are registered as QuickJS ES modules for `import`-style access at the top level, but inside a Compartment, the code runs as a script, not a module. Endowments are the only mechanism.

## Stage 3: Compartment Creation

Each deployment gets its own SES Compartment with per-source endowments:

```javascript
// kit/handlers_lifecycle.go — the actual eval code
var __endowments = globalThis.__kitEndowments("my-service.ts");
var __c = new globalThis.Compartment({ __options__: true, globals: __endowments });
globalThis.__kit_compartments["my-service.ts"] = __c;
```

The `__kitEndowments(source)` function (defined in kit_runtime.js) creates an endowments object with ~80 properties:

**brainkit infrastructure** — `bus` (scoped: `bus.on("topic")` auto-prefixes with `ts.my-service.topic`), `kit` (scoped: `kit.register` tracks resources against this source), `model`, `embeddingModel`, `provider`, `storage`, `vectorStore`, `registry`, `tools`, `tool`, `fs`, `mcp`, `output`, `generateWithApproval`

**AI SDK** — `generateText`, `streamText`, `generateObject`, `streamObject`, `embed`, `embedMany`, `z`

**Mastra** — `Agent`, `createTool`, `createWorkflow`, `createStep`, `Memory`, `InMemoryStore`, `LibSQLStore`, `PostgresStore`, `MongoDBStore`, `UpstashStore`, `LibSQLVector`, `PgVector`, `MongoDBVector`, `Workspace`, `LocalFilesystem`, `LocalSandbox`, `MDocument`, `GraphRAG`, `createVectorQueryTool`, `createDocumentChunkerTool`, `createGraphRAGTool`, `rerank`, `rerankWithScorer`, `Observability`, `DefaultExporter`, `createScorer`, `runEvals`, `RequestContext`, `ModelRouterEmbeddingModel`

**Web APIs** — `fetch`, `Headers`, `Request`, `Response`, `URL`, `URLSearchParams`, `AbortController`, `AbortSignal`, `TextEncoder`, `TextDecoder`, `ReadableStream`, `WritableStream`, `TransformStream`, `atob`, `btoa`, `crypto` (merged WebCrypto + Node.js), `structuredClone`

**Node.js compat** — `Buffer`, `process`, `EventEmitter`, `stream`, `net`, `os`, `dns`, `zlib`, `child_process`, `GoSocket`

**JS built-ins** — `JSON`, `Promise`, `setTimeout`, `setInterval`, `clearTimeout`, `clearInterval`, `queueMicrotask`, `console` (per-source tagged — `console.log` inside `my-service.ts` logs as `[my-service.ts] [log] message`), `Date` (BrainkitDate — restores `Date.now()` that SES blocks), `Math` (restores `Math.random()` that SES blocks)

The endowments are frozen via `harden()` — Compartment code cannot modify them.

### Deployment Namespace

Each .ts file gets a mailbox namespace derived from its filename:

```
my-service.ts    → ts.my-service
nested/svc.ts    → ts.nested.svc
agents.ts        → ts.agents
```

Inside the Compartment, `bus.on("greet", handler)` subscribes to `ts.my-service.greet`. External code sends messages to this topic via `sdk.SendToService(rt, ctx, "my-service.ts", "greet", payload)` from Go, or `bus.sendTo("my-service.ts", "greet", data)` from another .ts file.

## Stage 4: Code Evaluation

The user's code is wrapped in an async IIFE and evaluated inside the Compartment:

```javascript
await __c.evaluate('(async () => { ' + code + ' })()');
```

Top-level `await` works because the entire body is inside an async function. This means deployed .ts code can do things like:

```typescript
// This works — top-level await
const result = await generateText({
    model: model("openai", "gpt-4o-mini"),
    prompt: "Hello",
});
output({ text: result.text });
```

If evaluation fails (syntax error, runtime exception, API call failure), any resources created before the error are cleaned up automatically via `TeardownFile(source)`, and the Compartment reference is removed.

## Stage 5: Resource Tracking

After successful evaluation, brainkit collects all resources that were created during the eval. Resources are tracked by the `_resourceRegistry` in kit_runtime.js — every call to `kit.register(type, name, ref)` adds an entry with the current source filename.

```go
resources, err := k.ResourcesFrom(source)
k.deployments[source] = &deploymentInfo{
    Source:    source,
    CreatedAt: time.Now(),
    Resources: resources,
}
```

Tracked resource types:

| Type | Created by | Cleanup on teardown |
|------|-----------|---------------------|
| `tool` | `kit.register("tool", name, toolRef)` | Deregistered from Go tool registry + JS resource registry |
| `agent` | `kit.register("agent", name, agentRef)` | Unregistered from Go agent registry + JS resource registry |
| `workflow` | `kit.register("workflow", name, wfRef)` | Removed from JS resource registry |
| `memory` | `kit.register("memory", name, memRef)` | Removed from JS resource registry |
| `subscription` | `bus.on(topic, handler)` or `bus.subscribe(topic, handler)` | Unsubscribed from Go transport + removed from JS `__bus_subs` |

## Teardown

`kit.Teardown(ctx, "my-service.ts")` reverses the deployment:

1. Iterates all resources registered by this source file in LIFO order
2. For each resource, calls the cleanup function stored at registration time (unregister tools, unsubscribe bus handlers, etc.)
3. Drops the Compartment reference (`delete globalThis.__kit_compartments[source]`)
4. Removes the deployment entry from the Kit's tracking map

Teardown is idempotent — tearing down a source that was never deployed returns 0.

## Redeploy

`kit.Redeploy(ctx, source, newCode)` is teardown + deploy in one call. If teardown fails, it logs a warning but proceeds with the fresh deploy. The old resources are gone regardless.

## The EvalTS Wrapper

All .ts evaluation goes through `Kit.EvalTS`, which wraps user code with source tracking:

```go
func (k *Kit) EvalTS(ctx context.Context, filename, code string) (string, error) {
    wrapped := fmt.Sprintf(`(async () => {
        return await globalThis.__kitRunWithSource(%q, async () => {
            const { bus, kit, model, provider, storage, vectorStore, registry, tools, fs, mcp, output } = globalThis.__kit;
            %s
        });
    })()`, filename, code)

    if k.bridge.IsEvalBusy() {
        return k.bridge.EvalOnJSThread(filename, wrapped)
    }
    return k.agents.Eval(ctx, filename, wrapped)
}
```

`__kitRunWithSource` sets the current source filename so `kit.register` knows which deployment to attribute resources to. The destructuring makes all kit module exports available as local variables. If the bridge is already busy (another EvalTS is active — e.g., a bus handler triggered during deployment), it routes through `EvalOnJSThread` which uses `ctx.Schedule` to queue the eval on the JS thread.
