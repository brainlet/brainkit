# Go SDK

The SDK package (`sdk/`) is the public Go API for interacting with brainkit. It works with any `sdk.Runtime` — Kit or plugin client.

## The Async Pattern

brainkit is pure async pub/sub. Every operation follows the same pattern:

```go
// 1. Publish — returns routing info
pr, err := sdk.Publish(rt, ctx, sdk.ToolCallMsg{Name: "echo", Input: input})

// 2. Subscribe to the reply topic
done := make(chan sdk.ToolCallResp, 1)
unsub, err := sdk.SubscribeTo[sdk.ToolCallResp](rt, ctx, pr.ReplyTo,
    func(resp sdk.ToolCallResp, msg sdk.Message) {
        done <- resp
    })
defer unsub()

// 3. Wait
select {
case resp := <-done:
    fmt.Println("result:", string(resp.Result))
case <-ctx.Done():
    fmt.Println("timeout")
}
```

There is no `AskSync`, no `PublishAwait`, no blocking helper. You always publish, subscribe, and wait explicitly.

## Core Functions

### Publish — typed request with replyTo

```go
func Publish[T sdk.BrainkitMessage](rt Runtime, ctx context.Context, msg T, opts ...PublishOption) (PublishResult, error)
```

Sends a typed message. Generates a correlationID and replyTo topic automatically.

```go
type PublishResult struct {
    MessageID     string // Watermill message UUID
    CorrelationID string // for response filtering
    ReplyTo       string // where responses will be sent
    Topic         string // where the message was published
}
```

### Emit — fire-and-forget

```go
func Emit[T sdk.BrainkitMessage](rt Runtime, ctx context.Context, msg T) error
```

Sends a typed message with no replyTo. No response expected.

```go
sdk.Emit(rt, ctx, sdk.KitDeployedEvent{
    Source:    "my-service.ts",
    Resources: resources,
})
```

### SubscribeTo — typed subscription

```go
func SubscribeTo[T any](rt Runtime, ctx context.Context, topic string, handler func(T, sdk.Message)) (func(), error)
```

Subscribes to a topic and deserializes messages into type T. Returns a cancel function. The subscription is active before SubscribeTo returns (contract: no race between publish and subscribe).

### Reply — respond to a message

```go
func Reply(rt Runtime, ctx context.Context, msg sdk.Message, payload any) error
```

Sends a final response to the message's replyTo topic. Sets `done=true` in metadata. Returns `ErrNoReplyTo` if the message has no replyTo (was emitted, not published). Returns `ErrNotReplier` if the runtime doesn't implement the Replier interface.

### SendChunk — intermediate streaming response

```go
func SendChunk(rt Runtime, ctx context.Context, msg sdk.Message, payload any) error
```

Same as Reply but sets `done=false`. Use for streaming patterns where multiple responses precede a final Reply.

```go
// Pattern from test/bus/sdk_reply_test.go
sdk.SubscribeTo[json.RawMessage](rt, ctx, "request.topic",
    func(payload json.RawMessage, msg sdk.Message) {
        sdk.SendChunk(rt, ctx, msg, map[string]int{"chunk": 1})
        sdk.SendChunk(rt, ctx, msg, map[string]int{"chunk": 2})
        sdk.SendChunk(rt, ctx, msg, map[string]int{"chunk": 3})
        sdk.Reply(rt, ctx, msg, map[string]any{"done": true, "total": 3})
    })
```

### SendToService — address a .ts service

```go
func SendToService(rt Runtime, ctx context.Context, service, topic string, payload any, opts ...PublishOption) (PublishResult, error)
```

Resolves the naming convention and publishes:
- `"my-agent.ts"` + `"ask"` → topic `"ts.my-agent.ask"`
- `"my-agent"` + `"ask"` → topic `"ts.my-agent.ask"` (`.ts` suffix optional)
- `"nested/svc"` + `"rpc"` → topic `"ts.nested.svc.rpc"`

```go
// Pattern from test/bus/sdk_reply_test.go
pr, err := sdk.SendToService(rt, ctx, "calc.ts", "add", map[string]int{"a": 17, "b": 25})
```

### PublishTo — cross-Kit

```go
func PublishTo[T sdk.BrainkitMessage](rt Runtime, ctx context.Context, targetNamespace string, msg T, opts ...PublishOption) (PublishResult, error)
```

Publishes to a specific Kit's namespace. Requires `CrossNamespaceRuntime` (Kit, not plugin client). See [cross-kit.md](../concepts/cross-kit.md).

### WithReplyTo — override reply topic

```go
pr, err := sdk.SendToService(rt, ctx, "streamer.ts", "stream",
    json.RawMessage(`{}`),
    sdk.WithReplyTo("my-custom-reply-topic"),
)
// Responses go to "my-custom-reply-topic" instead of auto-generated UUID topic
```

## Typed Message Pairs

Every command has a request type and a response type in `sdk/`:

