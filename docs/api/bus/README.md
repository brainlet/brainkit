# Bus API Reference

Async-first message router. Supports fire-and-forget (`Send`), request/response (`Ask`), topic subscriptions (`On`), interceptors, job tracking, and pluggable transports.

---

## Core Types

### Message

The bus message envelope.

```go
type Message struct {
    Version  string            `json:"v"`
    Topic    string            `json:"topic"`
    Address  string            `json:"addr,omitempty"`
    ReplyTo  string            `json:"replyTo,omitempty"`
    CallerID string            `json:"caller"`
    ID       string            `json:"id"`
    ParentID string            `json:"parent,omitempty"`
    TraceID  string            `json:"trace"`
    Depth    int               `json:"depth,omitempty"`
    Payload  json.RawMessage   `json:"payload"`
    Metadata map[string]string `json:"meta,omitempty"`
}
```

| Field | Type | Description |
|-------|------|-------------|
| `Version` | `string` | Protocol version (`"v1"`) |
| `Topic` | `string` | Message topic for routing |
| `Address` | `string` | Target address. Empty = local. `"kit:X"` or `"host:X/kit:Y/agent:Z"` for remote. |
| `ReplyTo` | `string` | Reply topic (set by `Ask`, empty for `Send`) |
| `CallerID` | `string` | Sender identity |
| `ID` | `string` | Unique message ID (auto-generated if empty) |
| `ParentID` | `string` | ID of the message this replies to |
| `TraceID` | `string` | Trace ID for the entire message cascade (defaults to `ID`) |
| `Depth` | `int` | Cascade depth. Cycle detection triggers at `MaxDepth` (16). |
| `Payload` | `json.RawMessage` | Message content (JSON) |
| `Metadata` | `map[string]string` | Optional key-value metadata |

### SubscriptionID

```go
type SubscriptionID string
```

Opaque identifier returned by `On`, passed to `Off` for unsubscription.

### ReplyFunc

```go
type ReplyFunc func(payload json.RawMessage)
```

Passed to `On` handlers. Sends a response back to the `Ask` caller. No-op if the original message was a `Send`. Can only be called once -- second call is a no-op.

---

## Bus

### Constructor

```go
func NewBus(transport Transport, opts ...BusOption) *Bus
```

Creates a Bus backed by the given transport.

```go
b := bus.NewBus(bus.NewInProcessTransport())
```

### BusOption

| Option | Description |
|--------|-------------|
| `WithHandlerTimeout(d time.Duration)` | Default Ask timeout. Default: `30s`. |
| `WithJobTimeout(d time.Duration)` | Default cascade timeout. Default: `5m`. |
| `WithJobRetention(d time.Duration)` | How long completed jobs stay in memory. Default: `5m`. |

### Send

Fire-and-forget broadcast. Runs interceptors, tracks in job system, routes addressed messages via `Forward`.

```go
func (b *Bus) Send(msg Message) error
```

Returns error if cascade depth exceeds `MaxDepth` (16) or an interceptor rejects the message.

```go
payload, _ := json.Marshal(map[string]string{"name": "greeter"})
b.Send(bus.Message{
    Topic:    "agents.request",
    CallerID: "my-kit",
    Payload:  payload,
})
```

### Ask

Sends a message and registers a one-shot callback for the reply. The callback is guaranteed to fire exactly once (either reply or timeout). Returns a cancel function.

```go
func (b *Bus) Ask(msg Message, callback func(Message)) (cancel func())
```

```go
cancel := b.Ask(bus.Message{
    Topic:    "tools.call",
    CallerID: "my-kit",
    Payload:  payload,
}, func(reply bus.Message) {
    fmt.Println("got reply:", string(reply.Payload))
})
// cancel() to abort if no longer needed
```

### AskSync

Blocking convenience wrapper around `Ask`. Not a bus primitive -- a helper for Go code that needs synchronous request/response.

```go
func AskSync(b *Bus, ctx context.Context, msg Message) (*Message, error)
```

```go
reply, err := bus.AskSync(b, ctx, bus.Message{
    Topic:    "kit.list",
    CallerID: "my-kit",
    Payload:  json.RawMessage("{}"),
})
```

