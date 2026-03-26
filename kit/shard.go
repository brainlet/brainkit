package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// ShardDescriptor describes a deployed shard's registrations.
type ShardDescriptor struct {
	Module     string            `json:"module"`
	Mode       string            `json:"mode"`     // "stateless" | "persistent"
	Handlers   map[string]string `json:"handlers"` // topic pattern → exported function name
	DeployedAt time.Time         `json:"deployedAt"`
}

// WASMEventResult is the outcome of a shard handler invocation.
type WASMEventResult struct {
	ExitCode     int    `json:"exitCode"`               // kept for wasm.run compatibility (run() returns i32)
	ReplyPayload string `json:"replyPayload,omitempty"` // captured from reply() host function (shard handlers)
	Error        string `json:"error,omitempty"`
}

// deployedShard is an active shard with state.
type deployedShard struct {
	Descriptor    ShardDescriptor
	Binary        []byte // compiled WASM binary
	State         *shardStateStore
	subscriptions map[string]func()

	// Persistent mode: living instance (stays alive between events)
	persistRT   wazero.Runtime // nil for stateless
	persistInst api.Module     // nil for stateless
	persistHS   *hostState     // nil for stateless
	persistMu   sync.Mutex     // serialize handler calls on persistent instance
}

// topicMatches checks if a topic matches a pattern.
// "test.*" matches "test.foo", "test.foo.bar". "test.foo" matches only "test.foo".
func topicMatches(pattern, topic string) bool {
	if pattern == topic {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(topic, prefix)
	}
	return false
}

// ---------------------------------------------------------------------------
// State Store — manages state with per-mode concurrency control
// ---------------------------------------------------------------------------

