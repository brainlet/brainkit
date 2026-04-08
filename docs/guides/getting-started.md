# Getting Started

This guide walks through creating a Kernel, registering Go tools, deploying a .ts service, and calling it from Go. Every code example is real — taken from test helpers and test files.

## Minimal Kernel

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/brainlet/brainkit"
)

func main() {
    kit, err := brainkit.New(brainkit.Config{
        Namespace: "my-app",
        FSRoot:    "/tmp/my-app",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer kit.Close()

    // Kit is running — in-memory transport, AI providers auto-detected from env, no storage
    fmt.Println("Kit ready")
}
```

This creates a Kit with an in-memory message bus. No containers, no external services, no API keys needed.

## Full Kit with AI + Storage

```go
// AI providers are auto-detected from os.Getenv (e.g. OPENAI_API_KEY)
kit, err := brainkit.New(brainkit.Config{
    Namespace: "my-app",
    CallerID:  "my-app",
    FSRoot:    "/tmp/my-app",

    // SQLite storage — enables new LibSQLStore({ id: "x" }) in .ts code
    Storages: map[string]brainkit.StorageConfig{
        "default": brainkit.SQLiteStorage("/tmp/my-app/data.db"),
    },
})
```

## Registering Go Tools

Tools registered in Go are callable from every surface — .ts code, plugins, and other Go code.

```go
import (
    "github.com/brainlet/brainkit/internal/registry"
    "github.com/brainlet/brainkit"
)

// Define typed input struct
type AddInput struct {
    A int `json:"a"`
    B int `json:"b"`
}

// Register with typed generic helper
err := brainkit.RegisterTool(k,"add", registry.TypedTool[AddInput]{
    Description: "adds two numbers",
    Execute: func(ctx context.Context, input AddInput) (any, error) {
        return map[string]int{"sum": input.A + input.B}, nil
    },
})
```

The tool is now available:
- From Go: `sdk.Publish(rt, ctx, messages.ToolCallMsg{Name: "add", Input: map[string]any{"a": 10, "b": 32}})`
- From .ts: `await tools.call("add", { a: 10, b: 32 })`

## Deploying .ts Code

Deploy TypeScript code into a SES Compartment:

```go
// Pattern from test/e2e/scenarios_test.go
ctx := context.Background()

pr, err := sdk.Publish(rt, ctx, messages.KitDeployMsg{
    Source: "greeter.ts",
    Code: `
        const greetTool = createTool({
            id: "greet",
            description: "greets a person",
            execute: async ({ context: input }) => {
                return { greeting: "Hello, " + (input.name || "world") + "!" };
            },
        });
        kit.register("tool", "greet", greetTool);
    `,
})

// Wait for deploy to complete
deployCh := make(chan messages.KitDeployResp, 1)
unsub, _ := sdk.SubscribeTo[messages.KitDeployResp](rt, ctx, pr.ReplyTo,
    func(resp messages.KitDeployResp, msg messages.Message) { deployCh <- resp })
defer unsub()

resp := <-deployCh
fmt.Println("deployed:", resp.Deployed, "resources:", len(resp.Resources))
```

After deployment, the "greet" tool is callable from any surface — same as a Go-registered tool.

## Calling a .ts Service from Go

Deploy a .ts service with `bus.on`, then send it messages from Go:

```go
// Deploy — pattern from test/bus/api_test.go
sdk.Publish(rt, ctx, messages.KitDeployMsg{
    Source: "calc.ts",
    Code: `
        bus.on("add", (msg) => {
            const a = msg.payload.a || 0;
            const b = msg.payload.b || 0;
            msg.reply({ result: a + b });
        });
    `,
})
// ... wait for deploy ...

// Send to the service's mailbox — pattern from test/bus/sdk_reply_test.go
pr, err := sdk.SendToService(rt, ctx, "calc.ts", "add", map[string]int{"a": 17, "b": 25})

// Get the reply
replyCh := make(chan json.RawMessage, 1)
unsub, _ := sdk.SubscribeTo[json.RawMessage](rt, ctx, pr.ReplyTo,
    func(payload json.RawMessage, msg messages.Message) { replyCh <- payload })
defer unsub()

reply := <-replyCh
// reply: {"result":42}
```

`sdk.SendToService` resolves the naming convention: `"calc.ts"` + `"add"` → topic `"ts.calc.add"`. The .ts code's `bus.on("add")` is subscribed to that exact topic.

## Streaming from a .ts Service

```go
// Deploy — pattern from test/bus/api_test.go
sdk.Publish(rt, ctx, messages.KitDeployMsg{
    Source: "streamer.ts",
    Code: `
        bus.on("stream", (msg) => {
            msg.send({ chunk: "one" });
            msg.send({ chunk: "two" });
            msg.reply({ done: true });
        });
    `,
})

// Send and collect chunks
pr, _ := sdk.SendToService(rt, ctx, "streamer.ts", "stream", map[string]any{})

unsub, _ := rt.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
    if msg.Metadata["done"] == "true" {
        fmt.Println("final:", string(msg.Payload))
    } else {
        fmt.Println("chunk:", string(msg.Payload))
    }
})
defer unsub()
```

`msg.send(data)` sends intermediate chunks with `done=false`. `msg.reply(data)` sends the final response with `done=true`. The Go side distinguishes them via the `done` metadata flag.

## Teardown

```go
sdk.Publish(rt, ctx, messages.KitTeardownMsg{Source: "calc.ts"})
```

Teardown removes all resources created by the deployment — tools are deregistered, bus subscriptions cancelled, agent registrations removed, Compartment reference dropped.

## Using an External Transport

For multi-Kit communication or plugin support, set a transport backend:

```go
kit, err := brainkit.New(brainkit.Config{
    Namespace: "my-app",
    FSRoot:    "/tmp/my-app",
    Transport: "nats",
    NATSURL:   "nats://localhost:4222",
})
if err != nil {
    log.Fatal(err)
}
defer n.Close()

// Use n exactly like a Kernel — same sdk.Runtime interface
sdk.Publish(n, ctx, messages.ToolCallMsg{Name: "echo", Input: "hello"})
```

The Node's Kernel uses NATS instead of GoChannel. All bus messages flow through NATS, enabling cross-Kit communication and plugin subprocesses.

## Next Steps

- [TypeScript Services](ts-services.md) — bus.on patterns, msg.reply, streaming, HITL
- [Go SDK](go-sdk.md) — Publish, SubscribeTo, Reply, SendToService, error types
- [Plugins](plugins.md) — building out-of-process plugins
- [Storage and Memory](storage-and-memory.md) — storage backends, agent memory, vectors
