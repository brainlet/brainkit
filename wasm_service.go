package brainkit

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	asembed "github.com/brainlet/brainkit/internal/embed/compiler"
	"github.com/brainlet/brainkit/sdk"
	"github.com/tetratelabs/wazero"
)

//go:embed runtime/wasm_bundle.ts
var wasmBundleSource string

// WASMModule holds a compiled WASM module with metadata.
type WASMModule struct {
	Name       string    `json:"name"`
	Binary     []byte    `json:"-"`          // not serialized
	SourceHash string    `json:"sourceHash"` // SHA-256 of source
	Exports    []string  `json:"exports"`    // exported function names
	Size       int       `json:"size"`       // binary size in bytes
	CompiledAt time.Time `json:"compiledAt"`
}

// WASMModuleInfo is the serializable metadata (no binary).
type WASMModuleInfo struct {
	Name       string   `json:"name"`
	Size       int      `json:"size"`
	Exports    []string `json:"exports"`
	CompiledAt string   `json:"compiledAt"`
	SourceHash string   `json:"sourceHash"`
}

// WASMService handles wasm.compile, wasm.run, and module management bus messages.
// The AS compiler has its own QuickJS runtime (CPU-bound, separate from the Kit's JS runtime).
// wazero handles WASM execution.
type WASMService struct {
	kit      *Kernel
	compiler *asembed.Compiler

	mu        sync.Mutex
	compileMu sync.Mutex // serializes Compile() calls — Binaryen is NOT thread-safe
	modules   map[string]*WASMModule    // name → module
	shards    map[string]*deployedShard // name → active shard
	counter   int                       // for auto-naming unnamed modules
}

func newWASMService(kit *Kernel) *WASMService {
	return &WASMService{
		kit:     kit,
		modules: make(map[string]*WASMModule),
		shards:  make(map[string]*deployedShard),
	}
}

// loadFromStore restores modules and shard descriptors from persistent storage.
// Transport subscriptions are rebound later when a Node starts.
func (s *WASMService) loadFromStore(store KitStore) error {
	// Load compiled modules
	modules, err := store.LoadModules()
	if err != nil {
		return fmt.Errorf("load modules: %w", err)
	}
	s.mu.Lock()
	for name, mod := range modules {
		s.modules[name] = mod
	}
	s.mu.Unlock()

	// Load and auto-redeploy shards
	shards, err := store.LoadShards()
	if err != nil {
		return fmt.Errorf("load shards: %w", err)
	}
	for name, desc := range shards {
		// Verify module binary exists
		s.mu.Lock()
		mod, ok := s.modules[desc.Module]
		s.mu.Unlock()
		if !ok {
			log.Printf("[brainkit] skipping shard %q: module %q not found", name, desc.Module)
			continue
		}

		stateStore := newShardStateStore(desc.Mode)
		stateStore.shardName = name
		stateStore.store = store
		s.mu.Lock()
		s.shards[name] = &deployedShard{
			Descriptor:    desc,
			Binary:        mod.Binary,
			State:         stateStore,
			subscriptions: make(map[string]func()),
		}
		s.mu.Unlock()
		log.Printf("[brainkit] restored shard %q (%s mode, %d handlers)", name, desc.Mode, len(desc.Handlers))
	}

	return nil
}

func (s *WASMService) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.compiler != nil {
		s.compiler.Close()
		s.compiler = nil
	}
}

// ensureCompiler lazily creates the AS compiler (expensive — loads the AS bundle).
func (s *WASMService) ensureCompiler() (*asembed.Compiler, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.compiler != nil {
		if s.compiler.Dead() {
			s.compiler.Close()
			s.compiler = nil
		} else {
			return s.compiler, nil
		}
	}
	c, err := asembed.NewCompiler()
	if err != nil {
		return nil, fmt.Errorf("wasm: create compiler: %w", err)
	}
	s.compiler = c
	return c, nil
}

// handleRequest dispatches a WASM topic to the appropriate handler.
// Called by WASMDomain methods. Takes topic + raw JSON payload, returns raw JSON result.
func (s *WASMService) handleRequest(ctx context.Context, topic string, payload json.RawMessage) (json.RawMessage, error) {
	switch topic {
	case "wasm.compile":
		return s.handleCompile(ctx, payload)
	case "wasm.run":
		return s.handleRun(ctx, payload)
	case "wasm.list":
		return s.handleList(ctx, payload)
	case "wasm.get":
		return s.handleGet(ctx, payload)
	case "wasm.remove":
		return s.handleRemove(ctx, payload)
	case "wasm.deploy":
		return s.handleDeploy(ctx, payload)
	case "wasm.undeploy":
		return s.handleUndeploy(ctx, payload)
	case "wasm.describe":
		return s.handleDescribe(ctx, payload)
	default:
		return nil, fmt.Errorf("wasm: unknown topic %q", topic)
	}
}