| Request | Response | Topic |
|---------|----------|-------|
| `ToolCallMsg` | `ToolCallResp` | `tools.call` |
| `ToolListMsg` | `ToolListResp` | `tools.list` |
| `ToolResolveMsg` | `ToolResolveResp` | `tools.resolve` |
| `KitDeployMsg` | `KitDeployResp` | `kit.deploy` |
| `KitTeardownMsg` | `KitTeardownResp` | `kit.teardown` |
| `KitRedeployMsg` | `KitRedeployResp` | `kit.redeploy` |
| `KitListMsg` | `KitListResp` | `kit.list` |
| `AgentListMsg` | `AgentListResp` | `agents.list` |
| `AgentDiscoverMsg` | `AgentDiscoverResp` | `agents.discover` |
| `AgentGetStatusMsg` | `AgentGetStatusResp` | `agents.get-status` |
| `AgentSetStatusMsg` | `AgentSetStatusResp` | `agents.set-status` |
| `WorkflowStartMsg` | `WorkflowStartResp` | `workflow.start` |
| `WorkflowStartAsyncMsg` | `WorkflowStartAsyncResp` | `workflow.startAsync` |
| `WorkflowStatusMsg` | `WorkflowStatusResp` | `workflow.status` |
| `WorkflowResumeMsg` | `WorkflowResumeResp` | `workflow.resume` |
| `WorkflowCancelMsg` | `WorkflowCancelResp` | `workflow.cancel` |
| `WorkflowListMsg` | `WorkflowListResp` | `workflow.list` |
| `WorkflowRunsMsg` | `WorkflowRunsResp` | `workflow.runs` |
| `WorkflowRestartMsg` | `WorkflowRestartResp` | `workflow.restart` |
| `McpListToolsMsg` | `McpListToolsResp` | `mcp.listTools` |
| `McpCallToolMsg` | `McpCallToolResp` | `mcp.callTool` |
| `RegistryHasMsg` | `RegistryHasResp` | `registry.has` |
| `RegistryListMsg` | `RegistryListResp` | `registry.list` |
| `RegistryResolveMsg` | `RegistryResolveResp` | `registry.resolve` |
| `SecretsSetMsg` | `SecretsSetResp` | `secrets.set` |
| `SecretsGetMsg` | `SecretsGetResp` | `secrets.get` |
| `SecretsDeleteMsg` | `SecretsDeleteResp` | `secrets.delete` |
| `SecretsListMsg` | `SecretsListResp` | `secrets.list` |
| `SecretsRotateMsg` | `SecretsRotateResp` | `secrets.rotate` |

### CustomMsg — ad-hoc topics

For topics not in the command catalog (user-defined services, custom events):

```go
pr, err := sdk.Publish(rt, ctx, sdk.CustomMsg{
    Topic:   "my-custom-topic",
    Payload: json.RawMessage(`{"hello":"world"}`),
})
```

`CustomMsg.MarshalJSON` serializes only the Payload (not the Topic wrapper) so subscribers receive the inner payload directly. This means `json.Marshal(customMsg)` drops the Topic — use `customMsg.String()` for debugging.

### Events

Fire-and-forget events:

```go
sdk.Emit(rt, ctx, sdk.KitDeployedEvent{Source: "my-service.ts"})
sdk.Emit(rt, ctx, sdk.KitTeardownedEvent{Source: "my-service.ts", Removed: 3})
sdk.Emit(rt, ctx, sdk.PluginRegisteredEvent{Owner: "acme", Name: "cron", Version: "1.0.0", Tools: 5})
```

## Generated Typed Wrappers

`sdk/typed_gen.go` (generated by `codegen/sdkgen`) provides typed wrapper functions for every message pair:

```go
// These are thin aliases — no additional logic
pr, err := sdk.PublishToolCall(rt, ctx, sdk.ToolCallMsg{...})
unsub, err := sdk.SubscribeToolCallResp(rt, ctx, pr.ReplyTo, handler)

pr, err := sdk.PublishKitDeploy(rt, ctx, sdk.KitDeployMsg{...})
unsub, err := sdk.SubscribeKitDeployResp(rt, ctx, pr.ReplyTo, handler)
```

Use them for discoverability — your IDE's autocomplete shows every available operation. But they're strictly optional; `sdk.Publish` + `sdk.SubscribeTo` work with any message type.

## Error Handling

```go
import "github.com/brainlet/brainkit/sdk"

_, err := sdk.SendToService(rt, ctx, "calc.ts", "add", payload)
if err != nil {
    var notFound *sdk.NotFoundError
    var exists *sdk.AlreadyExistsError
    var valErr *sdk.ValidationError
    var timeout *sdk.TimeoutError

    switch {
    case errors.As(err, &notFound):
        fmt.Printf("%s %q not found\n", notFound.Resource, notFound.Name)
    case errors.As(err, &exists):
        fmt.Printf("%s %q already exists (%s)\n", exists.Resource, exists.Name, exists.Hint)
    case errors.As(err, &valErr):
        fmt.Printf("invalid %s: %s\n", valErr.Field, valErr.Message)
    case errors.As(err, &timeout):
        fmt.Printf("timeout: %s\n", timeout.Operation)
    case errors.Is(err, sdk.ErrNoReplyTo):
        fmt.Println("message has no reply destination")
    default:
        fmt.Printf("error: %v\n", err)
    }
}
```

See [error-handling.md](../concepts/error-handling.md) for the full error type inventory.

## The Runtime Interface

```go
type Runtime interface {
    PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (correlationID string, err error)
    SubscribeRaw(ctx context.Context, topic string, handler func(sdk.Message)) (cancel func(), err error)
    Close() error
}
```

Kit and plugin clients both implement this. Code that takes `sdk.Runtime` works with either.

## The Message Envelope

```go
// sdk/bus.go
type Message struct {
    Topic    string            `json:"topic"`
    Payload  []byte            `json:"payload"`
    CallerID string            `json:"callerId,omitempty"`
    TraceID  string            `json:"traceId,omitempty"`
    Metadata map[string]string `json:"metadata,omitempty"`
}
```

Five fields. `Metadata` carries `replyTo`, `correlationId`, `done`, `callerId`, and `depth`. This is the internal platform envelope — the typed message types (ToolCallMsg, etc.) are the Payload, deserialized by `SubscribeTo`.
