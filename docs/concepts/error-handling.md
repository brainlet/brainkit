# Error Handling

brainkit uses typed errors for conditions that callers need to handle semantically — "tool not found" is different from "tool execution failed" and should trigger different recovery logic. Internal plumbing errors (transport setup, marshal failures, JS bridge exceptions) stay as `fmt.Errorf` wrappers.

## Error Types

15 error types in `sdk/errors.go` (backed by `internal/sdkerrors/errors.go` to avoid import cycles). All implement `BrainkitError` interface with `Code() string` and `Details() map[string]any`:

### NotFoundError

Returned when a named resource doesn't exist.

```go
type NotFoundError struct {
    Resource string  // "tool", "agent", "shard", "module", "storage", "pool", "peer", "mcp-server"
    Name     string  // the name that was looked up
}
```

```go
pr, err := sdk.Publish(rt, ctx, sdk.ToolCallMsg{Name: "nonexistent"})
// ...
var notFound *sdk.NotFoundError
if errors.As(err, &notFound) {
    fmt.Printf("%s %q not found\n", notFound.Resource, notFound.Name)
    // → "tool "nonexistent" not found"
}
```

Used in: tool registry resolution (5 levels), agent get-status/unregister, shard invoke/undeploy, WASM module run/deploy, storage remove, pool scale/kill/info, discovery resolve, MCP server call.

### AlreadyExistsError

Returned when creating a resource that already exists.

```go
type AlreadyExistsError struct {
    Resource string  // "deployment", "shard", "storage", "pool"
    Name     string
    Hint     string  // optional, e.g. "use Redeploy"
}
```

```go
_, err := k.Deploy(ctx, "service.ts", code)
// err if already deployed:
var exists *sdk.AlreadyExistsError
if errors.As(err, &exists) {
    if exists.Hint == "use Redeploy" {
        k.Redeploy(ctx, exists.Name, code) // follow the hint
    }
}
```

Used in: Deploy (already deployed), wasm.deploy (shard already deployed), AddStorage (name taken), SpawnPool (name taken), wasm.remove (shard still deployed — hint: "undeploy first").

### ValidationError

Returned when input fails validation.

```go
type ValidationError struct {
    Field   string  // field name, e.g. "name", "status", "mode", "transport"
    Message string  // human-readable reason
}
```

```go
var valErr *sdk.ValidationError
if errors.As(err, &valErr) {
    fmt.Printf("invalid %s: %s\n", valErr.Field, valErr.Message)
    // → "invalid status: invalid value "unknown" (must be idle|busy|error)"
}
```

Used in: agent register/set-status (name/status required, invalid status value), shard descriptor validation (invalid mode, handler not in exports), plugin state transport (unsupported), plugin config (requires nats), registry registration (name required), agent Generate/Stream (prompt or messages required).

### TimeoutError

Returned when an operation exceeds its deadline.

```go
type TimeoutError struct {
    Operation string  // what timed out
}
```

```go
var timeout *sdk.TimeoutError
if errors.As(err, &timeout) {
    fmt.Printf("timeout: %s\n", timeout.Operation)
    // → "timeout: plugin manifest registration"
    // → "timeout: plugin READY"
    // → "timeout: router start (NATS JetStream provisioning)"
}
```

Used in: plugin manifest registration (30s), plugin READY line (StartTimeout), transport-connected Kit router start (2 minutes for NATS JetStream auto-provisioning).

### WorkspaceEscapeError

Returned when a filesystem path escapes the workspace boundary.

```go
type WorkspaceEscapeError struct {
    Path string  // the path that was attempted
}
```

```go
var escape *sdk.WorkspaceEscapeError
if errors.As(err, &escape) {
    fmt.Printf("blocked path traversal: %s\n", escape.Path)
    // → "blocked path traversal: ../../etc/passwd"
}
```

Used in: all fs.* operations when the resolved path is outside `Config.FSRoot`.

## Sentinel Errors

Nine sentinel errors for fixed conditions. Check with `errors.Is(err, sentinel)`:

### SDK sentinels (sdk/errors.go)

| Sentinel | When | Where |
|----------|------|-------|
| `sdk.ErrNoReplyTo` | Message has no replyTo metadata | `sdk.Reply`, `sdk.SendChunk` — the message was fire-and-forget (emitted, not published) |
| `sdk.ErrNotReplier` | Runtime doesn't implement Replier | `sdk.Reply`, `sdk.SendChunk` — the runtime can't send responses (shouldn't happen with Kit) |
| `sdk.ErrNotCrossNamespace` | Runtime doesn't support cross-Kit | `sdk.PublishTo` — plugin clients don't implement CrossNamespaceRuntime |

### Kit sentinels (kit/errors.go)

| Sentinel | When | Where |
|----------|------|-------|
| `kit.ErrNoWorkspace` | FSRoot not configured | All fs.* operations — set `Config.FSRoot` |
| `kit.ErrMCPNotConfigured` | No MCP servers registered | `mcp.listTools`, `mcp.callTool` — set `Config.MCPServers` |
| `kit.ErrCommandTopic` | Event emitted on command topic | `bus.emit` called with a catalog command topic like "tools.call" — use `bus.publish` instead |

### Internal sentinels

| Sentinel | Package | When |
|----------|---------|------|
| `agentembed.ErrSandboxClosed` | `internal/embed/agent` | Eval/CreateAgent called on a closed sandbox |
| `agentembed.ErrAgentClosed` | `internal/embed/agent` | Generate/Stream called on a closed agent |
| `messaging.ErrCycleDetected` | `internal/messaging` | Message cascade exceeded depth 16 — prevents infinite loops |

## Usage Patterns

### Handling tool calls

```go
pr, err := sdk.Publish(rt, ctx, sdk.ToolCallMsg{Name: name, Input: input})
if err != nil {
    var notFound *sdk.NotFoundError
    var valErr *sdk.ValidationError
    switch {
    case errors.As(err, &notFound):
        return fmt.Errorf("tool %q does not exist", notFound.Name)
    case errors.As(err, &valErr):
        return fmt.Errorf("invalid input: %s", valErr.Message)
    default:
        return fmt.Errorf("tool call failed: %w", err)
    }
}
```

### Handling deployments

```go
_, err := k.Deploy(ctx, source, code)
if err != nil {
    var exists *sdk.AlreadyExistsError
    if errors.As(err, &exists) {
        // Already deployed — redeploy instead
        _, err = k.Redeploy(ctx, source, code)
    }
    return err
}
```

### Handling replies

```go
err := sdk.Reply(rt, ctx, msg, response)
if err != nil {
    switch {
    case errors.Is(err, sdk.ErrNoReplyTo):
        // Message was fire-and-forget — no one is waiting for a reply
        log.Printf("no reply destination for message on %s", msg.Topic)
    case errors.Is(err, sdk.ErrNotReplier):
        // Bug — this runtime should support replies
        panic("runtime does not support Reply")
    default:
        return err // transport error
    }
}
```

## What Stays as fmt.Errorf

Not every error needs a type. These stay as wrapped string errors:

- **Marshal failures** (`"marshal %T: %w"`) — programming error, the type isn't JSON-serializable
- **Transport setup** (`"brainkit: transport: %w"`) — wraps Watermill/NATS/Redis driver errors
- **QuickJS evaluation** (`"deploy %s: %w"`) — wraps JS exceptions, the error message IS the diagnostic
- **WASM compile/instantiate** (`"wasm.compile: %w"`) — wraps wazero errors
- **Store operations** (`"kitstore: %w"`) — wraps SQLite errors
- **MCP protocol** (`"mcp: initialize %q: %w"`) — wraps mcp-go errors
- **JS bridge exceptions** — `ThrowError` in bridges.go delivers errors as JS exceptions, not Go errors

The rule: if a caller would handle this error differently from other errors of the same operation, it needs a type. If the caller just logs and returns, `fmt.Errorf` is fine.