### On

Subscribes to messages matching a topic pattern. Returns a `SubscriptionID` for later unsubscription.

```go
func (b *Bus) On(pattern string, handler func(Message, ReplyFunc), opts ...SubscribeOption) SubscriptionID
```

**Topic matching**:
- `"test.foo"` matches only `"test.foo"`
- `"test.*"` matches `"test.foo"`, `"test.foo.bar"`, etc.

```go
subID := b.On("tools.*", func(msg bus.Message, reply bus.ReplyFunc) {
    result, _ := json.Marshal(map[string]string{"ok": "true"})
    reply(result)
})
```

### SubscribeOption

| Option | Description |
|--------|-------------|
| `AsWorker(group string)` | Join a worker group. Only ONE subscriber in the group receives each message. Enables competing consumers. |
| `WithAddress(addr string)` | Filter: only receive messages addressed to this entity. |

```go
// Competing consumers -- 3 workers, each message goes to exactly one
b.On("tasks.*", handler, bus.AsWorker("task-pool"))
b.On("tasks.*", handler, bus.AsWorker("task-pool"))
b.On("tasks.*", handler, bus.AsWorker("task-pool"))

// Address filtering -- only messages addressed to "agent:greeter"
b.On("agents.*", handler, bus.WithAddress("agent:greeter"))
```

### Off

Removes a subscription.

```go
func (b *Bus) Off(id SubscriptionID)
```

### AddInterceptor

Registers an interceptor. Interceptors are sorted by priority (lowest first) and run before message dispatch.

```go
func (b *Bus) AddInterceptor(i Interceptor)
```

### Interceptor Interface

```go
type Interceptor interface {
    Name() string
    Priority() int
    Match(topic string) bool
    Process(msg *Message) error
}
```

| Method | Description |
|--------|-------------|
| `Name()` | Interceptor name (for logging) |
| `Priority()` | Sort order (lowest runs first) |
| `Match(topic)` | Return `true` to intercept this topic |
| `Process(msg)` | Modify `Payload`/`Metadata` or return error to reject. `Topic`, `CallerID`, and `Address` are immutable -- changes are reverted. |

### RegisterName / UnregisterName

Kit name collision detection on the bus.

```go
func (b *Bus) RegisterName(name string) error  // error if already taken
func (b *Bus) UnregisterName(name string)
```

### Jobs

```go
func (b *Bus) Jobs() []Job          // all tracked jobs
func (b *Bus) Job(traceID string) *Job  // specific job by trace ID
func (b *Bus) SetJobTimeout(timeout time.Duration)
```

### Job

Tracks a cascade of messages sharing a `TraceID`.

```go
type Job struct {
    TraceID     string    `json:"traceId"`
    Status      string    `json:"status"`      // running | completed | failed | timeout
    StartedAt   time.Time `json:"startedAt"`
    CompletedAt time.Time `json:"completedAt,omitempty"`
    Messages    int       `json:"messages"`
    Pending     int       `json:"pending"`
}
```

### Metrics

```go
func (b *Bus) Metrics() BusMetrics
```

```go
type BusMetrics struct {
    Transport   TransportMetrics `json:"transport"`
    ActiveJobs  int              `json:"activeJobs"`
    TotalJobs   int              `json:"totalJobs"`
    Subscribers int              `json:"subscribers"`
}
```

### Close

Shuts down the bus. Stops job tracking, cancels pending reply entries, closes the transport.

```go
func (b *Bus) Close()
```

---

## Transport Interface

Pluggable message delivery backend.

```go
type Transport interface {
    Publish(msg Message) error
    Forward(msg Message, target string) error
    Subscribe(info SubscriberInfo) error
    Unsubscribe(id SubscriptionID) error
    Metrics() TransportMetrics
    SubscriberCount() int
    Close() error
}
```

| Method | Description |
|--------|-------------|
| `Publish` | Dispatch a message to matching local subscribers |
| `Forward` | Send a message to a remote target. Returns `ErrNoRoute` if unreachable. |
| `Subscribe` | Register a subscriber |
| `Unsubscribe` | Remove a subscription |
| `Metrics` | Transport-level stats |
| `SubscriberCount` | Number of active subscribers |
| `Close` | Shut down the transport |

