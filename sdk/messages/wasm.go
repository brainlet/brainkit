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
	ResultMeta
	ModuleID string   `json:"moduleId"`
	Name     string   `json:"name"`
	Size     int      `json:"size"`
	Exports  []string `json:"exports"`
	Text     string   `json:"text,omitempty"`
}

type WasmRunResp struct {
	ResultMeta
	ExitCode int `json:"exitCode"`
	Value    any `json:"value,omitempty"`
}

type WasmDeployResp struct {
	ResultMeta
	Module   string            `json:"module"`
	Mode     string            `json:"mode"` // "stateless" | "persistent"
	Handlers map[string]string `json:"handlers"`
}

func (WasmCompileResp) BusTopic() string { return "wasm.compile.result" }

func (WasmRunResp) BusTopic() string { return "wasm.run.result" }

func (WasmDeployResp) BusTopic() string { return "wasm.deploy.result" }

type WasmUndeployResp struct {
	ResultMeta
	Undeployed bool `json:"undeployed"`
}

func (WasmUndeployResp) BusTopic() string { return "wasm.undeploy.result" }

type WasmListResp struct {
	ResultMeta
	Modules []WasmModuleInfo `json:"modules"`
}

func (WasmListResp) BusTopic() string { return "wasm.list.result" }

type WasmGetResp struct {
	ResultMeta
	Module *WasmModuleInfo `json:"module,omitempty"`
}

func (WasmGetResp) BusTopic() string { return "wasm.get.result" }

type WasmRemoveResp struct {
	ResultMeta
	Removed bool `json:"removed"`
}

func (WasmRemoveResp) BusTopic() string { return "wasm.remove.result" }

type WasmDescribeResp struct {
	ResultMeta
	Module   string            `json:"module"`
	Mode     string            `json:"mode"`
	Handlers map[string]string `json:"handlers"`
}

func (WasmDescribeResp) BusTopic() string { return "wasm.describe.result" }

// ── Shared types ──

type WasmModuleInfo struct {
	Name       string   `json:"name"`
	Size       int      `json:"size"`
	Exports    []string `json:"exports"`
	CompiledAt string   `json:"compiledAt"`
	SourceHash string   `json:"sourceHash"`
}
