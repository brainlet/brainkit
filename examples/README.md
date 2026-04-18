# brainkit examples

Tiny, focused demos. Each example is self-contained — `go run` it
from the repo root.

| Example | What it shows |
|---|---|
| [hello-embedded](./hello-embedded/) | Library mode: embed a Kit, deploy an inline `.ts`, call it, print the reply |
| [multi-kit](./multi-kit/) | Two Kits in one process, routed by peer name through `modules/topology` |
| [gateway-routes](./gateway-routes/) | HTTP gateway on a bare Kit — `GET /hello` forwards to a deployed `.ts` handler |

More examples (hello-server, plugin-author, harness-lite) land as the
surrounding modules mature.
