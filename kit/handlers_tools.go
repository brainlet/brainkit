package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/kit/tracing"
	"github.com/brainlet/brainkit/sdk/messages"
)

// ToolsDomain handles tool registry operations: call, resolve, register, list.
type ToolsDomain struct {
	tools    *registry.ToolRegistry
	eval     JSEvaluator
	tracer   *tracing.Tracer
	callerID string
}

func newToolsDomain(tools *registry.ToolRegistry, eval JSEvaluator, tracer *tracing.Tracer, callerID string) *ToolsDomain {
	return &ToolsDomain{tools: tools, eval: eval, tracer: tracer, callerID: callerID}
}

// Call executes a registered tool by name and returns the typed response.
func (d *ToolsDomain) Call(ctx context.Context, req messages.ToolCallMsg) (*messages.ToolCallResp, error) {
	tool, err := d.tools.Resolve(req.Name)
	if err != nil {
		return nil, err
	}

	span := d.tracer.StartSpan("tools.call:"+tool.ShortName, ctx)
	span.SetAttribute("tool", tool.Name)

	inputJSON, _ := json.Marshal(req.Input)
	result, err := tool.Executor.Call(ctx, d.callerID, inputJSON)
	span.End(err)
	if err != nil {
		return nil, err
	}
	return &messages.ToolCallResp{Result: result}, nil
}

// Resolve looks up a tool by name and returns its registration info.
func (d *ToolsDomain) Resolve(_ context.Context, req messages.ToolResolveMsg) (*messages.ToolResolveResp, error) {
	tool, err := d.tools.Resolve(req.Name)
	if err != nil {
		return nil, err
	}
	resp := &messages.ToolResolveResp{
		Name:        tool.Name,
		ShortName:   tool.ShortName,
		Description: tool.Description,
	}
	if tool.InputSchema != nil {
		resp.InputSchema = string(tool.InputSchema)
	}
	return resp, nil
}

// Register adds a tool to the registry. Returns the fully qualified name.
func (d *ToolsDomain) Register(_ context.Context, name, description string, inputSchema json.RawMessage, callerID string) (string, error) {
	var fullName string
	shortName := name
	if registry.IsNewFormat(name) {
		fullName = name
		_, _, _, shortName = registry.ParseToolName(name)
	} else {
		fullName = registry.ComposeName(callerID, callerID, "0.0.0", name)
	}

	if err := d.tools.Register(registry.RegisteredTool{
		Name:        fullName,
		ShortName:   shortName,
		Description: description,
		InputSchema: inputSchema,
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, _ string, input json.RawMessage) (json.RawMessage, error) {
				rawInput := strings.TrimSpace(string(input))
				if rawInput == "" {
					rawInput = "null"
				}
				script := fmt.Sprintf(`(async () => {
					var entry = globalThis.__kit_registry.get("tool", %q);
					var execFn = entry && entry.ref && typeof entry.ref.__brainkit_execute === "function"
						? entry.ref.__brainkit_execute
						: entry && entry.ref && typeof entry.ref.execute === "function"
							? entry.ref.execute
							: null;
					if (!execFn) {
						throw new Error("tool not found in JS registry: " + %q);
					}
					var __input = JSON.parse(%q);
					var __wrapped;
					if (__input && typeof __input === "object" && !Array.isArray(__input)) {
						__wrapped = {};
						for (var key in __input) __wrapped[key] = __input[key];
						__wrapped.context = __input;
					} else {
						__wrapped = { context: __input };
					}
					var result = await execFn(__wrapped, { requestContext: null });
					return JSON.stringify(result === undefined ? null : result);
				})()`, shortName, shortName, rawInput)
				out, err := d.eval.EvalOnJSThread("__brainkit_tool_exec__.js", script)
				if err != nil {
					return nil, err
				}
				out = strings.TrimSpace(out)
				if out == "" {
					out = "null"
				}
				return json.RawMessage(out), nil
			},
		},
	}); err != nil {
		return "", err
	}
	return fullName, nil
}

func (d *ToolsDomain) Unregister(_ context.Context, name string) error {
	tool, err := d.tools.Resolve(name)
	if err != nil {
		return err
	}
	d.tools.Unregister(tool.Name)
	return nil
}

// List returns all registered tools, optionally filtered.
func (d *ToolsDomain) List(_ context.Context, req messages.ToolListMsg) (*messages.ToolListResp, error) {
	toolList := d.tools.List(req.Namespace)
	var infos []messages.ToolInfo
	for _, t := range toolList {
		infos = append(infos, messages.ToolInfo{
			Name:        t.Name,
			ShortName:   t.ShortName,
			Description: t.Description,
		})
	}
	if infos == nil {
		infos = []messages.ToolInfo{}
	}
	return &messages.ToolListResp{Tools: infos}, nil
}
