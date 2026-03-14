package registry

import (
	"context"
	"encoding/json"
)

// RegisteredTool is a tool in the registry.
type RegisteredTool struct {
	Name        string          // full namespaced name: "plugin.postgres@1.0.0.db_query"
	ShortName   string          // just the tool name: "db_query"
	Description string
	InputSchema json.RawMessage // JSON Schema
	Owner       string          // CallerID of who registered it
	Namespace   string          // "platform", "plugin.postgres@1.0.0", "user", "agent.coder-1"
	Executor    ToolExecutor
}

// ToolExecutor abstracts over different execution backends.
type ToolExecutor interface {
	Call(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error)
}

// GoFuncExecutor wraps a Go function as a ToolExecutor.
type GoFuncExecutor struct {
	Fn func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error)
}

func (e *GoFuncExecutor) Call(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
	return e.Fn(ctx, callerID, input)
}
