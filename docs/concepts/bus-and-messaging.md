# Bus and Messaging

All communication in brainkit flows through a message bus built on Watermill. Go code, deployed .ts services, WASM shards, and plugins all use the same bus — different APIs, same underlying transport.

## The Command Catalog

The bus isn't a free-form pub/sub system. It has a typed command catalog — a fixed set of topics, each with a request type, a response type, and a Go handler. The catalog is defined once in `kit/catalog.go` and used by both Kernel (standalone) and Node (transport-connected).

```go
// kit/catalog.go — simplified
specs := []commandSpec{
    kernelCommand(func(ctx context.Context, k *Kernel, req messages.ToolCallMsg) (*messages.ToolCallResp, error) {
        return k.toolsDomain.Call(ctx, req)
    }),
    kernelCommand(func(ctx context.Context, k *Kernel, req messages.KitDeployMsg) (*messages.KitDeployResp, error) {
        return k.lifecycle.Deploy(ctx, req)
    }),
    nodeCommand(func(ctx context.Context, n *Node, req messages.PluginManifestMsg) (*messages.PluginManifestResp, error) {
        return n.processPluginManifest(ctx, req)
    }),
    // ... 30+ commands total
}
```

Each command has a topic (from `BusTopic()` on the message type), a decoder, and a handler. `kernelCommand` commands run on any Kernel. `nodeCommand` commands only run on a Node (they need plugin infrastructure).

### Current command topics

| Domain | Topics | Handler |
|--------|--------|---------|
| `tools.*` | call, resolve, list | ToolsDomain |
| `agents.*` | list, discover, get-status, set-status | AgentsDomain |
| `kit.*` | deploy, teardown, redeploy, list, deploy.file | LifecycleDomain |
| `workflow.*` | start, startAsync, status, resume, cancel, list, runs, restart | Catalog (inline JS eval) |
| `mcp.*` | listTools, callTool | Catalog (inline) |
| `registry.*` | has, list, resolve | Catalog (inline) |
| `secrets.*` | set, get, delete, list, rotate | SecretsDomain |
| `packages.*` | search, install, remove, update, list, info | PackagesDomain |
| `package.*` | deploy, teardown, redeploy, list, info | PackageDeployDomain |
| `metrics.get` | | Catalog (inline) |
| `trace.*` | get, list | Catalog (inline) |
| `rbac.*` | assign, revoke, list, roles | Catalog (inline) |
| `peers.*` | list, resolve (Node only) | Catalog (inline) |
| `test.run` | | TestingDomain |
| `plugin.*` | manifest, state.get, state.set, start, stop, restart, list, status (Node only) | PluginLifecycleDomain |
| `gateway.http.*` | route.add, route.remove, route.list, status | Gateway (bus subscriber) |

Topics NOT in the catalog are user-defined — `.ts` services use `bus.on("topic")` to create mailbox handlers, WASM shards register handlers via `bus_on`, and Go code can subscribe with `sdk.SubscribeTo`. These bypass the catalog entirely.

### How commands are routed

When `sdk.Publish(rt, ctx, messages.ToolCallMsg{Name: "echo", Input: ...})` is called:

1. `sdk.Publish` marshals the message, generates a correlationID and replyTo topic, stamps them in context
2. `RemoteClient.PublishRaw` resolves the namespace+topic, applies the transport's topic sanitizer, publishes via Watermill
3. The Host's consumer handler receives the Watermill message on the sanitized topic
4. The handler looks up the command in the catalog by topic, decodes the payload, calls the Go handler
5. The handler returns a response (or error)
6. The Host marshals the response and publishes it to the replyTo topic from the inbound message metadata
7. The caller's `sdk.SubscribeTo` handler receives the response on the replyTo topic

The replyTo topic is unique per Publish call (format: `<topic>.reply.<uuid>`), so multiple concurrent callers don't interfere.

### The LocalInvoker shortcut

When JS code calls `__go_brainkit_request("tools.call", payload)`, it doesn't go through the transport. The LocalInvoker looks up the command in the catalog and calls the handler directly — same Go function, no Watermill, no serialization round-trip. This is why tool calls from .ts code are fast: they skip the entire transport layer.

```go
// kit/local_invoker.go
func (i *LocalInvoker) Invoke(ctx context.Context, topic string, payload json.RawMessage) (json.RawMessage, error) {
    spec, ok := commandCatalog().Lookup(topic)
    if !ok || spec.invokeKernel == nil {
        return nil, fmt.Errorf("unknown topic: %s", topic)
    }
    return spec.invokeKernel(ctx, i.kernel, payload)
}
```

Workflow bus commands use `kernel.EvalTS()` to call Mastra's Workflow APIs via the JS registry — the Go catalog handler generates JS code that calls `workflow.createRun()`, `run.start()`, `run.resume()`, etc.

## The Async Pattern

brainkit uses pure async pub/sub. There is no blocking `AskSync`, no `PublishAwait`, no request-response helper that hides the async nature. The pattern is always:

```go
// 1. Publish — returns routing info
pr, err := sdk.Publish(rt, ctx, messages.ToolCallMsg{Name: "echo", Input: input})

// 2. Subscribe to the replyTo topic
done := make(chan messages.ToolCallResp, 1)
unsub, err := sdk.SubscribeTo[messages.ToolCallResp](rt, ctx, pr.ReplyTo,
    func(resp messages.ToolCallResp, msg messages.Message) {
        done <- resp
    })
defer unsub()

// 3. Wait with timeout
select {
case resp := <-done:
    // use resp
case <-ctx.Done():
    // timeout
}
```

