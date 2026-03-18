package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/brainlet/brainkit/bus"
	"github.com/tetratelabs/wazero"
)

// ShardDescriptor describes a deployed shard's registrations.
type ShardDescriptor struct {
	Module     string            `json:"module"`
	Mode       string            `json:"mode"`     // stateless | shared | keyed
	StateKey   string            `json:"stateKey"`  // "" if not keyed
	Handlers   map[string]string `json:"handlers"`  // topic pattern → exported function name
	DeployedAt time.Time         `json:"deployedAt"`
}

// WASMEventResult is the outcome of a shard handler invocation.
type WASMEventResult struct {
	ExitCode int    `json:"exitCode"`
	Error    string `json:"error,omitempty"`
}

// deployedShard is an active shard with bus subscriptions and state.
type deployedShard struct {
	Descriptor    ShardDescriptor
	Binary        []byte // compiled WASM binary
	Subscriptions []bus.SubscriptionID
	State         *shardStateStore
}

// ---------------------------------------------------------------------------
// State Store — manages state with per-mode concurrency control
// ---------------------------------------------------------------------------

type shardStateStore struct {
	mode      string
	shardName string
	store     KitStore // nil if no persistence

	// shared mode: one state map, serialized access
	sharedMu     sync.Mutex
	shared       map[string]string
	sharedLoaded bool

	// keyed mode: per-key state maps with per-key locks
	keyedMu     sync.Mutex
	keyLocks    map[string]*sync.Mutex
	keyed       map[string]map[string]string
	keyedLoaded map[string]bool
}

func newShardStateStore(mode string) *shardStateStore {
	return &shardStateStore{
		mode:        mode,
		shared:      make(map[string]string),
		keyLocks:    make(map[string]*sync.Mutex),
		keyed:       make(map[string]map[string]string),
		keyedLoaded: make(map[string]bool),
	}
}

// acquireState returns a copy of the state for the given key and acquires
// the appropriate lock. Caller MUST call releaseState when done.
func (s *shardStateStore) acquireState(key string) map[string]string {
	switch s.mode {
	case "stateless":
		return make(map[string]string)
	case "shared":
		s.sharedMu.Lock()
		// Load from store on first access
		if !s.sharedLoaded && s.store != nil {
			persisted, err := s.store.LoadState(s.shardName, "")
			if err == nil && persisted != nil {
				s.shared = persisted
			}
			s.sharedLoaded = true
		}
		cp := make(map[string]string, len(s.shared))
		for k, v := range s.shared {
			cp[k] = v
		}
		return cp
	case "keyed":
		s.keyedMu.Lock()
		mu, ok := s.keyLocks[key]
		if !ok {
			mu = &sync.Mutex{}
			s.keyLocks[key] = mu
		}
		s.keyedMu.Unlock()
		mu.Lock()
		// Load from store on first access per key
		if !s.keyedLoaded[key] && s.store != nil {
			persisted, err := s.store.LoadState(s.shardName, key)
			if err == nil && persisted != nil {
				s.keyed[key] = persisted
			}
			s.keyedMu.Lock()
			s.keyedLoaded[key] = true
			s.keyedMu.Unlock()
		}
		state := s.keyed[key]
		cp := make(map[string]string, len(state))
		for k, v := range state {
			cp[k] = v
		}
		return cp
	default:
		return make(map[string]string)
	}
}

// releaseState persists the state and releases the lock.
func (s *shardStateStore) releaseState(key string, state map[string]string) {
	switch s.mode {
	case "stateless":
		// discard
	case "shared":
		s.shared = state
		// Save to store BEFORE unlocking (consistency)
		if s.store != nil {
			s.store.SaveState(s.shardName, "", state)
		}
		s.sharedMu.Unlock()
	case "keyed":
		s.keyed[key] = state
		// Save to store BEFORE unlocking (consistency)
		if s.store != nil {
			s.store.SaveState(s.shardName, key, state)
		}
		s.keyedMu.Lock()
		mu := s.keyLocks[key]
		s.keyedMu.Unlock()
		if mu != nil {
			mu.Unlock()
		}
	}
}

// ---------------------------------------------------------------------------
// Key Extraction
// ---------------------------------------------------------------------------

