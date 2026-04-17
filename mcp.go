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
//
// Internally this wraps an engine-scoped module (Init(*Kernel)) so MCP can
// reach into kernel tool registry internals. The Kit-scoped Module contract
// is satisfied by a passthrough adapter — engine runs the real init, the
// adapter is a no-op.
func NewMCPModule(servers map[string]MCPServerConfig) Module {
	return &mcpModuleAdapter{inner: engine.NewMCPModule(servers)}
}

// mcpModuleAdapter bridges the legacy engine-scoped MCPModule to the public
// brainkit.Module contract. The engine iterates cfg.Modules separately and
// runs the inner module directly; the adapter's Init / Close are no-ops so
// the module isn't double-invoked. Once modules/mcp lands, delete this.
type mcpModuleAdapter struct {
	inner *engine.MCPModule
}

func (a *mcpModuleAdapter) Name() string     { return a.inner.Name() }
func (a *mcpModuleAdapter) Init(*Kit) error  { return nil }
func (a *mcpModuleAdapter) Close() error     { return a.inner.Close() }

// unwrapEngineModule lets brainkit.New pick the inner engine.Module out so
// it can still flow through engine.NewKernel's legacy init path.
func (a *mcpModuleAdapter) unwrapEngineModule() any { return a.inner }
