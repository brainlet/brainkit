# brainkit

Embeddable runtime for AI agent teams. Combines an in-process JS/TS
compartment (QuickJS + SES) with a typed pub/sub bus (Watermill)
behind a single `Kit` type. Compose opt-in subsystems — gateway,
plugins, schedules, audit, tracing — through the `Module` interface.

No daemons to manage in library mode. No schema changes to ship a new
agent. One Kit per deployable unit; many Kits per cluster.

## Quick taste

**Library mode — embed a Kit:**

```go
import "github.com/brainlet/brainkit"

kit, _ := brainkit.New(brainkit.Config{
    Namespace: "myapp",
    Transport: brainkit.EmbeddedNATS(),
    Providers: []brainkit.ProviderConfig{
        brainkit.OpenAI(os.Getenv("OPENAI_API_KEY")),
    },
})
defer kit.Close()

resp, err := brainkit.Call[sdk.PackageDeployMsg, sdk.PackageDeployResp](
    kit, ctx, sdk.PackageDeployMsg{Path: "./agents/support"})
```

**Service mode — run brainkit as a server:**

```go
import "github.com/brainlet/brainkit/server"

srv, _ := server.QuickStart("my-app", "/var/brainkit",
    server.WithSecretKey(os.Getenv("SECRET_KEY")))
defer srv.Close()
_ = srv.Start(ctx)
```

Or scaffold a server binary through the CLI:

```sh
brainkit new server my-service
cd my-service && go mod tidy
go run . --config brainkit.yaml
```

## Is this for me?

| Use case | Shape |
|---|---|
| Add AI agents to a Go service | Library mode, one Kit |
| Long-running agent backend | Service mode, `server.New` or `brainkit start` |
| Multi-kit routing (analytics + ingest + …) | Kits on a shared bus + `modules/topology` |
| Write a plugin in Go | Subprocess plugin via `sdk/plugin`, loaded by `modules/plugins` |
| Embed in tests | `brainkit.Memory()` transport, no persistence |

## Modules

| Module | Maturity | What it adds |
|---|---|---|
| `modules/gateway` | stable | HTTP gateway (routes, SSE, WebSocket) |
| `modules/mcp` | stable | MCP client (external servers as tools) |
| `modules/plugins` | beta | Subprocess plugin supervisor + WS control plane |
| `modules/schedules` | beta | Persisted cron-style scheduling |
| `modules/audit` | beta | Audit log query surface + SQLite/Postgres stores |
| `modules/tracing` | beta | Distributed tracing with SQLite backing |
| `modules/probes` | beta | Provider health probes |
| `modules/discovery` | beta | Bus-mode peer discovery |
| `modules/topology` | beta | Cross-kit routing + `peers.*` bus commands |
| `modules/workflow` | beta | Declarative agent workflows |
| `modules/harness` | **wip** | Agent orchestration layer |

## Design docs

Architecture, migration plans, and vision live in
[`../brainkit-maps/brainkit/designs/`](../brainkit-maps/brainkit/designs/).

## License

Apache 2.0.
