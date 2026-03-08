// Ported from: packages/core/src/agent/save-queue/index.ts
package savequeue

import (
	"sync"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
)

// ---------------------------------------------------------------------------
// Re-exported and stub types
// ---------------------------------------------------------------------------

// IMastraLogger is re-exported from logger.
type IMastraLogger = logger.IMastraLogger

// MemoryConfig is a stub for ../../memory.MemoryConfig.
// Real memory.MemoryConfig is a struct with ReadOnly, LastMessages, SemanticRecall, etc.
// Kept as = any because savequeue passes it opaquely to SaveMessages.
type MemoryConfig = any

// MastraMemory is a stub for ../../memory/memory.MastraMemory.
// MISMATCH: real memory.MastraMemory.SaveMessages signature differs:
//   stub:  SaveMessages(SaveMessagesParams) error
//   real:  SaveMessages(ctx context.Context, messages []MastraDBMessage, memoryConfig *MemoryConfig) (*SaveMessagesResult, error)
// Cannot wire without updating all call sites to use context.Context and separate params.
type MastraMemory interface {
	SaveMessages(params SaveMessagesParams) error
}

// SaveMessagesParams holds parameters for MastraMemory.SaveMessages.
// NOT DEFINED in memory package. This is a savequeue-local struct that bundles
// the separate params of memory.MastraMemory.SaveMessages into one struct.
type SaveMessagesParams struct {
	Messages     []any
	MemoryConfig MemoryConfig
}

// MessageList is a stub for ../../agent/message-list.MessageList.
// MISMATCH: real messagelist.MessageList has different method return types:
//   stub:  DrainUnsavedMessages() []any
//   real:  DrainUnsavedMessages() []*state.MastraDBMessage
//   stub:  GetEarliestUnsavedMessageTimestamp() int64
//   real:  GetEarliestUnsavedMessageTimestamp() *int64
// Cannot wire without updating all call sites to handle typed returns.
type MessageList interface {
	// DrainUnsavedMessages returns and clears unsaved messages.
	DrainUnsavedMessages() []any
	// GetEarliestUnsavedMessageTimestamp returns the timestamp of the earliest unsaved message.
	// Returns 0 if no unsaved messages.
	GetEarliestUnsavedMessageTimestamp() int64
}

// MaxStalenessMs is the maximum time (in milliseconds) an unsaved message can remain
// before forcing an immediate flush instead of debouncing.
const MaxStalenessMs = 1000

// SaveQueueManager manages debounced, ordered message persistence for agent threads.
// It ensures that saves are batched efficiently while preventing data loss from stale messages.
type SaveQueueManager struct {
	logger     IMastraLogger
	debounceMs int
	memory     MastraMemory

	saveQueues        map[string]chan struct{} // per-thread serialization
	saveDebounceTimers map[string]*time.Timer

	mu sync.Mutex
}

// SaveQueueManagerOptions holds constructor options for SaveQueueManager.
type SaveQueueManagerOptions struct {
	Logger     IMastraLogger
	DebounceMs int
	Memory     MastraMemory
}

// NewSaveQueueManager creates a new SaveQueueManager.
func NewSaveQueueManager(opts SaveQueueManagerOptions) *SaveQueueManager {
	debounceMs := opts.DebounceMs
	if debounceMs <= 0 {
		debounceMs = 100
	}

	return &SaveQueueManager{
		logger:             opts.Logger,
		debounceMs:         debounceMs,
		memory:             opts.Memory,
		saveQueues:         make(map[string]chan struct{}),
		saveDebounceTimers: make(map[string]*time.Timer),
	}
}

// debounceSave debounces save operations for a thread, ensuring that consecutive
// save requests are batched and only the latest is executed after a short delay.
func (s *SaveQueueManager) debounceSave(threadID string, messageList MessageList, memoryConfig MemoryConfig) {
	s.mu.Lock()
	if timer, ok := s.saveDebounceTimers[threadID]; ok {
		timer.Stop()
	}
	s.saveDebounceTimers[threadID] = time.AfterFunc(
		time.Duration(s.debounceMs)*time.Millisecond,
		func() {
			s.enqueueSave(threadID, messageList, memoryConfig)
			s.mu.Lock()
			delete(s.saveDebounceTimers, threadID)
			s.mu.Unlock()
		},
	)
	s.mu.Unlock()
}

// enqueueSave enqueues a save operation for a thread, ensuring saves are executed
// in order and only one save runs at a time per thread.
func (s *SaveQueueManager) enqueueSave(threadID string, messageList MessageList, memoryConfig MemoryConfig) {
	// Use a mutex-guarded approach to serialize saves per thread
	s.mu.Lock()
	if _, ok := s.saveQueues[threadID]; !ok {
		s.saveQueues[threadID] = make(chan struct{}, 1)
	}
	ch := s.saveQueues[threadID]
	s.mu.Unlock()

	// Acquire per-thread serialization slot
	ch <- struct{}{}
	defer func() {
		<-ch
	}()

	s.persistUnsavedMessages(messageList, memoryConfig)
}

// ClearDebounce clears any pending debounced save for a thread.
func (s *SaveQueueManager) ClearDebounce(threadID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if timer, ok := s.saveDebounceTimers[threadID]; ok {
		timer.Stop()
		delete(s.saveDebounceTimers, threadID)
	}
}

// persistUnsavedMessages drains unsaved messages from the MessageList and persists them.
func (s *SaveQueueManager) persistUnsavedMessages(messageList MessageList, memoryConfig MemoryConfig) {
	newMessages := messageList.DrainUnsavedMessages()
	if len(newMessages) > 0 && s.memory != nil {
		err := s.memory.SaveMessages(SaveMessagesParams{
			Messages:     newMessages,
			MemoryConfig: memoryConfig,
		})
		if err != nil && s.logger != nil {
			s.logger.Error("Error persisting unsaved messages", err)
		}
	}
}

// BatchMessages batches a save of unsaved messages for a thread using debouncing.
// If the oldest unsaved message is stale (older than MaxStalenessMs), the save
// is performed immediately. Otherwise, the save is delayed to batch multiple updates.
func (s *SaveQueueManager) BatchMessages(messageList MessageList, threadID string, memoryConfig MemoryConfig) {
	if threadID == "" {
		return
	}

	earliest := messageList.GetEarliestUnsavedMessageTimestamp()
	now := time.Now().UnixMilli()

	if earliest > 0 && now-earliest > MaxStalenessMs {
		s.FlushMessages(messageList, threadID, memoryConfig)
	} else {
		s.debounceSave(threadID, messageList, memoryConfig)
	}
}

// FlushMessages forces an immediate save of unsaved messages for a thread,
// bypassing any debounce delay.
func (s *SaveQueueManager) FlushMessages(messageList MessageList, threadID string, memoryConfig MemoryConfig) {
	if threadID == "" {
		return
	}
	s.ClearDebounce(threadID)
	s.enqueueSave(threadID, messageList, memoryConfig)
}
