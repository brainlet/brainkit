package wasm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/brainlet/brainkit/bus"
	asembed "github.com/brainlet/brainkit/internal/embed/compiler"
	"github.com/tetratelabs/wazero"
)

// Service handles wasm.compile, wasm.run, and module management bus messages.
// The AS compiler has its own QuickJS runtime (CPU-bound, separate from the Kit's JS runtime).
// wazero handles WASM execution.
type Service struct {
	bridge   BusBridge
	compiler *asembed.Compiler

	mu      sync.Mutex
	modules map[string]*Module        // name → module
	shards  map[string]*deployedShard // name → active shard
	counter int                       // for auto-naming unnamed modules
}

func NewService(bridge BusBridge) *Service {
	return &Service{
		bridge:  bridge,
		modules: make(map[string]*Module),
		shards:  make(map[string]*deployedShard),
	}
}

// LoadFromStore restores modules and shards from persistent storage.
// Called during Kit.New() if a store is configured.
func (s *Service) LoadFromStore(store Store) error {
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

		// Subscribe to bus topics (skip init — descriptor already has registrations)
		var subscriptions []bus.SubscriptionID
		for topic, funcName := range desc.Handlers {
			shardName := name
			fn := funcName
			tp := topic
			subID := s.bridge.Bus().On(topic, func(m bus.Message, _ bus.ReplyFunc) {
				result, err := s.invokeShardHandler(context.Background(), shardName, tp, m.Payload)
				if err != nil {
					log.Printf("[shard:%s] handler %s error: %v", shardName, fn, err)
				} else if result.Error != "" {
					log.Printf("[shard:%s] handler %s error: %s", shardName, fn, result.Error)
				}
			})
			subscriptions = append(subscriptions, subID)
		}

		stateStore := newShardStateStore(desc.Mode)
		stateStore.shardName = name
		stateStore.store = store
		s.mu.Lock()
		s.shards[name] = &deployedShard{
			Descriptor:    desc,
			Binary:        mod.Binary,
			Subscriptions: subscriptions,
			State:         stateStore,
		}
		s.mu.Unlock()
		log.Printf("[brainkit] restored shard %q (%s mode, %d handlers)", name, desc.Mode, len(desc.Handlers))
	}

	return nil
}

func (s *Service) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.compiler != nil {
		s.compiler.Close()
		s.compiler = nil
	}
}

