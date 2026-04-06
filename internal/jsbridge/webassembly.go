package jsbridge

import (
	"context"
	"fmt"
	"github.com/brainlet/brainkit/internal/syncx"
	"sync/atomic"

	quickjs "github.com/buke/quickjs-go"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// WebAssemblyPolyfill provides WebAssembly.instantiate() backed by wazero.
// Enables JS libraries that ship WASM modules (like xxhash-wasm) to work in QuickJS.
type WebAssemblyPolyfill struct {
	bridge  *Bridge
	mu      syncx.Mutex
	rt      wazero.Runtime
	modules map[int64]api.Module
	nextID  atomic.Int64
}

// WebAssembly creates a WebAssembly polyfill.
func WebAssembly() *WebAssemblyPolyfill {
	return &WebAssemblyPolyfill{
		modules: make(map[int64]api.Module),
	}
}

func (p *WebAssemblyPolyfill) Name() string      { return "webassembly" }
func (p *WebAssemblyPolyfill) SetBridge(b *Bridge) { p.bridge = b }

func (p *WebAssemblyPolyfill) Setup(ctx *quickjs.Context) error {
	p.rt = wazero.NewRuntime(context.Background())

	polyfill := p

	// __go_wasm_instantiate(wasmBytesArrayBuffer) → JSON descriptor string
	ctx.Globals().Set("__go_wasm_instantiate", ctx.NewFunction(
		func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
			if len(args) < 1 {
				return qctx.ThrowError(fmt.Errorf("wasm_instantiate: requires wasm bytes"))
			}

			size := args[0].ByteLen()
			if size <= 0 {
				return qctx.ThrowError(fmt.Errorf("wasm_instantiate: empty buffer"))
			}
			wasmBytes, err := args[0].ToByteArray(uint(size))
			if err != nil {
				return qctx.ThrowError(fmt.Errorf("wasm_instantiate: read bytes: %w", err))
			}

			goCtx := context.Background()

			compiled, err := polyfill.rt.CompileModule(goCtx, wasmBytes)
			if err != nil {
				return qctx.ThrowError(fmt.Errorf("wasm_instantiate: compile: %w", err))
			}

			modID := polyfill.nextID.Add(1)
			modName := fmt.Sprintf("wasm_%d", modID)

			mod, err := polyfill.rt.InstantiateModule(goCtx, compiled, wazero.NewModuleConfig().WithName(modName))
			if err != nil {
				return qctx.ThrowError(fmt.Errorf("wasm_instantiate: instantiate: %w", err))
			}

			polyfill.mu.Lock()
			polyfill.modules[modID] = mod
			polyfill.mu.Unlock()

			// Register Go bridge functions for each exported function
			fnNames := make([]string, 0)
			for name, def := range mod.ExportedFunctionDefinitions() {
				fnName := name
				_ = def

				bridgeName := fmt.Sprintf("__go_wasm_fn_%d_%s", modID, fnName)
				qctx.Globals().Set(bridgeName, qctx.NewFunction(
					func(qctx *quickjs.Context, this *quickjs.Value, callArgs []*quickjs.Value) *quickjs.Value {
						polyfill.mu.Lock()
						m := polyfill.modules[modID]
						polyfill.mu.Unlock()
						if m == nil {
							return qctx.ThrowError(fmt.Errorf("wasm module %d closed", modID))
						}
						fn := m.ExportedFunction(fnName)
						if fn == nil {
							return qctx.ThrowError(fmt.Errorf("wasm function %q not found", fnName))
						}
						wasmArgs := make([]uint64, len(callArgs))
						for i, a := range callArgs {
							wasmArgs[i] = uint64(a.ToInt64())
						}
						results, callErr := fn.Call(context.Background(), wasmArgs...)
						if callErr != nil {
							return qctx.ThrowError(callErr)
						}
						if len(results) == 0 {
							return qctx.NewUndefined()
						}
						return qctx.NewInt64(int64(results[0]))
					},
				))
				fnNames = append(fnNames, fnName)
			}

			// Check for memory exports
			hasMemory := false
			for range mod.ExportedMemoryDefinitions() {
				hasMemory = true
				break
			}

			if hasMemory {
				memPrefix := fmt.Sprintf("__go_wasm_mem_%d", modID)

				// Get memory buffer as ArrayBuffer
				qctx.Globals().Set(memPrefix+"_buffer", qctx.NewFunction(
					func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
						polyfill.mu.Lock()
						m := polyfill.modules[modID]
						polyfill.mu.Unlock()
						if m == nil {
							return qctx.NewArrayBuffer(nil)
						}
						// Try "mem" first (xxhash), then "memory" (convention)
						mem := m.ExportedMemory("mem")
						if mem == nil {
							mem = m.ExportedMemory("memory")
						}
						if mem == nil {
							return qctx.NewArrayBuffer(nil)
						}
						size := mem.Size()
						data, ok := mem.Read(0, size)
						if !ok {
							return qctx.NewArrayBuffer(nil)
						}
						return qctx.NewArrayBuffer(data)
					},
				))

				// Grow memory
				qctx.Globals().Set(memPrefix+"_grow", qctx.NewFunction(
					func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
						if len(args) < 1 {
							return qctx.NewInt32(-1)
						}
						pages := uint32(args[0].ToInt32())
						polyfill.mu.Lock()
						m := polyfill.modules[modID]
						polyfill.mu.Unlock()
						if m == nil {
							return qctx.NewInt32(-1)
						}
						mem := m.ExportedMemory("mem")
						if mem == nil {
							mem = m.ExportedMemory("memory")
						}
						if mem == nil {
							return qctx.NewInt32(-1)
						}
						oldPages, ok := mem.Grow(pages)
						if !ok {
							return qctx.NewInt32(-1)
						}
						return qctx.NewInt32(int32(oldPages))
					},
				))
			}

			// Build JSON descriptor
			desc := fmt.Sprintf(`{"id":%d,"fns":[`, modID)
			for i, name := range fnNames {
				if i > 0 {
					desc += ","
				}
				desc += fmt.Sprintf(`"%s"`, name)
			}
			desc += fmt.Sprintf(`],"hasMem":%v}`, hasMemory)

			return qctx.NewString(desc)
		},
	))

	return evalJS(ctx, wasmJS)
}

