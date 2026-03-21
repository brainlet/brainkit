package wasm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/brainlet/brainkit/bus"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// deployedShard is an active shard with bus subscriptions and state.
type deployedShard struct {
	Descriptor    ShardDescriptor
	Binary        []byte // compiled WASM binary
	Subscriptions []bus.SubscriptionID
	State         *shardStateStore

	// Persistent mode: living instance (stays alive between events)
	persistRT   wazero.Runtime // nil for stateless
	persistInst api.Module     // nil for stateless
	persistHS   *hostState     // nil for stateless
	persistMu   sync.Mutex     // serialize handler calls on persistent instance
}

// ---------------------------------------------------------------------------
// State Store — manages state with per-mode concurrency control
// ---------------------------------------------------------------------------

type shardStateStore struct {
	mode      string
	shardName string
	store     Store // nil if no persistence

	// persistent mode: one state map, serialized access
	mu     sync.Mutex
	state  map[string]string
	loaded bool
}

func newShardStateStore(mode string) *shardStateStore {
	return &shardStateStore{
		mode:  mode,
		state: make(map[string]string),
	}
}

// acquireState returns a copy of the state and acquires the lock.
// Caller MUST call releaseState when done.
func (s *shardStateStore) acquireState() map[string]string {
	switch s.mode {
	case "stateless":
		return make(map[string]string)
	case "persistent":
		s.mu.Lock()
		// Load from store on first access
		if !s.loaded && s.store != nil {
			persisted, err := s.store.LoadState(s.shardName, "")
			if err == nil && persisted != nil {
				s.state = persisted
			}
			s.loaded = true
		}
		cp := make(map[string]string, len(s.state))
		for k, v := range s.state {
			cp[k] = v
		}
		return cp
	default:
		return make(map[string]string)
	}
}

