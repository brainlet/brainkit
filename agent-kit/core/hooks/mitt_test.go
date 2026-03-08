// Ported from: packages/core/src/hooks/mitt.test.ts

package hooks

import (
	"sync"
	"testing"
)

// callTracker records calls to a handler for assertion purposes.
// Corresponds to TS: vi.fn()
type callTracker struct {
	mu    sync.Mutex
	calls []any
}

func newTracker() (*callTracker, Handler) {
	t := &callTracker{}
	h := func(event any) {
		t.mu.Lock()
		defer t.mu.Unlock()
		t.calls = append(t.calls, event)
	}
	return t, h
}

func (ct *callTracker) callCount() int {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return len(ct.calls)
}

func (ct *callTracker) calledWith(i int) any {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	return ct.calls[i]
}

// TestOff_RemoveSpecificHandler corresponds to TS:
// it('should remove only the specific handler when handler is provided')
func TestOff_RemoveSpecificHandler(t *testing.T) {
	// Arrange: Register multiple handlers for the 'foo' event
	emitter := New()
	tracker1, handler1 := newTracker()
	tracker2, handler2 := newTracker()

	emitter.On("foo", handler1)
	emitter.On("foo", handler2)

	// Act: Remove one specific handler
	emitter.Off("foo", handler1)

	// Assert: Verify handler removal and emission behavior
	emitter.Emit("foo", "test")

	if tracker1.callCount() != 0 {
		t.Errorf("handler1 should not have been called, got %d calls", tracker1.callCount())
	}
	if tracker2.callCount() != 1 {
		t.Fatalf("handler2 should have been called once, got %d calls", tracker2.callCount())
	}
	if tracker2.calledWith(0) != "test" {
		t.Errorf("handler2 should have been called with 'test', got %v", tracker2.calledWith(0))
	}
}

// TestOff_RemoveAllHandlers corresponds to TS:
// it('should remove all handlers when no handler is provided')
func TestOff_RemoveAllHandlers(t *testing.T) {
	// Arrange: Register multiple handlers for the 'foo' event
	emitter := New()
	tracker1, handler1 := newTracker()
	tracker2, handler2 := newTracker()

	emitter.On("foo", handler1)
	emitter.On("foo", handler2)

	// Act: Remove all handlers for the event type
	emitter.Off("foo")

	// Assert: Verify all handlers are removed
	emitter.Emit("foo", "test")

	if tracker1.callCount() != 0 {
		t.Errorf("handler1 should not have been called, got %d calls", tracker1.callCount())
	}
	if tracker2.callCount() != 0 {
		t.Errorf("handler2 should not have been called, got %d calls", tracker2.callCount())
	}
}

// TestOff_NoHandlersRegistered corresponds to TS:
// it('should safely handle calling off() on event type with no handlers')
func TestOff_NoHandlersRegistered(t *testing.T) {
	emitter := New()

	// Act & Assert: Verify both variants don't panic
	// TS: expect(() => { emitter.off('foo'); }).not.toThrow();
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Off('foo') should not panic, got: %v", r)
			}
		}()
		emitter.Off("foo") // Remove all handlers for non-existent event
	}()

	// TS: expect(() => { emitter.off('foo', unusedHandler); }).not.toThrow();
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Off('foo', handler) should not panic, got: %v", r)
			}
		}()
		_, unusedHandler := newTracker()
		emitter.Off("foo", unusedHandler) // Remove specific handler for non-existent event
	}()

	// Assert: Verify the emitter's state remains valid
	tracker1, handler1 := newTracker()
	emitter.On("foo", handler1)
	emitter.Emit("foo", "test")

	if tracker1.callCount() != 1 {
		t.Fatalf("handler1 should have been called once, got %d calls", tracker1.callCount())
	}
	if tracker1.calledWith(0) != "test" {
		t.Errorf("handler1 should have been called with 'test', got %v", tracker1.calledWith(0))
	}
}

// TestOff_PreserveHandlersOnNonExistentRemoval corresponds to TS:
// it('should preserve existing handlers when removing non-existent handler')
func TestOff_PreserveHandlersOnNonExistentRemoval(t *testing.T) {
	// Arrange: Register two valid handlers for 'foo' event
	emitter := New()
	tracker1, handler1 := newTracker()
	tracker2, handler2 := newTracker()
	trackerNonExistent, nonExistentHandler := newTracker()

	emitter.On("foo", handler1)
	emitter.On("foo", handler2)

	// Act: Attempt to remove non-existent handler and emit event
	emitter.Off("foo", nonExistentHandler)
	emitter.Emit("foo", "test")

	// Assert: Verify both original handlers were called and non-existent handler was never called
	if tracker1.callCount() != 1 {
		t.Fatalf("handler1 should have been called once, got %d calls", tracker1.callCount())
	}
	if tracker1.calledWith(0) != "test" {
		t.Errorf("handler1 should have been called with 'test', got %v", tracker1.calledWith(0))
	}
	if tracker2.callCount() != 1 {
		t.Fatalf("handler2 should have been called once, got %d calls", tracker2.callCount())
	}
	if tracker2.calledWith(0) != "test" {
		t.Errorf("handler2 should have been called with 'test', got %v", tracker2.calledWith(0))
	}
	if trackerNonExistent.callCount() != 0 {
		t.Errorf("nonExistentHandler should not have been called, got %d calls", trackerNonExistent.callCount())
	}
}