### SubscriberInfo

```go
type SubscriberInfo struct {
    ID      SubscriptionID
    Pattern string
    Group   string          // "" = broadcast, "name" = worker group
    Address string          // "" = all, "agent:X" = filter by address
    Handler func(Message)
}
```

### TransportMetrics

```go
type TransportMetrics struct {
    Topics  map[string]TopicMetrics       `json:"topics"`
    Workers map[string]WorkerGroupMetrics `json:"workers"`
}

type TopicMetrics struct {
    Pending int     `json:"pending"`
    Rate    float64 `json:"rate"`    // messages/sec
}

type WorkerGroupMetrics struct {
    Name       string  `json:"name"`
    Members    int     `json:"members"`
    Pending    int     `json:"pending"`
    Throughput float64 `json:"throughput"`
}
```

---

## Transport Implementations

### InProcessTransport

Default local transport. Delivers messages in-process via goroutines and channels.

```go
func NewInProcessTransport() *InProcessTransport
```

- Broadcast subscribers each get their own buffered channel (256)
- Worker group members share a single channel (competing consumers)
- `Forward` always returns `ErrNoRoute` (no remote targets)

```go
t := bus.NewInProcessTransport()
b := bus.NewBus(t)
```

### GRPCTransport

Combines local in-process delivery with gRPC-based forwarding to remote Kits. Uses an embedded `InProcessTransport` for local subscribers and forwards addressed messages over gRPC.

```go
func NewGRPCTransport() *GRPCTransport
```

- Local `Publish`/`Subscribe`/`Unsubscribe` delegated to an internal `InProcessTransport`
- `Forward` looks up the target peer by name and sends via gRPC stream
- Supports optional `Discovery` interface for resolving unknown peers on demand
- Address format: `"kit:<name>"` or `"host:<name>"`

```go
t := brainkit.NewGRPCTransport()
b := bus.NewBus(t)
```

### NATSTransport

Uses NATS as the message broker. Bus topics map directly to NATS subjects. Worker groups map to NATS queue subscriptions.

```go
func NewNATSTransport(url string, opts ...nats.Option) (*NATSTransport, error)
```

- Auto-reconnect with unlimited retries
- Worker groups use NATS queue subscriptions (competing consumers)
- Address routing uses subject prefixes: `"tools.call"` with address `"kit:staging"` becomes NATS subject `"kit.staging.tools.call"`
- Topic pattern `"events.*"` maps to NATS `"events.*"`, `"events.**"` maps to `"events.>"`

```go
t, err := brainkit.NewNATSTransport("nats://localhost:4222")
if err != nil {
    log.Fatal(err)
}
b := bus.NewBus(t)
```

---

## SDK Message Types

The SDK provides its own `Message` and `ReplyFunc` types for plugin interactions.

```go
import "github.com/brainlet/brainkit/sdk/messages"
```

### BusMessage Interface

```go
type BusMessage interface {
    BusTopic() string
}
```

All typed SDK messages implement this interface. The SDK uses `BusTopic()` to route messages automatically.

### messages.Message

```go
type Message struct {
    Topic    string            `json:"topic"`
    Payload  []byte            `json:"payload"`
    CallerID string            `json:"callerId,omitempty"`
    TraceID  string            `json:"traceId,omitempty"`
    Metadata map[string]string `json:"metadata,omitempty"`
}
```

### messages.ReplyFunc

```go
type ReplyFunc func(payload any) error
```

Used in SDK `On` handlers when the sender used `Ask`.

---

## Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `DefaultHandlerTimeout` | `30s` | Default Ask reply timeout |
| `DefaultJobTimeout` | `5m` | Default cascade timeout |
| `DefaultJobRetention` | `5m` | How long completed jobs stay in memory |
| `ProtocolVersion` | `"v1"` | Bus protocol version |
| `MaxDepth` | `16` | Maximum cascade depth before cycle detection |

## Sentinel Errors

| Error | Description |
|-------|-------------|
| `ErrNoRoute` | Returned by `Forward` when a message cannot be delivered to the target |
