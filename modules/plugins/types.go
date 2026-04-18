package plugins

import (
	"github.com/brainlet/brainkit/internal/types"
)

// Store is the narrow persistence surface the plugins module needs.
// brainkit's KitStore satisfies it structurally, so the common case is
// `Config{Store: kitStore}`.
//
// When nil: running-plugin state is not persisted and plugins installed
// via plugin.start aren't restored on restart.
type Store interface {
	LoadRunningPlugins() ([]types.RunningPluginRecord, error)
	SaveRunningPlugin(r types.RunningPluginRecord) error
	DeleteRunningPlugin(name string) error
	LoadInstalledPlugins() ([]types.InstalledPlugin, error)
}

// Config configures the plugins module.
type Config struct {
	// Plugins is the static list of subprocess plugins to start on Init.
	// May be nil; plugins can also be added at runtime via the plugin.start
	// bus command.
	Plugins []types.PluginConfig

	// Store is optional. When provided, dynamically-started plugins are
	// persisted and restored on module Init. The narrow Store interface is
	// satisfied by brainkit.KitStore.
	Store Store
}
