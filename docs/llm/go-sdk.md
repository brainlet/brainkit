# Go SDK — API Reference

> `import "github.com/brainlet/brainkit/sdk"`
> `import "github.com/brainlet/brainkit/sdk/messages"`

## Interfaces

```go
type Runtime interface {
    PublishRaw(ctx context.Context, topic string, payload json.RawMessage) (correlationID string, err error)
    SubscribeRaw(ctx context.Context, topic string, handler func(messages.Message)) (cancel func(), err error)
    Close() error
}

type CrossNamespaceRuntime interface {
    Runtime
    PublishRawTo(ctx context.Context, targetNamespace, topic string, payload json.RawMessage) (correlationID string, err error)
    SubscribeRawTo(ctx context.Context, targetNamespace, topic string, handler func(messages.Message)) (cancel func(), err error)
}

type Replier interface {
    ReplyRaw(ctx context.Context, replyTo, correlationID string, payload json.RawMessage, done bool) error
}
```

Kernel and Node implement all three. Plugin clients implement only Runtime.

## Core Functions

```go
func Publish[T messages.BrainkitMessage](rt Runtime, ctx context.Context, msg T, opts ...PublishOption) (PublishResult, error)
func Emit[T messages.BrainkitMessage](rt Runtime, ctx context.Context, msg T) error
func SubscribeTo[T any](rt Runtime, ctx context.Context, topic string, handler func(T, messages.Message)) (func(), error)
func Reply(rt Runtime, ctx context.Context, msg messages.Message, payload any) error
func SendChunk(rt Runtime, ctx context.Context, msg messages.Message, payload any) error
func SendToService(rt Runtime, ctx context.Context, service, topic string, payload any, opts ...PublishOption) (PublishResult, error)
func SendToShard(rt Runtime, ctx context.Context, shard, topic string, payload any, opts ...PublishOption) (PublishResult, error)
func PublishTo[T messages.BrainkitMessage](rt Runtime, ctx context.Context, targetNamespace string, msg T, opts ...PublishOption) (PublishResult, error)
```

## Types

```go
type PublishResult struct {
    MessageID     string
    CorrelationID string
    ReplyTo       string
    Topic         string
}

type PublishOption func(*publishConfig)
func WithReplyTo(topic string) PublishOption
```

## Error Types

```go
// sdk/errors.go — check with errors.As
type NotFoundError struct { Resource, Name string }
type AlreadyExistsError struct { Resource, Name, Hint string }
type ValidationError struct { Field, Message string }
type TimeoutError struct { Operation string }
type WorkspaceEscapeError struct { Path string }

// Sentinels — check with errors.Is
var ErrNoReplyTo error          // message has no replyTo
var ErrNotReplier error         // runtime doesn't implement Replier
var ErrNotCrossNamespace error  // runtime doesn't support cross-Kit
```

## Message Types — sdk/messages/

### BrainkitMessage interface

```go
type BrainkitMessage interface { BusTopic() string }
```

### Message envelope

```go
type Message struct {
    Topic    string            `json:"topic"`
    Payload  []byte            `json:"payload"`
    CallerID string            `json:"callerId,omitempty"`
    TraceID  string            `json:"traceId,omitempty"`
    Metadata map[string]string `json:"metadata,omitempty"`
}
```

### ResultMeta (embedded in all response types)

```go
type ResultMeta struct { Error string `json:"error,omitempty"` }
func (m *ResultMeta) SetError(err string)
func (m ResultMeta) ResultError() string
func ResultErrorOf(v any) string
```

### CustomMsg

```go
type CustomMsg struct {
    Topic   string          `json:"-"`
    Payload json.RawMessage `json:"payload"`
}
func (m CustomMsg) BusTopic() string           // returns Topic
func (m CustomMsg) MarshalJSON() ([]byte, error) // returns Payload only (not wrapped)
```

### Tools

```go
type ToolCallMsg struct { Name string; Input any }           // tools.call
type ToolListMsg struct { Namespace string }                  // tools.list
type ToolResolveMsg struct { Name string }                    // tools.resolve

type ToolCallResp struct { ResultMeta; Result json.RawMessage }
type ToolListResp struct { ResultMeta; Tools []ToolInfo }
type ToolResolveResp struct { ResultMeta; Name, ShortName, Description string; InputSchema any }
type ToolInfo struct { Name, ShortName, Namespace, Description string }
```

### Agents

```go
type AgentListMsg struct { Filter *AgentFilter }              // agents.list
type AgentDiscoverMsg struct { Capability, Model, Status string } // agents.discover
type AgentGetStatusMsg struct { Name string }                  // agents.get-status
type AgentSetStatusMsg struct { Name, Status string }          // agents.set-status

type AgentListResp struct { ResultMeta; Agents []AgentInfo }
type AgentDiscoverResp struct { ResultMeta; Agents []AgentInfo }
type AgentGetStatusResp struct { ResultMeta; Name, Status string }
type AgentSetStatusResp struct { ResultMeta; OK bool }
type AgentInfo struct { Name string; Capabilities []string; Model, Status, Kit string }
type AgentFilter struct { Capability, Model, Status string }
```

### Kit Lifecycle

