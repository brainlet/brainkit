package tools

import (
	"context"
	"encoding/json"
	"fmt"

	toolreg "github.com/brainlet/brainkit/internal/tools"
	bkmodule "github.com/brainlet/brainkit/module"
)

// TypedTool defines a Go function tool with typed JSON input.
type TypedTool[T any] = toolreg.TypedTool[T]

// GoTool returns a hot-mountable module that registers one typed Go tool for
// the lifetime of its module scope.
func GoTool[T any](name string, tool TypedTool[T]) bkmodule.Module {
	return &goToolModule{spec: goToolSpec(name, tool)}
}

type goToolModule struct {
	spec bkmodule.ToolSpec
}

func (m *goToolModule) ID() string {
	return "go-tool:" + m.spec.Name
}

func (m *goToolModule) Mount(ctx context.Context, host bkmodule.Host) error {
	_, err := host.Tools().Register(ctx, m.spec)
	return err
}

func (m *goToolModule) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:      m.ID(),
		Status:    bkmodule.StatusStable,
		Summary:   "Typed Go tool registration.",
		Resources: []bkmodule.ResourceDescriptor{bkmodule.ToolResource(m.spec)},
	}
}

func goToolSpec[T any](name string, tool TypedTool[T]) bkmodule.ToolSpec {
	var zero T
	spec := bkmodule.ToolSpec{
		Name:        name,
		Description: tool.Description,
		InputSchema: toolreg.StructToJSONSchema(zero),
		Executor: bkmodule.ToolExecutorFunc(func(ctx context.Context, _ string, input json.RawMessage) (json.RawMessage, error) {
			var typed T
			if err := json.Unmarshal(input, &typed); err != nil {
				return nil, fmt.Errorf("tool %s: unmarshal input: %w", name, err)
			}
			result, err := tool.Execute(ctx, typed)
			if err != nil {
				return nil, err
			}
			return json.Marshal(result)
		}),
	}

	if toolreg.IsNewFormat(name) {
		owner, pkg, version, short := toolreg.ParseToolName(name)
		spec.Owner = owner
		spec.Package = pkg
		spec.Version = version
		spec.ShortName = short
	} else {
		spec.ShortName = name
	}
	return spec
}
