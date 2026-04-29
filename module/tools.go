package module

import (
	"context"
	"encoding/json"
)

// ToolExecutor abstracts a module-owned tool implementation.
type ToolExecutor interface {
	Call(context.Context, string, json.RawMessage) (json.RawMessage, error)
}

// ToolExecutorFunc adapts a function to ToolExecutor.
type ToolExecutorFunc func(context.Context, string, json.RawMessage) (json.RawMessage, error)

// Call satisfies ToolExecutor.
func (f ToolExecutorFunc) Call(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
	return f(ctx, callerID, input)
}

// ToolSpec is the module-facing description of a registered tool.
type ToolSpec struct {
	Name        string
	ShortName   string
	Owner       string
	Package     string
	Version     string
	Description string
	InputSchema json.RawMessage
	Local       bool
	Executor    ToolExecutor
}

// ToolHost registers tools owned by a module scope.
type ToolHost interface {
	Register(context.Context, ToolSpec) (Handle, error)
}

// ToolCallRequest asks the core tool registry to execute a tool.
type ToolCallRequest struct {
	Name  string
	Input any
}

// ToolCallResponse is the neutral tool execution result returned by the
// runtime capability. The bus module owns the wire envelope around this shape.
type ToolCallResponse struct {
	Result json.RawMessage
}

// ToolListRequest asks for registered tools, optionally filtered by namespace.
type ToolListRequest struct {
	Namespace string
}

// ToolListResponse lists registered tool metadata.
type ToolListResponse struct {
	Tools []ToolInfo
}

// ToolResolveRequest asks the registry to resolve one tool.
type ToolResolveRequest struct {
	Name string
}

// ToolResolveResponse describes one registered tool.
type ToolResolveResponse struct {
	Name        string
	ShortName   string
	Description string
	InputSchema any
}

// ToolInfo is the neutral registry metadata exposed to modules.
type ToolInfo struct {
	Name        string
	ShortName   string
	Namespace   string
	Description string
}

// ToolCommands is the narrow capability consumed by modules/tools.
type ToolCommands interface {
	CallTool(context.Context, ToolCallRequest) (*ToolCallResponse, error)
	ResolveTool(context.Context, ToolResolveRequest) (*ToolResolveResponse, error)
	ListTools(context.Context, ToolListRequest) (*ToolListResponse, error)
}
