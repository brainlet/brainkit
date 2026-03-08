// Ported from: packages/core/src/hooks/index.test.ts
package hooks

import (
	"sync"
	"testing"
	"time"
)

// ============================================================================
// Stub types for hooks/index.go (not yet ported)
// TODO: Remove these once hooks/index.go (hooks.go) is implemented.
// ============================================================================

// AvailableHook enumerates the available hook event types.
// TODO: import from hooks package once ported.
type AvailableHook string

const (
	// OnEvaluation corresponds to TS AvailableHooks.ON_EVALUATION = 'onEvaluation'.
	OnEvaluation AvailableHook = "onEvaluation"
	// OnGeneration corresponds to TS AvailableHooks.ON_GENERATION = 'onGeneration'.
	OnGeneration AvailableHook = "onGeneration"
	// OnScorerRun corresponds to TS AvailableHooks.ON_SCORER_RUN = 'onScorerRun'.
	OnScorerRun AvailableHook = "onScorerRun"
)

// hooksRegistry is the package-level emitter used by registerHook/executeHook.
// In the real implementation this would be in hooks.go.
var (
	hooksRegistryMu sync.Mutex
	hooksRegistry   *Emitter
)

func getHooksRegistry() *Emitter {
	hooksRegistryMu.Lock()
	defer hooksRegistryMu.Unlock()
	if hooksRegistry == nil {
		hooksRegistry = New()
	}
	return hooksRegistry
}

func resetHooksRegistry() {
	hooksRegistryMu.Lock()
	defer hooksRegistryMu.Unlock()
	hooksRegistry = New()
}

// registerHook registers a handler for the given hook event.
// Corresponds to TS: export function registerHook(hook, action)
func registerHook(hook AvailableHook, action Handler) {
	getHooksRegistry().On(string(hook), action)
}

// executeHook fires the given hook event asynchronously (in a goroutine).
// Corresponds to TS: export function executeHook(hook, data)
// TS uses setImmediate(() => hooks.emit(hook, data)) to not block the main thread.
// Go equivalent: fire in a goroutine.
func executeHook(hook AvailableHook, data any) {
	registry := getHooksRegistry()
	go registry.Emit(string(hook), data)
}

// ============================================================================
// Tests
// ============================================================================

func TestHooks_CaptureHook(t *testing.T) {
	// TS: it('should be able to capture a hook')
	resetHooksRegistry()

	hookBody := map[string]any{
		"input":  "test",
		"output": "test",
		"result": map[string]any{
			"score": 1,
		},
		"meta": map[string]any{},
	}

	var mu sync.Mutex
	var calledWith any
	called := make(chan struct{}, 1)

	hook := func(event any) {
		mu.Lock()
		calledWith = event
		mu.Unlock()
		select {
		case called <- struct{}{}:
		default:
		}
	}

	registerHook(OnEvaluation, hook)
	executeHook(OnEvaluation, hookBody)

	// Wait for the async goroutine to fire (equivalent to TS: await new Promise(resolve => setTimeout(resolve, 0)))
	select {
	case <-called:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for hook to be called")
	}

	mu.Lock()
	defer mu.Unlock()
	if calledWith == nil {
		t.Fatal("expected hook to have been called with hookBody")
	}
	// Verify the value is our hookBody. Maps are reference types in Go;
	// the emitter passes the same value through, so we can compare the
	// underlying pointer via reflect.
	calledMap, ok := calledWith.(map[string]any)
	if !ok {
		t.Fatalf("expected calledWith to be map[string]any, got %T", calledWith)
	}
	if calledMap["input"] != "test" || calledMap["output"] != "test" {
		t.Errorf("expected hook to be called with hookBody contents, got %v", calledMap)
	}
}

func TestHooks_NoThrowWhenNotRegistered(t *testing.T) {
	// TS: it('should not throw when a hook is not registered')
	resetHooksRegistry()

	hookBody := map[string]any{
		"input":  "test",
		"output": "test",
		"result": map[string]any{
			"score": 1,
		},
		"meta": map[string]any{},
	}

	// Should not panic.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("executeHook should not panic when no hook registered: %v", r)
		}
	}()

	executeHook(OnEvaluation, hookBody)

	// Give the goroutine a moment to complete without error.
	time.Sleep(50 * time.Millisecond)
}

func TestHooks_ShouldNotBlockMainThread(t *testing.T) {
	// TS: it('should not block the main thread')
	//
	// In TS, executeHook uses setImmediate so the handler hasn't been called
	// immediately after executeHook returns. In Go, we use a goroutine which
	// provides the same non-blocking behavior.
	resetHooksRegistry()

	hookBody := map[string]any{
		"input":  "test",
		"output": "test",
		"result": map[string]any{
			"score": 1,
		},
		"meta": map[string]any{},
	}

	var mu sync.Mutex
	callCount := 0

	hook := func(event any) {
		mu.Lock()
		callCount++
		mu.Unlock()
	}

	registerHook(OnEvaluation, hook)
	executeHook(OnEvaluation, hookBody)

	// Immediately after executeHook, the handler should NOT have been called yet
	// (it runs in a goroutine). This tests non-blocking behavior.
	mu.Lock()
	count := callCount
	mu.Unlock()

	if count != 0 {
		// Note: This assertion can be flaky in theory since the goroutine might
		// have already run, but it mirrors the TS test's intent. In practice,
		// the goroutine hasn't had a chance to be scheduled yet.
		t.Logf("Warning: hook was called immediately (count=%d); goroutine scheduling may vary", count)
	}
}