This is verbose but explicit. You always know where the response comes from, you always handle timeouts, and you never block a goroutine waiting for a bus response.

For convenience, the SDK provides typed wrappers like `sdk.PublishToolCall` and `sdk.SubscribeToolCallResp` (generated by `codegen/sdkgen`), but they're thin aliases over `Publish` and `SubscribeTo`.

## Transport Backends

Six Watermill backends are supported. The transport is configured on Node (or injected into Kernel):

| Backend | Config Type | Topic Sanitizer | Container |
|---------|------------|-----------------|-----------|
| GoChannel (memory) | `"memory"` | none | none |
| SQLite | `"sql-sqlite"` | dots → underscores | none |
| NATS JetStream | `"nats"` | dots → dashes, slashes → dashes | `nats:latest -js` |
| AMQP (RabbitMQ) | `"amqp"` | slashes → dashes | `rabbitmq:management` |
| Redis Streams | `"redis"` | none | `redis:latest` |
| PostgreSQL | `"sql-postgres"` | dots → underscores | `postgres:16` |

### Topic sanitizers

Each transport has characters that are special or invalid in topic names. The sanitizer transforms logical topics (like `tools.call.reply.abc-123`) into transport-safe names before publishing and subscribing.

For NATS: dots are subject delimiters, so `tools.call` → `tools-call`. This is handled automatically by `RemoteClient` and `Host` — user code works with logical topic names.

For SQL backends: topics become table names, so dots → underscores.

GoChannel and Redis accept any string — no sanitization needed.

### NATS JetStream specifics

NATS uses JetStream with auto-provisioning. Each subscriber gets a durable consumer with a queue group. The `SubjectCalculator` maps topics to NATS subjects (replacing dots with dashes). The `DurablePrefix` is sanitized from the `NATSName` config.

Auto-provisioning means the first subscriber to a topic creates a JetStream stream. This can be slow — up to 30 seconds per topic on cold start. `NewNode` waits up to 2 minutes for `router.Running()` to account for this.

## Namespace Routing

Every Kernel has a namespace (default: `"user"`). All topics are prefixed with the namespace before hitting the transport:

```
Logical topic: tools.call
Namespace: my-kit
Wire topic: my-kit.tools.call (then sanitized per transport)
```

`RemoteClient.resolvedTopic` handles this:

```go
func (c *RemoteClient) resolvedTopic(logicalTopic string) string {
    topic := NamespacedTopic(c.namespace, logicalTopic)
    if c.topicSanitizer != nil {
        topic = c.topicSanitizer(topic)
    }
    return topic
}
```

Cross-Kit communication uses `PublishRawToNamespace` which prefixes with the TARGET namespace instead of the sender's:

```go
func (c *RemoteClient) PublishRawToNamespace(ctx context.Context, targetNamespace, logicalTopic string, payload json.RawMessage) (string, error) {
    // ... publish to targetNamespace.logicalTopic (sanitized)
}
```

See [cross-kit.md](cross-kit.md) for the full cross-Kit communication model.

## Middleware

Three middleware functions run on every inbound command message:

1. **DepthMiddleware** — reads the `depth` metadata field. If depth >= 16 (MaxDepth), returns `ErrCycleDetected`. This prevents infinite loops where handler A publishes to handler B which publishes back to handler A.

2. **CallerIDMiddleware** — stamps a default `callerId` in message metadata if not already set. Callers can override this via `messaging.WithPublishMeta`.

3. **MetricsMiddleware** — records processing time and error counts per topic. Available via `Metrics.Snapshot()`.

## The Event Catalog

Separate from the command catalog, the event catalog validates fire-and-forget events. It prevents publishing a known event type on a command topic (which would be silently ignored since no command handler would process it).

```go
// kit/events.go
func (r *knownEventRegistry) Validate(topic string, payload json.RawMessage) error {
    if commandCatalog().HasCommand(topic) {
        return fmt.Errorf("%w: %s", ErrCommandTopic, topic)
    }
    // ...
}
```

Known events: `kit.deployed`, `kit.teardown.done`, `plugin.registered`. User-defined events are not validated — they pass through freely.

## The JS Bus API

From deployed .ts code, the bus is accessed via the `bus` object (endowment from kit_runtime.js):

```typescript
// Publish with replyTo (request/response pattern)
const { replyTo, correlationId } = bus.publish("topic", data);

// Fire-and-forget
bus.emit("topic", data);

// Subscribe to any topic
const subId = bus.subscribe("topic", (msg) => {
    msg.reply(response);     // final response (done=true)
    msg.send(chunk);         // intermediate chunk (done=false)
});

// Mailbox subscribe (auto-prefixed with deployment namespace)
bus.on("localTopic", handler);  // subscribes to ts.<source>.<localTopic>

// Send to another .ts service
bus.sendTo("other-service.ts", "topic", data);

// Unsubscribe
bus.unsubscribe(subId);
```

Each of these calls a Go bridge function (`__go_brainkit_bus_publish`, `__go_brainkit_bus_emit`, etc.) which interacts with the Kernel's RemoteClient and transport.

