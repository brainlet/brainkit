// Ported from: packages/core/src/processors/memory/message-history.ts
package memory

import (
	"context"
	"strings"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/processors"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"

	wmutils "github.com/brainlet/brainkit/agent-kit/core/memory"
)

// ---------------------------------------------------------------------------
// Stub types for unported dependencies
// ---------------------------------------------------------------------------

// MemoryStorage is the storage interface used by memory processors.
// STUB REASON: The real storage/domains/memory.MemoryStorage interface has different
// method signatures (different param/return types, more methods). Replacing would
// require updating all call sites throughout message_history.go, semantic_recall.go,
// and working_memory.go to use the real storage domain types.
type MemoryStorage interface {
	// ListMessages lists messages with optional filtering.
	ListMessages(ctx context.Context, args ListMessagesInput) (ListMessagesOutput, error)

	// GetThreadByID retrieves a thread by its ID.
	GetThreadByID(ctx context.Context, threadID string) (*StorageThread, error)

	// SaveThread creates or saves a thread.
	SaveThread(ctx context.Context, thread StorageThread) error

	// UpdateThread updates an existing thread.
	UpdateThread(ctx context.Context, input UpdateThreadInput) error

	// SaveMessages persists messages.
	SaveMessages(ctx context.Context, messages []processors.MastraDBMessage) error

	// GetResourceByID retrieves a resource by its ID.
	GetResourceByID(ctx context.Context, resourceID string) (*StorageResource, error)
}

// ListMessagesInput holds parameters for listing messages.
// STUB REASON: Part of the MemoryStorage stub interface contract.
type ListMessagesInput struct {
	ThreadID   string           `json:"threadId"`
	ResourceID string           `json:"resourceId,omitempty"`
	Page       int              `json:"page"`
	PerPage    int              `json:"perPage,omitempty"`
	OrderBy    *OrderByClause   `json:"orderBy,omitempty"`
	Include    []IncludeClause  `json:"include,omitempty"`
}

// OrderByClause describes a sort order.
type OrderByClause struct {
	Field     string `json:"field"`
	Direction string `json:"direction"` // "ASC" | "DESC"
}

// IncludeClause identifies a specific message to include plus context range.
type IncludeClause struct {
	ID                   string `json:"id"`
	ThreadID             string `json:"threadId,omitempty"`
	WithNextMessages     int    `json:"withNextMessages,omitempty"`
	WithPreviousMessages int    `json:"withPreviousMessages,omitempty"`
}

// ListMessagesOutput is the result of listing messages.
type ListMessagesOutput struct {
	Messages []processors.MastraDBMessage `json:"messages"`
}

