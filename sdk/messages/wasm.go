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

// ── WASM Command Allowlist ──

type WasmAllowlistGetMsg struct{}

func (WasmAllowlistGetMsg) BusTopic() string { return "wasm.allowlist.get" }

type WasmAllowlistGetResp struct {
	ResultMeta
	Allowlist map[string]bool `json:"allowlist"`
}

type WasmAllowlistSetMsg struct {
	Allowlist map[string]bool `json:"allowlist"`
}

func (WasmAllowlistSetMsg) BusTopic() string { return "wasm.allowlist.set" }

type WasmAllowlistSetResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

type WasmAllowlistAddMsg struct {
	Command string `json:"command"`
}

func (WasmAllowlistAddMsg) BusTopic() string { return "wasm.allowlist.add" }

type WasmAllowlistAddResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

type WasmAllowlistRemoveMsg struct {
	Command string `json:"command"`
}

func (WasmAllowlistRemoveMsg) BusTopic() string { return "wasm.allowlist.remove" }

type WasmAllowlistRemoveResp struct {
	ResultMeta
	OK bool `json:"ok"`
}

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




type WasmUndeployResp struct {
	ResultMeta
	Undeployed bool `json:"undeployed"`
}


type WasmListResp struct {
	ResultMeta
	Modules []WasmModuleInfo `json:"modules"`
}


type WasmGetResp struct {
	ResultMeta
	Module *WasmModuleInfo `json:"module,omitempty"`
}


type WasmRemoveResp struct {
	ResultMeta
	Removed bool `json:"removed"`
}


type WasmDescribeResp struct {
	ResultMeta
	Module   string            `json:"module"`
	Mode     string            `json:"mode"`
	Handlers map[string]string `json:"handlers"`
}


// ── Shared types ──

type WasmModuleInfo struct {
	Name       string   `json:"name"`
	Size       int      `json:"size"`
	Exports    []string `json:"exports"`
	CompiledAt string   `json:"compiledAt"`
	SourceHash string   `json:"sourceHash"`
}
