package tools

import (
	"context"
	"encoding/json"
)

// RegisteredTool is a tool in the registry.
type RegisteredTool struct {
	// Name is the canonical registered key: "owner/pkg@version/tool".
	Name        string          `json:"name"`
	ShortName   string          `json:"shortName"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`

	// Naming fields — always populated from the canonical name.
	Owner   string `json:"owner,omitempty"`   // "brainlet", "acme-corp"
	Package string `json:"package,omitempty"` // "cron", "postgres"
	Version string `json:"version,omitempty"` // "1.0.0", "2.1.0-beta.1"

	// Local marks tools that must only be called from the same runtime.
	// Plugin tools are local — they execute in a subprocess attached to this Kit.
	// Remote Kit instances cannot invoke local tools via cross-namespace calls.
	Local bool `json:"local,omitempty"`

	Executor ToolExecutor `json:"-"`
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
