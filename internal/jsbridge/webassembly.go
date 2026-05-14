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

func (p *WebAssemblyPolyfill) Name() string        { return "webassembly" }
func (p *WebAssemblyPolyfill) SetBridge(b *Bridge) { p.bridge = b }

func (p *WebAssemblyPolyfill) debugResources() map[string]int {
	p.mu.Lock()
	defer p.mu.Unlock()
	runtimeCount := 0
	if p.rt != nil {
		runtimeCount = 1
	}
	return map[string]int{
		"wasm.runtimes": runtimeCount,
		"wasm.modules":  len(p.modules),
	}
}

func (p *WebAssemblyPolyfill) Setup(ctx *quickjs.Context) error {
	p.rt = wazero.NewRuntime(context.Background())
	if p.bridge != nil {
		p.bridge.RegisterResourceProvider(p.debugResources)
		p.bridge.Go(func(goCtx context.Context) {
			<-goCtx.Done()
			p.Close()
		})
	}

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
				paramTypes := append([]api.ValueType(nil), def.ParamTypes()...)
				resultTypes := append([]api.ValueType(nil), def.ResultTypes()...)

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
							var typ api.ValueType
							if i < len(paramTypes) {
								typ = paramTypes[i]
							}
							switch typ {
							case api.ValueTypeI64:
								if a.IsBigInt() {
									wasmArgs[i] = a.ToBigInt().Uint64()
								} else {
									wasmArgs[i] = uint64(a.ToInt64())
								}
							case api.ValueTypeF32:
								wasmArgs[i] = api.EncodeF32(float32(a.ToFloat64()))
							case api.ValueTypeF64:
								wasmArgs[i] = api.EncodeF64(a.ToFloat64())
							default:
								wasmArgs[i] = uint64(uint32(a.ToInt32()))
							}
						}
						results, callErr := fn.Call(context.Background(), wasmArgs...)
						if callErr != nil {
							return qctx.ThrowError(callErr)
						}
						if len(results) == 0 {
							return qctx.NewUndefined()
						}
						var resultType api.ValueType
						if len(resultTypes) > 0 {
							resultType = resultTypes[0]
						}
						switch resultType {
						case api.ValueTypeI64:
							return qctx.NewBigUint64(results[0])
						case api.ValueTypeF32:
							return qctx.NewFloat64(float64(api.DecodeF32(results[0])))
						case api.ValueTypeF64:
							return qctx.NewFloat64(api.DecodeF64(results[0]))
						default:
							return qctx.NewInt32(int32(results[0]))
						}
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

				// Copy a JavaScript ArrayBuffer back into wazero memory. QuickJS
				// cannot expose wazero memory as a shared backing store, so the
				// JS wrapper syncs its current memory snapshot before each wasm
				// function call and refreshes it after the call.
				qctx.Globals().Set(memPrefix+"_write", qctx.NewFunction(
					func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
						if len(args) < 1 {
							return qctx.NewUndefined()
						}
						size := args[0].ByteLen()
						if size <= 0 {
							return qctx.NewUndefined()
						}
						data, err := args[0].ToByteArray(uint(size))
						if err != nil {
							return qctx.ThrowError(fmt.Errorf("wasm memory write: read bytes: %w", err))
						}
						polyfill.mu.Lock()
						m := polyfill.modules[modID]
						polyfill.mu.Unlock()
						if m == nil {
							return qctx.NewUndefined()
						}
						mem := m.ExportedMemory("mem")
						if mem == nil {
							mem = m.ExportedMemory("memory")
						}
						if mem == nil {
							return qctx.NewUndefined()
						}
						memSize := int(mem.Size())
						if len(data) > memSize {
							data = data[:memSize]
						}
						if !mem.Write(0, data) {
							return qctx.ThrowError(fmt.Errorf("wasm memory write: failed"))
						}
						return qctx.NewUndefined()
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
      var memoryExport = null;
      var memBufferFn = null;
      var memGrowFn = null;
      var memWriteFn = null;

      function syncMemoryToWasm() {
        if (memoryExport && memoryExport._buffer && memWriteFn) {
          memWriteFn(memoryExport._buffer);
        }
      }

      function syncMemoryFromWasm() {
        if (!memoryExport || !memBufferFn) return;
        var next = memBufferFn();
        if (!memoryExport._buffer || memoryExport._buffer.byteLength !== next.byteLength) {
          memoryExport._buffer = next;
          return;
        }
        new Uint8Array(memoryExport._buffer).set(new Uint8Array(next));
      }

      // Wrap each exported function
      for (var i = 0; i < desc.fns.length; i++) {
        (function(fnName) {
          var bridgeFn = globalThis["__go_wasm_fn_" + modId + "_" + fnName];
          exports[fnName] = function() {
            syncMemoryToWasm();
            var result = bridgeFn.apply(null, arguments);
            syncMemoryFromWasm();
            return result;
          };
        })(desc.fns[i]);
      }

      // Wrap memory export
      if (desc.hasMem) {
        memBufferFn = globalThis["__go_wasm_mem_" + modId + "_buffer"];
        memGrowFn = globalThis["__go_wasm_mem_" + modId + "_grow"];
        memWriteFn = globalThis["__go_wasm_mem_" + modId + "_write"];
        memoryExport = {
          _buffer: memBufferFn(),
          get buffer() {
            return this._buffer;
          },
          grow: function(pages) {
            syncMemoryToWasm();
            var oldPages = memGrowFn(pages);
            this._buffer = memBufferFn();
            return oldPages;
          },
        };
        exports.mem = memoryExport;
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
