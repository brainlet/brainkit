package brainkit

import (
	"encoding/json"
	"time"
)

// PluginConfig configures a single plugin.
type PluginConfig struct {
	Name            string            // human-readable name
	Binary          string            // path to plugin binary
	Args            []string          // command-line arguments
	Env             map[string]string // environment variables
	Config          json.RawMessage   // plugin-specific config (passed via PLUGIN_CONFIG env)
	AutoRestart     bool              // restart on crash (default: true)
	MaxRestarts     int               // max restarts before giving up (default: 5)
	StartTimeout    time.Duration     // max time to wait for READY line (default: 10s)
	ShutdownTimeout time.Duration     // max time to wait for graceful stop (default: 5s)
	Role            string            // RBAC role (default: "service")
}

func pluginDefaults(cfg *PluginConfig) {
	if cfg.MaxRestarts == 0 {
		cfg.MaxRestarts = 5
	}
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = 10 * time.Second
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 5 * time.Second
	}
}
