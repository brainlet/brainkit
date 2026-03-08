// Ported from: packages/core/src/agent/save-queue/save-queue.test.ts
package savequeue

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Mock types
// ---------------------------------------------------------------------------

// mockMemory tracks save calls and records saved messages.
type mockMemory struct {
	mu        sync.Mutex
	saved     []any
	saveCalls int32
	saveFn    func(params SaveMessagesParams) error // optional custom handler
}

func (m *mockMemory) SaveMessages(params SaveMessagesParams) error {
	if m.saveFn != nil {
		return m.saveFn(params)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	atomic.AddInt32(&m.saveCalls, 1)
	m.saved = append(m.saved, params.Messages...)
	return nil
}

func (m *mockMemory) getSaved() []any {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := make([]any, len(m.saved))
	copy(cp, m.saved)
	return cp
}

func (m *mockMemory) getSaveCalls() int {
	return int(atomic.LoadInt32(&m.saveCalls))
}

// mockMessageList implements the MessageList interface for tests.
type mockMessageList struct {
	mu                sync.Mutex
	unsaved           []any
	earliestTimestamp int64
}

func newMockMessageList() *mockMessageList {
	return &mockMessageList{}
}

func (ml *mockMessageList) Add(msg any) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.unsaved = append(ml.unsaved, msg)
}

func (ml *mockMessageList) DrainUnsavedMessages() []any {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	msgs := ml.unsaved
	ml.unsaved = nil
	return msgs
}

func (ml *mockMessageList) GetEarliestUnsavedMessageTimestamp() int64 {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	return ml.earliestTimestamp
}

