package types

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

// RunningPlugin describes a running plugin process.
type RunningPlugin struct {
	Name     string
	PID      int
	Uptime   time.Duration
	Status   string // "running"
	Restarts int
	Config   PluginConfig
}

// InstalledPlugin is the on-disk format for an installed plugin binary.
type InstalledPlugin struct {
	Name        string    `json:"name"`
	Owner       string    `json:"owner"`
	Version     string    `json:"version"`
	BinaryPath  string    `json:"binaryPath"`
	Manifest    string    `json:"manifest"` // raw JSON
	InstalledAt time.Time `json:"installedAt"`
}

// RunningPluginRecord is the on-disk format for a running plugin (for restart recovery).
type RunningPluginRecord struct {
	Name       string            `json:"name"`
	Owner      string            `json:"owner"`
	Version    string            `json:"version"`
	BinaryPath string            `json:"binaryPath"`
	Env        map[string]string `json:"env"`
	Config     json.RawMessage   `json:"config,omitempty"`
	StartOrder int               `json:"startOrder"`
	StartedAt  time.Time         `json:"startedAt"`
	Role       string            `json:"role,omitempty"`
}
