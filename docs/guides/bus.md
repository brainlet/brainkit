# Bus

The bus is the async-first message router at the core of brainkit. All communication between Kit components (agents, tools, workflows, WASM shards, plugins, cross-Kit networking) flows through the bus.

---

## Three Primitives

### Send (fire-and-forget)

Broadcast a message. No response expected. Subscribers receive it asynchronously.

```go
kit.Bus.Send(bus.Message{
    Topic:    "order.created",
    CallerID: "order-service",
    Payload:  json.RawMessage(`{"orderId":"123","total":99.99}`),
})
```

### Ask (request/response)

Send a message and register a callback for the reply. The callback is guaranteed to fire exactly once (either a reply or a timeout). Returns a cancel function.

```go
cancel := kit.Bus.Ask(bus.Message{
    Topic:    "tools.call",
    CallerID: "user",
    Payload:  json.RawMessage(`{"name":"search","input":{"q":"brainkit"}}`),
}, func(reply bus.Message) {
    fmt.Println("result:", string(reply.Payload))
})
// cancel() aborts the pending ask
```

Default timeout: 30 seconds. Configurable via `bus.WithHandlerTimeout(d)`.

### On (subscribe)

Subscribe to messages matching a topic pattern. The handler receives the message and a `ReplyFunc`. If the sender used `Ask`, calling `reply(payload)` sends the response back. If it was a `Send`, reply is a no-op.

```go
subID := kit.Bus.On("order.*", func(msg bus.Message, reply bus.ReplyFunc) {
    fmt.Println("got:", msg.Topic, string(msg.Payload))
    result, _ := json.Marshal(map[string]bool{"ok": true})
    reply(result) // sends response if this was an Ask
})
// kit.Bus.Off(subID) to unsubscribe
```

Topic patterns: `"order.created"` matches exactly. `"order.*"` matches `"order.created"`, `"order.shipped"`, etc.

---

## AskSync (convenience wrapper)

`AskSync` wraps `Ask` into a blocking call with context cancellation. It is NOT a bus primitive -- it is a convenience for Go code that needs synchronous request/response (bridges, handlers, Go API methods).

```go
resp, err := bus.AskSync(kit.Bus, ctx, bus.Message{
    Topic:    "memory.createThread",
    CallerID: "user",
    Payload:  json.RawMessage(`{"opts":{"title":"chat"}}`),
})
if err != nil {
    // context cancelled or other error
}
fmt.Println(string(resp.Payload))
```

---

## Worker Groups (AsWorker)

By default, every subscriber receives every matching message (broadcast). Worker groups enable competing consumers -- only ONE subscriber in the group gets each message.

```go
// Three Kit instances in the "processors" group:
kit.Bus.On("order.process", handler, bus.AsWorker("processors"))
```

Messages to `order.process` are load-balanced across group members. This is how the scaling system (InstanceManager) distributes work across pooled Kit instances.

When `WorkerGroup` is set in Kit config, all Kit bus handlers automatically use `AsWorker`:

```go
kit, _ := brainkit.New(brainkit.Config{
    WorkerGroup: "order-pool",  // all handlers compete within this group
})
```

---

## Address Routing

Messages can target specific entities using the `Address` field. Non-local addresses are routed via the transport's `Forward` method for cross-Kit delivery.

```go
// Send to a specific Kit
kit.Bus.Send(bus.Message{
    Topic:   "agents.request",
    Address: "kit:analytics/agent:summarizer",
    Payload: json.RawMessage(`{"name":"summarizer","prompt":"hello"}`),
})
```

Address formats:
- `""` -- local (default)
- `"kit:name"` -- route to a named Kit
- `"host:X/kit:Y/agent:Z"` -- full path

Subscribers can filter by address:

```go
kit.Bus.On("agents.*", handler, bus.WithAddress("agent:summarizer"))
```

---

## Interceptor Pipeline

Interceptors process messages before dispatch. They can modify `Payload` and `Metadata`, but `Topic`, `CallerID`, and `Address` are immutable. Return an error to reject the message.

