# brainkit examples

Tiny, focused demos. Each example is self-contained — `go run` it
from the repo root.

| Example | What it shows |
|---|---|
| [ai-chat](./ai-chat/) | Register an AI provider, deploy a `.ts` that calls `generateText`, print the model's reply |
| [hello-embedded](./hello-embedded/) | Library mode: embed a Kit, deploy an inline `.ts`, call it, print the reply |
| [hello-server](./hello-server/) | Service mode: `brainkit.yaml` + `server.New` + `Start` |
| [multi-kit](./multi-kit/) | Two Kits in one process, routed by peer name through `modules/topology` |
| [gateway-routes](./gateway-routes/) | HTTP gateway on a bare Kit — `GET /hello` forwards to a deployed `.ts` handler |
| [go-tools](./go-tools/) | Register typed Go functions as first-class bus tools; invoke from `.ts` and from Go |
| [plugin-author](./plugin-author/) | Minimal subprocess plugin (own go.mod) — one tool + one subscription, built as a standalone binary |
| [plugin-host](./plugin-host/) | Live round-trip for plugin-author — builds the plugin, boots a Kit, calls its tool, prints the reply (with integration test) |
| [secrets](./secrets/) | Encrypted secret store lifecycle — Set / Get / Rotate / Delete via the `Kit.Secrets()` accessor |
| [storage-vectors](./storage-vectors/) | Persistent KV (Mastra Memory + SQLite) + vector store / similarity search from `.ts` |

More examples (harness-lite) land as the surrounding modules mature.
