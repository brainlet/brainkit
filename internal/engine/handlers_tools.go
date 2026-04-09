package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// ToolsDomain handles tool registry operations: call, resolve, register, list.
type ToolsDomain struct {
	tools     *tools.ToolRegistry
	eval      JSEvaluator
	tracer    *tracing.Tracer
	callerID  string
	runtimeID string // local runtime ID — used to reject remote calls to local-only tools
}

func newToolsDomain(tools *tools.ToolRegistry, eval JSEvaluator, tracer *tracing.Tracer, callerID, runtimeID string) *ToolsDomain {
	return &ToolsDomain{tools: tools, eval: eval, tracer: tracer, callerID: callerID, runtimeID: runtimeID}
}

// Call executes a registered tool by name and returns the typed response.
func (d *ToolsDomain) Call(ctx context.Context, req sdk.ToolCallMsg) (*sdk.ToolCallResp, error) {
	tool, err := d.tools.Resolve(req.Name)
	if err != nil {
		return nil, err
	}

	// Reject remote calls to local-only tools (plugin tools).
	// Local tools are bound to this Kit's subprocess — remote Kit instances must not invoke them.
	if tool.Local && d.runtimeID != "" {
		callerRuntimeID := transport.RuntimeIDFromContext(ctx)
		if callerRuntimeID != "" && callerRuntimeID != d.runtimeID {
			return nil, &sdkerrors.PermissionDeniedError{
				Source: callerRuntimeID,
				Action: "call",
				Topic:  tool.Name,
				Role:   "remote",
			}
		}
	}

	span := d.tracer.StartSpan("tools.call:"+tool.ShortName, ctx)
	span.SetAttribute("tool", tool.Name)

	inputJSON, _ := json.Marshal(req.Input)
	result, err := tool.Executor.Call(ctx, d.callerID, inputJSON)
	span.End(err)
	if err != nil {
		return nil, err
	}
	// nil result + nil error = pass-through (plugin responds directly to caller).
	// Return nil so the command handler skips publishing a response.
	if result == nil {
		return nil, nil
	}
	return &sdk.ToolCallResp{Result: result}, nil
}

// Resolve looks up a tool by name and returns its registration info.
func (d *ToolsDomain) Resolve(_ context.Context, req sdk.ToolResolveMsg) (*sdk.ToolResolveResp, error) {
	tool, err := d.tools.Resolve(req.Name)
	if err != nil {
		return nil, err
	}
	resp := &sdk.ToolResolveResp{
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
	if tools.IsNewFormat(name) {
		fullName = name
		_, _, _, shortName = tools.ParseToolName(name)
	} else {
		fullName = tools.ComposeName(callerID, callerID, "0.0.0", name)
	}

	if err := d.tools.Register(tools.RegisteredTool{
		Name:        fullName,
		ShortName:   shortName,
		Description: description,
		InputSchema: inputSchema,
		Executor: &tools.GoFuncExecutor{
			Fn: func(ctx context.Context, _ string, input json.RawMessage) (json.RawMessage, error) {
				rawInput := strings.TrimSpace(string(input))
				if rawInput == "" {
					rawInput = "null"
				}
				argsJSON, _ := json.Marshal(map[string]any{"name": shortName, "input": json.RawMessage(rawInput)})
				script := fmt.Sprintf(`(async () => { return JSON.stringify(await __brainkit.tools.execute(JSON.parse(%q))); })()`, string(argsJSON))
				out, err := d.eval.EvalOnJSThread("__dispatch_tool__.js", script)
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
func (d *ToolsDomain) List(_ context.Context, req sdk.ToolListMsg) (*sdk.ToolListResp, error) {
	toolList := d.tools.List(req.Namespace)
	var infos []sdk.ToolInfo
	for _, t := range toolList {
		infos = append(infos, sdk.ToolInfo{
			Name:        t.Name,
			ShortName:   t.ShortName,
			Description: t.Description,
		})
	}
	if infos == nil {
		infos = []sdk.ToolInfo{}
	}
	return &sdk.ToolListResp{Tools: infos}, nil
}
