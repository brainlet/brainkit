package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
)

// Kit is the brainkit execution engine.
type Kit struct {
	Bus    *bus.Bus
	Tools  *registry.ToolRegistry
	ai     *AIService
	config Config

	mu        sync.Mutex
	sandboxes map[string]*Sandbox
	closed    bool
}

// New creates a new Kit.
func New(cfg Config) (*Kit, error) {
	k := &Kit{
		Bus:       bus.New(),
		Tools:     registry.New(),
		config:    cfg,
		sandboxes: make(map[string]*Sandbox),
	}

	// Create AI service (has its own QuickJS runtime via ai-embed)
	// Only initialize if we have provider configs (avoids loading the bundle for bus-only tests)
	if len(cfg.Providers) > 0 {
		ai, err := newAIService(k)
		if err != nil {
			return nil, err
		}
		k.ai = ai
	} else {
		k.ai = &AIService{kit: k}
	}

	k.registerHandlers()

	return k, nil
}

// Close shuts down all sandboxes and the bus.
func (k *Kit) Close() {
	k.mu.Lock()
	if k.closed {
		k.mu.Unlock()
		return
	}
	k.closed = true
	sandboxes := make([]*Sandbox, 0, len(k.sandboxes))
	for _, s := range k.sandboxes {
		sandboxes = append(sandboxes, s)
	}
	k.mu.Unlock()

	for _, s := range sandboxes {
		s.Close()
	}
	if k.ai != nil {
		k.ai.close()
	}
	k.Bus.Close()
}

func (k *Kit) registerHandlers() {
	k.Bus.Handle("ai.*", k.ai.handleBusMessage)

	k.Bus.Handle("tools.*", func(ctx context.Context, msg bus.Message) (*bus.Message, error) {
		var req struct {
			Name  string          `json:"name"`
			Input json.RawMessage `json:"input"`
		}
		if err := json.Unmarshal(msg.Payload, &req); err != nil {
			return nil, fmt.Errorf("tools: invalid request: %w", err)
		}

		tool, err := k.Tools.Resolve(req.Name, msg.CallerID)
		if err != nil {
			return nil, err
		}

		result, err := tool.Executor.Call(ctx, msg.CallerID, req.Input)
		if err != nil {
			return nil, err
		}

		return &bus.Message{
			Topic:   msg.ReplyTo,
			Payload: result,
		}, nil
	})
}
