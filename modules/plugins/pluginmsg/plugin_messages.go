package pluginmsg

import "encoding/json"

// -- Plugin Lifecycle --

type PluginStartMsg struct {
	Name   string            `json:"name"`
	Binary string            `json:"binary,omitempty"`
	Env    map[string]string `json:"env,omitempty"`
	Config json.RawMessage   `json:"config,omitempty"`
}

func (PluginStartMsg) BusTopic() string { return "plugin.start" }

type PluginStartResp struct {
	Started bool   `json:"started"`
	Name    string `json:"name"`
	PID     int    `json:"pid"`
}

type PluginStopMsg struct {
	Name string `json:"name"`
}

func (PluginStopMsg) BusTopic() string { return "plugin.stop" }

type PluginStopResp struct {
	Stopped bool `json:"stopped"`
}

type PluginRestartMsg struct {
	Name string `json:"name"`
}

func (PluginRestartMsg) BusTopic() string { return "plugin.restart" }

type PluginRestartResp struct {
	Restarted bool `json:"restarted"`
	PID       int  `json:"pid"`
}

type PluginListRunningMsg struct{}

func (PluginListRunningMsg) BusTopic() string { return "plugin.list" }

type PluginListRunningResp struct {
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
	Name     string   `json:"name"`
	PID      int      `json:"pid"`
	Status   string   `json:"status"`
	Uptime   string   `json:"uptime"`
	Restarts int      `json:"restarts"`
	Tools    []string `json:"tools,omitempty"`
}

// -- Plugin Manifest --

type PluginManifestMsg struct {
	Owner         string              `json:"owner"`
	Name          string              `json:"name"`
	Version       string              `json:"version"`
	Description   string              `json:"description,omitempty"`
	Tools         []PluginToolDef     `json:"tools,omitempty"`
	Subscriptions []string            `json:"subscriptions,omitempty"`
	Events        []string            `json:"events,omitempty"`
	HostFunctions []PluginHostFuncDef `json:"host_functions,omitempty"`
}

func (PluginManifestMsg) BusTopic() string { return "plugin.manifest" }

type PluginManifestResp struct {
	Registered bool `json:"registered"`
}

type PluginToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema string `json:"inputSchema,omitempty"`
}

// PluginHostFuncDef declares a host function a plugin provides.
type PluginHostFuncDef struct {
	Module      string          `json:"module"` // wazero module name: "telegram", "db"
	Name        string          `json:"name"`   // function name: "send", "query"
	Description string          `json:"description"`
	Params      []HostFuncParam `json:"params"`
	Returns     string          `json:"returns"`    // "string", "i32", "i64", "void"
	ToolTopic   string          `json:"tool_topic"` // bus topic to route calls to
}

type HostFuncParam struct {
	Name string `json:"name"`
	Type string `json:"type"` // "string", "i32", "i64", "f64"
}

// -- Plugin Events --

type PluginRegisteredEvent struct {
	Owner   string `json:"owner"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Tools   int    `json:"tools"`
}

func (PluginRegisteredEvent) BusTopic() string { return "plugin.registered" }

// PluginStartedEvent is emitted when a plugin is started dynamically.
type PluginStartedEvent struct {
	Name    string `json:"name"`
	PID     int    `json:"pid"`
	Version string `json:"version,omitempty"`
}

func (PluginStartedEvent) BusTopic() string { return "plugin.started" }

// PluginStoppedEvent is emitted when a plugin is stopped.
type PluginStoppedEvent struct {
	Name   string `json:"name"`
	Reason string `json:"reason,omitempty"`
}

func (PluginStoppedEvent) BusTopic() string { return "plugin.stopped" }
