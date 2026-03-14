package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	asembed "github.com/brainlet/brainkit/as-embed"
	"github.com/brainlet/brainkit/bus"
	"github.com/tetratelabs/wazero"
	// "github.com/tetratelabs/wazero/api" // available for future use
)

// WASMService handles wasm.compile and wasm.run bus messages.
// The AS compiler has its own QuickJS runtime (CPU-bound, separate from the Kit's JS runtime).
// wazero handles WASM execution.
type WASMService struct {
	kit      *Kit
	compiler *asembed.Compiler

	mu      sync.Mutex
	modules map[string][]byte // moduleID → compiled WASM binary
}

func newWASMService(kit *Kit) *WASMService {
	return &WASMService{
		kit:     kit,
		modules: make(map[string][]byte),
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
	default:
		return nil, fmt.Errorf("wasm: unknown topic %q", msg.Topic)
	}
}

type wasmCompileRequest struct {
	Source  string                `json:"source"`
	Options asembed.CompileOptions `json:"options"`
}

type wasmCompileResponse struct {
	ModuleID string `json:"moduleId"`
	Text     string `json:"text,omitempty"`
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
	result, err := compiler.Compile(sources, req.Options)
	if err != nil {
		return nil, fmt.Errorf("wasm.compile: %w", err)
	}

	// Store the compiled binary
	moduleID := fmt.Sprintf("mod_%d", len(s.modules))
	s.mu.Lock()
	s.modules[moduleID] = result.Binary
	s.mu.Unlock()

	resp := wasmCompileResponse{
		ModuleID: moduleID,
		Text:     result.Text,
	}
	payload, _ := json.Marshal(resp)
	return &bus.Message{Payload: payload}, nil
}

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

	// Resolve module binary
	var moduleID string
	if req.ModuleID != "" {
		moduleID = req.ModuleID
	} else if req.Module != nil {
		// Module might be passed as {moduleId: "..."} from the compile result
		var mod struct{ ModuleID string `json:"moduleId"` }
		json.Unmarshal(req.Module, &mod)
		moduleID = mod.ModuleID
	}

	s.mu.Lock()
	binary, ok := s.modules[moduleID]
	s.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("wasm.run: module %q not found", moduleID)
	}

	// Execute with wazero
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	compiled, err := rt.CompileModule(ctx, binary)
	if err != nil {
		return nil, fmt.Errorf("wasm.run: compile: %w", err)
	}

	mod, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		return nil, fmt.Errorf("wasm.run: instantiate: %w", err)
	}
	defer mod.Close(ctx)

	// Call the "run" export if it exists, otherwise call "_start"
	var resultVal uint64
	if fn := mod.ExportedFunction("run"); fn != nil {
		results, err := fn.Call(ctx)
		if err != nil {
			return nil, fmt.Errorf("wasm.run: call run(): %w", err)
		}
		if len(results) > 0 {
			resultVal = results[0]
		}
	} else if fn := mod.ExportedFunction("_start"); fn != nil {
		_, err := fn.Call(ctx)
		if err != nil {
			return nil, fmt.Errorf("wasm.run: call _start(): %w", err)
		}
	}

	// Return the result
	result := map[string]any{
		"exitCode": resultVal,
	}

	// Try to read exported memory for string results
	if mem := mod.ExportedMemory("memory"); mem != nil {
		_ = mem // Available for future string I/O
	}
	// Try to read exported globals
	if g := mod.ExportedGlobal("result"); g != nil {
		result["value"] = g.Get()
	}

	payload, _ := json.Marshal(result)
	return &bus.Message{Payload: payload}, nil
}
