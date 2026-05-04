// Package tools owns tools.* registry commands as a hot-mountable Kit module.
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
)

// Module exposes tools.call/resolve/list. Construct via New and include in
// brainkit.Config.Modules when the runtime should expose the tool registry over
// the bus.
type Module struct {
	commands bkmodule.ToolCommands
}

// New creates the tools module.
func New() *Module { return &Module{} }

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "tools" }

// Status reports maturity.
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusStable }

// Mount registers tools.* command handlers against the running Kit.
func (m *Module) Mount(_ context.Context, host bkmodule.Host) error {
	commands, err := bkmodule.RequireCapability[bkmodule.ToolCommands](host, bkmodule.CapabilityToolCommands)
	if err != nil {
		return fmt.Errorf("tools: %w", err)
	}
	m.commands = commands
	host.Scope().Defer(func(context.Context) error {
		m.commands = nil
		return nil
	})

	for _, spec := range []bkmodule.CommandSpec{
		m.callCommand(),
		bkmodule.Command(m.Resolve),
		bkmodule.Command(m.List),
	} {
		if _, err := host.Commands().Handle(spec); err != nil {
			return err
		}
	}
	return nil
}

// Close detaches the module from Kit capabilities. Command handles are owned by
// the module scope, so unmounting unregisters them.
func (m *Module) Close() error {
	m.commands = nil
	return nil
}

// Factory is the registered ModuleFactory for tools.
type Factory struct{}

// YAML is reserved for future module options.
type YAML struct{}

// Build decodes YAML and returns the tools module.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	return New(), nil
}

// Describe surfaces module metadata for module manifests.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:    "tools",
		Status:  bkmodule.StatusStable,
		Summary: "Tool registry bus commands (tools.call, resolve, list).",
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](),
			bkmodule.CommandMessage[toolmsg.ToolListMsg, toolmsg.ToolListResp](),
			bkmodule.CommandMessage[toolmsg.ToolResolveMsg, toolmsg.ToolResolveResp](),
		},
		Capabilities: []bkmodule.CapabilityDescriptor{
			bkmodule.RequiredCapabilityOf[bkmodule.ToolCommands](bkmodule.CapabilityToolCommands),
		},
	}
}

func init() { bkmodule.Register("tools", Factory{}) }

func (m *Module) callCommand() bkmodule.CommandSpec {
	var zero toolmsg.ToolCallMsg
	topic := zero.BusTopic()
	desc := bkmodule.CommandMessage[toolmsg.ToolCallMsg, toolmsg.ToolCallResp]()
	return bkmodule.CommandSpec{
		Name:     topic,
		Topic:    topic,
		Request:  desc.Request,
		Response: desc.Response,
		Handle: func(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
			var req toolmsg.ToolCallMsg
			if len(payload) > 0 {
				if err := json.Unmarshal(payload, &req); err != nil {
					return nil, fmt.Errorf("decode %s: %w", topic, err)
				}
			}
			resp, err := m.Call(ctx, req)
			if err != nil {
				return nil, err
			}
			if resp == nil {
				return nil, nil
			}
			return json.Marshal(resp)
		},
	}
}

// Call handles tools.call.
func (m *Module) Call(ctx context.Context, req toolmsg.ToolCallMsg) (*toolmsg.ToolCallResp, error) {
	resp, err := m.commands.CallTool(ctx, bkmodule.ToolCallRequest{Name: req.Name, Input: req.Input})
	if err != nil || resp == nil {
		return nil, err
	}
	return &toolmsg.ToolCallResp{Result: resp.Result}, nil
}

// Resolve handles tools.resolve.
func (m *Module) Resolve(ctx context.Context, req toolmsg.ToolResolveMsg) (*toolmsg.ToolResolveResp, error) {
	resp, err := m.commands.ResolveTool(ctx, bkmodule.ToolResolveRequest{Name: req.Name})
	if err != nil || resp == nil {
		return nil, err
	}
	return &toolmsg.ToolResolveResp{
		Name:        resp.Name,
		ShortName:   resp.ShortName,
		Description: resp.Description,
		InputSchema: resp.InputSchema,
	}, nil
}

// List handles tools.list.
func (m *Module) List(ctx context.Context, req toolmsg.ToolListMsg) (*toolmsg.ToolListResp, error) {
	resp, err := m.commands.ListTools(ctx, bkmodule.ToolListRequest{Namespace: req.Namespace})
	if err != nil || resp == nil {
		return nil, err
	}
	infos := make([]toolmsg.ToolInfo, 0, len(resp.Tools))
	for _, info := range resp.Tools {
		infos = append(infos, toolmsg.ToolInfo{
			Name:        info.Name,
			ShortName:   info.ShortName,
			Namespace:   info.Namespace,
			Description: info.Description,
		})
	}
	return &toolmsg.ToolListResp{Tools: infos}, nil
}
