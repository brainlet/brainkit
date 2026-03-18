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

	asembed "github.com/brainlet/brainkit/as-embed"
	"github.com/brainlet/brainkit/bus"
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
	kit      *Kit
	compiler *asembed.Compiler

	mu      sync.Mutex
	modules map[string]*WASMModule // name → module
	counter int                    // for auto-naming unnamed modules
}

func newWASMService(kit *Kit) *WASMService {
	return &WASMService{
		kit:     kit,
		modules: make(map[string]*WASMModule),
	}
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

func (s *WASMService) handleBusMessage(ctx context.Context, msg bus.Message) (*bus.Message, error) {
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

func (s *WASMService) handleCompile(ctx context.Context, msg bus.Message) (*bus.Message, error) {
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
	// AS resolves bare "wasm" → ~lib/wasm, which nextFile finds in onDemandSources.
	if wasmBundleSource != "" {
		sources["~lib/wasm"] = wasmBundleSource
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

func (s *WASMService) handleRun(ctx context.Context, msg bus.Message) (*bus.Message, error) {
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
			var mod struct{ ModuleID string `json:"moduleId"` }
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
// List / Get / Remove
// ---------------------------------------------------------------------------

func (s *WASMService) handleList(ctx context.Context, msg bus.Message) (*bus.Message, error) {
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
	payload, _ := json.Marshal(infos)
	return &bus.Message{Payload: payload}, nil
}

func (s *WASMService) handleGet(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct{ Name string `json:"name"` }
	json.Unmarshal(msg.Payload, &req)

	s.mu.Lock()
	mod, ok := s.modules[req.Name]
	s.mu.Unlock()

	if !ok {
		payload, _ := json.Marshal(nil)
		return &bus.Message{Payload: payload}, nil
	}

	info := WASMModuleInfo{
		Name:       mod.Name,
		Size:       mod.Size,
		Exports:    mod.Exports,
		CompiledAt: mod.CompiledAt.Format(time.RFC3339),
		SourceHash: mod.SourceHash,
	}
	payload, _ := json.Marshal(info)
	return &bus.Message{Payload: payload}, nil
}

func (s *WASMService) handleRemove(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct{ Name string `json:"name"` }
	json.Unmarshal(msg.Payload, &req)

	s.mu.Lock()
	_, ok := s.modules[req.Name]
	if ok {
		delete(s.modules, req.Name)
	}
	s.mu.Unlock()

	payload, _ := json.Marshal(map[string]bool{"removed": ok})
	return &bus.Message{Payload: payload}, nil
}
