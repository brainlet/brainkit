# brainkit examples

Tiny, focused demos. Each example is self-contained — `go run` it
from the repo root.

| Example | What it shows |
|---|---|
| [agent-forge](./agent-forge/) | Flagship meta-programming example — multi-agent pipeline (architect → coder → 3 parallel reviewers in a dountil loop → deploy) that designs, writes, reviews, and deploys a brand-new agent at runtime |
| [agent-spawner](./agent-spawner/) | Minimal meta-programming example — an agent with a `deploy_agent` tool that templates a .ts package and deploys it, for learning the primitive before agent-forge |
| [ai-chat](./ai-chat/) | Register an AI provider, deploy a `.ts` that calls `generateText`, print the model's reply |
| [cross-kit](./cross-kit/) | Two Kits on a shared in-process NATS, routed by peer name through `modules/topology` + `WithCallTo` |
| [hello-embedded](./hello-embedded/) | Library mode: embed a Kit, deploy an inline `.ts`, call it, print the reply |
| [hello-server](./hello-server/) | Service mode: `brainkit.yaml` + `server.New` + `Start` |
| [multi-kit](./multi-kit/) | Two Kits in one process, routed by peer name through `modules/topology` |
| [observability](./observability/) | `audit.query` + `audit.stats` + `trace.list` round-trip via `modules/audit` + `modules/tracing` |
| [gateway-routes](./gateway-routes/) | HTTP gateway on a bare Kit — `GET /hello` forwards to a deployed `.ts` handler |
| [go-tools](./go-tools/) | Register typed Go functions as first-class bus tools; invoke from `.ts` and from Go |
| [harness-lite](./harness-lite/) | WIP — frozen `modules/harness` surface: `NewModule`, `Instance`, and the six frozen event types |
| [mcp](./mcp/) | Wire an external Model Context Protocol server (npx filesystem server) as first-class tools |
| [plugin-author](./plugin-author/) | Minimal subprocess plugin (own go.mod) — one tool + one subscription, built as a standalone binary |
| [plugin-host](./plugin-host/) | Live round-trip for plugin-author — builds the plugin, boots a Kit, calls its tool, prints the reply (with integration test) |
| [schedules](./schedules/) | Cron-style scheduled bus messages — `modules/schedules`, create / cancel via generated wrappers |
| [secrets](./secrets/) | Encrypted secret store lifecycle — Set / Get / Rotate / Delete via the `Kit.Secrets()` accessor |
| [storage-vectors](./storage-vectors/) | Persistent KV (Mastra Memory + SQLite) + vector store / similarity search from `.ts` |
| [streaming](./streaming/) | Every streaming surface: bus `CallStream`, gateway SSE, WebSocket, Webhook |
| [workflows](./workflows/) | Declarative 3-step pipeline through `modules/workflow` (`createStep` + `createWorkflow`) |