func extractShardKey(payload json.RawMessage, keyField string) string {
	if keyField == "" {
		return ""
	}
	var payloadMap map[string]json.RawMessage
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		return ""
	}
	raw, ok := payloadMap[keyField]
	if !ok {
		return ""
	}
	var keyValue string
	if err := json.Unmarshal(raw, &keyValue); err != nil {
		// Coerce non-string to string
		keyValue = string(raw)
	}
	return keyValue
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func validateShardDescriptor(desc *ShardDescriptor, exports []string) error {
	if desc.Mode != "stateless" && desc.Mode != "shared" && desc.Mode != "keyed" {
		return fmt.Errorf("invalid shard mode: %q", desc.Mode)
	}
	if desc.Mode == "keyed" && desc.StateKey == "" {
		return fmt.Errorf("keyed mode requires a state key field")
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
// This is used both by bus subscription callbacks and by InjectWASMEvent.
func (s *WASMService) invokeShardHandler(ctx context.Context, shardName, topic string, payload json.RawMessage) (*WASMEventResult, error) {
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
	key := extractShardKey(payload, shard.Descriptor.StateKey)
	state := shard.State.acquireState(key)

	// Create wazero runtime
	rt := wazero.NewRuntime(ctx)
	defer rt.Close(ctx)

	// Register host functions
	hs := newHostState(s.kit, nil)
	hs.state = state
	if err := hs.registerHostFunctions(ctx, rt); err != nil {
		shard.State.releaseState(key, state)
		return nil, fmt.Errorf("shard %q: register host functions: %w", shardName, err)
	}

	// Compile + instantiate
	compiled, err := rt.CompileModule(ctx, binary)
	if err != nil {
		shard.State.releaseState(key, state)
		return nil, fmt.Errorf("shard %q: compile: %w", shardName, err)
	}

	inst, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		shard.State.releaseState(key, state)
		return nil, fmt.Errorf("shard %q: instantiate: %w", shardName, err)
	}
	defer inst.Close(ctx)

	// Write payload to WASM memory
	payloadPtr, err := writeASString(ctx, inst, string(payload))
	if err != nil {
		shard.State.releaseState(key, hs.state)
		return nil, fmt.Errorf("shard %q: write payload: %w", shardName, err)
	}

	// Call the handler function
	fn := inst.ExportedFunction(funcName)
	if fn == nil {
		shard.State.releaseState(key, hs.state)
		return nil, fmt.Errorf("shard %q: function %q not exported", shardName, funcName)
	}

	results, err := fn.Call(ctx, uint64(payloadPtr))
	exitCode := 0
	if err != nil {
		shard.State.releaseState(key, hs.state)
		return &WASMEventResult{ExitCode: -1, Error: err.Error()}, nil
	}
	if len(results) > 0 {
		exitCode = int(results[0])
	}

	// Save state back
	shard.State.releaseState(key, hs.state)

	return &WASMEventResult{ExitCode: exitCode}, nil
}

// ---------------------------------------------------------------------------
// Deploy / Undeploy / Describe
// ---------------------------------------------------------------------------

func (s *WASMService) handleDeploy(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct{ Name string `json:"name"` }
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
		StateKey:   hs.shardStateKey,
		Handlers:   hs.shardHandlers,
		DeployedAt: time.Now(),
	}

	// Validate
	if err := validateShardDescriptor(&desc, exports); err != nil {
		return nil, fmt.Errorf("wasm.deploy: %w", err)
	}

	// Subscribe to bus topics
	var subscriptions []bus.SubscriptionID
	for topic, funcName := range desc.Handlers {
		shardName := req.Name
		fn := funcName
		tp := topic
		subID := s.kit.Bus.On(topic, func(m bus.Message, _ bus.ReplyFunc) {
			result, err := s.invokeShardHandler(context.Background(), shardName, tp, m.Payload)
			if err != nil {
				log.Printf("[shard:%s] handler %s error: %v", shardName, fn, err)
			} else if result.ExitCode != 0 {
				log.Printf("[shard:%s] handler %s returned exit code %d", shardName, fn, result.ExitCode)
			}
		})
		subscriptions = append(subscriptions, subID)
	}

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
	s.shards[req.Name].State.store = s.kit.config.Store
	s.mu.Unlock()

	// Persist shard descriptor
	if s.kit.config.Store != nil {
		s.kit.config.Store.SaveShard(req.Name, desc)
	}

	payload, _ := json.Marshal(desc)
	return &bus.Message{Payload: payload}, nil
}

func (s *WASMService) handleUndeploy(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct{ Name string `json:"name"` }
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
		s.kit.Bus.Off(subID)
	}
	delete(s.shards, req.Name)
	s.mu.Unlock()

	// Delete shard + state from store
	if s.kit.config.Store != nil {
		s.kit.config.Store.DeleteShard(req.Name)
		s.kit.config.Store.DeleteState(req.Name)
	}

	payload, _ := json.Marshal(map[string]bool{"undeployed": true})
	return &bus.Message{Payload: payload}, nil
}

func (s *WASMService) handleDescribe(ctx context.Context, msg bus.Message) (*bus.Message, error) {
	var req struct{ Name string `json:"name"` }
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

func (s *WASMService) listDeployedShards() []ShardDescriptor {
	s.mu.Lock()
	defer s.mu.Unlock()
	descs := make([]ShardDescriptor, 0, len(s.shards))
	for _, shard := range s.shards {
		descs = append(descs, shard.Descriptor)
	}
	return descs
}
