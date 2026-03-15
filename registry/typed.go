package registry

import (
	"context"
	"encoding/json"
	"fmt"
)

// TypedTool defines a tool with a typed Go struct for input.
// The JSON Schema is generated automatically from the Input struct's fields and tags.
//
// Example:
//
//	registry.TypedTool[MyInput]{
//	    Description: "Does something useful",
//	    Execute: func(ctx context.Context, input MyInput) (any, error) {
//	        return map[string]any{"result": input.A + input.B}, nil
//	    },
//	}
type TypedTool[T any] struct {
	Description string
	Execute     func(ctx context.Context, input T) (any, error)
}

// Register registers a typed tool on a ToolRegistry.
// The name should be fully qualified: "namespace.toolName" (e.g., "platform.math.add").
// JSON Schema is generated from the Input type parameter's struct tags.
func Register[T any](r *ToolRegistry, name string, tool TypedTool[T]) error {
	var zero T
	schema := StructToJSONSchema(zero)
	ns, short := ParseNamespace(name)

	return r.Register(RegisteredTool{
		Name:        name,
		ShortName:   short,
		Namespace:   ns,
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
	})
}
