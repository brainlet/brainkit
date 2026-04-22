// Package mcp connects a brainkit Kit to external Model Context Protocol
// servers and exposes their tools as Kit-registered tools. It also registers
// mcp.listTools / mcp.callTool bus commands for direct server-side calls.
package mcp

import (
	"context"
	"encoding/json"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/types"
	toolreg "github.com/brainlet/brainkit/internal/tools"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// Module is a Kit-scoped MCP module. Construct via New and include in
// brainkit.Config.Modules.
type Module struct {
	servers map[string]ServerConfig
	manager *MCPManager
}

// New creates an MCP module that will connect to the given servers at
// Kit.Init time.
func New(servers map[string]ServerConfig) *Module {
	return &Module{servers: servers}
}

// Name reports the module identifier.
func (m *Module) Name() string { return "mcp" }

// Status reports maturity (stable).
func (m *Module) Status() brainkit.ModuleStatus { return brainkit.ModuleStatusStable }

// Init connects to every configured MCP server, registers discovered tools
// with the Kit's tool registry, and wires the mcp.listTools / mcp.callTool
// bus commands. Individual server connect failures are reported through the
// Kit's error handler but do not fail Init — other servers still initialize.
func (m *Module) Init(k *brainkit.Kit) error {
	if len(m.servers) == 0 {
		return nil
	}

	m.manager = NewManager()

	for name, cfg := range m.servers {
		if err := m.manager.Connect(context.Background(), name, cfg); err != nil {
			k.ReportError(&sdkerrors.TransportError{
				Operation: "MCP.Connect:" + name, Cause: err,
			}, types.ErrorContext{Operation: "ConnectMCP", Component: "mcp", Source: name})
			continue
		}
		for _, tool := range m.manager.ListToolsForServer(name) {
			toolCopy := tool
			fullName := toolreg.ComposeName("mcp", toolCopy.ServerName, "1.0.0", toolCopy.Name)
			_ = k.RegisterRawTool(toolreg.RegisteredTool{
				Name:        fullName,
				ShortName:   toolCopy.Name,
				Owner:       "mcp",
				Package:     toolCopy.ServerName,
				Version:     "1.0.0",
				Description: toolCopy.Description,
				InputSchema: toolCopy.InputSchema,
				Executor: &toolreg.GoFuncExecutor{
					Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
						return m.manager.CallTool(ctx, toolCopy.ServerName, toolCopy.Name, input)
					},
				},
			})
		}
	}

	k.RegisterCommand(brainkit.Command(func(ctx context.Context, req sdk.McpListToolsMsg) (*sdk.McpListToolsResp, error) {
		return m.listTools(ctx, req)
	}))
	k.RegisterCommand(brainkit.Command(func(ctx context.Context, req sdk.McpCallToolMsg) (*sdk.McpCallToolResp, error) {
		return m.callTool(ctx, req)
	}))

	return nil
}

// Close disconnects from every MCP server.
func (m *Module) Close() error {
	if m.manager != nil {
		return m.manager.Close()
	}
	return nil
}

func (m *Module) listTools(_ context.Context, _ sdk.McpListToolsMsg) (*sdk.McpListToolsResp, error) {
	if m.manager == nil {
		return nil, &sdkerrors.NotConfiguredError{Feature: "mcp"}
	}
	tools := m.manager.ListTools()
	var infos []sdk.McpToolInfo
	for _, t := range tools {
		infos = append(infos, sdk.McpToolInfo{Name: t.Name, Server: t.ServerName, Description: t.Description})
	}
	return &sdk.McpListToolsResp{Tools: infos}, nil
}

func (m *Module) callTool(ctx context.Context, req sdk.McpCallToolMsg) (*sdk.McpCallToolResp, error) {
	if m.manager == nil {
		return nil, &sdkerrors.NotConfiguredError{Feature: "mcp"}
	}
	argsJSON, _ := json.Marshal(req.Args)
	result, err := m.manager.CallTool(ctx, req.Server, req.Tool, argsJSON)
	if err != nil {
		return nil, err
	}
	return &sdk.McpCallToolResp{Result: result}, nil
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
// every listed server during Init.
func (Factory) Build(ctx brainkit.ModuleContext) (brainkit.Module, error) {
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
func (Factory) Describe() brainkit.ModuleDescriptor {
	return brainkit.ModuleDescriptor{
		Name:    "mcp",
		Status:  brainkit.ModuleStatusStable,
		Summary: "Model Context Protocol client: discovers + proxies external tools.",
	}
}

func init() { brainkit.RegisterModule("mcp", Factory{}) }
