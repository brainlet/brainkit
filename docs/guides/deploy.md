# Deploy and Teardown

Deploy evaluates `.ts` code inside a **SES Compartment** with isolated globals and a hardened Kit API. Each deployment is tracked by source name. Resources created during evaluation (agents, tools, workflows, bus subscriptions, memory configs) are automatically tracked and cleaned up on teardown.

> **Key concept**: A Compartment is a sandboxed JavaScript execution context. It gets its own global object with only the Kit API (agent, createTool, bus, ai, z, etc.) exposed as frozen endowments. Code inside a Compartment cannot touch the host runtime's globals.

---

## Go API

### Deploy

```go
resources, err := kit.Deploy(ctx, "team.ts", `
    agent({
        name: "researcher",
        model: "openai/gpt-4o-mini",
        instructions: "Research topics thoroughly.",
    });
    agent({
        name: "writer",
        model: "openai/gpt-4o-mini",
        instructions: "Write clear documentation.",
    });
`)
// resources = [{type:"agent", id:"researcher", source:"team.ts"}, ...]
```

Deploy fails if the source is already deployed. Use `Redeploy` to replace.

### Teardown

```go
removed, err := kit.Teardown(ctx, "team.ts")
// removed = 2 (number of resources cleaned up)
```

Teardown removes all resources created by that source file (agents unregistered, bus subscriptions removed, tools deregistered) and drops the Compartment reference. Idempotent -- returns 0 if the source was not deployed.

### Redeploy

```go
resources, err := kit.Redeploy(ctx, "team.ts", newCode)
```

Tears down the old deployment, then deploys new code. If teardown fails, it logs a warning but proceeds with the fresh deploy.

### List Deployments

```go
deployments := kit.ListDeployments()
for _, d := range deployments {
    fmt.Printf("%s: %d resources (deployed %s)\n", d.Source, len(d.Resources), d.CreatedAt)
}
```

---

## Bus Topics

All deploy operations are available as bus messages for use from JS, WASM, or plugins.

| Topic | Payload | Response |
|-------|---------|----------|
| `kit.deploy` | `{"source":"x.ts","code":"..."}` | `{"deployed":true,"resources":[...]}` |
| `kit.teardown` | `{"source":"x.ts"}` | `{"removed":2}` |
| `kit.list` | `{}` | `[{"source":"x.ts","createdAt":"...","resources":[...]}]` |
| `kit.redeploy` | `{"source":"x.ts","code":"..."}` | `{"deployed":true,"resources":[...]}` |

### From .ts code (via bus)

```typescript
import { bus } from "kit";

// Deploy
const result = await bus.ask("kit.deploy", {
    source: "helper.ts",
    code: `agent({ name: "helper", model: "openai/gpt-4o-mini", instructions: "Help users." });`,
});

// Teardown
await bus.ask("kit.teardown", { source: "helper.ts" });

// List
const deployments = await bus.ask("kit.list", {});
```

### From WASM (via askAsync)

```assemblyscript
import { bus } from "brainkit";

export function deployAgent(topic: string, payload: string): void {
    bus.askAsyncRaw(
        "kit.deploy",
        '{"source":"wasm-agent.ts","code":"agent({ name: \\"bot\\", model: \\"openai/gpt-4o-mini\\", instructions: \\"Hi\\" });"}',
        "onDeployed"
    );
}

export function onDeployed(topic: string, payload: string): void {
    log("deployed: " + payload);
}
```

### From plugins (via SDK client)

```go
// Deploy
err := client.Deploy(ctx, "plugin-agent.ts", `
    agent({ name: "plugin-bot", model: "openai/gpt-4o-mini", instructions: "test" });
`)

// Teardown
err := client.Teardown(ctx, "plugin-agent.ts")
```

---

## Resource Tracking

When code runs inside `Deploy`, every Kit API call that creates a resource is tracked against the source filename. On teardown, resources are removed in LIFO order (last created, first destroyed).

### Tracked resource types

| Type | Created by | Cleanup on teardown |
|------|-----------|---------------------|
| `agent` | `agent({ name, ... })` | Unregistered from agent registry, JS reference dropped |
| `tool` | `createTool(...)` | Deregistered from tool registry |
| `workflow` | `createWorkflow(...)` | Removed from workflow store |
| `memory` | `createMemory(...)` | Config reference dropped |
| `subscription` | `bus.on(...)` | Unsubscribed from bus |

### Querying resources

```go
// All resources
resources, _ := kit.ListResources()

// Filter by type
agents, _ := kit.ListResources("agent")

// Resources from a specific file
res, _ := kit.ResourcesFrom("team.ts")

// Remove a single resource
kit.RemoveResource("agent", "researcher")
```

---

## SES Compartment Details

Each `Deploy` call creates a new SES `Compartment` with per-source endowments:

- `agent`, `createTool`, `createSubagent`, `createWorkflow`, `createStep`, `createMemory`
- `z` (Zod schema builder)
- `ai`, `tools`, `bus`, `agents`, `mcp`
- `console`, `JSON`
- All storage constructors: `LibSQLStore`, `PostgresStore`, `MongoDBStore`, etc.

Endowments are frozen (hardened). Compartment code cannot modify them or add properties to the shared globals. Each Compartment evaluates independently -- one deployment cannot see or interfere with another's variables.

---

## Examples

### Deploy an agent team

```go
resources, err := kit.Deploy(ctx, "support-team.ts", `
    const researcher = agent({
        name: "researcher",
        model: "openai/gpt-4o",
        instructions: "Research questions thoroughly. Cite sources.",
    });

    const responder = agent({
        name: "responder",
        model: "openai/gpt-4o-mini",
        instructions: "Answer user questions clearly and concisely.",
        agents: { researcher },
        maxSteps: 5,
    });
`)
// resources: [{type:"agent", id:"researcher"}, {type:"agent", id:"responder"}]
```

### Redeploy with new configuration

```go
resources, err := kit.Redeploy(ctx, "support-team.ts", `
    const responder = agent({
        name: "responder",
        model: "openai/gpt-4o",  // upgraded model
        instructions: "Answer user questions. Be thorough.",
        maxSteps: 10,
    });
`)
// Old researcher + responder torn down, new responder deployed
```

### Teardown everything from a file

```go
removed, err := kit.Teardown(ctx, "support-team.ts")
fmt.Printf("cleaned up %d resources\n", removed)
```

---

## Error Handling

- Deploying a source that is already deployed returns an error. Use `Redeploy` instead.
- If code evaluation fails mid-deploy, partial resources are cleaned up automatically.
- Teardown is idempotent -- tearing down a source that was never deployed returns 0.
- Both `source` and `code` are required for `kit.deploy`. Omitting either returns an error.