func (ml *mockMessageList) SetEarliestTimestamp(ts int64) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.earliestTimestamp = ts
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestSaveQueueManager(t *testing.T) {
	t.Run("batches saves with debounce", func(t *testing.T) {
		mem := &mockMemory{}
		manager := NewSaveQueueManager(SaveQueueManagerOptions{Memory: mem})

		list := newMockMessageList()
		list.Add(map[string]any{"id": "m1", "content": "Hello"})
		manager.BatchMessages(list, "thread-1", nil)

		list.Add(map[string]any{"id": "m2", "content": "World"})
		manager.BatchMessages(list, "thread-1", nil)

		// Wait for debounce to fire.
		time.Sleep(time.Duration(manager.debounceMs+50) * time.Millisecond)

		saveCalls := mem.getSaveCalls()
		if saveCalls != 1 {
			t.Errorf("expected 1 save call, got %d", saveCalls)
		}
		saved := mem.getSaved()
		if len(saved) != 2 {
			t.Errorf("expected 2 saved messages, got %d", len(saved))
		}
	})

	t.Run("does nothing if no unsaved messages", func(t *testing.T) {
		mem := &mockMemory{}
		manager := NewSaveQueueManager(SaveQueueManagerOptions{Memory: mem})

		list := newMockMessageList()
		manager.FlushMessages(list, "thread-4", nil)

		saveCalls := mem.getSaveCalls()
		if saveCalls != 0 {
			t.Errorf("expected 0 save calls, got %d", saveCalls)
		}
	})

	t.Run("handles batchMessages with stale messages (forces flush)", func(t *testing.T) {
		mem := &mockMemory{}
		manager := NewSaveQueueManager(SaveQueueManagerOptions{Memory: mem})

		list := newMockMessageList()
		list.Add(map[string]any{"id": "m1", "content": "Hello"})
		// Set earliest timestamp to be older than MaxStalenessMs.
		staleTime := time.Now().UnixMilli() - MaxStalenessMs - 100
		list.SetEarliestTimestamp(staleTime)

		manager.BatchMessages(list, "thread-5", nil)

		// Since it was stale, it should flush immediately (synchronously via enqueueSave).
		// Give a small buffer for the goroutine to complete.
		time.Sleep(50 * time.Millisecond)

		saveCalls := mem.getSaveCalls()
		if saveCalls != 1 {
			t.Errorf("expected 1 save call for stale message, got %d", saveCalls)
		}
		saved := mem.getSaved()
		if len(saved) != 1 {
			t.Errorf("expected 1 saved message, got %d", len(saved))
		}
	})

	t.Run("clearDebounce cancels pending debounce", func(t *testing.T) {
		mem := &mockMemory{}
		manager := NewSaveQueueManager(SaveQueueManagerOptions{Memory: mem})

		list := newMockMessageList()
		list.Add(map[string]any{"id": "m1", "content": "Hello"})
		manager.BatchMessages(list, "thread-6", nil)

		// Cancel the debounce before it fires.
		manager.ClearDebounce("thread-6")

		// Wait for the original debounce time to pass.
		time.Sleep(time.Duration(manager.debounceMs+50) * time.Millisecond)

		saveCalls := mem.getSaveCalls()
		if saveCalls != 0 {
			t.Errorf("expected 0 save calls after clearDebounce, got %d", saveCalls)
		}
	})

	t.Run("should serialize saves under rapid step completion", func(t *testing.T) {
		var concurrent int32
		var maxConcurrent int32
		var totalSaves int32

		mem := &mockMemory{
			saveFn: func(params SaveMessagesParams) error {
				cur := atomic.AddInt32(&concurrent, 1)
				for {
					old := atomic.LoadInt32(&maxConcurrent)
					if cur <= old || atomic.CompareAndSwapInt32(&maxConcurrent, old, cur) {
						break
					}
				}
				time.Sleep(20 * time.Millisecond)
				atomic.AddInt32(&concurrent, -1)
				atomic.AddInt32(&totalSaves, 1)
				return nil
			},
		}
		manager := NewSaveQueueManager(SaveQueueManagerOptions{Memory: mem})

		list := newMockMessageList()
		threadID := "thread-concurrency"

		// Add and trigger saves rapidly.
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			list.Add(map[string]any{"id": i, "content": "msg"})
			wg.Add(1)
			go func() {
				defer wg.Done()
				manager.FlushMessages(list, threadID, nil)
			}()
		}
		wg.Wait()

		mc := atomic.LoadInt32(&maxConcurrent)
		if mc != 1 {
			t.Errorf("expected maxConcurrent=1 (serialized saves), got %d", mc)
		}
		ts := atomic.LoadInt32(&totalSaves)
		if ts == 0 {
			t.Error("expected at least 1 save to occur")
		}
	})

	t.Run("should flush buffered parts via drainUnsavedMessages", func(t *testing.T) {
		var savedMessages []any
		var mu sync.Mutex

		mem := &mockMemory{
			saveFn: func(params SaveMessagesParams) error {
				mu.Lock()
				defer mu.Unlock()
				savedMessages = append(savedMessages, params.Messages...)
				return nil
			},
		}
		manager := NewSaveQueueManager(SaveQueueManagerOptions{Memory: mem})

		list := newMockMessageList()
		threadID := "thread-drain"

		list.Add(map[string]any{"id": "m1", "role": "user"})
		list.Add(map[string]any{"id": "m2", "role": "assistant"})
		list.Add(map[string]any{"id": "m3", "role": "user"})

		mu.Lock()
		if len(savedMessages) != 0 {
			t.Errorf("expected 0 saved messages before flush, got %d", len(savedMessages))
		}
		mu.Unlock()

		manager.FlushMessages(list, threadID, nil)

		mu.Lock()
		if len(savedMessages) != 3 {
			t.Errorf("expected 3 saved messages after flush, got %d", len(savedMessages))
		}
		mu.Unlock()

		// After flush, drainUnsavedMessages should return empty.
		remaining := list.DrainUnsavedMessages()
		if len(remaining) != 0 {
			t.Errorf("expected 0 remaining unsaved messages, got %d", len(remaining))
		}
	})

	t.Run("should not save when threadID is empty", func(t *testing.T) {
		mem := &mockMemory{}
		manager := NewSaveQueueManager(SaveQueueManagerOptions{Memory: mem})

		list := newMockMessageList()
		list.Add(map[string]any{"id": "m1"})

		manager.BatchMessages(list, "", nil)
		manager.FlushMessages(list, "", nil)

		saveCalls := mem.getSaveCalls()
		if saveCalls != 0 {
			t.Errorf("expected 0 save calls with empty threadID, got %d", saveCalls)
		}
	})

	t.Run("should use custom debounce time", func(t *testing.T) {
		mem := &mockMemory{}
		manager := NewSaveQueueManager(SaveQueueManagerOptions{
			Memory:     mem,
			DebounceMs: 200,
		})

		if manager.debounceMs != 200 {
			t.Errorf("expected debounceMs=200, got %d", manager.debounceMs)
		}

		list := newMockMessageList()
		list.Add(map[string]any{"id": "m1"})
		manager.BatchMessages(list, "thread-1", nil)

		// At 100ms the debounce should not have fired yet.
		time.Sleep(100 * time.Millisecond)
		if mem.getSaveCalls() != 0 {
			t.Error("expected 0 save calls at 100ms with 200ms debounce")
		}

		// Wait for the remaining debounce time plus buffer.
		time.Sleep(150 * time.Millisecond)
		if mem.getSaveCalls() != 1 {
			t.Errorf("expected 1 save call after debounce, got %d", mem.getSaveCalls())
		}
	})

	t.Run("should default debounceMs to 100", func(t *testing.T) {
		mem := &mockMemory{}
		manager := NewSaveQueueManager(SaveQueueManagerOptions{Memory: mem})
		if manager.debounceMs != 100 {
			t.Errorf("expected default debounceMs=100, got %d", manager.debounceMs)
		}
	})

	t.Run("should handle nil memory gracefully", func(t *testing.T) {
		manager := NewSaveQueueManager(SaveQueueManagerOptions{Memory: nil})
		list := newMockMessageList()
		list.Add(map[string]any{"id": "m1"})

		// Should not panic.
		manager.FlushMessages(list, "thread-1", nil)
	})
}
