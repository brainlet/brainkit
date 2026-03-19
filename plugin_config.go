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
	HealthInterval  time.Duration     // health check interval (default: 30s)
	StartTimeout    time.Duration     // max time to wait for LISTEN line (default: 10s)
	ShutdownTimeout time.Duration     // max time to wait for graceful stop (default: 5s)
	MaxPending      int               // max pending events before backpressure drop (default: 1000)
}

// NetworkConfig configures Kit-to-Kit networking.
type NetworkConfig struct {
	Listen    string            // ":9090" — listen for incoming connections
	Peers     map[string]string // name → address: {"server-2": "10.0.1.5:9090"}
	Discovery DiscoveryConfig   // optional discovery configuration
}

func pluginDefaults(cfg *PluginConfig) {
	if cfg.MaxRestarts == 0 {
		cfg.MaxRestarts = 5
	}
	if cfg.HealthInterval == 0 {
		cfg.HealthInterval = 30 * time.Second
	}
	if cfg.StartTimeout == 0 {
		cfg.StartTimeout = 10 * time.Second
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 5 * time.Second
	}
	if cfg.MaxPending == 0 {
		cfg.MaxPending = 1000
	}
}
