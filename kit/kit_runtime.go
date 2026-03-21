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

// loadRuntime sets up the kit runtime:
// 1. Evaluates kit_runtime.js (sets globalThis.__kit)
// 2. Registers "kit" as an ES module (enables import { ... } from "kit")
func (k *Kit) loadRuntime() error {
	val, err := k.bridge.Eval("kit-runtime.js", kitRuntimeJS)
	if err != nil {
		return fmt.Errorf("brainkit: load runtime: %w", err)
	}
	val.Free()

	ctx := k.bridge.Context()
	modVal := ctx.LoadModule(kitModuleJS, "kit", quickjs.EvalLoadOnly(true))
	if modVal.IsException() {
		exc := ctx.Exception()
		modVal.Free()
		return fmt.Errorf("brainkit: register kit module: %v", exc)
	}
	modVal.Free()

	return nil
}
