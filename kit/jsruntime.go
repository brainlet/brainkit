package kit

import (
	_ "embed"
	"fmt"

	quickjs "github.com/buke/quickjs-go"
)

//go:embed runtime/kit_runtime.js
var kitRuntimeJS string

//go:embed runtime/kit_module.js
var kitModuleJS string

//go:embed runtime/ai_module.js
var aiModuleJS string

//go:embed runtime/agent_module.js
var agentModuleJS string

//go:embed runtime/compiler_module.js
var compilerModuleJS string

//go:embed runtime/test_module.js
var testModuleJS string

//go:embed runtime/test_runtime.js
var testRuntimeJS string

//go:embed runtime/patches.js
var patchesJS string

//go:embed runtime/bridges.js
var bridgesJS string

//go:embed runtime/approval.js
var approvalJS string

//go:embed runtime/infrastructure.js
var infrastructureJS string

//go:embed runtime/resolve.js
var resolveJS string

//go:embed runtime/bus.js
var busJS string

// loadRuntime sets up the kit runtime:
// Loads 8 JS files in dependency order, then registers 5 ES modules.
func (k *Kernel) loadRuntime() error {
	// Load runtime files in dependency order
	runtimeFiles := []struct {
		source string
		name   string
	}{
		{patchesJS, "patches.js"},              // 1. prototype patches (before Mastra)
		{bridgesJS, "bridges.js"},              // 2. bridge helpers
		{approvalJS, "approval.js"},            // 3. HITL
		{infrastructureJS, "infrastructure.js"}, // 4. tools, fs, mcp, registry, secrets, output
		{resolveJS, "resolve.js"},              // 5. model/provider/storage/vector resolution
		{busJS, "bus.js"},                      // 6. resource registry, bus API, kit.register
		{kitRuntimeJS, "kit-runtime.js"},       // 7. export + endowments
		{testRuntimeJS, "test-runtime.js"},     // 8. test framework
	}
	for _, rf := range runtimeFiles {
		val, err := k.bridge.Eval(rf.name, rf.source)
		if err != nil {
			return fmt.Errorf("brainkit: load %s: %w", rf.name, err)
		}
		val.Free()
	}

	ctx := k.bridge.Context()

	modules := []struct {
		source string
		name   string
	}{
		{kitModuleJS, "kit"},
		{aiModuleJS, "ai"},
		{agentModuleJS, "agent"},
		{compilerModuleJS, "compiler"},
		{testModuleJS, "test"},
	}

	for _, mod := range modules {
		modVal := ctx.LoadModule(mod.source, mod.name, quickjs.EvalLoadOnly(true))
		if modVal.IsException() {
			exc := ctx.Exception()
			modVal.Free()
			return fmt.Errorf("brainkit: register %s module: %v", mod.name, exc)
		}
		modVal.Free()
	}

	return nil
}
