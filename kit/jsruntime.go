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

// loadRuntime sets up the kit runtime:
// 1. Evaluates kit_runtime.js (sets globalThis.__kit)
// 2. Registers five ES modules: "kit", "ai", "agent", "compiler", "test"
func (k *Kernel) loadRuntime() error {
	val, err := k.bridge.Eval("kit-runtime.js", kitRuntimeJS)
	if err != nil {
		return fmt.Errorf("brainkit: load runtime: %w", err)
	}
	val.Free()

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