// Close cleans up all wazero modules and the runtime.
func (p *WebAssemblyPolyfill) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	goCtx := context.Background()
	for _, mod := range p.modules {
		mod.Close(goCtx)
	}
	p.modules = nil
	if p.rt != nil {
		p.rt.Close(goCtx)
		p.rt = nil
	}
}

const wasmJS = `
(function() {
  globalThis.WebAssembly = globalThis.WebAssembly || {};

  WebAssembly.instantiate = function(bufferSource, importObject) {
    var bytes;
    if (bufferSource instanceof Uint8Array) {
      // Extract the underlying ArrayBuffer, accounting for offset/length
      if (bufferSource.byteOffset === 0 && bufferSource.byteLength === bufferSource.buffer.byteLength) {
        bytes = bufferSource.buffer;
      } else {
        // Create a copy of just this view's portion
        var copy = new Uint8Array(bufferSource.byteLength);
        for (var i = 0; i < bufferSource.byteLength; i++) copy[i] = bufferSource[i];
        bytes = copy.buffer;
      }
    } else if (bufferSource instanceof ArrayBuffer) {
      bytes = bufferSource;
    } else {
      return Promise.reject(new Error("WebAssembly.instantiate: invalid buffer source"));
    }

    try {
      var descJSON = __go_wasm_instantiate(bytes);
      var desc = JSON.parse(descJSON);
      var modId = desc.id;

      var exports = {};

      // Wrap each exported function
      for (var i = 0; i < desc.fns.length; i++) {
        (function(fnName) {
          var bridgeFn = globalThis["__go_wasm_fn_" + modId + "_" + fnName];
          exports[fnName] = function() {
            return bridgeFn.apply(null, arguments);
          };
        })(desc.fns[i]);
      }

      // Wrap memory export
      if (desc.hasMem) {
        var memBufferFn = globalThis["__go_wasm_mem_" + modId + "_buffer"];
        var memGrowFn = globalThis["__go_wasm_mem_" + modId + "_grow"];
        exports.mem = {
          get buffer() {
            return memBufferFn();
          },
          grow: function(pages) {
            return memGrowFn(pages);
          },
        };
        // Also expose as "memory" (common convention)
        exports.memory = exports.mem;
      }

      return Promise.resolve({
        module: {},
        instance: { exports: exports },
      });
    } catch (e) {
      return Promise.reject(e);
    }
  };

  WebAssembly.compile = function(bufferSource) {
    return WebAssembly.instantiate(bufferSource).then(function(r) { return r.module; });
  };

  WebAssembly.validate = function() { return true; };
})();
`