// releaseState persists the state and releases the lock.
func (s *shardStateStore) releaseState(state map[string]string) {
	switch s.mode {
	case "stateless":
		// discard
	case "persistent":
		s.state = state
		// Save to store BEFORE unlocking (consistency)
		if s.store != nil {
			s.store.SaveState(s.shardName, "", state)
		}
		s.mu.Unlock()
	}
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func ValidateShardDescriptor(desc *ShardDescriptor, exports []string) error {
	if desc.Mode != "stateless" && desc.Mode != "persistent" {
		return fmt.Errorf("invalid shard mode: %q (must be \"stateless\" or \"persistent\")", desc.Mode)
	}
	exportSet := make(map[string]bool, len(exports))
	for _, e := range exports {
		exportSet[e] = true
	}
	for topic, funcName := range desc.Handlers {
		if !exportSet[funcName] {
			return fmt.Errorf("handler %q for topic %q not found in module exports", funcName, topic)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Shard Handler Invocation
// ---------------------------------------------------------------------------

// invokeShardHandler runs a single shard handler with the given payload.
// Handler signature: (topicPtr: u32, payloadPtr: u32) → void
// Reply via reply() host function. askAsync callbacks run after handler returns.
// (module-protocol §12.1)
func (s *Service) invokeShardHandler(ctx context.Context, shardName, topic string, payload json.RawMessage) (*EventResult, error) {
	s.mu.Lock()
	shard, ok := s.shards[shardName]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("shard %q not deployed", shardName)
	}
	// Find handler function for this topic
	funcName := ""
	for pattern, fn := range shard.Descriptor.Handlers {
		if pattern == topic || bus.TopicMatches(pattern, topic) {
			funcName = fn
			break
		}
	}
	if funcName == "" {
		s.mu.Unlock()
		return nil, fmt.Errorf("shard %q has no handler for topic %q", shardName, topic)
	}
	binary := shard.Binary
	s.mu.Unlock()

	// Resolve state
	state := shard.State.acquireState()

	// Create wazero runtime
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	// Register host functions
	hs := newHostState(s.bridge, nil)
	hs.state = state

	// Set up askAsync context for cancellation
	askCtx, askCancel := context.WithCancel(ctx)
	defer askCancel()
	hs.askCtx = askCtx
	hs.askCancel = askCancel

	if err := hs.registerHostFunctions(ctx, rt); err != nil {
		shard.State.releaseState(state)
		return nil, fmt.Errorf("shard %q: register host functions: %w", shardName, err)
	}

	// Compile + instantiate
	compiled, err := rt.CompileModule(ctx, binary)
	if err != nil {
		shard.State.releaseState(state)
		return nil, fmt.Errorf("shard %q: compile: %w", shardName, err)
	}

	inst, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		shard.State.releaseState(state)
		return nil, fmt.Errorf("shard %q: instantiate: %w", shardName, err)
	}
	defer inst.Close(ctx)

	// Store instance reference for askAsync callbacks
	hs.inst = inst

	// Write topic + payload to WASM memory
	topicPtr, err := writeASString(ctx, inst, topic)
	if err != nil {
		shard.State.releaseState(hs.state)
		return nil, fmt.Errorf("shard %q: write topic: %w", shardName, err)
	}
	payloadPtr, err := writeASString(ctx, inst, string(payload))
	if err != nil {
		shard.State.releaseState(hs.state)
		return nil, fmt.Errorf("shard %q: write payload: %w", shardName, err)
	}

	// Call the handler function with (topicPtr, payloadPtr) → void
	fn := inst.ExportedFunction(funcName)
	if fn == nil {
		shard.State.releaseState(hs.state)
		return nil, fmt.Errorf("shard %q: function %q not exported", shardName, funcName)
	}

	_, err = fn.Call(ctx, uint64(topicPtr), uint64(payloadPtr))
	if err != nil {
		shard.State.releaseState(hs.state)
		return &EventResult{Error: err.Error()}, nil
	}

	// Wait for all pending askAsync callbacks to complete
	hs.pendingAsks.Wait()

	// Clear instance reference (no more callbacks allowed)
	hs.askMu.Lock()
	hs.inst = nil
	hs.askMu.Unlock()

	// Save state back
	shard.State.releaseState(hs.state)

	return &EventResult{ReplyPayload: hs.replyPayload}, nil
}

// ---------------------------------------------------------------------------
// Deploy / Undeploy / Describe
// ---------------------------------------------------------------------------

func (s *Service) handleDeploy(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("wasm.deploy: invalid request: %w", err)
	}

	s.mu.Lock()
	mod, ok := s.modules[req.Name]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("wasm.deploy: module %q not found (compile first)", req.Name)
	}
	if _, deployed := s.shards[req.Name]; deployed {
		s.mu.Unlock()
		return nil, fmt.Errorf("wasm.deploy: shard %q already deployed (undeploy first)", req.Name)
	}
	binary := mod.Binary
	exports := mod.Exports
	s.mu.Unlock()

	// Create init-phase wazero runtime
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	hs := newHostState(s.bridge, mod)
	hs.initPhase = true
	if err := hs.registerHostFunctions(ctx, rt); err != nil {
		return nil, fmt.Errorf("wasm.deploy: register host functions: %w", err)
	}

	compiled, err := rt.CompileModule(ctx, binary)
	if err != nil {
		return nil, fmt.Errorf("wasm.deploy: compile: %w", err)
	}

	inst, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		return nil, fmt.Errorf("wasm.deploy: instantiate: %w", err)
	}

	// Call init() if it exists
	if initFn := inst.ExportedFunction("init"); initFn != nil {
		_, err := initFn.Call(ctx)
		if err != nil {
			inst.Close(ctx)
			return nil, fmt.Errorf("wasm.deploy: init() failed: %w", err)
		}
	}
	inst.Close(ctx)

	// Build descriptor from collected registrations
	mode := hs.shardMode
	if mode == "" {
		mode = "stateless" // default
	}
	desc := ShardDescriptor{
		Module:     req.Name,
		Mode:       mode,
		Handlers:   hs.shardHandlers,
		DeployedAt: time.Now(),
	}

	// Validate
	if err := ValidateShardDescriptor(&desc, exports); err != nil {
		return nil, fmt.Errorf("wasm.deploy: %w", err)
	}

	// Subscribe to bus topics
	var subscriptions []bus.SubscriptionID
	for topic, funcName := range desc.Handlers {
		shardName := req.Name
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

	store := s.bridge.WASMStore()

	// Store deployed shard
	s.mu.Lock()
	s.shards[req.Name] = &deployedShard{
		Descriptor:    desc,
		Binary:        binary,
		Subscriptions: subscriptions,
		State:         newShardStateStore(mode),
	}
	// Set store reference for persistence
	s.shards[req.Name].State.shardName = req.Name
	s.shards[req.Name].State.store = store
	s.mu.Unlock()

	// Persist shard descriptor
	if store != nil {
		store.SaveShard(req.Name, desc)
	}

	payload, _ := json.Marshal(desc)
	return &bus.Message{Payload: payload}, nil
}

func (s *Service) handleUndeploy(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("wasm.undeploy: invalid request: %w", err)
	}

	s.mu.Lock()
	shard, ok := s.shards[req.Name]
	if !ok {
		s.mu.Unlock()
		return nil, fmt.Errorf("wasm.undeploy: shard %q not deployed", req.Name)
	}

	// Unsubscribe all
	for _, subID := range shard.Subscriptions {
		s.bridge.Bus().Off(subID)
	}
	delete(s.shards, req.Name)
	s.mu.Unlock()

	// Delete shard + state from store
	store := s.bridge.WASMStore()
	if store != nil {
		store.DeleteShard(req.Name)
		store.DeleteState(req.Name)
	}

	payload, _ := json.Marshal(map[string]bool{"undeployed": true})
	return &bus.Message{Payload: payload}, nil
}

func (s *Service) handleDescribe(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(msg.Payload, &req); err != nil {
		return nil, fmt.Errorf("wasm.describe: invalid request: %w", err)
	}

	s.mu.Lock()
	shard, ok := s.shards[req.Name]
	s.mu.Unlock()

	if !ok {
		payload, _ := json.Marshal(nil)
		return &bus.Message{Payload: payload}, nil
	}

	payload, _ := json.Marshal(shard.Descriptor)
	return &bus.Message{Payload: payload}, nil
}

func (s *Service) ListDeployedShards() []ShardDescriptor {
	s.mu.Lock()
	defer s.mu.Unlock()
	descs := make([]ShardDescriptor, 0, len(s.shards))
	for _, shard := range s.shards {
		descs = append(descs, shard.Descriptor)
	}
	return descs
}
