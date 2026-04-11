package brainkit

import "github.com/brainlet/brainkit/internal/engine"

// NewMCPModule creates a Module that connects to external MCP tool servers.
// Pass this to Config.Modules to enable MCP support.
//
//	kit, _ := brainkit.New(brainkit.Config{
//	    Modules: []brainkit.Module{
//	        brainkit.NewMCPModule(map[string]brainkit.MCPServerConfig{...}),
//	    },
//	})
func NewMCPModule(servers map[string]MCPServerConfig) Module {
	return engine.NewMCPModule(servers)
}
