package registry

import (
	"context"
	"encoding/json"
	"fmt"
)

// TypedTool defines a tool with a typed Go struct for input.
type TypedTool[T any] struct {
	Description string
	Execute     func(ctx context.Context, input T) (any, error)
}

// Register registers a typed tool on a ToolRegistry.
// The name should be new-format ("brainlet/cron@1.0.0/create") or a bare short name.
// Fields are auto-populated from the name.
func Register[T any](r *ToolRegistry, name string, tool TypedTool[T]) error {
	var zero T
	schema := StructToJSONSchema(zero)

	rt := RegisteredTool{
		Name:        name,
		Description: tool.Description,
		InputSchema: schema,
		Executor: &GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var typed T
				if err := json.Unmarshal(input, &typed); err != nil {
					return nil, fmt.Errorf("tool %s: unmarshal input: %w", name, err)
				}
				result, err := tool.Execute(ctx, typed)
				if err != nil {
					return nil, err
				}
				return json.Marshal(result)
			},
		},
	}

	if IsNewFormat(name) {
		owner, pkg, version, short := ParseToolName(name)
		rt.Owner = owner
		rt.Package = pkg
		rt.Version = version
		rt.ShortName = short
	} else {
		// Bare short name — no owner/pkg/version.
		rt.ShortName = name
	}

	return r.Register(rt)
}
