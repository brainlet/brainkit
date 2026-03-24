package kit

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
	"unicode/utf16"

	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// hostState holds shared state for one WASM execution.
// Created fresh for each wasm.run() or shard handler invocation.
type hostState struct {
	kit    *Kernel
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

	// Async callback tracking
	pendingInvokes sync.WaitGroup
	callbackMu     sync.Mutex // serializes callback WASM calls
	invokeCtx      context.Context
	invokeCancel   context.CancelFunc
	inst           api.Module // set after instantiation, before handler call
}

func newHostState(kit *Kernel, module *WASMModule) *hostState {
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

func pinASObject(ctx context.Context, m api.Module, ptr uint32) error {
	if ptr == 0 {
		return nil
	}
	pinFn := m.ExportedFunction("__pin")
	if pinFn == nil {
		return nil
	}
	_, err := pinFn.Call(ctx, uint64(ptr))
	if err != nil {
		return fmt.Errorf("__pin failed: %w", err)
	}
	return nil
}

func unpinASObject(ctx context.Context, m api.Module, ptr uint32) error {
	if ptr == 0 {
		return nil
	}
	unpinFn := m.ExportedFunction("__unpin")
	if unpinFn == nil {
		return nil
	}
	_, err := unpinFn.Call(ctx, uint64(ptr))
	if err != nil {
		return fmt.Errorf("__unpin failed: %w", err)
	}
	return nil
}

// callExportedFunc calls a WASM exported function with string arguments.
// Handles the full lifecycle: lock → find export → write strings → pin → call → unpin.
func (hs *hostState) callExportedFunc(funcName string, args ...string) error {
	hs.callbackMu.Lock()
	defer hs.callbackMu.Unlock()

	inst := hs.inst
	if inst == nil {
		return fmt.Errorf("instance closed")
	}

	fn := inst.ExportedFunction(funcName)
	if fn == nil {
		return fmt.Errorf("export %q not found", funcName)
	}

	ptrs := make([]uint32, 0, len(args))
	wasmArgs := make([]uint64, 0, len(args))
	defer func() {
		for _, ptr := range ptrs {
			unpinASObject(hs.invokeCtx, inst, ptr)
		}
	}()

	for _, arg := range args {
		ptr, err := writeASString(hs.invokeCtx, inst, arg)
		if err != nil {
			return err
		}
		if err := pinASObject(hs.invokeCtx, inst, ptr); err != nil {
			return err
		}
		ptrs = append(ptrs, ptr)
		wasmArgs = append(wasmArgs, uint64(ptr))
	}

	_, err := fn.Call(hs.invokeCtx, wasmArgs...)
	return err
}

// registerHostFunctions registers the "host" and "env" modules with wazero.
// 10 host functions in "host" module + abort in "env" module.
func (hs *hostState) registerHostFunctions(ctx context.Context, rt wazero.Runtime) error {
	invoker := newLocalInvoker(hs.kit)

	// Register "env" module — AS runtime imports abort() from here.
	// Fixed: actually terminates the module instead of just logging.
	_, err := rt.NewHostModuleBuilder("env").
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, msgPtr, filePtr, line, col uint32) {
			msg := readASString(m, msgPtr)
			file := readASString(m, filePtr)
			log.Printf("[wasm:abort] %s at %s:%d:%d", msg, file, line, col)
			m.CloseWithExitCode(ctx, 255)
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
			// Determine source name from module or shard
			source := "wasm"
			if hs.module != nil && hs.module.Name != "" {
				source = "wasm:" + hs.module.Name
			}
			var levelStr string
			switch level {
			case 0:
				levelStr = "debug"
			case 1:
				levelStr = "info"
			case 2:
				levelStr = "warn"
			case 3:
				levelStr = "error"
			default:
				levelStr = "log"
			}
			hs.kit.emitLog(source, levelStr, msg)
		}).Export("log").

		// ── 2. bus_emit(topic: string, payload: string) ──
		// Fire-and-forget bus publish. No replyTo.
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, topicPtr, payloadPtr uint32) {
			topic := readASString(m, topicPtr)
			payload := readASString(m, payloadPtr)
			if err := hs.kit.publish(ctx, topic, json.RawMessage(payload)); err != nil {
				log.Printf("[wasm:bus_emit] publish to %s failed: %v", topic, err)
			}
		}).Export("bus_emit").

		// ── 3. bus_publish(topic: string, payload: string, callbackFuncName: string) ──
		// Publish to bus with replyTo. Subscribes to replyTo topic and routes the
		// response back to the WASM callback. Can talk to .ts services, other WASM
		// shards, plugins — anything on the bus.
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, topicPtr, payloadPtr, callbackPtr uint32) {
			topic := readASString(m, topicPtr)
			payload := readASString(m, payloadPtr)
			callbackName := readASString(m, callbackPtr)

			hs.pendingInvokes.Add(1)

			go func() {
				defer hs.pendingInvokes.Done()

				invokeCtx := hs.invokeCtx
				if invokeCtx == nil {
					invokeCtx = ctx
				}

				// First try the catalog (for infrastructure commands like tools.call, fs.read)
				if spec, ok := commandCatalog().Lookup(topic); ok && spec.invokeKernel != nil {
					resultPayload, err := invoker.Invoke(invokeCtx, topic, json.RawMessage(payload))
					resultTopic := topic + ".result"
					if err != nil {
						resultPayload, _ = json.Marshal(map[string]string{"error": err.Error()})
					}
					if callErr := hs.callExportedFunc(callbackName, resultTopic, string(resultPayload)); callErr != nil {
						log.Printf("[wasm:bus_publish] callback delivery failed: %v", callErr)
					}
					return
				}

				// Not a catalog command — publish to the bus with replyTo
				correlationID := fmt.Sprintf("wasm-%d", time.Now().UnixNano())
				replyTo := topic + ".reply." + correlationID

				// Subscribe to replyTo first
				replyCtx, replyCancel := context.WithTimeout(invokeCtx, 30*time.Second)
				defer replyCancel()

				replyCh := make(chan messages.Message, 1)
				unsub, subErr := hs.kit.SubscribeRaw(replyCtx, replyTo, func(msg messages.Message) {
					select {
					case replyCh <- msg:
					default:
					}
				})
				if subErr != nil {
					errPayload := `{"error":"bus_publish: subscribe failed: ` + subErr.Error() + `"}`
					hs.callExportedFunc(callbackName, topic+".reply", errPayload)
					return
				}
				defer unsub()

				// Publish with replyTo metadata
				publishCtx := messaging.WithPublishMeta(invokeCtx, correlationID, replyTo)
				if _, pubErr := hs.kit.PublishRaw(publishCtx, topic, json.RawMessage(payload)); pubErr != nil {
					errPayload := `{"error":"bus_publish: publish failed: ` + pubErr.Error() + `"}`
					hs.callExportedFunc(callbackName, topic+".reply", errPayload)
					return
				}

				// Wait for reply
				select {
				case msg := <-replyCh:
					if callErr := hs.callExportedFunc(callbackName, topic+".reply", string(msg.Payload)); callErr != nil {
						log.Printf("[wasm:bus_publish] callback delivery failed: %v", callErr)
					}
				case <-replyCtx.Done():
					errPayload := `{"error":"bus_publish: timeout waiting for reply"}`
					hs.callExportedFunc(callbackName, topic+".reply", errPayload)
				}
			}()
		}).Export("bus_publish").

		// ── 4. bus_on(topic: string, funcName: string) ──
		// Subscribe to topic pattern. Init phase only.
		NewFunctionBuilder().
		WithFunc(func(ctx context.Context, m api.Module, topicPtr, funcPtr uint32) {
			if !hs.initPhase {
				return
			}
			topic := readASString(m, topicPtr)
			funcName := readASString(m, funcPtr)
			hs.shardHandlers[topic] = funcName
		}).Export("bus_on").

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

			if hs.currentReplyTo != "" {
				if err := hs.kit.publish(ctx, hs.currentReplyTo, json.RawMessage(payload)); err != nil {
					log.Printf("[wasm:reply] publish to %s failed: %v", hs.currentReplyTo, err)
				}
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
