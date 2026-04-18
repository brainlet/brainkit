# brainkit examples

Tiny, focused demos. Each example is self-contained — `go run` it
from the repo root.

| Example | What it shows |
|---|---|
| [agent-forge](./agent-forge/) | Flagship meta-programming example — multi-agent pipeline (architect → coder → 3 parallel reviewers in a dountil loop → deploy) that designs, writes, reviews, and deploys a brand-new agent at runtime |
| [agent-spawner](./agent-spawner/) | Minimal meta-programming example — an agent with a `deploy_agent` tool that templates a .ts package and deploys it, for learning the primitive before agent-forge |
| [agent-stream](./agent-stream/) | `agent.stream()` from inside a deployment — plain text streaming + `structuredOutput` → typed partials, both piped over bus `CallStream` + SSE gateway |
| [ai-chat](./ai-chat/) | Register an AI provider, deploy a `.ts` that calls `generateText`, print the model's reply |
| [cross-kit](./cross-kit/) | Two Kits on a shared in-process NATS, routed by peer name through `modules/topology` + `WithCallTo` |
| [custom-scorer](./custom-scorer/) | Domain-specific `createScorer` — regex scorer + LLM-judge scorer side by side on the same dataset; highlights regex-vs-LLM tradeoffs |
| [evals](./evals/) | Batch `runEvals` + prebuilt scorers (answer-relevancy + completeness) + baseline/tolerance regression gate — CI quality gate pattern |
| [hello-embedded](./hello-embedded/) | Library mode: embed a Kit, deploy an inline `.ts`, call it, print the reply |
| [hello-server](./hello-server/) | Service mode: `brainkit.yaml` + `server.New` + `Start` |
| [multi-kit](./multi-kit/) | Two Kits in one process, routed by peer name through `modules/topology` |
| [observability](./observability/) | `audit.query` + `audit.stats` + `trace.list` round-trip via `modules/audit` + `modules/tracing` |
| [package-workflow](./package-workflow/) | The on-disk package lifecycle: `ScaffoldPackage` → edit → add a sibling file → `PackageFromDir` deploy → teardown. The shape `brainkit new package` produces, unpacked into Go. |
| [gateway-routes](./gateway-routes/) | HTTP gateway on a bare Kit — `GET /hello` forwards to a deployed `.ts` handler |
| [go-tools](./go-tools/) | Register typed Go functions as first-class bus tools; invoke from `.ts` and from Go |
| [guardrails](./guardrails/) | Input processors on an Agent — `PromptInjectionDetector` rewrites hostile input, `PIIDetector` masks PII |
| [harness-lite](./harness-lite/) | WIP — frozen `modules/harness` surface: `NewModule`, `Instance`, and the six frozen event types |
| [hitl-tool-approval](./hitl-tool-approval/) | Synchronous HITL: a tool marked `requireApproval:true` pauses the agent; Go approves/declines via a bus topic; uses `generateWithApproval` |
| [hitl-workflow](./hitl-workflow/) | Workflow-based HITL: a step calls `suspend(reason)`, Go resumes with `CallWorkflowResume` — durable across process restart when storage is configured |
| [mcp](./mcp/) | Wire an external Model Context Protocol server (npx filesystem server) as first-class tools |
| [plugin-author](./plugin-author/) | Minimal subprocess plugin (own go.mod) — one tool + one subscription, built as a standalone binary |
| [plugin-host](./plugin-host/) | Live round-trip for plugin-author — builds the plugin, boots a Kit, calls its tool, prints the reply (with integration test) |
| [rag-pipeline](./rag-pipeline/) | Full Mastra RAG flow — `MDocument.chunk` + embeddings + pgvector + `createVectorQueryTool` on an Agent, with positive / negative questions + optional `rerankWithScorer` path |
| [schedules](./schedules/) | Cron-style scheduled bus messages — `modules/schedules`, create / cancel via generated wrappers |
| [secrets](./secrets/) | Encrypted secret store lifecycle — Set / Get / Rotate / Delete via the `Kit.Secrets()` accessor |
| [storage-vectors](./storage-vectors/) | Persistent KV (Mastra Memory + SQLite) + vector store / similarity search from `.ts` |
| [streaming](./streaming/) | Every streaming surface: bus `CallStream`, gateway SSE, WebSocket, Webhook |
| [voice-agent](./voice-agent/) | Full speak → listen → generate → speak round trip via `OpenAIVoice` — TTS to MP3, STT back to text, generate answer, TTS to a second MP3 |
| [voice-broadcast](./voice-broadcast/) | Single TTS fans through `audio.Composite` to three sinks at once — desktop speakers + MP3 file + bus topic — with a subscriber watching; shows the Sink primitive |
| [voice-chat](./voice-chat/) | Minimum canonical "agent speaks answers" — stdin question, `agent.generate` + `voice.speak`, played via `new Audio(stream).play()`. The baseline you add voice to an existing agent with |
| [voice-realtime](./voice-realtime/) | Live bidirectional voice in a browser — mic PCM16 streams up over WS, `OpenAIRealtimeVoice` replies as the model speaks, plays back in the page + desktop speakers |
| [working-memory](./working-memory/) | Multi-turn agent with `Memory` — remembers names across turns on the same thread; different threads are isolated |
| [workflows](./workflows/) | Declarative 3-step pipeline through `modules/workflow` (`createStep` + `createWorkflow`) |
| [workspace-agent](./workspace-agent/) | Coding agent — reads/writes real files + runs shell commands through brainkit's `fs` + `exec` polyfills, sandboxed under `FSRoot` |
