# Plugins

Plugins are separate Go processes that extend a Kit's capabilities. They connect to the host Kit via WebSocket, register tools, and handle calls through a JSON protocol. No Watermill or transport dependency — plugins only need the SDK module.

## Building a Plugin

```go
package main

import (
    "context"
    "log"

    bkplugin "github.com/brainlet/brainkit/sdk/plugin"
)

func main() {
    p := bkplugin.New("acme", "my-plugin", "1.0.0",
        bkplugin.WithDescription("A sample plugin"),
    )

    bkplugin.Tool(p, "echo", "echoes the input", handleEcho)
    bkplugin.Tool(p, "concat", "concatenates strings", handleConcat)

    p.OnStart(func(client bkplugin.Client) error {
        log.Println("plugin started")
        return nil
    })

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

func handleEcho(ctx context.Context, client bkplugin.Client, input EchoInput) (EchoOutput, error) {
    return EchoOutput{Echoed: input.Message}, nil
}

type ConcatInput struct {
    A string `json:"a"`
    B string `json:"b"`
}
type ConcatOutput struct {
    Result string `json:"result"`
}

func handleConcat(ctx context.Context, client bkplugin.Client, input ConcatInput) (ConcatOutput, error) {
    return ConcatOutput{Result: input.A + input.B}, nil
}
```

## Plugin Lifecycle

```
1. Host starts plugin subprocess, passes BRAINKIT_PLUGIN_WS_URL env var
2. Plugin connects to host via WebSocket
3. Plugin sends manifest (tool definitions) over WS
4. Host registers tools, sends manifest.ack
5. Plugin prints "READY:acme/my-plugin@1.0.0" to stdout
6. Host reads READY → plugin is operational
7. Host sends tool.call messages over WS
8. Plugin executes tools, sends tool.result back
9. Plugin runs until SIGTERM
```

Environment variables injected by the host:

| Var | Value |
|-----|-------|
| `BRAINKIT_PLUGIN_WS_URL` | WebSocket URL to connect to host |
| `BRAINKIT_NAMESPACE` | Kit namespace |
| `BRAINKIT_NODE_ID` | Host node ID |
| `BRAINKIT_PLUGIN_CONFIG` | Plugin-specific config JSON |

## Host Configuration

```go
kit, err := brainkit.New(brainkit.Config{
    Namespace: "my-app",
    Transport: "nats",
    NATSURL:   "nats://localhost:4222",
    Plugins: []brainkit.PluginConfig{{
        Name:        "my-plugin",
        Binary:      "./plugins/my-plugin",
        AutoRestart: true,
        Env: map[string]string{
            "DB_PATH": "/data/plugin.db",
        },
    }},
})
```

Plugins require a transport backend (not memory) because the host needs a running Node for the WS server.

## Plugin Dependencies

Plugins depend only on the SDK module:

```
go get github.com/brainlet/brainkit/sdk
```

This gives you: `sdk/plugin` (framework), `sdk` (message types, Runtime interface), `sdk/pluginws` (protocol types). Dependencies: `coder/websocket` + `google/uuid`. No quickjs, no watermill, no esbuild.

## Calling Plugin Tools from .ts

Deployed .ts code calls plugin tools via `tools.call()`:

```typescript
bus.on("process", async (msg) => {
    const result = await tools.call("set", { key: "hello", value: "world" });
    msg.reply(result);
});
```

The tool name matches the short name registered by the plugin. The bus routes the call through the host's WS connection to the plugin.

## Available Plugins

| Plugin | Tools | Description |
|--------|-------|-------------|
| `brainkit-plugin-kv` | set, get, delete, list | SQLite key-value store |
| `brainkit-plugin-hackernews` | top, new, best, item, search, user | Hacker News reader |
| `brainkit-plugin-wikipedia` | search, summary, article, random, links | Wikipedia reader |
| `brainkit-plugin-cron` | create, list, remove, pause, resume | Job scheduling |
