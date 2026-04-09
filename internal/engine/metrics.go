package engine

import (
	"time"

	"github.com/brainlet/brainkit/internal/types"
)


// Metrics returns a point-in-time snapshot of internal Kernel state.
func (k *Kernel) Metrics() types.KernelMetrics {
	m := types.KernelMetrics{
		ActiveHandlers:    k.activeHandlers.Load(),
		ActiveDeployments: len(k.ListDeployments()),
		ActiveSchedules:   len(k.ListSchedules()),
		PumpCycles:        k.pumpCycles.Load(),
	}

	if !k.startedAt.IsZero() {
		m.Uptime = time.Since(k.startedAt)
	}

	// Plugin metrics from Node's WS server
	if k.node != nil && k.node.plugins != nil && k.node.plugins.wsServer != nil {
		k.node.plugins.wsServer.mu.Lock()
		m.ActivePlugins = len(k.node.plugins.wsServer.conns)
		for _, pc := range k.node.plugins.wsServer.conns {
			pc.mu.Lock()
			pm := types.PluginMetrics{
				Name:       pc.name,
				Healthy:    pc.healthy,
				ToolCalls:  pc.toolCalls,
				ToolErrors: pc.toolErrors,
				LastPong:   pc.lastPong,
			}
			pc.mu.Unlock()
			m.Plugins = append(m.Plugins, pm)
		}
		k.node.plugins.wsServer.mu.Unlock()
	}

	return m
}