type shardStateStore struct {
	mode      string
	shardName string
	store     KitStore // nil if no persistence

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

func validateShardDescriptor(desc *ShardDescriptor, exports []string) error {
	if desc.Mode != "stateless" && desc.Mode != "persistent" {
		return &sdk.ValidationError{Field: "mode", Message: fmt.Sprintf("invalid value %q (must be stateless or persistent)", desc.Mode)}
	}
	exportSet := make(map[string]bool, len(exports))
	for _, e := range exports {
		exportSet[e] = true
	}
	for topic, funcName := range desc.Handlers {
		if !exportSet[funcName] {
			return &sdk.ValidationError{Field: "handlers", Message: fmt.Sprintf("handler %q for topic %q not found in module exports", funcName, topic)}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Shard Handler Invocation
// ---------------------------------------------------------------------------

// invokeShardHandler runs a single shard handler with the given payload.
// Handler signature: (topicPtr: u32, payloadPtr: u32) → void
// Reply via reply() host function. Async invocation callbacks run after handler returns.
// (module-protocol §12.1)
func (s *WASMService) invokeShardHandler(ctx context.Context, shardName, topic string, payload json.RawMessage) (*WASMEventResult, error) {
	s.mu.Lock()
	shard, ok := s.shards[shardName]
	if !ok {
		s.mu.Unlock()
		return nil, &sdk.NotFoundError{Resource: "shard", Name: shardName}
	}
	// Find handler function for this topic
	funcName := ""
	for pattern, fn := range shard.Descriptor.Handlers {
		if pattern == topic || topicMatches(pattern, topic) {
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
	hs := newHostState(s.kit, nil)
	hs.state = state

	// Set up async callback context for cancellation
	invokeCtx, invokeCancel := context.WithCancel(ctx)
	defer invokeCancel()
	hs.invokeCtx = invokeCtx
	hs.invokeCancel = invokeCancel

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

	// Store instance reference for async callbacks
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
		return &WASMEventResult{Error: err.Error()}, nil
	}

	// Wait for all pending async callbacks to complete
	hs.pendingInvokes.Wait()

	// Clear instance reference (no more callbacks allowed)
	hs.callbackMu.Lock()
	hs.inst = nil
	hs.callbackMu.Unlock()

	// Save state back
	shard.State.releaseState(hs.state)

	return &WASMEventResult{ReplyPayload: hs.replyPayload}, nil
}

// ---------------------------------------------------------------------------
// Deploy / Undeploy / Describe
// ---------------------------------------------------------------------------

func (s *WASMService) handleDeploy(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("wasm.deploy: invalid request: %w", err)
	}

	s.mu.Lock()
	mod, ok := s.modules[req.Name]
	if !ok {
		s.mu.Unlock()
		return nil, &sdk.NotFoundError{Resource: "module", Name: req.Name}
	}
	if _, deployed := s.shards[req.Name]; deployed {
		s.mu.Unlock()
		return nil, &sdk.AlreadyExistsError{Resource: "shard", Name: req.Name, Hint: "undeploy first"}
	}
	binary := mod.Binary
	exports := mod.Exports
	s.mu.Unlock()

	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	hs := newHostState(s.kit, mod)
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

	if initFn := inst.ExportedFunction("init"); initFn != nil {
		_, err := initFn.Call(ctx)
		if err != nil {
			inst.Close(ctx)
			return nil, fmt.Errorf("wasm.deploy: init() failed: %w", err)
		}
	}
	inst.Close(ctx)

	mode := hs.shardMode
	if mode == "" {
		mode = "stateless"
	}
	desc := ShardDescriptor{
		Module:     req.Name,
		Mode:       mode,
		Handlers:   hs.shardHandlers,
		DeployedAt: time.Now(),
	}

	if err := validateShardDescriptor(&desc, exports); err != nil {
		return nil, fmt.Errorf("wasm.deploy: %w", err)
	}

	deployed := &deployedShard{
		Descriptor: desc,
		Binary:     binary,
		State:      newShardStateStore(mode),
	}
	deployed.State.shardName = req.Name
	deployed.State.store = s.kit.config.Store
	if err := s.bindShardSubscriptions(req.Name, deployed); err != nil {
		return nil, fmt.Errorf("wasm.deploy: %w", err)
	}

	s.mu.Lock()
	s.shards[req.Name] = deployed
	s.mu.Unlock()

	if s.kit.config.Store != nil {
		s.kit.config.Store.SaveShard(req.Name, desc)
	}

	out, _ := json.Marshal(desc)
	return out, nil
}

func (s *WASMService) handleUndeploy(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("wasm.undeploy: invalid request: %w", err)
	}

	s.mu.Lock()
	shard, ok := s.shards[req.Name]
	if !ok {
		s.mu.Unlock()
		return nil, &sdk.NotFoundError{Resource: "shard", Name: req.Name}
	}
	delete(s.shards, req.Name)
	s.mu.Unlock()
	s.cancelShardSubscriptions(shard)

	if s.kit.config.Store != nil {
		s.kit.config.Store.DeleteShard(req.Name)
		s.kit.config.Store.DeleteState(req.Name)
	}

	out, _ := json.Marshal(map[string]bool{"undeployed": true})
	return out, nil
}

func (s *WASMService) handleDescribe(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("wasm.describe: invalid request: %w", err)
	}

	s.mu.Lock()
	shard, ok := s.shards[req.Name]
	s.mu.Unlock()

	if !ok {
		out, _ := json.Marshal(nil)
		return out, nil
	}

	out, _ := json.Marshal(shard.Descriptor)
	return out, nil
}

func (s *WASMService) listDeployedShards() []ShardDescriptor {
	s.mu.Lock()
	defer s.mu.Unlock()
	descs := make([]ShardDescriptor, 0, len(s.shards))
	for _, shard := range s.shards {
		descs = append(descs, shard.Descriptor)
	}
	return descs
}

func (s *WASMService) restoreTransportSubscriptions() error {
	s.mu.Lock()
	shards := make(map[string]*deployedShard, len(s.shards))
	for name, shard := range s.shards {
		shards[name] = shard
	}
	s.mu.Unlock()

	for name, shard := range shards {
		if err := s.bindShardSubscriptions(name, shard); err != nil {
			return fmt.Errorf("restore shard %q subscriptions: %w", name, err)
		}
	}
	return nil
}

func (s *WASMService) bindShardSubscriptions(shardName string, shard *deployedShard) error {
	s.cancelShardSubscriptions(shard)

	s.kit.mu.Lock()
	transport := s.kit.transport
	s.kit.mu.Unlock()
	if transport == nil {
		return nil
	}

	if shard.subscriptions == nil {
		shard.subscriptions = make(map[string]func(), len(shard.Descriptor.Handlers))
	}
	for topic := range shard.Descriptor.Handlers {
		if strings.Contains(topic, "*") {
			s.cancelShardSubscriptions(shard)
			return fmt.Errorf("shard %q uses unsupported wildcard subscription %q on transport-backed runtime", shardName, topic)
		}
		cancel, err := s.kit.subscribe(topic, func(msg messages.Message) {
			if _, invokeErr := s.invokeShardHandler(context.Background(), shardName, topic, json.RawMessage(msg.Payload)); invokeErr != nil {
				log.Printf("[brainkit] shard %q handler for %s failed: %v", shardName, topic, invokeErr)
			}
		})
		if err != nil {
			s.cancelShardSubscriptions(shard)
			return err
		}
		shard.subscriptions[topic] = cancel
	}
	return nil
}

func (s *WASMService) cancelShardSubscriptions(shard *deployedShard) {
	if shard == nil || len(shard.subscriptions) == 0 {
		return
	}
	for topic, cancel := range shard.subscriptions {
		if cancel != nil {
			cancel()
		}
		delete(shard.subscriptions, topic)
	}
}
