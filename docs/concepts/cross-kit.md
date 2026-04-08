# Cross-Kit Communication

Multiple Kits can run on the same transport and communicate across namespace boundaries. This enables isolation patterns (dev/prod, multi-tenant) while allowing controlled inter-Kit messaging.

## Why Cross-Kit

A single Kit is a self-contained runtime — its own QuickJS heap, its own tool registry, its own deployed .ts services. But real deployments need multiple Kits:

- **Dev/prod isolation**: A dev Kit and a prod Kit share the same NATS cluster but shouldn't see each other's messages by default
- **Multi-tenant**: Each tenant gets a Kit with its own namespace, tools, and agents
- **Microservice decomposition**: One Kit per domain (orders, payments, notifications) communicating over the bus

Cross-Kit communication lets Kit A send a message to Kit B's namespace. Kit B's handlers process it as if it were a local message. The response comes back through the replyTo mechanism.

## The CrossNamespaceRuntime Interface

```go
// sdk/runtime.go
type CrossNamespaceRuntime interface {
    Runtime
    PublishRawTo(ctx context.Context, targetNamespace, topic string, payload json.RawMessage) (correlationID string, err error)
    SubscribeRawTo(ctx context.Context, targetNamespace, topic string, handler func(sdk.Message)) (cancel func(), err error)
}
```

Both standalone and transport-connected Kit implement this. Plugin clients do NOT — plugins talk to their host Kit only.

## How Namespace Routing Works

Every Kit has a namespace (from `Config.Namespace`, default `"user"`). When a Kit publishes to topic `tools.call`, the RemoteClient prefixes it with the namespace:

```
Logical: tools.call
Namespace: kit-a
Wire topic: kit-a.tools.call (then sanitized per transport)
```

`PublishRawTo` uses the TARGET namespace instead:

```go
// internal/messaging/client.go
func (c *RemoteClient) PublishRawToNamespace(ctx context.Context, targetNamespace, logicalTopic string, payload json.RawMessage) (string, error) {
    // resolves to: targetNamespace.logicalTopic (sanitized)
    // replyTo resolves to: callerNamespace.replyTopic (sanitized)
}
```

The replyTo topic is always resolved with the CALLER's namespace — so responses come back to the sender, not the target.

## Using PublishTo from Go

The SDK provides a typed generic function:

```go
// sdk/cross.go
func PublishTo[T sdk.BrainkitMessage](rt Runtime, ctx context.Context, targetNamespace string, msg T, opts ...PublishOption) (PublishResult, error)
```

Example — Kit A sends a tool call to Kit B:

```go
// Kit A (namespace: "kit-a")
pr, err := sdk.PublishTo(kitA, ctx, "kit-b", sdk.ToolCallMsg{
    Name:  "echo",
    Input: map[string]string{"message": "hello from kit-a"},
})

// Subscribe to replyTo — comes back on kit-a's namespace
unsub, err := sdk.SubscribeTo[sdk.ToolCallResp](kitA, ctx, pr.ReplyTo,
    func(resp sdk.ToolCallResp, msg sdk.Message) {
        fmt.Println("response:", string(resp.Result))
    })
defer unsub()
```

## Reaching .ts Services Across Kits

To send a message to a .ts service deployed on another Kit, use `PublishTo` with a `CustomMsg` targeting the service's mailbox topic:

```go
// Kit A sends to Kit B's "greeter.ts" service
pr, err := sdk.PublishTo(kitA, ctx, "kit-b", sdk.CustomMsg{
    Topic:   "ts.greeter.greet",
    Payload: json.RawMessage(`{"name":"world"}`),
})
```

The wire topic becomes `kit-b.ts.greeter.greet` (sanitized per transport). Kit B's handler — registered via `bus.on("greet")` in greeter.ts — receives it.

## Discovery

Discovery resolves Kit names to addresses. Two providers:

### Static

Fixed configuration — you know your peers at startup:

```go
discovery.NewStaticFromConfig([]discovery.PeerConfig{
    {Name: "kit-b", Namespace: "kit-b", Address: "10.0.1.2:9090"},
})
```

`Resolve("kit-b")` returns the namespace for routing. `Browse()` returns all known peers.

### Multicast (LAN)

Zero-config UDP multicast for development. Kits announce themselves on `224.0.0.251:5353` with a custom protocol (NOT actual mDNS):

```
BRAINKIT|_brainkit._tcp|kit-a|10.0.1.1:9001
```

Each Kit listens for announcements and builds a peer map. Re-announces on read timeout (1 second) to handle late joiners.

```go
d, err := discovery.NewMulticast("_brainkit._tcp")
d.Register(discovery.Peer{Name: "kit-a", Address: "10.0.1.1:9001"})
// Other Kits on the LAN discover kit-a automatically
```

## Cross-Kit Test Coverage

14 cross-Kit tests in `test/cross/` run on NATS:

| Test | What it covers |
|------|---------------|
| ts→Go | .ts deploys tool, Go on other Kit calls it |
| Go→ts | Go registers tool, .ts on other Kit calls it via tools.call |
| ts→WASM | .ts deploys tool, WASM shard on other Kit calls it via _busPublish |
| WASM→Go | WASM calls Go-registered tool across Kits |
| Plugin→Go | Plugin registers tool, Go on other Kit calls it |
| WASM→Plugin | WASM calls plugin tool across Kits |
| Chain (A→B→C) | Three-Kit chain — message flows through all three |

All tests use `testutil.NewTestKernelPairFull` which creates two Kits on the same NATS transport with different namespaces.
