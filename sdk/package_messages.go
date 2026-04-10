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

