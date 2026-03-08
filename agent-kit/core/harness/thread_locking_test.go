// Ported from: packages/core/src/harness/thread-locking.test.ts
package harness

import (
	"fmt"
	"sync/atomic"
	"testing"
)

// idSeq is a package-level atomic counter for generating unique test IDs.
var idSeq int64

func newTestIDGenerator() func() string {
	return func() string {
		n := atomic.AddInt64(&idSeq, 1)
		return fmt.Sprintf("thread-%d", n)
	}
}

func TestHarnessThreadLocking(t *testing.T) {
	// The TS tests use a full InMemoryStore and threadLock callbacks with vi.fn() mocks.
	// Many tests rely on storage-backed createThread, switchThread, and selectOrCreateThread
	// which persist threads and call lock acquire/release. The Go Harness currently has
	// TODO stubs for storage integration, so most thread locking tests need to be skipped.
	//
	// However, we can test the basic ThreadLock integration that IS wired up.

	t.Run("createThread", func(t *testing.T) {
		t.Skip("not yet implemented - requires storage-backed createThread with lock acquire/release integration")

		// The TS tests verify:
		// 1. acquires lock on the new thread
		// 2. releases lock on previous thread when creating a new one
		// 3. acquire is called before release on createThread
		// 4. re-acquires old lock if acquire on new thread fails
		// 5. waits for an async acquire promise before releasing previous thread lock
	})

	t.Run("switchThread", func(t *testing.T) {
		t.Skip("not yet implemented - requires storage-backed switchThread with lock acquire/release integration")

		// The TS tests verify:
		// 1. acquires lock on the target thread
		// 2. releases lock on previous thread
		// 3. acquire is called before release on switchThread
		// 4. propagates errors from acquire (e.g., lock conflict)
		// 5. waits for an async release promise before resolving switchThread
	})

	t.Run("selectOrCreateThread", func(t *testing.T) {
		t.Run("acquires lock when selecting an existing thread", func(t *testing.T) {
			t.Skip("not yet implemented - requires storage-backed selectOrCreateThread with lock integration")
		})

		t.Run("acquires lock when creating a new thread (no existing threads)", func(t *testing.T) {
			t.Skip("not yet implemented - requires storage-backed selectOrCreateThread with lock integration")
		})
	})

	t.Run("deleteThread", func(t *testing.T) {
		t.Run("emits thread_deleted event", func(t *testing.T) {
			h, err := New(HarnessConfig{
				ID:          "test-harness",
				IDGenerator: newTestIDGenerator(),
				Modes: []HarnessMode{
					{ID: "default", Name: "Default", Default: true},
				},
			})
			if err != nil {
				t.Fatalf("failed to create harness: %v", err)
			}

			var deletedThreadIDs []string
			h.Subscribe(func(event HarnessEvent) {
				if event.Type == "thread_deleted" {
					deletedThreadIDs = append(deletedThreadIDs, event.ThreadID)
				}
			})

			thread, err := h.CreateThread("to-delete")
			if err != nil {
				t.Fatalf("CreateThread failed: %v", err)
			}

			err = h.DeleteThread(thread.ID)
			if err != nil {
				t.Fatalf("DeleteThread failed: %v", err)
			}

			if len(deletedThreadIDs) != 1 {
				t.Fatalf("expected 1 thread_deleted event, got %d", len(deletedThreadIDs))
			}
			if deletedThreadIDs[0] != thread.ID {
				t.Errorf("expected deleted threadID = %q, got %q", thread.ID, deletedThreadIDs[0])
			}
		})

		t.Run("clears currentThreadId when deleting the current thread", func(t *testing.T) {
			h, err := New(HarnessConfig{
				ID:          "test-harness",
				IDGenerator: newTestIDGenerator(),
				Modes: []HarnessMode{
					{ID: "default", Name: "Default", Default: true},
				},
			})
			if err != nil {
				t.Fatalf("failed to create harness: %v", err)
			}

			thread, err := h.CreateThread("current")
			if err != nil {
				t.Fatalf("CreateThread failed: %v", err)
			}

			if got := h.GetCurrentThreadID(); got != thread.ID {
				t.Fatalf("expected currentThreadID = %q, got %q", thread.ID, got)
			}

			err = h.DeleteThread(thread.ID)
			if err != nil {
				t.Fatalf("DeleteThread failed: %v", err)
			}

			if got := h.GetCurrentThreadID(); got != "" {
				t.Errorf("expected currentThreadID to be empty after delete, got %q", got)
			}
		})

		t.Run("resets token usage when deleting the current thread", func(t *testing.T) {
			h, err := New(HarnessConfig{
				ID:          "test-harness",
				IDGenerator: newTestIDGenerator(),
				Modes: []HarnessMode{
					{ID: "default", Name: "Default", Default: true},
				},
			})
			if err != nil {
				t.Fatalf("failed to create harness: %v", err)
			}

			// Set some token usage
			h.mu.Lock()
			h.tokenUsage = TokenUsage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150}
			h.mu.Unlock()

			thread, err := h.CreateThread("current")
			if err != nil {
				t.Fatalf("CreateThread failed: %v", err)
			}

			// CreateThread already resets, set again
			h.mu.Lock()
			h.tokenUsage = TokenUsage{PromptTokens: 200, CompletionTokens: 80, TotalTokens: 280}
			h.mu.Unlock()

			err = h.DeleteThread(thread.ID)
			if err != nil {
				t.Fatalf("DeleteThread failed: %v", err)
			}

			usage := h.GetTokenUsage()
			if usage.PromptTokens != 0 || usage.CompletionTokens != 0 || usage.TotalTokens != 0 {
				t.Errorf("expected zero token usage after delete, got %+v", usage)
			}
		})

		t.Run("does not clear currentThreadId when deleting a non-current thread", func(t *testing.T) {
			h, err := New(HarnessConfig{
				ID:          "test-harness",
				IDGenerator: newTestIDGenerator(),
				Modes: []HarnessMode{
					{ID: "default", Name: "Default", Default: true},
				},
			})
			if err != nil {
				t.Fatalf("failed to create harness: %v", err)
			}

			first, err := h.CreateThread("first")
			if err != nil {
				t.Fatalf("CreateThread failed: %v", err)
			}

			second, err := h.CreateThread("second")
			if err != nil {
				t.Fatalf("CreateThread failed: %v", err)
			}

			// Current thread should be second
			if got := h.GetCurrentThreadID(); got != second.ID {
				t.Fatalf("expected currentThreadID = %q, got %q", second.ID, got)
			}

			// Delete first (non-current) thread
			err = h.DeleteThread(first.ID)
			if err != nil {
				t.Fatalf("DeleteThread failed: %v", err)
			}

			// Current should still be second
			if got := h.GetCurrentThreadID(); got != second.ID {
				t.Errorf("expected currentThreadID to remain %q, got %q", second.ID, got)
			}
		})
	})

	t.Run("without threadLock config", func(t *testing.T) {
		t.Run("works normally without locking", func(t *testing.T) {
			h, err := New(HarnessConfig{
				ID:          "test-harness",
				IDGenerator: newTestIDGenerator(),
				Modes: []HarnessMode{
					{ID: "default", Name: "Default", Default: true},
				},
				// No ThreadLock
			})
			if err != nil {
				t.Fatalf("failed to create harness: %v", err)
			}

			threadA, err := h.CreateThread("test")
			if err != nil {
				t.Fatalf("CreateThread failed: %v", err)
			}

			_, err = h.CreateThread("test2")
			if err != nil {
				t.Fatalf("CreateThread failed: %v", err)
			}

			err = h.SwitchThread(threadA.ID)
			if err != nil {
				t.Fatalf("SwitchThread failed: %v", err)
			}
			// No errors thrown — locking is optional
		})
	})
}
