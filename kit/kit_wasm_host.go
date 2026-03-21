package kit

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"unicode/utf16"

	"github.com/brainlet/brainkit/internal/bus"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// hostState holds shared state for one WASM execution.
// Created fresh for each wasm.run() or shard handler invocation.
type hostState struct {
	kit    *Kit
	module *WASMModule
	state  map[string]string
	logsMu sync.Mutex
	logs   []string

	// Shard registration (init phase only)
	initPhase     bool
	shardMode     string            // "stateless" or "persistent"
	shardHandlers map[string]string // topic → exported func name
	shardTools    map[string]string // tool name → exported func name

	// Reply mechanism
	currentReplyTo string // replyTo topic for the inbound message being handled
	replyPayload   string // captured payload from reply() call
	hasReplied     bool

	// askAsync tracking
	pendingAsks sync.WaitGroup
	askMu       sync.Mutex         // serializes callback WASM calls
	askCtx      context.Context
	askCancel   context.CancelFunc
	inst        api.Module // set after instantiation, before handler call
}

func newHostState(kit *Kit, module *WASMModule) *hostState {
	return &hostState{
		kit:           kit,
		module:        module,
		state:         make(map[string]string),
		shardHandlers: make(map[string]string),
		shardTools:    make(map[string]string),
	}
}

// readASString reads an AssemblyScript string from WASM linear memory.
// AS strings are UTF-16LE encoded. The object header stores rtSize (byte length)
// at offset -4 from the pointer. The pointer points to the start of the UTF-16 payload.
func readASString(m api.Module, ptr uint32) string {
	if ptr == 0 {
		return ""
	}
	mem := m.Memory()
	if mem == nil {
		return ""
	}
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
	u16s := utf16.Encode([]rune(s))
	byteLen := len(u16s) * 2
	results, err := newFn.Call(ctx, uint64(byteLen), 2)
	if err != nil {
		return 0, fmt.Errorf("__new failed: %w", err)
	}
	ptr := uint32(results[0])
	data := make([]byte, byteLen)
	for i, c := range u16s {
		binary.LittleEndian.PutUint16(data[i*2:], c)
	}
	m.Memory().Write(ptr, data)
	return ptr, nil
}

