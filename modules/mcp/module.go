// Package mcp connects a brainkit Kit to external Model Context Protocol
// servers and exposes their tools as Kit-registered tools. It also registers
// mcp.listTools / mcp.callTool bus commands for direct server-side calls.
package mcp

import (
	"context"
	"encoding/json"

	toolreg "github.com/brainlet/brainkit/internal/tools"
	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/modules/mcp/mcpmsg"
	_ "github.com/brainlet/brainkit/modules/tools"

	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// Module is a Kit-scoped MCP module. Construct via New and include in
// brainkit.Config.Modules.
type Module struct {
	servers map[string]ServerConfig
	manager *MCPManager
}

// New creates an MCP module that will connect to the given servers at
// module mount time.
func New(servers map[string]ServerConfig) *Module {
	return &Module{servers: servers}
}

// ID reports the hot-mount module identifier.
func (m *Module) ID() string { return "mcp" }

// Dependencies reports modules that must mount before mcp. MCP-discovered tools
// are registered in the shared tool registry and need the tools.* bus surface
// for normal tool invocation.
func (m *Module) Dependencies() []string { return []string{"tools"} }

// Status reports maturity (stable).
func (m *Module) Status() bkmodule.Status { return bkmodule.StatusStable }

// Mount connects configured MCP servers and scopes the discovered tools plus
// mcp.* commands to the module lifetime.
func (m *Module) Mount(ctx context.Context, host bkmodule.Host) error {
	if len(m.servers) == 0 {
		return nil
	}

	m.manager = NewManager()
	host.Scope().Defer(func(context.Context) error {
		if m.manager != nil {
			return m.manager.Close()
		}
		return nil
	})

	for name, cfg := range m.servers {
		if err := m.manager.Connect(ctx, name, cfg); err != nil {
			host.Logger().Warn("mcp connect failed", "server", name, "error", err)
			continue
		}
		for _, tool := range m.manager.ListToolsForServer(name) {
			toolCopy := tool
			fullName := toolreg.ComposeName("mcp", toolCopy.ServerName, "1.0.0", toolCopy.Name)
			if _, err := host.Tools().Register(ctx, bkmodule.ToolSpec{
				Name:        fullName,
				ShortName:   toolCopy.Name,
				Owner:       "mcp",
				Package:     toolCopy.ServerName,
				Version:     "1.0.0",
				Description: toolCopy.Description,
				InputSchema: toolCopy.InputSchema,
				Executor: bkmodule.ToolExecutorFunc(func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
					return m.manager.CallTool(ctx, toolCopy.ServerName, toolCopy.Name, input)
				}),
			}); err != nil {
				return err
			}
		}
	}

	if _, err := host.Commands().Handle(bkmodule.Command(func(ctx context.Context, req mcpmsg.McpListToolsMsg) (*mcpmsg.McpListToolsResp, error) {
		return m.listTools(ctx, req)
	})); err != nil {
		return err
	}
	if _, err := host.Commands().Handle(bkmodule.Command(func(ctx context.Context, req mcpmsg.McpCallToolMsg) (*mcpmsg.McpCallToolResp, error) {
		return m.callTool(ctx, req)
	})); err != nil {
		return err
	}

	return nil
}

// Close disconnects from every MCP server.
func (m *Module) Close() error {
	if m.manager != nil {
		return m.manager.Close()
	}
	return nil
}

func (m *Module) listTools(_ context.Context, _ mcpmsg.McpListToolsMsg) (*mcpmsg.McpListToolsResp, error) {
	if m.manager == nil {
		return nil, &sdkerrors.NotConfiguredError{Feature: "mcp"}
	}
	tools := m.manager.ListTools()
	var infos []mcpmsg.McpToolInfo
	for _, t := range tools {
		infos = append(infos, mcpmsg.McpToolInfo{Name: t.Name, Server: t.ServerName, Description: t.Description})
	}
	return &mcpmsg.McpListToolsResp{Tools: infos}, nil
}

func (m *Module) callTool(ctx context.Context, req mcpmsg.McpCallToolMsg) (*mcpmsg.McpCallToolResp, error) {
	if m.manager == nil {
		return nil, &sdkerrors.NotConfiguredError{Feature: "mcp"}
	}
	argsJSON, _ := json.Marshal(req.Args)
	result, err := m.manager.CallTool(ctx, req.Server, req.Tool, argsJSON)
	if err != nil {
		return nil, err
	}
	return &mcpmsg.McpCallToolResp{Result: result}, nil
}

// ServerYAML is one entry in the YAML `servers:` map. Exactly one of
// Command or URL must be set (subprocess vs. remote HTTP).
type ServerYAML struct {
	Command string            `yaml:"command"`
	Args    []string          `yaml:"args"`
	Env     map[string]string `yaml:"env"`
	URL     string            `yaml:"url"`
}

// YAML is the config shape decoded by the registry factory.
type YAML struct {
	Servers map[string]ServerYAML `yaml:"servers"`
}

// Factory is the registered ModuleFactory for mcp.
type Factory struct{}

// Build decodes YAML and returns an MCP module that will connect to
// every listed server during Mount.
func (Factory) Build(ctx bkmodule.BuildContext) (bkmodule.Module, error) {
	var y YAML
	if err := ctx.Decode(&y); err != nil {
		return nil, err
	}
	servers := make(map[string]ServerConfig, len(y.Servers))
	for name, s := range y.Servers {
		servers[name] = ServerConfig{
			Command: s.Command,
			Args:    s.Args,
			Env:     s.Env,
			URL:     s.URL,
		}
	}
	return New(servers), nil
}

// Describe surfaces module metadata for `brainkit modules list`.
func (Factory) Describe() bkmodule.Descriptor {
	return bkmodule.Descriptor{
		Name:     "mcp",
		Status:   bkmodule.StatusStable,
		Summary:  "Model Context Protocol client: discovers + proxies external tools.",
		Requires: []string{"tools"},
		Commands: []bkmodule.MessageDescriptor{
			bkmodule.CommandMessage[mcpmsg.McpCallToolMsg, mcpmsg.McpCallToolResp](),
			bkmodule.CommandMessage[mcpmsg.McpListToolsMsg, mcpmsg.McpListToolsResp](),
		},
	}
}

func init() { bkmodule.Register("mcp", Factory{}) }
