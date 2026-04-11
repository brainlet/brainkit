package engine

import (
	"context"
	"encoding/json"

	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
)

// MCPDomain handles mcp.listTools and mcp.callTool bus commands.
type MCPDomain struct {
	mcp *mcppkg.MCPManager
}

func newMCPDomain(mcp *mcppkg.MCPManager) *MCPDomain {
	return &MCPDomain{mcp: mcp}
}

func (d *MCPDomain) ListTools(_ context.Context, req sdk.McpListToolsMsg) (*sdk.McpListToolsResp, error) {
	if d.mcp == nil {
		return nil, &sdkerrors.NotConfiguredError{Feature: "mcp"}
	}
	tools := d.mcp.ListTools()
	var infos []sdk.McpToolInfo
	for _, t := range tools {
		infos = append(infos, sdk.McpToolInfo{Name: t.Name, Server: t.ServerName, Description: t.Description})
	}
	return &sdk.McpListToolsResp{Tools: infos}, nil
}

func (d *MCPDomain) CallTool(ctx context.Context, req sdk.McpCallToolMsg) (*sdk.McpCallToolResp, error) {
	if d.mcp == nil {
		return nil, &sdkerrors.NotConfiguredError{Feature: "mcp"}
	}
	argsJSON, _ := json.Marshal(req.Args)
	result, err := d.mcp.CallTool(ctx, req.Server, req.Tool, argsJSON)
	if err != nil {
		return nil, err
	}
	return &sdk.McpCallToolResp{Result: result}, nil
}
