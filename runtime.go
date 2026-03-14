package brainkit

import (
	_ "embed"
	"fmt"

	quickjs "github.com/buke/quickjs-go"
)

//go:embed runtime/brainlet_runtime.js
var brainletRuntimeJS string

//go:embed runtime/brainlet_module.js
var brainletModuleJS string

// loadRuntime sets up the brainlet runtime in the Kit:
// 1. Evaluates brainlet-runtime.js (sets globalThis.__brainlet)
// 2. Registers "brainlet" as an ES module (enables import { ... } from "brainlet")
func (k *Kit) loadRuntime() error {
	val, err := k.bridge.Eval("brainlet-runtime.js", brainletRuntimeJS)
	if err != nil {
		return fmt.Errorf("brainkit: load runtime: %w", err)
	}
	val.Free()

	ctx := k.bridge.Context()
	modVal := ctx.LoadModule(brainletModuleJS, "brainlet", quickjs.EvalLoadOnly(true))
	if modVal.IsException() {
		exc := ctx.Exception()
		modVal.Free()
		return fmt.Errorf("brainkit: register brainlet module: %v", exc)
	}
	modVal.Free()

	return nil
}