```go
type LogInterceptor struct{}

func (LogInterceptor) Name() string        { return "logger" }
func (LogInterceptor) Priority() int       { return 10 }
func (LogInterceptor) Match(topic string) bool { return true }
func (LogInterceptor) Process(msg *bus.Message) error {
    log.Printf("[bus] %s: %s", msg.Topic, string(msg.Payload))
    return nil
}

kit.Bus.AddInterceptor(LogInterceptor{})
```

Interceptors are sorted by priority (lowest first). They run on both `Send` and `Ask`.

---

## Message Envelope

Every message carries:

| Field | Type | Description |
|-------|------|-------------|
| `Version` | `string` | Protocol version (`"v1"`) |
| `Topic` | `string` | Routing topic |
| `Address` | `string` | Target address (empty = local) |
| `ReplyTo` | `string` | Reply topic (set by Ask) |
| `CallerID` | `string` | Sender identity |
| `ID` | `string` | Unique message ID (auto-generated) |
| `ParentID` | `string` | ID of the message this replies to |
| `TraceID` | `string` | Trace ID for cascading operations |
| `Depth` | `int` | Cascade depth (cycle detection at 16) |
| `Payload` | `json.RawMessage` | Message content |
| `Metadata` | `map[string]string` | Optional key-value metadata |

---

## All Bus Topic Domains

These are all the topic domains registered by Kit's `registerHandlers`:

| Domain | Topics | Handler |
|--------|--------|---------|
| `wasm.*` | compile, run, deploy, undeploy, describe, remove, list, list-deployed | WASM service |
| `tools.*` | call, resolve, register, list | Tool registry |
| `mcp.*` | listTools, callTool | MCP manager |
| `agents.*` | register, unregister, list, discover, get-status, set-status, request, message | Agent registry |
| `fs.*` | read, write, list, mkdir, delete, stat | Filesystem (sandboxed) |
| `ai.*` | generate, embed, embedMany, generateObject | AI model calls |
| `memory.*` | createThread, getThread, listThreads, save, recall, deleteThread | Memory (Mastra) |
| `workflows.*` | run, resume, cancel, status | Workflows (Mastra) |
| `vectors.*` | createIndex, listIndexes, upsert, query, deleteIndex | Vector store |
| `kit.*` | deploy, teardown, list, redeploy | Deploy/teardown |
| `plugin.state.*` | get, set | Plugin state persistence |

---

## Transport

The bus delegates message delivery to a `Transport` implementation:

| Transport | Use case |
|-----------|----------|
| `InProcessTransport` | Default. Local in-memory delivery via goroutines and channels. |
| `GRPCTransport` | Kit-to-Kit networking. Routes addressed messages to remote peers. |
| `NATSTransport` | NATS-based transport for distributed deployments. |

Configured via `Config.Transport`:

```go
// Default (in-process, or GRPC if Network is configured)
kit, _ := brainkit.New(brainkit.Config{})

// NATS
kit, _ := brainkit.New(brainkit.Config{
    Transport: "nats",
    NATS: brainkit.NATSConfig{URL: "nats://localhost:4222"},
})
```

---

## Jobs and Metrics

The bus tracks cascading operations as "jobs" via trace IDs.

```go
// List all tracked jobs
jobs := kit.Bus.Jobs()

// Get a specific job
job := kit.Bus.Job(traceID)

// Bus metrics snapshot
metrics := kit.Bus.Metrics()
fmt.Printf("subscribers: %d, active jobs: %d\n", metrics.Subscribers, metrics.ActiveJobs)
```

---

## Lifecycle

```go
// Create
b := bus.NewBus(bus.NewInProcessTransport(),
    bus.WithHandlerTimeout(10 * time.Second),
    bus.WithJobTimeout(2 * time.Minute),
)

// Use
b.Send(msg)
b.Ask(msg, callback)
subID := b.On("topic", handler)
b.Off(subID)
b.AddInterceptor(myInterceptor)

// Shutdown
b.Close()
```
