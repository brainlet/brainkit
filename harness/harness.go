package harness

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	iharness "github.com/brainlet/brainkit/internal/harness"
	"github.com/brainlet/brainkit/jsbridge"
)

// Harness orchestrates agent execution with thread persistence,
// mode management, tool approval, and event streaming.
// It wraps Mastra's JS Harness via a brainkit QuickJS bridge.
type Harness struct {
	bridge *jsbridge.Bridge
	evalTS iharness.EvalFunc
	rt     *iharness.Runtime
	config HarnessConfig

	subscribers []harnessSubscriber
	subMu       sync.RWMutex
	nextSubID   int

	displayState *DisplayState
	dsMu         sync.RWMutex

	tokenUsage TokenUsage
	tuMu       sync.RWMutex

	threadLock *ThreadLock

	heartbeats map[string]*time.Ticker
	hbMu       sync.Mutex

	initialized bool
	closed      bool
}

// New creates and initializes a Harness from a bridge and EvalTS function.
// Agents referenced by modes must already exist in the underlying Kit runtime.
func New(bridge *jsbridge.Bridge, evalTS iharness.EvalFunc, cfg HarnessConfig) (*Harness, error) {
	if err := validateHarnessConfig(cfg); err != nil {
		return nil, err
	}

	lock := cfg.ThreadLock
	if lock == nil {
		lock = defaultThreadLock()
	}

	h := &Harness{
		bridge:       bridge,
		evalTS:       evalTS,
		config:       cfg,
		displayState: NewDisplayState(),
		threadLock:   lock,
		heartbeats:   make(map[string]*time.Ticker),
	}

	h.initRuntime()

	jsConfig := h.buildJSConfig()
	configJSON, err := json.Marshal(jsConfig)
	if err != nil {
		return nil, fmt.Errorf("harness: marshal config: %w", err)
	}

	if err := h.rt.InitHarnessJS(string(configJSON)); err != nil {
		return nil, err
	}

	h.startHeartbeats()

	h.initialized = true
	return h, nil
}

// Close stops heartbeats, releases locks, and tears down the Harness.
func (h *Harness) Close() error {
	if h.closed {
		return nil
	}
	h.closed = true
	h.stopHeartbeats()
	return nil
}

// GetDisplayState returns a thread-safe deep copy of the canonical display state.
func (h *Harness) GetDisplayState() *DisplayState {
	h.dsMu.RLock()
	defer h.dsMu.RUnlock()
	return h.displayState.clone()
}

// GetTokenUsage returns the accumulated token usage.
func (h *Harness) GetTokenUsage() TokenUsage {
	h.tuMu.RLock()
	defer h.tuMu.RUnlock()
	return h.tokenUsage
}

// IsRunning returns true if the agent is currently processing.
func (h *Harness) IsRunning() bool {
	h.dsMu.RLock()
	defer h.dsMu.RUnlock()
	return h.displayState.IsRunning
}