// ---------------------------------------------------------------------------
// Compile
// ---------------------------------------------------------------------------

type wasmCompileRequest struct {
	Source  string             `json:"source"`
	Options wasmCompileOptions `json:"options"`
}

type wasmCompileOptions struct {
	asembed.CompileOptions
	Name string `json:"name"` // module name (optional, auto-generated if empty)
}

type wasmCompileResponse struct {
	ModuleID string   `json:"moduleId"`
	Name     string   `json:"name"`
	Text     string   `json:"text,omitempty"`
	Size     int      `json:"size"`
	Exports  []string `json:"exports"`
}

func (s *WASMService) handleCompile(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var req wasmCompileRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("wasm.compile: invalid request: %w", err)
	}

	// Serialize all compilations — Binaryen C library is NOT thread-safe.
	// Concurrent Compile() calls cause SIGSEGV in BinaryenModuleAllocateAndWrite.
	s.compileMu.Lock()
	defer s.compileMu.Unlock()

	compiler, err := s.ensureCompiler()
	if err != nil {
		return nil, err
	}

	sources := map[string]string{"input.ts": req.Source}
	// Auto-inject wasm library for `import { ... } from "wasm"` resolution.
	// AS resolves bare "brainkit" → ~lib/brainkit, which nextFile finds in onDemandSources.
	if wasmBundleSource != "" {
		sources["~lib/brainkit"] = wasmBundleSource
	}
	compileOpts := req.Options.CompileOptions
	// Always export runtime — needed for host function string interop (__new, memory)
	compileOpts.ExportRuntime = true
	result, err := compiler.Compile(sources, compileOpts)
	if err != nil {
		return nil, fmt.Errorf("wasm.compile: %w", err)
	}
	if len(result.Binary) == 0 {
		return nil, fmt.Errorf("wasm.compile: compiler produced empty binary (warnings: %s)", result.Text)
	}

	// Determine module name
	name := req.Options.Name
	s.mu.Lock()
	if name == "" {
		name = fmt.Sprintf("mod_%d", s.counter)
		s.counter++
	}

	// Extract exports from compiled binary
	exports := extractExports(ctx, result.Binary)

	// Source hash for change detection
	h := sha256.Sum256([]byte(req.Source))
	sourceHash := hex.EncodeToString(h[:])

	mod := &WASMModule{
		Name:       name,
		Binary:     result.Binary,
		SourceHash: sourceHash,
		Exports:    exports,
		Size:       len(result.Binary),
		CompiledAt: time.Now(),
	}
	s.modules[name] = mod
	s.mu.Unlock()

	// Persist module if store configured
	if s.kit.config.Store != nil {
		s.kit.config.Store.SaveModule(name, mod.Binary, WASMModuleInfo{
			Name:       mod.Name,
			Size:       mod.Size,
			Exports:    mod.Exports,
			CompiledAt: mod.CompiledAt.Format(time.RFC3339),
			SourceHash: mod.SourceHash,
		})
	}

	resp := wasmCompileResponse{
		ModuleID: name,
		Name:     name,
		Text:     result.Text,
		Size:     mod.Size,
		Exports:  exports,
	}
	out, _ := json.Marshal(resp)
	return out, nil
}

// extractExports uses wazero to list exported functions from a WASM binary.
func extractExports(ctx context.Context, binary []byte) []string {
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	compiled, err := rt.CompileModule(ctx, binary)
	if err != nil {
		return nil
	}
	defer compiled.Close(ctx)

	var names []string
	for name := range compiled.ExportedFunctions() {
		names = append(names, name)
	}
	return names
}

// ---------------------------------------------------------------------------
// Run
// ---------------------------------------------------------------------------

type wasmRunRequest struct {
	ModuleID string          `json:"moduleId"`
	Module   json.RawMessage `json:"module"`
	Input    json.RawMessage `json:"input"`
}

