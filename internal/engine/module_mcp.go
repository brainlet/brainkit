package engine

import (
	"context"
	"encoding/json"

	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	toolreg "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// MCPModule manages MCP server connections as a Kernel module.
type MCPModule struct {
	servers map[string]types.MCPServerConfig
	manager *mcppkg.MCPManager
	domain  *MCPDomain
}

// NewMCPModule creates an MCP module that connects to the given servers.
func NewMCPModule(servers map[string]types.MCPServerConfig) *MCPModule {
	return &MCPModule{servers: servers}
}

func (m *MCPModule) Name() string { return "mcp" }

func (m *MCPModule) Init(k *Kernel) error {
	if len(m.servers) == 0 {
		return nil
	}

	m.manager = mcppkg.New()
	m.domain = newMCPDomain(m.manager)

	for name, cfg := range m.servers {
		if err := m.manager.Connect(context.Background(), name, cfg); err != nil {
			types.InvokeErrorHandler(k.config.ErrorHandler, &sdkerrors.TransportError{
				Operation: "MCP.Connect:" + name, Cause: err,
			}, types.ErrorContext{Operation: "ConnectMCP", Component: "mcp", Source: name})
			continue
		}
		for _, tool := range m.manager.ListToolsForServer(name) {
			toolCopy := tool
			fullName := toolreg.ComposeName("mcp", toolCopy.ServerName, "1.0.0", toolCopy.Name)
			_ = k.Tools.Register(toolreg.RegisteredTool{
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

	// Register bus commands
	k.RegisterCommand(kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.McpListToolsMsg) (*sdk.McpListToolsResp, error) {
		return m.domain.ListTools(ctx, req)
	}))
	k.RegisterCommand(kernelCommand(func(ctx context.Context, kernel *Kernel, req sdk.McpCallToolMsg) (*sdk.McpCallToolResp, error) {
		return m.domain.CallTool(ctx, req)
	}))

	return nil
}

func (m *MCPModule) Close() error {
	if m.manager != nil {
		return m.manager.Close()
	}
	return nil
}