// StorageThread represents a thread record.
// STUB REASON: Part of the MemoryStorage stub interface contract.
type StorageThread struct {
	ID         string         `json:"id"`
	ResourceID string         `json:"resourceId"`
	Title      string         `json:"title"`
	Metadata   map[string]any `json:"metadata"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

// UpdateThreadInput holds the fields for updating a thread.
type UpdateThreadInput struct {
	ID       string         `json:"id"`
	Title    string         `json:"title"`
	Metadata map[string]any `json:"metadata"`
}

// StorageResource represents a resource record.
// STUB REASON: Part of the MemoryStorage stub interface contract.
type StorageResource struct {
	ID            string  `json:"id"`
	WorkingMemory string  `json:"workingMemory,omitempty"`
}

// ---------------------------------------------------------------------------
// MemoryRequestContext helpers
// ---------------------------------------------------------------------------

// MemoryRequestContext holds memory-specific context passed via RequestContext
// under the 'MastraMemory' key.
// Defined locally for the processors/memory subpackage. The memory package has
// its own version of this type.
type MemoryRequestContext struct {
	Thread       *MemoryRequestThread `json:"thread,omitempty"`
	ResourceID   string               `json:"resourceId,omitempty"`
	MemoryConfig *MemoryConfig        `json:"memoryConfig,omitempty"`
}

// MemoryRequestThread is a partial thread with at least an ID.
type MemoryRequestThread struct {
	ID string `json:"id"`
}

// MemoryConfig holds runtime memory configuration.
// Defined locally for the processors/memory subpackage.
type MemoryConfig struct {
	ReadOnly bool `json:"readOnly,omitempty"`
}

// ParseMemoryRequestContext extracts and validates memory context from a
// RequestContext.  Returns nil when no memory context is set.
func ParseMemoryRequestContext(rc *requestcontext.RequestContext) *MemoryRequestContext {
	if rc == nil {
		return nil
	}
	raw := rc.Get("MastraMemory")
	if raw == nil {
		return nil
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil
	}

	result := &MemoryRequestContext{}

	// Extract thread
	if threadRaw, ok := m["thread"]; ok {
		if threadMap, ok := threadRaw.(map[string]any); ok {
			if id, ok := threadMap["id"].(string); ok {
				result.Thread = &MemoryRequestThread{ID: id}
			}
		}
	}

	// Extract resourceId
	if rid, ok := m["resourceId"].(string); ok {
		result.ResourceID = rid
	}

	// Extract memoryConfig
	if cfgRaw, ok := m["memoryConfig"]; ok {
		if cfgMap, ok := cfgRaw.(map[string]any); ok {
			cfg := &MemoryConfig{}
			if ro, ok := cfgMap["readOnly"].(bool); ok {
				cfg.ReadOnly = ro
			}
			result.MemoryConfig = cfg
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// MessageHistoryOptions
// ---------------------------------------------------------------------------

// MessageHistoryOptions configures the MessageHistory processor.
type MessageHistoryOptions struct {
	Storage      MemoryStorage
	LastMessages int // 0 means no limit
}

// ---------------------------------------------------------------------------
// MessageHistory
// ---------------------------------------------------------------------------

// MessageHistory is a hybrid processor that handles both retrieval and
// persistence of message history.
//   - On input:  Fetches historical messages from storage and prepends them.
//   - On output: Persists new messages to storage (excluding system messages).
//
// It retrieves threadId and resourceId from RequestContext at execution time,
// making it decoupled from memory-specific context.
type MessageHistory struct {
	processors.BaseProcessor
	storage      MemoryStorage
	lastMessages int
}

// NewMessageHistory creates a new MessageHistory processor.
func NewMessageHistory(opts MessageHistoryOptions) *MessageHistory {
	return &MessageHistory{
		BaseProcessor: processors.NewBaseProcessor("message-history", "MessageHistory"),
		storage:       opts.Storage,
		lastMessages:  opts.LastMessages,
	}
}

// getMemoryContext gets threadId and resourceId from either RequestContext or
// MessageList's memoryInfo.
func (mh *MessageHistory) getMemoryContext(
	rc *requestcontext.RequestContext,
	messageList *processors.MessageList,
) *MemoryRequestContext {
	// First try RequestContext (set by Memory class)
	memCtx := ParseMemoryRequestContext(rc)
	if memCtx != nil && memCtx.Thread != nil && memCtx.Thread.ID != "" {
		return memCtx
	}

	// Fallback to MessageList's memoryInfo (set when MessageList is created with threadId).
	// Ported from TS: const serialized = messageList.serialize();
	//                  if (serialized.memoryInfo?.threadId) { ... }
	if messageList != nil {
		serialized := messageList.Serialize()
		if serialized.MemoryInfo != nil && serialized.MemoryInfo.ThreadID != "" {
			return &MemoryRequestContext{
				Thread:     &MemoryRequestThread{ID: serialized.MemoryInfo.ThreadID},
				ResourceID: serialized.MemoryInfo.ResourceID,
			}
		}
	}
	return nil
}

// ProcessInput fetches historical messages from storage and adds them to the
// MessageList. Implements InputProcessorMethods.
func (mh *MessageHistory) ProcessInput(args processors.ProcessInputArgs) (
	[]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error,
) {
	messageList := args.MessageList
	rc := args.RequestContext

	ctx := context.Background()

	memCtx := mh.getMemoryContext(rc, messageList)
	if memCtx == nil {
		return nil, messageList, nil, nil
	}

	threadID := memCtx.Thread.ID
	resourceID := memCtx.ResourceID

	// 1. Fetch historical messages from storage (as DB format)
	perPage := 0
	if mh.lastMessages > 0 {
		perPage = mh.lastMessages
	}
	result, err := mh.storage.ListMessages(ctx, ListMessagesInput{
		ThreadID:   threadID,
		ResourceID: resourceID,
		Page:       0,
		PerPage:    perPage,
		OrderBy:    &OrderByClause{Field: "createdAt", Direction: "DESC"},
	})
	if err != nil {
		return nil, messageList, nil, err
	}

	// 2. Filter out system messages (they should never be stored in DB)
	var filteredMessages []processors.MastraDBMessage
	for _, msg := range result.Messages {
		if msg.Role != "system" {
			filteredMessages = append(filteredMessages, msg)
		}
	}

	// 3. Merge with incoming messages and messages already in MessageList (avoiding duplicates by ID).
	// This includes messages added by previous processors like SemanticRecall.
	// Ported from TS: const existingMessages = messageList.get.all.db();
	//                  const messageIds = new Set(existingMessages.map(m => m.id).filter(Boolean));
	existingIDs := make(map[string]bool)
	if messageList != nil {
		for _, existing := range messageList.GetAllDB() {
			if existing.ID != "" {
				existingIDs[existing.ID] = true
			}
		}
	}

	var uniqueHistorical []processors.MastraDBMessage
	for _, msg := range filteredMessages {
		if msg.ID == "" || !existingIDs[msg.ID] {
			uniqueHistorical = append(uniqueHistorical, msg)
		}
	}

	// Reverse to chronological order (oldest first) since we fetched DESC.
	for i, j := 0, len(uniqueHistorical)-1; i < j; i, j = i+1, j-1 {
		uniqueHistorical[i], uniqueHistorical[j] = uniqueHistorical[j], uniqueHistorical[i]
	}

	if len(uniqueHistorical) == 0 {
		return nil, messageList, nil, nil
	}

	// Add historical messages with source: 'memory'.
	// Ported from TS: for (const msg of chronologicalMessages) {
	//                    if (msg.role === 'system') continue;
	//                    messageList.add(msg, 'memory');
	//                  }
	for _, msg := range uniqueHistorical {
		if msg.Role == "system" {
			continue // memory should not store system messages
		}
		messageList.Add(msg, "memory")
	}

	return nil, messageList, nil, nil
}

// ProcessInputStep is a no-op for MessageHistory.
func (mh *MessageHistory) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	return nil, nil, nil
}

// ---------------------------------------------------------------------------
// Output processing
// ---------------------------------------------------------------------------

// filterMessagesForPersistence filters messages before persisting to storage:
//  1. Removes streaming tool calls (state === 'partial-call')
//  2. Removes updateWorkingMemory tool invocations
//  3. Strips <working_memory> tags from text content
func (mh *MessageHistory) filterMessagesForPersistence(messages []processors.MastraDBMessage) []processors.MastraDBMessage {
	var result []processors.MastraDBMessage

	for _, m := range messages {
		newMsg := m // shallow copy

		// Strip working memory tags from string content.
		if newMsg.Content.Content != "" {
			newMsg.Content.Content = strings.TrimSpace(
				wmutils.RemoveWorkingMemoryTags(newMsg.Content.Content),
			)
		}

		if len(newMsg.Content.Parts) > 0 {
			var filteredParts []processors.MessagePart
			for _, p := range newMsg.Content.Parts {
				// Filter out streaming tool calls (partial-call is intermediate).
				if p.Type == "tool-invocation" && p.ToolInvocationData != nil &&
					p.ToolInvocationData.State == "partial-call" {
					continue
				}
				// Filter out updateWorkingMemory tool invocations.
				if p.Type == "tool-invocation" && p.ToolInvocationData != nil &&
					p.ToolInvocationData.ToolName == "updateWorkingMemory" {
					continue
				}
				// Strip working memory tags from text parts.
				if p.Type == "text" {
					text := p.Text
					p.Text = strings.TrimSpace(wmutils.RemoveWorkingMemoryTags(text))
				}
				filteredParts = append(filteredParts, p)
			}

			// If all parts were filtered out, skip the whole message.
			if len(filteredParts) == 0 {
				continue
			}
			newMsg.Content.Parts = filteredParts
		}

		result = append(result, newMsg)
	}

	return result
}

// ProcessOutputResult persists new messages to storage.
// Implements OutputProcessorMethods.
func (mh *MessageHistory) ProcessOutputResult(args processors.ProcessOutputResultArgs) (
	[]processors.MastraDBMessage, *processors.MessageList, error,
) {
	messageList := args.MessageList
	rc := args.RequestContext

	ctx := context.Background()

	memCtx := mh.getMemoryContext(rc, messageList)

	// Check if readOnly from memoryConfig.
	parsedCtx := ParseMemoryRequestContext(rc)
	readOnly := parsedCtx != nil && parsedCtx.MemoryConfig != nil && parsedCtx.MemoryConfig.ReadOnly

	if memCtx == nil || readOnly {
		return nil, messageList, nil
	}

	// Get new input and response messages from the MessageList.
	// Ported from TS: const newInput = messageList.get.input.db();
	//                  const newOutput = messageList.get.response.db();
	//                  const messagesToSave = [...newInput, ...newOutput];
	newInput := messageList.GetInputDB()
	newOutput := messageList.GetResponseDB()
	messagesToSave := append(newInput, newOutput...)

	if len(messagesToSave) == 0 {
		return nil, messageList, nil
	}

	threadID := memCtx.Thread.ID
	resourceID := memCtx.ResourceID

	if err := mh.PersistMessages(ctx, messagesToSave, threadID, resourceID); err != nil {
		return nil, nil, err
	}

	// The TS version adds a 10ms delay to avoid timestamp collisions.
	time.Sleep(10 * time.Millisecond)

	return nil, messageList, nil
}

// ProcessOutputStream is a no-op for MessageHistory.
func (mh *MessageHistory) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	return &args.Part, nil
}

// ProcessOutputStep is a no-op for MessageHistory.
func (mh *MessageHistory) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, args.MessageList, nil
}

// PersistMessages persists messages to storage, filtering out partial tool
// calls and working memory tags.  Also ensures the thread exists (creates if
// needed).
//
// This method can be called externally by other processors (e.g.,
// ObservationalMemory) that need to save messages incrementally.
func (mh *MessageHistory) PersistMessages(
	ctx context.Context,
	messages []processors.MastraDBMessage,
	threadID string,
	resourceID string,
) error {
	if len(messages) == 0 {
		return nil
	}

	filtered := mh.filterMessagesForPersistence(messages)
	if len(filtered) == 0 {
		return nil
	}

	// Ensure thread exists (create if needed) before saving messages.
	thread, err := mh.storage.GetThreadByID(ctx, threadID)
	if err != nil {
		return err
	}
	if thread != nil {
		if err := mh.storage.UpdateThread(ctx, UpdateThreadInput{
			ID:       threadID,
			Title:    thread.Title,
			Metadata: thread.Metadata,
		}); err != nil {
			return err
		}
	} else {
		// Auto-create thread if it doesn't exist.
		rid := resourceID
		if rid == "" {
			rid = threadID
		}
		now := time.Now()
		if err := mh.storage.SaveThread(ctx, StorageThread{
			ID:         threadID,
			ResourceID: rid,
			Title:      "",
			Metadata:   map[string]any{},
			CreatedAt:  now,
			UpdatedAt:  now,
		}); err != nil {
			return err
		}
	}

	// Persist messages after thread is guaranteed to exist.
	return mh.storage.SaveMessages(ctx, filtered)
}