func (s *WASMService) handleRun(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var req wasmRunRequest
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("wasm.run: invalid request: %w", err)
	}

	// Resolve module name
	var moduleName string
	if req.ModuleID != "" {
		moduleName = req.ModuleID
	} else if req.Module != nil {
		// Try as string (name)
		var name string
		if err := json.Unmarshal(req.Module, &name); err == nil && name != "" {
			moduleName = name
		} else {
			// Try as object with moduleId field
			var mod struct {
				ModuleID string `json:"moduleId"`
			}
			json.Unmarshal(req.Module, &mod)
			moduleName = mod.ModuleID
		}
	}

	s.mu.Lock()
	mod, ok := s.modules[moduleName]
	s.mu.Unlock()
	if !ok {
		return nil, &sdk.NotFoundError{Resource: "module", Name: moduleName}
	}

	// Execute with wazero + host functions
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	// Register host functions BEFORE instantiation so WASM can import from "host"
	hs := newHostState(s.kit, mod)
	if err := hs.registerHostFunctions(ctx, rt); err != nil {
		return nil, fmt.Errorf("wasm.run: register host functions: %w", err)
	}

	compiled, err := rt.CompileModule(ctx, mod.Binary)
	if err != nil {
		return nil, fmt.Errorf("wasm.run: compile: %w", err)
	}

	// Check what functions the module exports
	for name := range compiled.ExportedFunctions() {
		log.Printf("[wasm.run] export: %s", name)
	}

	inst, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		return nil, fmt.Errorf("wasm.run: instantiate: %w", err)
	}
	defer inst.Close(ctx)

	// Set up async callback support for wasm.run
	invokeCtx, invokeCancel := context.WithCancel(ctx)
	defer invokeCancel()
	hs.invokeCtx = invokeCtx
	hs.invokeCancel = invokeCancel
	hs.inst = inst

	// Call the "run" export if it exists, otherwise call "_start"
	var resultVal uint64
	if fn := inst.ExportedFunction("run"); fn != nil {
		results, err := fn.Call(ctx)
		if err != nil {
			return nil, fmt.Errorf("wasm.run: call run(): %w", err)
		}
		if len(results) > 0 {
			resultVal = results[0]
		}
	} else if fn := inst.ExportedFunction("_start"); fn != nil {
		_, err := fn.Call(ctx)
		if err != nil {
			return nil, fmt.Errorf("wasm.run: call _start(): %w", err)
		}
	}

	// Wait for any pending async callbacks
	hs.pendingInvokes.Wait()
	hs.callbackMu.Lock()
	hs.inst = nil
	hs.callbackMu.Unlock()

	result := map[string]any{
		"exitCode": resultVal,
	}

	if g := inst.ExportedGlobal("result"); g != nil {
		result["value"] = g.Get()
	}

	out, _ := json.Marshal(result)
	return out, nil
}

// ---------------------------------------------------------------------------
// List / Get / Remove
// ---------------------------------------------------------------------------

func (s *WASMService) handleList(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	infos := make([]WASMModuleInfo, 0, len(s.modules))
	for _, mod := range s.modules {
		infos = append(infos, WASMModuleInfo{
			Name:       mod.Name,
			Size:       mod.Size,
			Exports:    mod.Exports,
			CompiledAt: mod.CompiledAt.Format(time.RFC3339),
			SourceHash: mod.SourceHash,
		})
	}
	out, _ := json.Marshal(infos)
	return out, nil
}

func (s *WASMService) handleGet(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var req struct {
		Name string `json:"name"`
	}
	json.Unmarshal(payload, &req)

	s.mu.Lock()
	mod, ok := s.modules[req.Name]
	s.mu.Unlock()

	if !ok {
		out, _ := json.Marshal(map[string]any{"module": nil})
		return out, nil
	}

	info := WASMModuleInfo{
		Name:       mod.Name,
		Size:       mod.Size,
		Exports:    mod.Exports,
		CompiledAt: mod.CompiledAt.Format(time.RFC3339),
		SourceHash: mod.SourceHash,
	}
	out, _ := json.Marshal(map[string]any{"module": info})
	return out, nil
}

func (s *WASMService) handleRemove(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var req struct {
		Name string `json:"name"`
	}
	json.Unmarshal(payload, &req)

	s.mu.Lock()
	if _, deployed := s.shards[req.Name]; deployed {
		s.mu.Unlock()
		return nil, &sdk.AlreadyExistsError{Resource: "shard", Name: req.Name, Hint: "undeploy shard first before removing module"}
	}
	_, ok := s.modules[req.Name]
	if ok {
		delete(s.modules, req.Name)
	}
	s.mu.Unlock()

	// Delete from store
	if ok && s.kit.config.Store != nil {
		s.kit.config.Store.DeleteModule(req.Name)
	}

	out, _ := json.Marshal(map[string]bool{"removed": ok})
	return out, nil
}
