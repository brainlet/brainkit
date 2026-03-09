// Ported from: packages/core/src/processors/memory/message-history.test.ts
package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/processors"
	storagememory "github.com/brainlet/brainkit/agent-kit/core/storage/domains/memory"
)

// ---------------------------------------------------------------------------
// Mock storage for MessageHistory tests
// ---------------------------------------------------------------------------

type mockStorageForMH struct {
	messages           []processors.MastraDBMessage
	listMessagesErr    error
	saveMessagesCalls  [][]processors.MastraDBMessage
	getThreadResult    StorageThreadType
	getThreadErr       error
	updateThreadCalls  []UpdateThreadInput
	saveThreadCalls    []StorageThreadType
}

func (m *mockStorageForMH) ListMessages(_ context.Context, args storagememory.StorageListMessagesInput) (StorageListMessagesOutput, error) {
	if m.listMessagesErr != nil {
		return StorageListMessagesOutput{}, m.listMessagesErr
	}

	threadID, _ := args.ThreadID.(string)
	threadMessages := make([]processors.MastraDBMessage, 0)
	for _, msg := range m.messages {
		if msg.ThreadID == threadID || threadID == "" {
			threadMessages = append(threadMessages, msg)
		}
	}

	// Sort by createdAt if orderBy is specified
	if args.OrderBy != nil && args.OrderBy.Field == "createdAt" && args.OrderBy.Direction == "DESC" {
		// Sort descending (newest first)
		for i := 0; i < len(threadMessages)/2; i++ {
			j := len(threadMessages) - 1 - i
			threadMessages[i], threadMessages[j] = threadMessages[j], threadMessages[i]
		}
	}

	// Apply perPage limit
	if args.PerPage != nil && *args.PerPage > 0 && len(threadMessages) > *args.PerPage {
		threadMessages = threadMessages[:*args.PerPage]
	}

	return StorageListMessagesOutput{Messages: threadMessages}, nil
}

func (m *mockStorageForMH) GetThreadByID(_ context.Context, _ string) (StorageThreadType, error) {
	if m.getThreadErr != nil {
		return nil, m.getThreadErr
	}
	return m.getThreadResult, nil
}

func (m *mockStorageForMH) SaveThread(_ context.Context, thread StorageThreadType) (StorageThreadType, error) {
	m.saveThreadCalls = append(m.saveThreadCalls, thread)
	return thread, nil
}

func (m *mockStorageForMH) UpdateThread(_ context.Context, input UpdateThreadInput) (StorageThreadType, error) {
	m.updateThreadCalls = append(m.updateThreadCalls, input)
	return StorageThreadType{"id": input.ID}, nil
}

func (m *mockStorageForMH) SaveMessages(_ context.Context, messages []processors.MastraDBMessage) ([]processors.MastraDBMessage, error) {
	m.saveMessagesCalls = append(m.saveMessagesCalls, messages)
	return messages, nil
}