// registerHostFunctions registers the "host" and "env" modules with wazero.
// 11 host functions total (module-protocol §12.1):
//
//	send, askAsync, on, tool, reply, log, get_state, set_state, has_state, set_mode
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

	// Register "host" module — 11 host functions.
	_, err = rt.NewHostModuleBuilder("host").

		// ── 1. log(msg: string, level: i32) ──
		// (module-protocol §9)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, msgPtr, level uint32) {
			msg := readASString(m, msgPtr)
			hs.logsMu.Lock()
			hs.logs = append(hs.logs, msg)
			hs.logsMu.Unlock()
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

		// ── 2. send(topic: string, payload: string) ──
		// Fire-and-forget bus publish. (module-protocol §4.1, §12.1)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, topicPtr, payloadPtr uint32) {
			topic := readASString(m, topicPtr)
			payload := readASString(m, payloadPtr)
			hs.kit.Bus.Send(bus.Message{
				Topic:    topic,
				CallerID: hs.kit.callerID,
				Payload:  json.RawMessage(payload),
			})
		}).Export("send").

		// ── 3. askAsync(topic: string, payload: string, callbackFuncName: string) ──
		// Async request/response. Goroutine fires bus.Ask, calls exported callback
		// when response arrives. Instance stays alive until all pending asks complete.
		// (module-protocol §4.2, §12.1)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, topicPtr, payloadPtr, callbackPtr uint32) {
			topic := readASString(m, topicPtr)
			payload := readASString(m, payloadPtr)
			callbackName := readASString(m, callbackPtr)

			hs.pendingAsks.Add(1)

			go func() {
				defer hs.pendingAsks.Done()

				askCtx := hs.askCtx
				if askCtx == nil {
					askCtx = ctx
				}

				resp, err := bus.AskSync(hs.kit.Bus, askCtx, bus.Message{
					Topic:    topic,
					CallerID: hs.kit.callerID,
					Payload:  json.RawMessage(payload),
				})

				var respTopic string
				var respPayload string
				if err != nil {
					respTopic = topic
					respPayload = fmt.Sprintf(`{"error":%q}`, err.Error())
				} else {
					respTopic = resp.Topic
					if respTopic == "" {
						respTopic = topic
					}
					respPayload = string(resp.Payload)
				}

				// Serialize access to the WASM instance
				hs.askMu.Lock()
				defer hs.askMu.Unlock()

				inst := hs.inst
				if inst == nil {
					return // instance already closed
				}

				callbackFn := inst.ExportedFunction(callbackName)
				if callbackFn == nil {
					log.Printf("[wasm:askAsync] callback %q not found in exports", callbackName)
					return
				}

				topicStrPtr, werr := writeASString(askCtx, inst, respTopic)
				if werr != nil {
					log.Printf("[wasm:askAsync] write topic failed: %v", werr)
					return
				}
				payloadStrPtr, werr := writeASString(askCtx, inst, respPayload)
				if werr != nil {
					log.Printf("[wasm:askAsync] write payload failed: %v", werr)
					return
				}

				_, cerr := callbackFn.Call(askCtx, uint64(topicStrPtr), uint64(payloadStrPtr))
				if cerr != nil {
					log.Printf("[wasm:askAsync] callback %q call failed: %v", callbackName, cerr)
				}
			}()
		}).Export("askAsync").

		// ── 4. on(topic: string, funcName: string) ──
		// Subscribe to topic pattern. Init phase only.
		// (module-protocol §12.1)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, topicPtr, funcPtr uint32) {
			if !hs.initPhase {
				return
			}
			topic := readASString(m, topicPtr)
			funcName := readASString(m, funcPtr)
			hs.shardHandlers[topic] = funcName
		}).Export("on").

		// ── 5. tool(name: string, funcName: string) ──
		// Register a tool this shard provides. Init phase only.
		// (module-protocol §7.3, §12.1)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, namePtr, funcPtr uint32) {
			if !hs.initPhase {
				return
			}
			name := readASString(m, namePtr)
			funcName := readASString(m, funcPtr)
			hs.shardTools[name] = funcName
		}).Export("tool").

		// ── 6. reply(payload: string) ──
		// Reply to the current inbound message.
		// (module-protocol §6.3, §12.1)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, payloadPtr uint32) {
			payload := readASString(m, payloadPtr)
			hs.replyPayload = payload
			hs.hasReplied = true

			// If there's a replyTo topic, send the reply on the bus immediately
			if hs.currentReplyTo != "" {
				hs.kit.Bus.Send(bus.Message{
					Topic:    hs.currentReplyTo,
					CallerID: hs.kit.callerID,
					Payload:  json.RawMessage(payload),
				})
			}
		}).Export("reply").

		// ── 7. get_state(key: string) → string ──
		// (module-protocol §8)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr uint32) uint32 {
			key := readASString(m, keyPtr)
			val := hs.state[key]
			ptr, err := writeASString(ctx, m, val)
			if err != nil {
				return 0
			}
			return ptr
		}).Export("get_state").

		// ── 8. set_state(key: string, value: string) ──
		// (module-protocol §8)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr, valPtr uint32) {
			key := readASString(m, keyPtr)
			val := readASString(m, valPtr)
			hs.state[key] = val
		}).Export("set_state").

		// ── 9. has_state(key: string) → i32 ──
		// (module-protocol §8)
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, keyPtr uint32) uint32 {
			key := readASString(m, keyPtr)
			_, exists := hs.state[key]
			if exists {
				return 1
			}
			return 0
		}).Export("has_state").

		// ── 10. set_mode(mode: string) ──
		// Sets execution mode: "stateless" or "persistent". Init phase only.
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, modePtr uint32) {
			if !hs.initPhase {
				return
			}
			hs.shardMode = readASString(m, modePtr)
		}).Export("set_mode").

		Instantiate(ctx)
	return err
}
