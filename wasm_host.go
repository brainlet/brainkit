package brainkit

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"unicode/utf16"

	"github.com/brainlet/brainkit/bus"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// hostState holds shared state for one WASM execution.
// Created fresh for each wasm.run() call.
type hostState struct {
	kit        *Kit
	module     *WASMModule
	lastResult string            // buffer for host→WASM result passing
	state      map[string]string // per-execution key-value state
	logs       []string          // captured log messages (for testing)

	// Shard registration (populated during init phase only)
	initPhase     bool
	shardMode     string
	shardStateKey string
	shardHandlers map[string]string // topic → exported function name
}

func newHostState(kit *Kit, module *WASMModule) *hostState {
	return &hostState{
		kit:           kit,
		module:        module,
		state:         make(map[string]string),
		shardHandlers: make(map[string]string),
	}
}

// readASString reads an AssemblyScript string from WASM linear memory.
// AS strings are UTF-16LE encoded. The object header stores rtSize (byte length)
// at offset -4 from the pointer. The pointer points to the start of the UTF-16 payload.
// See: https://www.assemblyscript.org/runtime.html#memory-layout
func readASString(m api.Module, ptr uint32) string {
	if ptr == 0 {
		return ""
	}
	mem := m.Memory()
	if mem == nil {
		return ""
	}

	// rtSize is at offset -4 from the pointer (payload byte length)
	rtSize, ok := mem.ReadUint32Le(ptr - 4)
	if !ok || rtSize == 0 {
		return ""
	}

	data, ok := mem.Read(ptr, rtSize)
	if !ok {
		return ""
	}

	return decodeUTF16LE(data)
}