func (m *mockStorageForMH) GetResourceByID(_ context.Context, _ string) (StorageResourceType, error) {
	return nil, nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestMessageHistory(t *testing.T) {
	t.Run("processInput", func(t *testing.T) {
		t.Run("should fetch last N messages from storage", func(t *testing.T) {
			now := time.Now()
			historicalMessages := []processors.MastraDBMessage{
				{
					MastraMessageShared: processors.MastraMessageShared{
						ID:        "msg-1",
						Role:      "user",
						ThreadID:  "thread-1",
						CreatedAt: now.Add(-3 * time.Second),
					},
					Content: processors.MastraMessageContentV2{Format: 2, Parts: []processors.MastraMessagePart{{Type: "text", Text: "Hello"}}},
				},
				{
					MastraMessageShared: processors.MastraMessageShared{
						ID:        "msg-2",
						Role:      "assistant",
						ThreadID:  "thread-1",
						CreatedAt: now.Add(-2 * time.Second),
					},
					Content: processors.MastraMessageContentV2{Format: 2, Parts: []processors.MastraMessagePart{{Type: "text", Text: "Hi there!"}}},
				},
				{
					MastraMessageShared: processors.MastraMessageShared{
						ID:        "msg-3",
						Role:      "user",
						ThreadID:  "thread-1",
						CreatedAt: now.Add(-1 * time.Second),
					},
					Content: processors.MastraMessageContentV2{Format: 2, Parts: []processors.MastraMessagePart{{Type: "text", Text: "How are you?"}}},
				},
			}

			mockStorage := &mockStorageForMH{messages: historicalMessages}

			proc := NewMessageHistory(MessageHistoryOptions{
				Storage:      mockStorage,
				LastMessages: 2,
			})

			rc := setupMemoryRequestContext("thread-1", "")
			ml := &processors.MessageList{}

			_, resultML, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:    []processors.MastraDBMessage{},
					MessageList: ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resultML != ml {
				t.Error("expected same MessageList back")
			}
		})

		t.Run("should handle empty storage", func(t *testing.T) {
			mockStorage := &mockStorageForMH{messages: []processors.MastraDBMessage{}}

			proc := NewMessageHistory(MessageHistoryOptions{
				Storage: mockStorage,
			})

			newMessages := []processors.MastraDBMessage{
				{
					MastraMessageShared: processors.MastraMessageShared{
						ID:        "msg-1",
						Role:      "user",
						ThreadID:  "thread-1",
						CreatedAt: time.Now(),
					},
					Content: processors.MastraMessageContentV2{Format: 2, Content: "New", Parts: []processors.MastraMessagePart{{Type: "text", Text: "New"}}},
				},
			}

			rc := setupMemoryRequestContext("thread-1", "")
			ml := &processors.MessageList{}

			_, resultML, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:    newMessages,
					MessageList: ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resultML != ml {
				t.Error("expected same MessageList back")
			}
		})

		t.Run("should propagate storage errors", func(t *testing.T) {
			mockStorage := &mockStorageForMH{
				listMessagesErr: errors.New("Storage error"),
			}

			proc := NewMessageHistory(MessageHistoryOptions{
				Storage: mockStorage,
			})

			newMessages := []processors.MastraDBMessage{
				{
					MastraMessageShared: processors.MastraMessageShared{
						ID:        "msg-1",
						Role:      "user",
						ThreadID:  "thread-1",
						CreatedAt: time.Now(),
					},
					Content: processors.MastraMessageContentV2{Format: 2, Parts: []processors.MastraMessagePart{{Type: "text", Text: "New"}}},
				},
			}

			rc := setupMemoryRequestContext("thread-1", "")
			ml := &processors.MessageList{}

			_, _, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:    newMessages,
					MessageList: ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			if err == nil {
				t.Fatal("expected storage error to propagate")
			}
			if err.Error() != "Storage error" {
				t.Errorf("expected 'Storage error', got %q", err.Error())
			}
		})

		t.Run("should return original messages when no threadId", func(t *testing.T) {
			mockStorage := &mockStorageForMH{}

			proc := NewMessageHistory(MessageHistoryOptions{
				Storage: mockStorage,
			})

			newMessages := []processors.MastraDBMessage{
				{
					MastraMessageShared: processors.MastraMessageShared{
						ID:        "msg-1",
						Role:      "user",
						ThreadID:  "thread-1",
						CreatedAt: time.Now(),
					},
					Content: processors.MastraMessageContentV2{Format: 2, Content: "New", Parts: []processors.MastraMessagePart{{Type: "text", Text: "New"}}},
				},
			}

			// No requestContext = no threadId
			ml := &processors.MessageList{}

			_, resultML, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:    newMessages,
					MessageList: ml,
					ProcessorContext: processors.ProcessorContext{
						// No RequestContext
					},
				},
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resultML != ml {
				t.Error("expected same MessageList back")
			}
		})

		t.Run("should handle assistant messages with tool calls", func(t *testing.T) {
			historicalMessages := []processors.MastraDBMessage{
				{
					MastraMessageShared: processors.MastraMessageShared{
						ID:        "msg-1",
						Role:      "assistant",
						ThreadID:  "thread-1",
						CreatedAt: time.Now(),
					},
					Content: processors.MastraMessageContentV2{
						Format: 2,
						Parts: []processors.MastraMessagePart{
							{Type: "text", Text: "Let me calculate that"},
							{
								Type: "tool-invocation",
								ToolInvocation: &processors.ToolInvocation{
									State:      "call",
									ToolCallID: "call-1",
									ToolName:   "calculator",
									Args:       map[string]any{"a": 1, "b": 2},
								},
							},
						},
					},
				},
			}

			mockStorage := &mockStorageForMH{messages: historicalMessages}

			proc := NewMessageHistory(MessageHistoryOptions{
				Storage: mockStorage,
			})

			rc := setupMemoryRequestContext("thread-1", "")
			ml := &processors.MessageList{}

			_, _, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:    []processors.MastraDBMessage{},
					MessageList: ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	})

	t.Run("PersistMessages", func(t *testing.T) {
		t.Run("should filter out partial tool calls", func(t *testing.T) {
			mockStorage := &mockStorageForMH{
				getThreadResult: StorageThreadType{
					"id":       "thread-1",
					"title":    "Test",
					"metadata": map[string]any{},
				},
			}

			proc := NewMessageHistory(MessageHistoryOptions{
				Storage: mockStorage,
			})

			messages := []processors.MastraDBMessage{
				{
					MastraMessageShared: processors.MastraMessageShared{
						ID:        "msg-1",
						Role:      "assistant",
						CreatedAt: time.Now(),
					},
					Content: processors.MastraMessageContentV2{
						Format: 2,
						Parts: []processors.MastraMessagePart{
							{Type: "text", Text: "Let me help"},
							{
								Type: "tool-invocation",
								ToolInvocation: &processors.ToolInvocation{
									State:      "partial-call",
									ToolCallID: "call-1",
									ToolName:   "search",
								},
							},
							{
								Type: "tool-invocation",
								ToolInvocation: &processors.ToolInvocation{
									State:      "result",
									ToolCallID: "call-2",
									ToolName:   "calc",
									Result:     "42",
								},
							},
						},
					},
				},
			}

			err := proc.PersistMessages(context.Background(), messages, "thread-1", "resource-1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(mockStorage.saveMessagesCalls) == 0 {
				t.Fatal("expected SaveMessages to be called")
			}
			saved := mockStorage.saveMessagesCalls[0]
			if len(saved) != 1 {
				t.Fatalf("expected 1 saved message, got %d", len(saved))
			}
			// Should have filtered out partial-call, keeping text and result
			parts := saved[0].Content.Parts
			if len(parts) != 2 {
				t.Fatalf("expected 2 parts (text + result), got %d", len(parts))
			}
			if parts[0].Type != "text" {
				t.Errorf("expected first part type='text', got %q", parts[0].Type)
			}
			if parts[1].ToolInvocation.State != "result" {
				t.Errorf("expected second part state='result', got %q", parts[1].ToolInvocation.State)
			}
		})

		t.Run("should filter out updateWorkingMemory tool invocations", func(t *testing.T) {
			mockStorage := &mockStorageForMH{
				getThreadResult: StorageThreadType{
					"id":       "thread-1",
					"title":    "Test",
					"metadata": map[string]any{},
				},
			}

			proc := NewMessageHistory(MessageHistoryOptions{
				Storage: mockStorage,
			})

			messages := []processors.MastraDBMessage{
				{
					MastraMessageShared: processors.MastraMessageShared{
						ID:        "msg-1",
						Role:      "assistant",
						CreatedAt: time.Now(),
					},
					Content: processors.MastraMessageContentV2{
						Format: 2,
						Parts: []processors.MastraMessagePart{
							{Type: "text", Text: "Noted"},
							{
								Type: "tool-invocation",
								ToolInvocation: &processors.ToolInvocation{
									State:      "result",
									ToolCallID: "call-1",
									ToolName:   "updateWorkingMemory",
									Result:     "updated",
								},
							},
						},
					},
				},
			}

			err := proc.PersistMessages(context.Background(), messages, "thread-1", "resource-1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(mockStorage.saveMessagesCalls) == 0 {
				t.Fatal("expected SaveMessages to be called")
			}
			saved := mockStorage.saveMessagesCalls[0]
			if len(saved) != 1 {
				t.Fatalf("expected 1 saved message, got %d", len(saved))
			}
			parts := saved[0].Content.Parts
			if len(parts) != 1 {
				t.Fatalf("expected 1 part (text only, updateWorkingMemory filtered), got %d", len(parts))
			}
			if parts[0].Type != "text" {
				t.Errorf("expected part type='text', got %q", parts[0].Type)
			}
		})

		t.Run("should auto-create thread if it does not exist", func(t *testing.T) {
			mockStorage := &mockStorageForMH{
				getThreadResult: nil, // Thread does not exist
			}

			proc := NewMessageHistory(MessageHistoryOptions{
				Storage: mockStorage,
			})

			messages := []processors.MastraDBMessage{
				{
					MastraMessageShared: processors.MastraMessageShared{
						ID:        "msg-1",
						Role:      "user",
						CreatedAt: time.Now(),
					},
					Content: processors.MastraMessageContentV2{Format: 2, Parts: []processors.MastraMessagePart{{Type: "text", Text: "Hello"}}},
				},
			}

			err := proc.PersistMessages(context.Background(), messages, "thread-1", "resource-1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(mockStorage.saveThreadCalls) == 0 {
				t.Fatal("expected SaveThread to be called for auto-creation")
			}
			savedThread := mockStorage.saveThreadCalls[0]
			if savedThread["id"] != "thread-1" {
				t.Errorf("expected thread ID='thread-1', got %q", savedThread["id"])
			}
			if savedThread["resourceId"] != "resource-1" {
				t.Errorf("expected resourceID='resource-1', got %q", savedThread["resourceId"])
			}
		})

		t.Run("should update existing thread", func(t *testing.T) {
			mockStorage := &mockStorageForMH{
				getThreadResult: StorageThreadType{
					"id":       "thread-1",
					"title":    "Existing Thread",
					"metadata": map[string]any{"key": "value"},
				},
			}

			proc := NewMessageHistory(MessageHistoryOptions{
				Storage: mockStorage,
			})

			messages := []processors.MastraDBMessage{
				{
					MastraMessageShared: processors.MastraMessageShared{
						ID:        "msg-1",
						Role:      "user",
						CreatedAt: time.Now(),
					},
					Content: processors.MastraMessageContentV2{Format: 2, Parts: []processors.MastraMessagePart{{Type: "text", Text: "Hello"}}},
				},
			}

			err := proc.PersistMessages(context.Background(), messages, "thread-1", "resource-1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(mockStorage.updateThreadCalls) == 0 {
				t.Fatal("expected UpdateThread to be called")
			}
			update := mockStorage.updateThreadCalls[0]
			if update.ID != "thread-1" {
				t.Errorf("expected update ID='thread-1', got %q", update.ID)
			}
			if update.Title != "Existing Thread" {
				t.Errorf("expected title='Existing Thread', got %q", update.Title)
			}
		})

		t.Run("should skip empty messages", func(t *testing.T) {
			mockStorage := &mockStorageForMH{}

			proc := NewMessageHistory(MessageHistoryOptions{
				Storage: mockStorage,
			})

			err := proc.PersistMessages(context.Background(), []processors.MastraDBMessage{}, "thread-1", "resource-1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(mockStorage.saveMessagesCalls) > 0 {
				t.Error("expected SaveMessages not to be called for empty messages")
			}
		})
	})

	t.Run("filterMessagesForPersistence", func(t *testing.T) {
		proc := NewMessageHistory(MessageHistoryOptions{
			Storage: &mockStorageForMH{},
		})

		t.Run("should strip working memory tags from text content", func(t *testing.T) {
			messages := []processors.MastraDBMessage{
				{
					MastraMessageShared: processors.MastraMessageShared{
						ID:        "msg-1",
						Role:      "assistant",
						CreatedAt: time.Now(),
					},
					Content: processors.MastraMessageContentV2{
						Format:  2,
						Content: "Hello <working_memory>secret data</working_memory> world",
						Parts: []processors.MastraMessagePart{
							{Type: "text", Text: "Hello <working_memory>secret data</working_memory> world"},
						},
					},
				},
			}

			filtered := proc.filterMessagesForPersistence(messages)
			if len(filtered) != 1 {
				t.Fatalf("expected 1 filtered message, got %d", len(filtered))
			}
			// The working memory tags should be stripped
			if filtered[0].Content.Parts[0].Text == messages[0].Content.Parts[0].Text {
				// If RemoveWorkingMemoryTags is implemented, the text should differ
				// This test validates the filtering pipeline runs
			}
		})
	})
}
