package messages

// ── Plugin Manifest ──

type PluginManifestMsg struct {
	Owner         string          `json:"owner"`
	Name          string          `json:"name"`
	Version       string          `json:"version"`
	Description   string          `json:"description,omitempty"`
	Tools         []PluginToolDef `json:"tools,omitempty"`
	Subscriptions []string        `json:"subscriptions,omitempty"`
	Events        []string        `json:"events,omitempty"`
}

func (PluginManifestMsg) BusTopic() string { return "plugin.manifest" }

type PluginToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema string `json:"inputSchema,omitempty"`
}

type PluginManifestResp struct {
	ResultMeta
	Registered bool `json:"registered"`
}

func (PluginManifestResp) BusTopic() string { return "plugin.manifest.result" }

// ── Plugin State ──

type PluginStateGetMsg struct {
	Key string `json:"key"`
}

func (PluginStateGetMsg) BusTopic() string { return "plugin.state.get" }

type PluginStateGetResp struct {
	ResultMeta
	Value string `json:"value"`
}

func (PluginStateGetResp) BusTopic() string { return "plugin.state.get.result" }

type PluginStateSetMsg struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (PluginStateSetMsg) BusTopic() string { return "plugin.state.set" }

type PluginStateSetResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

func (PluginStateSetResp) BusTopic() string { return "plugin.state.set.result" }