// ensureCompiler lazily creates the AS compiler (expensive — loads the AS bundle).
func (s *Service) ensureCompiler() (*asembed.Compiler, error) {
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

func (s *Service) HandleBusMessage(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	switch msg.Topic {
	case "wasm.compile":
		return s.handleCompile(ctx, msg)
	case "wasm.run":
		return s.handleRun(ctx, msg)
	case "wasm.list":
		return s.handleList(ctx, msg)
	case "wasm.get":
		return s.handleGet(ctx, msg)
	case "wasm.remove":
		return s.handleRemove(ctx, msg)
	case "wasm.deploy":
		return s.handleDeploy(ctx, msg)
	case "wasm.undeploy":
		return s.handleUndeploy(ctx, msg)
	case "wasm.describe":
		return s.handleDescribe(ctx, msg)
	default:
		return nil, fmt.Errorf("wasm: unknown topic %q", msg.Topic)
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

func (s *Service) handleCompile(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req wasmCompileRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("wasm.compile: invalid request: %w", err)
	}

	compiler, err := s.ensureCompiler()
	if err != nil {
		return nil, err
	}

	sources := map[string]string{"input.ts": req.Source}
	// Auto-inject wasm library for `import { ... } from "wasm"` resolution.
	bundleSource := s.bridge.WASMBundleSource()
	if bundleSource != "" {
		sources["~lib/brainkit"] = bundleSource
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

	mod := &Module{
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
	store := s.bridge.WASMStore()
	if store != nil {
		store.SaveModule(name, mod.Binary, ModuleInfo{
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
	payload, _ := json.Marshal(resp)
	return &bus.Message{Payload: payload}, nil
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

func (s *Service) handleRun(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req wasmRunRequest
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
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
		return nil, fmt.Errorf("wasm.run: module %q not found", moduleName)
	}

	// Execute with wazero + host functions
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	// Register host functions BEFORE instantiation so WASM can import from "host"
	hs := newHostState(s.bridge, mod)
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

	// Set up askAsync support for wasm.run
	askCtx, askCancel := context.WithCancel(ctx)
	defer askCancel()
	hs.askCtx = askCtx
	hs.askCancel = askCancel
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

	// Wait for any pending askAsync callbacks
	hs.pendingAsks.Wait()
	hs.askMu.Lock()
	hs.inst = nil
	hs.askMu.Unlock()

	result := map[string]any{
		"exitCode": resultVal,
	}

	if g := inst.ExportedGlobal("result"); g != nil {
		result["value"] = g.Get()
	}

	payload, _ := json.Marshal(result)
	return &bus.Message{Payload: payload}, nil
}

// ---------------------------------------------------------------------------
// Public module accessors (used by Kit convenience methods)
// ---------------------------------------------------------------------------

// ListModules returns metadata for all compiled modules.
func (s *Service) ListModules() []ModuleInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	infos := make([]ModuleInfo, 0, len(s.modules))
	for _, mod := range s.modules {
		infos = append(infos, ModuleInfo{
			Name:       mod.Name,
			Size:       mod.Size,
			Exports:    mod.Exports,
			CompiledAt: mod.CompiledAt.Format(time.RFC3339),
			SourceHash: mod.SourceHash,
		})
	}
	return infos
}

// GetModule returns metadata for a specific module, or nil if not found.
func (s *Service) GetModule(name string) *ModuleInfo {
	s.mu.Lock()
	mod, ok := s.modules[name]
	s.mu.Unlock()
	if !ok {
		return nil
	}
	return &ModuleInfo{
		Name:       mod.Name,
		Size:       mod.Size,
		Exports:    mod.Exports,
		CompiledAt: mod.CompiledAt.Format(time.RFC3339),
		SourceHash: mod.SourceHash,
	}
}

// ---------------------------------------------------------------------------
// List / Get / Remove (bus handlers)
// ---------------------------------------------------------------------------

func (s *Service) handleList(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	infos := make([]ModuleInfo, 0, len(s.modules))
	for _, mod := range s.modules {
		infos = append(infos, ModuleInfo{
			Name:       mod.Name,
			Size:       mod.Size,
			Exports:    mod.Exports,
			CompiledAt: mod.CompiledAt.Format(time.RFC3339),
			SourceHash: mod.SourceHash,
		})
	}
	payload, _ := json.Marshal(infos)
	return &bus.Message{Payload: payload}, nil
}

func (s *Service) handleGet(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name string `json:"name"`
	}
	json.Unmarshal(msg.Payload, &req)

	s.mu.Lock()
	mod, ok := s.modules[req.Name]
	s.mu.Unlock()

	if !ok {
		payload, _ := json.Marshal(nil)
		return &bus.Message{Payload: payload}, nil
	}

	info := ModuleInfo{
		Name:       mod.Name,
		Size:       mod.Size,
		Exports:    mod.Exports,
		CompiledAt: mod.CompiledAt.Format(time.RFC3339),
		SourceHash: mod.SourceHash,
	}
	payload, _ := json.Marshal(info)
	return &bus.Message{Payload: payload}, nil
}

func (s *Service) handleRemove(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name string `json:"name"`
	}
	json.Unmarshal(msg.Payload, &req)

	s.mu.Lock()
	if _, deployed := s.shards[req.Name]; deployed {
		s.mu.Unlock()
		return nil, fmt.Errorf("wasm.remove: cannot remove module %q: shard is deployed (undeploy first)", req.Name)
	}
	_, ok := s.modules[req.Name]
	if ok {
		delete(s.modules, req.Name)
	}
	s.mu.Unlock()

	// Delete from store
	store := s.bridge.WASMStore()
	if ok && store != nil {
		store.DeleteModule(req.Name)
	}

	payload, _ := json.Marshal(map[string]bool{"removed": ok})
	return &bus.Message{Payload: payload}, nil
}
