package kit

import (
	"context"
	"encoding/json"

	mcppkg "github.com/brainlet/brainkit/internal/mcp"
	"github.com/brainlet/brainkit/sdk/messages"
)

// MCPDomain handles MCP server operations.
type MCPDomain struct {
	kit *Kernel
	mgr *mcppkg.MCPManager
}

func newMCPDomain(k *Kernel, mgr *mcppkg.MCPManager) *MCPDomain {
	return &MCPDomain{kit: k, mgr: mgr}
}

func (d *MCPDomain) ListTools(_ context.Context, req messages.McpListToolsMsg) (*messages.McpListToolsResp, error) {
	if d.mgr == nil {
		return nil, ErrMCPNotConfigured
	}
	tools := d.mgr.ListTools()
	var infos []messages.McpToolInfo
	for _, t := range tools {
		infos = append(infos, messages.McpToolInfo{
			Name:        t.Name,
			Server:      t.ServerName,
			Description: t.Description,
		})
	}
	return &messages.McpListToolsResp{Tools: infos}, nil
}

func (d *MCPDomain) CallTool(ctx context.Context, req messages.McpCallToolMsg) (*messages.McpCallToolResp, error) {
	if d.mgr == nil {
		return nil, ErrMCPNotConfigured
	}
	argsJSON, _ := json.Marshal(req.Args)
	result, err := d.mgr.CallTool(ctx, req.Server, req.Tool, argsJSON)
	if err != nil {
		return nil, err
	}
	return &messages.McpCallToolResp{Result: result}, nil
}