```go
type KitDeployMsg struct { Source, Code string }               // kit.deploy
type KitTeardownMsg struct { Source string }                    // kit.teardown
type KitRedeployMsg struct { Source, Code string }              // kit.redeploy
type KitListMsg struct{}                                       // kit.list

type KitDeployResp struct { ResultMeta; Deployed bool; Resources []ResourceInfo }
type KitTeardownResp struct { ResultMeta; Removed int }
type KitRedeployResp struct { ResultMeta; Deployed bool; Resources []ResourceInfo }
type KitListResp struct { ResultMeta; Deployments []DeploymentInfo }
type ResourceInfo struct { Type, ID, Name, Source string; CreatedAt int64 }
type DeploymentInfo struct { Source, CreatedAt string; Resources []ResourceInfo }
```

### Filesystem

```go
type FsReadMsg struct { Path string }                          // fs.read
type FsWriteMsg struct { Path, Data string }                   // fs.write
type FsListMsg struct { Path, Pattern string }                 // fs.list
type FsStatMsg struct { Path string }                          // fs.stat
type FsDeleteMsg struct { Path string }                        // fs.delete
type FsMkdirMsg struct { Path string }                         // fs.mkdir

type FsReadResp struct { ResultMeta; Data string }
type FsWriteResp struct { ResultMeta; OK bool }
type FsListResp struct { ResultMeta; Files []FsFileInfo }
type FsStatResp struct { ResultMeta; Size int64; IsDir bool; ModTime string }
type FsDeleteResp struct { ResultMeta; OK bool }
type FsMkdirResp struct { ResultMeta; OK bool }
type FsFileInfo struct { Name string; Size int64; IsDir bool }
```

### WASM

```go
type WasmCompileMsg struct { Source string; Options *WasmCompileOpts }  // wasm.compile
type WasmRunMsg struct { ModuleID string; Input any }                    // wasm.run
type WasmDeployMsg struct { Name string }                                // wasm.deploy
type WasmUndeployMsg struct { Name string }                              // wasm.undeploy
type WasmListMsg struct{}                                                // wasm.list
type WasmGetMsg struct { Name string }                                   // wasm.get
type WasmRemoveMsg struct { Name string }                                // wasm.remove
type WasmDescribeMsg struct { Name string }                              // wasm.describe

type WasmCompileOpts struct { Name, Runtime string }
type WasmCompileResp struct { ResultMeta; ModuleID, Name, Text string; Size int; Exports []string }
type WasmRunResp struct { ResultMeta; ExitCode int; Value any }
type WasmDeployResp struct { ResultMeta; Module, Mode string; Handlers map[string]string }
type WasmUndeployResp struct { ResultMeta; Undeployed bool }
type WasmListResp struct { ResultMeta; Modules []WasmModuleInfo }
type WasmGetResp struct { ResultMeta; Module *WasmModuleInfo }
type WasmRemoveResp struct { ResultMeta; Removed bool }
type WasmDescribeResp struct { ResultMeta; Module, Mode string; Handlers map[string]string }
type WasmModuleInfo struct { Name string; Size int; Exports []string; CompiledAt, SourceHash string }
```

### MCP

```go
type McpListToolsMsg struct { Server string }                  // mcp.listTools
type McpCallToolMsg struct { Server, Tool string; Args any }   // mcp.callTool

type McpListToolsResp struct { ResultMeta; Tools []McpToolInfo }
type McpCallToolResp struct { ResultMeta; Result json.RawMessage }
type McpToolInfo struct { Name, Server, Description string }
```

### Registry

```go
type RegistryHasMsg struct { Category, Name string }           // registry.has
type RegistryListMsg struct { Category string }                // registry.list
type RegistryResolveMsg struct { Category, Name string }       // registry.resolve

type RegistryHasResp struct { ResultMeta; Found bool }
type RegistryListResp struct { ResultMeta; Items json.RawMessage }
type RegistryResolveResp struct { ResultMeta; Config json.RawMessage }
```

### Plugin

```go
type PluginManifestMsg struct { Owner, Name, Version, Description string; Tools []PluginToolDef; Subscriptions, Events []string }
type PluginManifestResp struct { ResultMeta; Registered bool }
type PluginStateGetMsg struct { Key string }                   // plugin.state.get
type PluginStateSetMsg struct { Key, Value string }            // plugin.state.set
type PluginStateGetResp struct { ResultMeta; Value string }
type PluginStateSetResp struct { ResultMeta; OK bool }
type PluginToolDef struct { Name, Description, InputSchema string }
```

### Events

```go
type KitDeployedEvent struct { Source string; Resources []ResourceInfo }      // kit.deployed
type KitTeardownedEvent struct { Source string; Removed int }                  // kit.teardown.done
type PluginRegisteredEvent struct { Owner, Name, Version string; Tools int }   // plugin.registered
type CustomEvent struct { Topic string; Payload json.RawMessage }             // dynamic topic
```

## Plugin SDK

```go
func New(owner, name, version string, opts ...PluginOption) *Plugin
func WithDescription(desc string) PluginOption

func Tool[In, Out any](p *Plugin, name, description string, handler func(ctx context.Context, client Client, in In) (Out, error))
func On[E any](p *Plugin, topic string, handler func(ctx context.Context, event E, client Client))
func Event[E messages.BrainkitMessage](p *Plugin, description string)
func Intercept(p *Plugin, name string, priority int, topicFilter string, handler func(ctx context.Context, msg InterceptMessage) (*InterceptMessage, error))

func (p *Plugin) OnStart(fn func(Client) error)
func (p *Plugin) OnStop(fn func() error)
func (p *Plugin) Run() error

type Client = Runtime
```
