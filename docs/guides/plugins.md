# Plugins

Plugins are separate Go processes that connect to a Kit over an external transport (NATS). They register tools, subscribe to events, and communicate through the same bus as .ts services.

## Building a Plugin

```go
// test/testplugin/main.go — real working plugin
package main

import (
    "context"
    "encoding/json"
    "log"

    "github.com/brainlet/brainkit/sdk"
)

func main() {
    p := sdk.New("acme", "my-plugin", "1.0.0",
        sdk.WithDescription("A test plugin"),
    )

    // Register a typed tool
    sdk.Tool[EchoInput, EchoOutput](p, "echo", "echoes the input", handleEcho)

    // Register another tool
    sdk.Tool[ConcatInput, ConcatOutput](p, "concat", "concatenates strings", handleConcat)

    // Register an event subscription
    sdk.On[DeployEvent](p, "kit.deployed", func(ctx context.Context, event DeployEvent, client sdk.Client) {
        log.Printf("deployment: %s", event.Source)
    })

    // OnStart — called after manifest is accepted
    p.OnStart(func(client sdk.Client) error {
        log.Println("plugin started, can use client to publish/subscribe")
        return nil
    })

    // Run blocks until SIGTERM
    if err := p.Run(); err != nil {
        log.Fatal(err)
    }
}

type EchoInput struct {
    Message string `json:"message"`
}
type EchoOutput struct {
    Echoed string `json:"echoed"`
}

func handleEcho(ctx context.Context, client sdk.Client, input EchoInput) (EchoOutput, error) {
    return EchoOutput{Echoed: input.Message}, nil
}

type ConcatInput struct {
    A string `json:"a"`
    B string `json:"b"`
}
type ConcatOutput struct {
    Result string `json:"result"`
}

func handleConcat(ctx context.Context, client sdk.Client, input ConcatInput) (ConcatOutput, error) {
    return ConcatOutput{Result: input.A + input.B}, nil
}
```

## Plugin Lifecycle

```
1. Plugin starts → connects to transport (NATS) via env vars
2. Creates Watermill router, registers tool/event handlers
3. Publishes PluginManifestMsg to "plugin.manifest"
4. Host Kit receives manifest → registers tools in shared registry
5. Host sends PluginManifestResp → plugin receives on replyTo
6. Plugin prints "READY:acme/my-plugin@1.0.0" to stdout
7. Host reads READY line → plugin is operational
8. Plugin runs until SIGTERM
```

Environment variables injected by the host:

| Var | Value |
|-----|-------|
| `BRAINKIT_TRANSPORT` | Transport type (e.g., `"nats"`) |
| `BRAINKIT_NATS_URL` | NATS URL |
| `BRAINKIT_NATS_NAME` | NATS durable prefix |
| `BRAINKIT_NAMESPACE` | Kit namespace |
| `BRAINKIT_NODE_ID` | Host node ID |
| `BRAINKIT_PLUGIN_CONFIG` | Plugin-specific config JSON |

## Host Configuration

```go
kit, err := brainkit.New(brainkit.Config{
    Namespace: "my-app",
    FSRoot:    "/tmp/my-app",
    Transport: "nats",
    NATSURL:   "nats://localhost:4222",
    Plugins: []brainkit.PluginConfig{
        {
            Name:        "my-plugin",
            Binary:      "./my-plugin",          // path to compiled binary
            AutoRestart: true,                    // restart on crash
            MaxRestarts: 5,                       // give up after 5 restarts
            StartTimeout: 10 * time.Second,       // time to wait for READY
            ShutdownTimeout: 5 * time.Second,     // time to wait for graceful stop
        },
    },
})
```

Plugins REQUIRE an external transport. Memory transport returns `ValidationError{Field: "transport", Message: "plugins require nats transport"}`.

## Auto-Restart

When a plugin crashes, the host detects it via `cmd.Wait()` and auto-restarts with exponential backoff:

```
Crash #1 → wait 1s → restart
Crash #2 → wait 2s → restart
Crash #3 → wait 4s → restart
Crash #4 → wait 8s → restart
Crash #5 → wait 16s → restart
Crash #6 → give up (MaxRestarts reached)
```

Backoff is capped at 30 seconds. Each crash logs the exit code and signal:

```
[plugin:my-plugin] crashed: signal killed (exit code -1)
[plugin:my-plugin] restarting in 2s (2/5)
[plugin:my-plugin] restarted (pid=12345, restart #2)
```

During intentional shutdown (`Kit.Close`), the `stopping` flag prevents auto-restart. The host sends SIGTERM, waits `ShutdownTimeout`, then SIGKILL if needed.

## Plugin State

Plugins can persist key-value state through the bus:

```go
p.OnStart(func(client sdk.Client) error {
    // Get state
    pr, _ := sdk.Publish(client, ctx, sdk.PluginStateGetMsg{Key: "counter"})
    sdk.SubscribeTo[sdk.PluginStateGetResp](client, ctx, pr.ReplyTo,
        func(resp sdk.PluginStateGetResp, msg sdk.Message) {
            fmt.Println("counter:", resp.Value)
        })

    // Set state
    sdk.Publish(client, ctx, sdk.PluginStateSetMsg{Key: "counter", Value: "42"})
    return nil
})
```

State is stored per-plugin (keyed by plugin identity). Storage backend:
- Memory transport → in-memory map (lost on restart)
- NATS transport → NATS KV bucket (persistent across restarts)

## Plugin Tools

When the host receives a plugin's manifest, it registers each tool in the shared tool registry with an executor that routes calls through the bus:

```
Go/TS calls "echo" tool
  → ToolsDomain.Call resolves to plugin tool
  → Publishes to "plugin.tool.acme/my-plugin@1.0.0/echo"
  → Plugin's router handles it
  → Plugin calls handleEcho
  → Plugin publishes result to "plugin.tool.acme/my-plugin@1.0.0/echo.result"
  → Go/TS receives result
```

Plugin tools appear in `tools.list` alongside Go-registered and .ts-registered tools. The naming convention is `owner/package@version/tool` — e.g., `acme/my-plugin@1.0.0/echo`. Resolution supports short names: `tools.call("echo")` finds it if there's no ambiguity.

## Stderr Capture

Plugin stderr is captured and routed to `log.Printf` with a prefix:

```
[plugin:my-plugin] some error output from the plugin
```

## Interceptors

Plugins can register interceptors that run before message dispatch:

```go
sdk.Intercept(p, "audit-log", 10, "*", func(ctx context.Context, msg sdk.InterceptMessage) (*sdk.InterceptMessage, error) {
    log.Printf("[audit] %s from %s: %s", msg.Topic, msg.CallerID, string(msg.Payload))
    return &msg, nil // pass through
})
```

Interceptors can modify `Payload` and `Metadata` but not `Topic`, `CallerID`, or `Address`. Return an error to reject the message.

## Plugin Limitations

- Plugins can NOT do cross-Kit operations (no `CrossNamespaceRuntime`)
- Plugins talk to their host Kit only — they don't have direct access to other Kits
- Plugin state is scoped to the plugin identity — no cross-plugin state sharing
- Health checks after startup are not yet implemented (see codebase assessment)
