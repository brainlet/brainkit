package messages

// ── Plugin Manifest ──

type PluginManifestMsg struct {
	Owner          string              `json:"owner"`
	Name           string              `json:"name"`
	Version        string              `json:"version"`
	Description    string              `json:"description,omitempty"`
	Tools          []PluginToolDef     `json:"tools,omitempty"`
	Subscriptions  []string            `json:"subscriptions,omitempty"`
	Events         []string            `json:"events,omitempty"`
	HostFunctions  []PluginHostFuncDef `json:"host_functions,omitempty"`
}

// PluginHostFuncDef declares a host function a plugin provides to WASM workflows.
type PluginHostFuncDef struct {
	Module      string           `json:"module"`      // wazero module name: "telegram", "db"
	Name        string           `json:"name"`        // function name: "send", "query"
	Description string           `json:"description"`
	Params      []HostFuncParam  `json:"params"`
	Returns     string           `json:"returns"`     // "string", "i32", "i64", "void"
	ToolTopic   string           `json:"tool_topic"`  // bus topic to route calls to
}

type HostFuncParam struct {
	Name string `json:"name"`
	Type string `json:"type"` // "string", "i32", "i64", "f64"
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


// ── Plugin State ──

type PluginStateGetMsg struct {
	Key string `json:"key"`
}

func (PluginStateGetMsg) BusTopic() string { return "plugin.state.get" }

type PluginStateGetResp struct {
	ResultMeta
	Value string `json:"value"`
}


type PluginStateSetMsg struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (PluginStateSetMsg) BusTopic() string { return "plugin.state.set" }

type PluginStateSetResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

