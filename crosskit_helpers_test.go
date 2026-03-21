package brainkit

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit/registry"
)

func newSharedTools() *registry.ToolRegistry {
	return registry.New()
}

func newTestGoTool(name, description string, fn func(map[string]any) (any, error)) registry.RegisteredTool {
	rt := registry.RegisteredTool{
		Name:        name,
		Description: description,
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var m map[string]any
				if len(input) > 0 {
					json.Unmarshal(input, &m)
				}
				result, err := fn(m)
				if err != nil {
					return nil, err
				}
				out, _ := json.Marshal(result)
				return out, nil
			},
		},
	}
	if registry.IsNewFormat(name) {
		owner, pkg, version, short := registry.ParseToolName(name)
		rt.Owner = owner
		rt.Package = pkg
		rt.Version = version
		rt.ShortName = short
	} else {
		rt.ShortName = name
	}
	return rt
}
