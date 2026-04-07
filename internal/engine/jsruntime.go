package engine

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

//go:embed runtime/dispatch.js
var dispatchJS string

// Type definitions — embedded for CLI scaffolding (brainkit new module)
//go:embed runtime/kit.d.ts
var KitDTS string

//go:embed runtime/ai.d.ts
var AiDTS string

//go:embed runtime/agent.d.ts
var AgentDTS string

//go:embed runtime/brainkit.d.ts
var BrainkitDTS string

//go:embed runtime/globals.d.ts
var GlobalsDTS string

const fsPromisesModuleJS = `export const readFile = globalThis.fs.promises.readFile;
export const writeFile = globalThis.fs.promises.writeFile;
export const appendFile = globalThis.fs.promises.appendFile;
export const readdir = globalThis.fs.promises.readdir;
export const stat = globalThis.fs.promises.stat;
export const lstat = globalThis.fs.promises.lstat;
export const access = globalThis.fs.promises.access;
export const mkdir = globalThis.fs.promises.mkdir;
export const mkdtemp = globalThis.fs.promises.mkdtemp;
export const rmdir = globalThis.fs.promises.rmdir;
export const rm = globalThis.fs.promises.rm;
export const unlink = globalThis.fs.promises.unlink;
export const rename = globalThis.fs.promises.rename;
export const copyFile = globalThis.fs.promises.copyFile;
export const cp = globalThis.fs.promises.cp;
export const link = globalThis.fs.promises.link;
export const symlink = globalThis.fs.promises.symlink;
export const readlink = globalThis.fs.promises.readlink;
export const realpath = globalThis.fs.promises.realpath;
export const chmod = globalThis.fs.promises.chmod;
export const chown = globalThis.fs.promises.chown;
export const lchown = globalThis.fs.promises.lchown;
export const truncate = globalThis.fs.promises.truncate;
export const utimes = globalThis.fs.promises.utimes;
export const open = globalThis.fs.promises.open;
export const watch = globalThis.fs.promises.watch;
export default globalThis.fs.promises;
`

// loadRuntime sets up the kit runtime:
// Loads 9 JS files in dependency order, then registers 6 ES modules.
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
		{dispatchJS, "dispatch.js"},            // 9. Go-callable dispatch functions
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
		{testModuleJS, "test"},
		// fs module — re-exports from globalThis.fs (set by jsbridge/fs.go polyfill)
		{"export default globalThis.fs;", "fs"},
		{fsPromisesModuleJS, "fs/promises"},
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
