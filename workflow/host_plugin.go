package workflow

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// registerPluginHostFunctions registers all plugin-provided host functions
// into the wazero runtime for the current workflow execution.
//
// Each plugin host function uses a string-in/string-out pattern:
// args are JSON-serialized into a single AS string, result returned as AS string.
// The AS type declarations (from codegen.go) provide typed wrappers at compile time.
//
// Call routing: workflow calls host function → check journal for replay →
// if not replayed, call plugin via bus (PluginCaller) → record in journal.
func (e *Engine) registerPluginHostFunctions(ctx context.Context, rt wazero.Runtime, ar *activeRun) {
	if e.hostRegistry == nil {
		return
	}

	modules := e.hostRegistry.ListModules()
	for _, modName := range modules {
		// Skip built-in modules — they have dedicated registrations
		if modName == "brainkit" || modName == "ai" || modName == "env" {
			continue
		}

		funcs := e.hostRegistry.ListFunctions(modName)
		if len(funcs) == 0 {
			continue
		}

		builder := rt.NewHostModuleBuilder(modName)
		for _, hf := range funcs {
			hf := hf // capture for closure
			builder = builder.NewFunctionBuilder().
				WithFunc(func(ctx context.Context, m api.Module, argsPtr uint32) uint32 {
					return e.callPluginHostFunc(ctx, m, ar, hf, argsPtr)
				}).Export(hf.Name)
		}

		if _, err := builder.Instantiate(ctx); err != nil {
			log.Printf("[workflow] warning: failed to register plugin module %q: %v", modName, err)
		}
	}
}

// callPluginHostFunc handles a single plugin host function call from WASM.
// 1. Deserialize args from WASM string pointer
// 2. Check journal for replay (return recorded result if replaying)
// 3. Call plugin via bus using PluginCaller
// 4. Record the call in journal
// 5. Return result as WASM string pointer
func (e *Engine) callPluginHostFunc(ctx context.Context, m api.Module, ar *activeRun, hf *HostFunctionDef, argsPtr uint32) uint32 {
	argsStr := readASString(m, argsPtr)
	argsJSON := json.RawMessage(argsStr)

	// Check journal for replay
	if result, ok := ar.journal.GetRecordedResult(hf.Module, hf.Name, argsJSON); ok {
		ptr, _ := writeASString(ctx, m, string(result))
		return ptr
	}

	// No plugin caller configured — can't route
	if e.pluginCaller == nil {
		log.Printf("[workflow] plugin host function %s.%s called but no PluginCaller configured", hf.Module, hf.Name)
		ar.journal.RecordCall(hf.Module, hf.Name, argsJSON, nil, errNoPluginCaller, 0)
		ptr, _ := writeASString(ctx, m, "")
		return ptr
	}

	// Call plugin via bus
	topic := hf.PluginTopic
	if topic == "" {
		log.Printf("[workflow] plugin host function %s.%s has no PluginTopic", hf.Module, hf.Name)
		ptr, _ := writeASString(ctx, m, "")
		return ptr
	}

	// Step-level timeout: 30s per host function call
	callCtx, callCancel := context.WithTimeout(ctx, 30*time.Second)
	defer callCancel()

	start := time.Now()
	result, err := e.pluginCaller(callCtx, topic, argsJSON)
	duration := time.Since(start)

	// Record in journal
	ar.journal.RecordCall(hf.Module, hf.Name, argsJSON, result, err, duration)

	if err != nil {
		log.Printf("[workflow] plugin %s.%s failed: %v", hf.Module, hf.Name, err)
		ptr, _ := writeASString(ctx, m, "")
		return ptr
	}

	ptr, _ := writeASString(ctx, m, string(result))
	return ptr
}

type pluginCallerError string

func (e pluginCallerError) Error() string { return string(e) }

var errNoPluginCaller = pluginCallerError("no PluginCaller configured on workflow engine")
