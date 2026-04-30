package toolhost

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	auditpkg "github.com/brainlet/brainkit/internal/audit"
	toolreg "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/internal/transport"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// JSEvaluator runs JavaScript on the bridge.
type JSEvaluator interface {
	EvalOnJSThread(filename, code string) (string, error)
}

// Domain handles host-side tool registry operations: call, resolve, register,
// unregister, and list. The modules/tools package is only the bus command
// adapter for this host capability.
type Domain struct {
	tools     *toolreg.ToolRegistry
	eval      JSEvaluator
	tracer    *tracing.Tracer
	audit     *auditpkg.Recorder
	callerID  string
	runtimeID string // local runtime ID; used to reject remote calls to local-only tools
}

func NewDomain(tools *toolreg.ToolRegistry, eval JSEvaluator, tracer *tracing.Tracer, audit *auditpkg.Recorder, callerID, runtimeID string) *Domain {
	return &Domain{tools: tools, eval: eval, tracer: tracer, audit: audit, callerID: callerID, runtimeID: runtimeID}
}

func (d *Domain) SetEvaluator(eval JSEvaluator) {
	d.eval = eval
}

func (d *Domain) CallTool(ctx context.Context, req bkmodule.ToolCallRequest) (*bkmodule.ToolCallResponse, error) {
	return d.Call(ctx, req)
}

func (d *Domain) ResolveTool(ctx context.Context, req bkmodule.ToolResolveRequest) (*bkmodule.ToolResolveResponse, error) {
	return d.Resolve(ctx, req)
}

func (d *Domain) ListTools(ctx context.Context, req bkmodule.ToolListRequest) (*bkmodule.ToolListResponse, error) {
	return d.List(ctx, req)
}

// Call executes a registered tool by name and returns the typed response.
func (d *Domain) Call(ctx context.Context, req bkmodule.ToolCallRequest) (*bkmodule.ToolCallResponse, error) {
	tool, err := d.tools.Resolve(req.Name)
	if err != nil {
		return nil, err
	}

	// Reject remote calls to local-only tools (plugin tools).
	if tool.Local && d.runtimeID != "" {
		callerRuntimeID := transport.RuntimeIDFromContext(ctx)
		fromTransport := transport.TopicFromContext(ctx) != ""
		if (callerRuntimeID != "" && callerRuntimeID != d.runtimeID) || (callerRuntimeID == "" && fromTransport) {
			d.audit.ToolCallDenied(tool.Name, callerRuntimeID, "local-only tool called from remote runtime")
			if callerRuntimeID == "" {
				callerRuntimeID = "unknown"
			}
			return nil, &sdkerrors.ValidationError{
				Field:   "runtimeId",
				Message: fmt.Sprintf("tool %q is local-only and cannot be called from remote runtime %s", tool.Name, callerRuntimeID),
			}
		}
	}

	callStart := time.Now()
	span := d.tracer.StartSpan("tools.call:"+tool.ShortName, ctx)
	span.SetAttribute("tool", tool.Name)

	inputJSON, _ := json.Marshal(req.Input)
	result, err := tool.Executor.Call(ctx, d.callerID, inputJSON)
	span.End(err)
	callDuration := time.Since(callStart)

	if err != nil {
		d.audit.ToolCallFailed(tool.Name, d.callerID, callDuration, err)
		return nil, err
	}
	// nil result + nil error = pass-through (plugin responds directly to caller).
	if result == nil {
		return nil, nil
	}
	d.audit.ToolCallCompleted(tool.Name, d.callerID, callDuration)
	return &bkmodule.ToolCallResponse{Result: result}, nil
}

// Resolve looks up a tool by name and returns its registration info.
func (d *Domain) Resolve(_ context.Context, req bkmodule.ToolResolveRequest) (*bkmodule.ToolResolveResponse, error) {
	tool, err := d.tools.Resolve(req.Name)
	if err != nil {
		return nil, err
	}
	resp := &bkmodule.ToolResolveResponse{
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
func (d *Domain) Register(_ context.Context, name, description string, inputSchema json.RawMessage, callerID string) (string, error) {
	if d.eval == nil {
		return "", &sdkerrors.NotConfiguredError{Feature: "js runtime"}
	}
	var fullName string
	shortName := name
	if toolreg.IsNewFormat(name) {
		fullName = name
		_, _, _, shortName = toolreg.ParseToolName(name)
	} else {
		fullName = toolreg.ComposeName(callerID, callerID, "0.0.0", name)
	}

	if err := d.tools.Register(toolreg.RegisteredTool{
		Name:        fullName,
		ShortName:   shortName,
		Description: description,
		InputSchema: inputSchema,
		Executor: &toolreg.GoFuncExecutor{
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

func (d *Domain) Unregister(_ context.Context, name string) error {
	tool, err := d.tools.Resolve(name)
	if err != nil {
		return err
	}
	d.tools.Unregister(tool.Name)
	return nil
}

// List returns all registered tools, optionally filtered.
func (d *Domain) List(_ context.Context, req bkmodule.ToolListRequest) (*bkmodule.ToolListResponse, error) {
	toolList := d.tools.List(req.Namespace)
	var infos []bkmodule.ToolInfo
	for _, t := range toolList {
		infos = append(infos, bkmodule.ToolInfo{
			Name:        t.Name,
			ShortName:   t.ShortName,
			Description: t.Description,
		})
	}
	if infos == nil {
		infos = []bkmodule.ToolInfo{}
	}
	return &bkmodule.ToolListResponse{Tools: infos}, nil
}
