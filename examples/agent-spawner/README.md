# agent-spawner

The peak brainkit use case: **an agent that designs and deploys
other agents at runtime**.

```
  Go                     Kit (JS/TS compartment)
  ────────────────       ──────────────────────────────────────
  kit.Deploy             ─▶ architect.ts
                              ├── Agent "architect" + deploy_agent tool
                              └── bus.on("create", …)

  Call ts.architect.create("I need a haiku agent…")
                              │
                              └─▶ architect.generate(prompt)
                                   │  ┌────────────────────────┐
                                   │  │ LLM decides to call    │
                                   ▼  │   deploy_agent({name,   │
                              tool execute:                    │
                              bus.call("package.deploy", {     │
                                manifest, files               │
                              })                               │
                                   │                           │
                                   ▼                           │
                              new deployment: haiku-bot.ts     │
                              (Agent + bus.on("ask") handler)  │
                                                                
  Call ts.haiku-bot.ask("autumn leaves…")
                              │
                              └─▶ haiku-bot.generate(prompt)
                                   │
                                   ▼
                              text + usage → Go
```

The architect is a standard Mastra `Agent` with one tool. The tool
generates a tiny `.ts` source string, packages it, and deploys it
via `bus.call("package.deploy", …)` — the same command the CLI's
`brainkit deploy` uses. Once the new package is live, the spawned
agent registers itself (`kit.register("agent", …)`) and exposes a
stable bus topic (`ts.<name>.ask`) so external callers can reach
it without the architect in the loop.

## Run

```sh
OPENAI_API_KEY=sk-... go run ./examples/agent-spawner
```

Flags:

| Flag | Default | Effect |
|------|---------|--------|
| `-model`    | `gpt-4o-mini` | OpenAI model id used by both agents |
| `-request`  | haiku-bot request | instruction given to the architect |
| `-ask`      | `autumn leaves…` | prompt sent to the spawned agent |

Example:

```sh
OPENAI_API_KEY=sk-... go run ./examples/agent-spawner \
    -request "I need an agent that turns a JSON object into a concise bullet-list summary. Name it summarizer." \
    -ask '{"project":"brainkit","stars":1200,"langs":["Go","TS"]}'
```

## Expected output

```
[1/3] architect deployed
[2/3] architect request: "I need an agent that writes a single short haiku…"
        architect picked name="haiku-bot"
        architect wrote instructions="Write a single short haiku…"
        architect deployed haiku-bot on ts.haiku-bot.ask
[3/3] calling ts.haiku-bot.ask with prompt="autumn leaves drifting past a mountain stream"

--- spawned agent reply ---
Crimson whispers fall,
Dancing on the cool, clear flow—
Nature's soft farewell.
---
usage: prompt=42 completion=22 total=64
```

## How the architect deploys a new package

Inside the `deploy_agent` tool:

```ts
const spawnSource =
    `const a = new Agent({ name: ${JSON.stringify(name)}, model: model("openai", "gpt-4o-mini"), instructions: ${JSON.stringify(instructions)} });\n` +
    `kit.register("agent", ${JSON.stringify(name)}, a);\n` +
    `bus.on("ask", async (msg) => {\n` +
    `  const r = await a.generate(msg.payload.prompt);\n` +
    `  msg.reply({ text: r.text, usage: r.usage });\n` +
    `});\n`;

await bus.call("package.deploy", {
    manifest: { name, entry: `${name}.ts` },
    files:    { [`${name}.ts`]: spawnSource },
}, { timeoutMs: 30000 });
```

`package.deploy` is a first-class bus command — anything that
speaks the bus (Go, JS inside a deployment, CLI, gateway HTTP
API) can drive it.

## Walking the tool-result stream

Mastra + AI SDK 5 put tool calls and their results in
`step.content[]` entries, not on a convenient top-level array. The
example's `_findToolCall` / `_findToolResult` helpers walk the
steps looking for entries whose `type === "tool-result"` and
`toolName === "deploy_agent"`, and pull the payload out of
`output.value`.

Keep this pattern in mind if you build your own tool-calling
surface on top of brainkit — it's the shape the runtime gives you.

## Extend this

- **Pass tools to the spawned agent.** The architect could
  include a set of tools (fetch, secrets, calc, RAG lookup) in
  the generated source — just template another `createTool(…)`
  block before the `new Agent(…)` line.
- **Memory.** Add `kit.register("memory", …)` inside the spawn
  template and pass it into `new Agent({ memory })` to give the
  child its own Mastra memory store.
- **Multiple architects.** Register several
  architect-style agents, each specialized in a domain (UI
  helpers, data-transform agents, interview bots), and let a
  router agent pick which architect to delegate to.
- **Teardown.** Call `brainkit.CallPackageTeardown` with the
  spawned name when you're done so the agent + its bus
  subscriptions are cleaned up.

## Under the hood

| Primitive | Where |
|-----------|-------|
| `kit.Deploy` from Go | `brainkit.PackageInline(name, file, code)` + `kit.Deploy` |
| `package.deploy` from JS | `bus.call("package.deploy", {manifest, files}, {timeoutMs})` |
| Agent registration | `kit.register("agent", name, agent)` |
| Tool | `createTool({ id, description, inputSchema: z.object(…), execute })` |
| Public call surface | `bus.on("ask", …)` in the spawned package → callable as `ts.<name>.ask` |
| Typed Go caller | `brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{Topic, Payload})` |

The architect is just a brainkit deployment with access to
`bus.call` — nothing about it is privileged. A plugin or a
second Kit on the same transport could drive the same flow.
