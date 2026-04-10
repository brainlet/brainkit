package sdk

import "encoding/json"

// ── Plugin Lifecycle ──

type PluginStartMsg struct {
	Name   string            `json:"name"`
	Binary string            `json:"binary,omitempty"`
	Env    map[string]string `json:"env,omitempty"`
	Config json.RawMessage   `json:"config,omitempty"`
}

func (PluginStartMsg) BusTopic() string { return "plugin.start" }

type PluginStartResp struct {
	ResultMeta
	Started bool   `json:"started"`
	Name    string `json:"name"`
	PID     int    `json:"pid"`
}

type PluginStopMsg struct {
	Name string `json:"name"`
}

func (PluginStopMsg) BusTopic() string { return "plugin.stop" }

type PluginStopResp struct {
	ResultMeta
	Stopped bool `json:"stopped"`
}

type PluginRestartMsg struct {
	Name string `json:"name"`
}

func (PluginRestartMsg) BusTopic() string { return "plugin.restart" }

type PluginRestartResp struct {
	ResultMeta
	Restarted bool `json:"restarted"`
	PID       int  `json:"pid"`
}

type PluginListRunningMsg struct{}

func (PluginListRunningMsg) BusTopic() string { return "plugin.list" }

type PluginListRunningResp struct {
	ResultMeta
	Plugins []RunningPluginInfo `json:"plugins"`
}

type RunningPluginInfo struct {
	Name     string `json:"name"`
	PID      int    `json:"pid"`
	Uptime   string `json:"uptime"`
	Status   string `json:"status"`
	Restarts int    `json:"restarts"`
}

type PluginStatusMsg struct {
	Name string `json:"name"`
}

func (PluginStatusMsg) BusTopic() string { return "plugin.status" }

type PluginStatusResp struct {
	ResultMeta
	Name     string   `json:"name"`
	PID      int      `json:"pid"`
	Status   string   `json:"status"`
	Uptime   string   `json:"uptime"`
	Restarts int      `json:"restarts"`
	Tools    []string `json:"tools,omitempty"`
}

// ── Package Manager ──

type PackagesSearchMsg struct {
	Query        string   `json:"query,omitempty"`
	Capabilities []string `json:"capabilities,omitempty"`
}

func (PackagesSearchMsg) BusTopic() string { return "packages.search" }

type PackagesSearchResp struct {
	ResultMeta
	Plugins []PluginSummary `json:"plugins"`
}

type PluginSummary struct {
	Name         string   `json:"name"`
	Owner        string   `json:"owner"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities,omitempty"`
}

type PackagesInstallMsg struct {
	Name    string `json:"name"`    // "brainlet/telegram-gateway"
	Version string `json:"version"` // "1.2.0" or "" for latest
}

func (PackagesInstallMsg) BusTopic() string { return "packages.install" }

type PackagesInstallResp struct {
	ResultMeta
	Installed bool   `json:"installed"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Path      string `json:"path"`
}

type PackagesRemoveMsg struct {
	Name string `json:"name"`
}

func (PackagesRemoveMsg) BusTopic() string { return "packages.remove" }

type PackagesRemoveResp struct {
	ResultMeta
	Removed bool `json:"removed"`
}

type PackagesUpdateMsg struct {
	Name string `json:"name"`
}

func (PackagesUpdateMsg) BusTopic() string { return "packages.update" }

type PackagesUpdateResp struct {
	ResultMeta
	Updated    bool   `json:"updated"`
	OldVersion string `json:"oldVersion"`
	NewVersion string `json:"newVersion"`
}

type PackagesListMsg struct{}

func (PackagesListMsg) BusTopic() string { return "packages.list" }

type PackagesListResp struct {
	ResultMeta
	Plugins []InstalledPluginInfo `json:"plugins"`
}

type InstalledPluginInfo struct {
	Name        string `json:"name"`
	Owner       string `json:"owner"`
	Version     string `json:"version"`
	BinaryPath  string `json:"binaryPath"`
	InstalledAt string `json:"installedAt"`
	Running     bool   `json:"running"`
}

type PackagesInfoMsg struct {
	Name string `json:"name"`
}

func (PackagesInfoMsg) BusTopic() string { return "packages.info" }

type PackagesInfoResp struct {
	ResultMeta
	Manifest json.RawMessage `json:"manifest"`
}
