package messages

// ── Options ──

type WasmCompileOpts struct {
	Name    string `json:"name,omitempty"`
	Runtime string `json:"runtime,omitempty"`
}

// ── Requests ──

type WasmCompileMsg struct {
	Source  string           `json:"source"`
	Options *WasmCompileOpts `json:"options,omitempty"`
}

func (WasmCompileMsg) BusTopic() string { return "wasm.compile" }

type WasmRunMsg struct {
	ModuleID string `json:"moduleId"`
	Input    any    `json:"input,omitempty"`
}

func (WasmRunMsg) BusTopic() string { return "wasm.run" }

type WasmDeployMsg struct {
	Name string `json:"name"`
}

func (WasmDeployMsg) BusTopic() string { return "wasm.deploy" }

type WasmUndeployMsg struct {
	Name string `json:"name"`
}

func (WasmUndeployMsg) BusTopic() string { return "wasm.undeploy" }

type WasmListMsg struct{}

func (WasmListMsg) BusTopic() string { return "wasm.list" }

type WasmGetMsg struct {
	Name string `json:"name"`
}

func (WasmGetMsg) BusTopic() string { return "wasm.get" }

type WasmRemoveMsg struct {
	Name string `json:"name"`
}

func (WasmRemoveMsg) BusTopic() string { return "wasm.remove" }

type WasmDescribeMsg struct {
	Name string `json:"name"`
}

func (WasmDescribeMsg) BusTopic() string { return "wasm.describe" }

// ── Responses ──

type WasmCompileResp struct {
	ModuleID string   `json:"moduleId"`
	Name     string   `json:"name"`
	Size     int      `json:"size"`
	Exports  []string `json:"exports"`
	Text     string   `json:"text,omitempty"`
}

type WasmRunResp struct {
	ExitCode int `json:"exitCode"`
	Value    any `json:"value,omitempty"`
}

type WasmDeployResp struct {
	Module   string            `json:"module"`
	Mode     string            `json:"mode"` // "stateless" | "persistent"
	Handlers map[string]string `json:"handlers"`
}

type WasmListResp struct {
	Modules []WasmModuleInfo `json:"modules"`
}

// ── Shared types ──

type WasmModuleInfo struct {
	Name       string   `json:"name"`
	Size       int      `json:"size"`
	Exports    []string `json:"exports"`
	CompiledAt string   `json:"compiledAt"`
	SourceHash string   `json:"sourceHash"`
}