// decodeUTF16LE converts UTF-16 little-endian bytes to a Go string.
func decodeUTF16LE(b []byte) string {
	if len(b) < 2 {
		return ""
	}
	u16s := make([]uint16, len(b)/2)
	for i := range u16s {
		u16s[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	return string(utf16.Decode(u16s))
}

// writeASString allocates an AS string in WASM memory and returns its pointer.
// Requires the module to export __new(size, classId) — all AS runtimes provide this.
// String class ID in AS is 2. Data is written as UTF-16LE.
func writeASString(ctx context.Context, m api.Module, s string) (uint32, error) {
	newFn := m.ExportedFunction("__new")
	if newFn == nil {
		return 0, fmt.Errorf("module does not export __new (compile with --exportRuntime)")
	}

	// Encode Go string to UTF-16LE
	u16s := utf16.Encode([]rune(s))
	byteLen := len(u16s) * 2

	// Allocate: __new(size, classId=2 for String)
	results, err := newFn.Call(ctx, uint64(byteLen), 2)
	if err != nil {
		return 0, fmt.Errorf("__new failed: %w", err)
	}
	ptr := uint32(results[0])

	// Write UTF-16LE data at the pointer
	data := make([]byte, byteLen)
	for i, c := range u16s {
		binary.LittleEndian.PutUint16(data[i*2:], c)
	}
	m.Memory().Write(ptr, data)

	return ptr, nil
}

// registerHostFunctions registers the "host" and "env" modules with wazero.
// Must be called BEFORE instantiating the WASM module.
//
// Host functions use AS's native string passing: the WASM module passes a pointer
// to a managed String object, and the host reads it using the AS object header layout
// (rtSize at offset -4, UTF-16LE payload at the pointer).
//
// For returning strings, the host uses __new to allocate a String in WASM memory
// and returns the pointer. The WASM module reads it as a normal AS string.
func (hs *hostState) registerHostFunctions(ctx context.Context, rt wazero.Runtime) error {
	// Register "env" module — AS runtime imports abort() from here.
	_, err := rt.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, msgPtr, filePtr, line, col uint32) {
			msg := readASString(m, msgPtr)
			file := readASString(m, filePtr)
			log.Printf("[wasm:abort] %s at %s:%d:%d", msg, file, line, col)
		}).Export("abort").
		Instantiate(ctx)
	if err != nil {
		return fmt.Errorf("register env module: %w", err)
	}

	// Register "host" module — Kit host functions callable from WASM.
	// AS passes strings as pointers. Host reads via readASString.
	_, err = rt.NewHostModuleBuilder("host").

		// log(msg: string, level: i32)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, msgPtr, level uint32) {
			msg := readASString(m, msgPtr)
			hs.logs = append(hs.logs, msg)
			switch level {
			case 0:
				log.Printf("[wasm:debug] %s", msg)
			case 1:
				log.Printf("[wasm:info] %s", msg)
			case 2:
				log.Printf("[wasm:warn] %s", msg)
			case 3:
				log.Printf("[wasm:error] %s", msg)
			default:
				log.Printf("[wasm] %s", msg)
			}
		}).Export("log").

		// call_tool(name: string, argsJSON: string) → string
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, namePtr, argsPtr uint32) uint32 {
			name := readASString(m, namePtr)
			argsJSON := readASString(m, argsPtr)

			resp, err := bus.AskSync(hs.kit.Bus, ctx, bus.Message{
				Topic:    "tools.call",
				CallerID: hs.kit.callerID,
				Payload:  json.RawMessage(fmt.Sprintf(`{"name":%q,"input":%s}`, name, argsJSON)),
			})
			if err != nil {
				hs.lastResult = fmt.Sprintf(`{"error":%q}`, err.Error())
			} else {
				hs.lastResult = string(resp.Payload)
			}

			// Return result as AS string pointer
			ptr, werr := writeASString(ctx, m, hs.lastResult)
			if werr != nil {
				return 0
			}
			return ptr
		}).Export("call_tool").

		// call_agent(name: string, prompt: string) → string (JSON: {"text":"..."} or {"error":"..."})
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, namePtr, promptPtr uint32) uint32 {
			name := readASString(m, namePtr)
			prompt := readASString(m, promptPtr)

			code := fmt.Sprintf(`
				var _agent = globalThis.__kit_registry.get("agent", %q);
				if (!_agent || !_agent.ref) return JSON.stringify({error: "agent " + %q + " not found"});
				var _result = await _agent.ref.generate(%q);
				return JSON.stringify({text: _result.text});
			`, name, name, prompt)

			result, err := hs.kit.EvalTS(ctx, "__wasm_call_agent.ts", code)
			if err != nil {
				hs.lastResult = fmt.Sprintf(`{"error":%q}`, err.Error())
			} else {
				// Pass through the JSON directly — it's already {"text":"..."} or {"error":"..."}
				hs.lastResult = result
			}

			ptr, werr := writeASString(ctx, m, hs.lastResult)
			if werr != nil {
				return 0
			}
			return ptr
		}).Export("call_agent").

		// get_state(key: string) → string (returns empty string "" if not found)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr uint32) uint32 {
			key := readASString(m, keyPtr)
			val := hs.state[key] // returns "" if not found

			ptr, err := writeASString(ctx, m, val)
			if err != nil {
				return 0
			}
			return ptr
		}).Export("get_state").

		// set_state(key: string, value: string)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr, valPtr uint32) {
			key := readASString(m, keyPtr)
			val := readASString(m, valPtr)
			hs.state[key] = val
		}).Export("set_state").

		// has_state(key: string) → i32 (0 = not found, 1 = found)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr uint32) uint32 {
			key := readASString(m, keyPtr)
			_, exists := hs.state[key]
			if exists {
				return 1
			}
			return 0
		}).Export("has_state").

		// set_mode(mode: string) — shard init only
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, modePtr uint32) {
			if !hs.initPhase {
				return
			}
			hs.shardMode = readASString(m, modePtr)
		}).Export("set_mode").

		// set_mode_key(keyField: string) — implies keyed mode, init only
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr uint32) {
			if !hs.initPhase {
				return
			}
			hs.shardMode = "keyed"
			hs.shardStateKey = readASString(m, keyPtr)
		}).Export("set_mode_key").

		// on_event(topic: string, funcName: string) — register handler, init only
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, topicPtr, funcPtr uint32) {
			if !hs.initPhase {
				return
			}
			topic := readASString(m, topicPtr)
			funcName := readASString(m, funcPtr)
			hs.shardHandlers[topic] = funcName
		}).Export("on_event").

		// bus_send(topic: string, payloadJSON: string)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, topicPtr, payloadPtr uint32) {
			topic := readASString(m, topicPtr)
			payload := readASString(m, payloadPtr)
			hs.kit.Bus.Send(bus.Message{
				Topic:    topic,
				CallerID: hs.kit.callerID,
				Payload:  json.RawMessage(payload),
			})
		}).Export("bus_send").

		Instantiate(ctx)
	return err
}
