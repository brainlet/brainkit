// Package plugincap defines module-facing plugin capability contracts.
package plugincap

import (
	"context"

	"github.com/brainlet/brainkit/internal/types"
)

// Restarter is the narrow surface the plugins module exposes for
// rotation-driven plugin restarts.
type Restarter interface {
	ListRunningPlugins() []types.RunningPlugin
	RestartPlugin(ctx context.Context, name string) error
}
