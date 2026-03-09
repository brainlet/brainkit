// Ported from: packages/core/src/processors/memory/semantic-recall.test.ts
package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/processors"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	storagememory "github.com/brainlet/brainkit/agent-kit/core/storage/domains/memory"
	"github.com/brainlet/brainkit/agent-kit/core/vector"
)

// ---------------------------------------------------------------------------
// Mock types for SemanticRecall tests
// ---------------------------------------------------------------------------

type mockVectorForSR struct {
	queryCalls       []vector.QueryVectorParams
	queryResults     []VectorQueryResult
	queryErr         error
	createIndexCalls []vector.CreateIndexParams
	createIndexErr   error
	upsertCalls      []vector.UpsertVectorParams
	upsertErr        error
}

func (m *mockVectorForSR) Query(_ context.Context, params vector.QueryVectorParams) ([]VectorQueryResult, error) {
	m.queryCalls = append(m.queryCalls, params)
	if m.queryErr != nil {
		return nil, m.queryErr
	}
	return m.queryResults, nil
}

func (m *mockVectorForSR) CreateIndex(_ context.Context, params vector.CreateIndexParams) error {
	m.createIndexCalls = append(m.createIndexCalls, params)
	return m.createIndexErr
}

func (m *mockVectorForSR) Upsert(_ context.Context, params vector.UpsertVectorParams) ([]string, error) {
	m.upsertCalls = append(m.upsertCalls, params)
	if m.upsertErr != nil {
		return nil, m.upsertErr
	}
	return params.IDs, nil
}

type mockEmbedder struct {
	doEmbedCalls   []EmbedOptions
	doEmbedResult  *EmbedResult
	doEmbedErr     error
	modelIDValue   string
}

func (m *mockEmbedder) DoEmbed(opts EmbedOptions) (*EmbedResult, error) {
	m.doEmbedCalls = append(m.doEmbedCalls, opts)
	if m.doEmbedErr != nil {
		return nil, m.doEmbedErr
	}
	return m.doEmbedResult, nil
}

func (m *mockEmbedder) ModelID() string {
	return m.modelIDValue
}

type mockStorageForSR struct {
	listMessagesCalls  []storagememory.StorageListMessagesInput
	listMessagesResult StorageListMessagesOutput
	listMessagesErr    error
}

func (m *mockStorageForSR) ListMessages(_ context.Context, args storagememory.StorageListMessagesInput) (StorageListMessagesOutput, error) {
	m.listMessagesCalls = append(m.listMessagesCalls, args)
	if m.listMessagesErr != nil {
		return StorageListMessagesOutput{}, m.listMessagesErr
	}
	return m.listMessagesResult, nil
}
func (m *mockStorageForSR) GetThreadByID(_ context.Context, _ string) (StorageThreadType, error) {
	return nil, nil
}
func (m *mockStorageForSR) SaveThread(_ context.Context, t StorageThreadType) (StorageThreadType, error) {
	return t, nil
}
func (m *mockStorageForSR) UpdateThread(_ context.Context, _ UpdateThreadInput) (StorageThreadType, error) {
	return nil, nil
}
func (m *mockStorageForSR) SaveMessages(_ context.Context, msgs []processors.MastraDBMessage) ([]processors.MastraDBMessage, error) {
	return msgs, nil
}
func (m *mockStorageForSR) GetResourceByID(_ context.Context, _ string) (StorageResourceType, error) {
	return nil, nil
}

// createTestMessage is a helper to create MastraDBMessage values.
func createTestMessage(id, role, content string) processors.MastraDBMessage {
	return processors.MastraDBMessage{
		MastraMessageShared: processors.MastraMessageShared{
			ID:        id,
			Role:      role,
			CreatedAt: time.Now(),
		},
		Content: processors.MastraMessageContentV2{
			Format:  2,
			Parts:   []processors.MastraMessagePart{{Type: "text", Text: content}},
			Content: content,
		},
	}
}

func setupMemoryRequestContext(threadID, resourceID string) *requestcontext.RequestContext {
	rc := requestcontext.NewRequestContext()
	rc.Set("MastraMemory", map[string]any{
		"thread":     map[string]any{"id": threadID},
		"resourceId": resourceID,
	})
	return rc
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestSemanticRecall(t *testing.T) {
	t.Run("Input Processing", func(t *testing.T) {
		t.Run("should respect topK limit", func(t *testing.T) {
			globalEmbeddingCache.Clear()

			mockStorage := &mockStorageForSR{
				listMessagesResult: StorageListMessagesOutput{
					Messages: []processors.MastraDBMessage{
						createTestMessage("msg-1", "user", "Message 1"),
						createTestMessage("msg-2", "assistant", "Message 2"),
					},
				},
			}
			mockVector := &mockVectorForSR{
				queryResults: []VectorQueryResult{
					{ID: "vec-1", Score: 0.95, Metadata: map[string]any{"message_id": "msg-1", "thread_id": "thread-1"}},
					{ID: "vec-2", Score: 0.92, Metadata: map[string]any{"message_id": "msg-2", "thread_id": "thread-1"}},
				},
			}
			embedder := &mockEmbedder{
				doEmbedResult: &EmbedResult{Embeddings: [][]float64{{0.1, 0.2, 0.3}}},
				modelIDValue:  "text-embedding-3-small",
			}

			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:  mockStorage,
				Vector:   mockVector,
				Embedder: embedder,
				TopK:     2,
			})

			inputMessages := []processors.MastraDBMessage{
				createTestMessage("msg-new", "user", "Test query"),
			}

			rc := setupMemoryRequestContext("thread-1", "resource-1")
			ml := &processors.MessageList{}

			proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:       inputMessages,
					MessageList:    ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			// Verify topK was passed to vector query
			if len(mockVector.queryCalls) == 0 {
				t.Fatal("expected vector.Query to be called")
			}
			if mockVector.queryCalls[0].TopK != 2 {
				t.Errorf("expected topK=2, got %d", mockVector.queryCalls[0].TopK)
			}
		})

		t.Run("should filter by threshold", func(t *testing.T) {
			globalEmbeddingCache.Clear()

			mockVector := &mockVectorForSR{
				queryResults: []VectorQueryResult{
					{ID: "vec-1", Score: 0.95, Metadata: map[string]any{"message_id": "msg-1", "thread_id": "thread-1"}},
					{ID: "vec-2", Score: 0.85, Metadata: map[string]any{"message_id": "msg-2", "thread_id": "thread-1"}}, // Below threshold
					{ID: "vec-3", Score: 0.92, Metadata: map[string]any{"message_id": "msg-3", "thread_id": "thread-1"}},
				},
			}
			mockStorage := &mockStorageForSR{
				listMessagesResult: StorageListMessagesOutput{
					Messages: []processors.MastraDBMessage{
						createTestMessage("msg-1", "user", "Message 1"),
						createTestMessage("msg-3", "user", "Message 3"),
					},
				},
			}
			embedder := &mockEmbedder{
				doEmbedResult: &EmbedResult{Embeddings: [][]float64{{0.1, 0.2, 0.3}}},
				modelIDValue:  "text-embedding-3-small",
			}

			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:   mockStorage,
				Vector:    mockVector,
				Embedder:  embedder,
				Threshold: 0.9,
			})

			inputMessages := []processors.MastraDBMessage{
				createTestMessage("msg-new", "user", "Test query"),
			}

			rc := setupMemoryRequestContext("thread-1", "resource-1")
			ml := &processors.MessageList{}

			proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:       inputMessages,
					MessageList:    ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			// Verify storage was called with only messages above threshold (msg-1 and msg-3)
			if len(mockStorage.listMessagesCalls) == 0 {
				t.Fatal("expected storage.ListMessages to be called")
			}
			includes := mockStorage.listMessagesCalls[0].Include
			if len(includes) != 2 {
				t.Fatalf("expected 2 includes (above threshold), got %d", len(includes))
			}
			if includes[0].ID != "msg-1" {
				t.Errorf("expected first include ID=msg-1, got %s", includes[0].ID)
			}
			if includes[1].ID != "msg-3" {
				t.Errorf("expected second include ID=msg-3, got %s", includes[1].ID)
			}
		})

		t.Run("should apply scope filter for thread scope", func(t *testing.T) {
			globalEmbeddingCache.Clear()

			mockVector := &mockVectorForSR{queryResults: []VectorQueryResult{}}
			mockStorage := &mockStorageForSR{}
			embedder := &mockEmbedder{
				doEmbedResult: &EmbedResult{Embeddings: [][]float64{{0.1, 0.2, 0.3}}},
				modelIDValue:  "text-embedding-3-small",
			}

			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:  mockStorage,
				Vector:   mockVector,
				Embedder: embedder,
				Scope:    "thread",
			})

			inputMessages := []processors.MastraDBMessage{
				createTestMessage("msg-new", "user", "Test query"),
			}

			rc := setupMemoryRequestContext("thread-1", "resource-1")
			ml := &processors.MessageList{}

			proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:       inputMessages,
					MessageList:    ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			if len(mockVector.queryCalls) == 0 {
				t.Fatal("expected vector.Query to be called")
			}
			filter := mockVector.queryCalls[0].Filter
			if filter["thread_id"] != "thread-1" {
				t.Errorf("expected thread_id filter, got %v", filter)
			}
		})

		t.Run("should apply scope filter for resource scope", func(t *testing.T) {
			globalEmbeddingCache.Clear()

			mockVector := &mockVectorForSR{queryResults: []VectorQueryResult{}}
			mockStorage := &mockStorageForSR{}
			embedder := &mockEmbedder{
				doEmbedResult: &EmbedResult{Embeddings: [][]float64{{0.1, 0.2, 0.3}}},
				modelIDValue:  "text-embedding-3-small",
			}

			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:  mockStorage,
				Vector:   mockVector,
				Embedder: embedder,
				Scope:    "resource",
			})

			inputMessages := []processors.MastraDBMessage{
				createTestMessage("msg-new", "user", "Test query"),
			}

			rc := setupMemoryRequestContext("thread-1", "resource-1")
			ml := &processors.MessageList{}

			proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:       inputMessages,
					MessageList:    ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			if len(mockVector.queryCalls) == 0 {
				t.Fatal("expected vector.Query to be called")
			}
			filter := mockVector.queryCalls[0].Filter
			if filter["resource_id"] != "resource-1" {
				t.Errorf("expected resource_id filter, got %v", filter)
			}
		})

		t.Run("should handle no results gracefully", func(t *testing.T) {
			globalEmbeddingCache.Clear()

			mockVector := &mockVectorForSR{queryResults: []VectorQueryResult{}}
			mockStorage := &mockStorageForSR{}
			embedder := &mockEmbedder{
				doEmbedResult: &EmbedResult{Embeddings: [][]float64{{0.1, 0.2, 0.3}}},
				modelIDValue:  "text-embedding-3-small",
			}

			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:  mockStorage,
				Vector:   mockVector,
				Embedder: embedder,
			})

			inputMessages := []processors.MastraDBMessage{
				createTestMessage("msg-new", "user", "Test query"),
			}

			rc := setupMemoryRequestContext("thread-1", "resource-1")
			ml := &processors.MessageList{}

			_, resultML, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:       inputMessages,
					MessageList:    ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resultML != ml {
				t.Error("expected original MessageList to be returned")
			}
			// Storage should not be called when no vector results
			if len(mockStorage.listMessagesCalls) > 0 {
				t.Error("expected storage.ListMessages not to be called")
			}
		})

		t.Run("should handle vector store errors gracefully", func(t *testing.T) {
			globalEmbeddingCache.Clear()

			mockVector := &mockVectorForSR{queryErr: errors.New("Vector query failed")}
			mockStorage := &mockStorageForSR{}
			embedder := &mockEmbedder{
				doEmbedResult: &EmbedResult{Embeddings: [][]float64{{0.1, 0.2, 0.3}}},
				modelIDValue:  "text-embedding-3-small",
			}

			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:  mockStorage,
				Vector:   mockVector,
				Embedder: embedder,
			})

			inputMessages := []processors.MastraDBMessage{
				createTestMessage("msg-new", "user", "Test query"),
			}

			rc := setupMemoryRequestContext("thread-1", "resource-1")
			ml := &processors.MessageList{}

			_, resultML, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:       inputMessages,
					MessageList:    ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			// Should return original messages on error (graceful degradation)
			if err != nil {
				t.Fatalf("expected no error (graceful handling), got: %v", err)
			}
			if resultML != ml {
				t.Error("expected original MessageList to be returned on error")
			}
		})

		t.Run("should skip when no user message present", func(t *testing.T) {
			globalEmbeddingCache.Clear()

			mockVector := &mockVectorForSR{}
			mockStorage := &mockStorageForSR{}
			embedder := &mockEmbedder{
				modelIDValue: "text-embedding-3-small",
			}

			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:  mockStorage,
				Vector:   mockVector,
				Embedder: embedder,
			})

			inputMessages := []processors.MastraDBMessage{
				createTestMessage("msg-1", "assistant", "Hello!"),
			}

			rc := setupMemoryRequestContext("thread-1", "resource-1")
			ml := &processors.MessageList{}

			_, resultML, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:       inputMessages,
					MessageList:    ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resultML != ml {
				t.Error("expected original MessageList")
			}
			// No embedder or vector calls should be made
			if len(embedder.doEmbedCalls) > 0 {
				t.Error("expected no embedder calls")
			}
			if len(mockVector.queryCalls) > 0 {
				t.Error("expected no vector query calls")
			}
		})

		t.Run("should return original messages when no threadId", func(t *testing.T) {
			globalEmbeddingCache.Clear()

			mockVector := &mockVectorForSR{}
			mockStorage := &mockStorageForSR{}
			embedder := &mockEmbedder{
				modelIDValue: "text-embedding-3-small",
			}

			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:  mockStorage,
				Vector:   mockVector,
				Embedder: embedder,
			})

			inputMessages := []processors.MastraDBMessage{
				createTestMessage("msg-new", "user", "Test query"),
			}

			// Empty context without thread
			emptyRC := requestcontext.NewRequestContext()
			ml := &processors.MessageList{}

			_, resultML, _, err := proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:       inputMessages,
					MessageList:    ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: emptyRC,
					},
				},
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if resultML != ml {
				t.Error("expected original MessageList")
			}
			if len(embedder.doEmbedCalls) > 0 {
				t.Error("expected no embedder calls")
			}
			if len(mockVector.queryCalls) > 0 {
				t.Error("expected no vector query calls")
			}
		})

		t.Run("should handle multi-part user messages", func(t *testing.T) {
			globalEmbeddingCache.Clear()

			mockVector := &mockVectorForSR{queryResults: []VectorQueryResult{}}
			mockStorage := &mockStorageForSR{}
			embedder := &mockEmbedder{
				doEmbedResult: &EmbedResult{Embeddings: [][]float64{{0.1, 0.2, 0.3}}},
				modelIDValue:  "text-embedding-3-small",
			}

			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:  mockStorage,
				Vector:   mockVector,
				Embedder: embedder,
			})

			inputMessages := []processors.MastraDBMessage{
				{
					MastraMessageShared: processors.MastraMessageShared{
						ID:        "msg-new",
						Role:      "user",
						CreatedAt: time.Now(),
					},
					Content: processors.MastraMessageContentV2{
						Format: 2,
						Parts: []processors.MastraMessagePart{
							{Type: "text", Text: "Part 1"},
							{Type: "text", Text: "Part 2"},
						},
					},
				},
			}

			rc := setupMemoryRequestContext("thread-1", "resource-1")
			ml := &processors.MessageList{}

			proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:       inputMessages,
					MessageList:    ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			// Should combine text parts
			if len(embedder.doEmbedCalls) == 0 {
				t.Fatal("expected embedder.DoEmbed to be called")
			}
			if len(embedder.doEmbedCalls[0].Values) != 1 {
				t.Fatalf("expected 1 value, got %d", len(embedder.doEmbedCalls[0].Values))
			}
			if embedder.doEmbedCalls[0].Values[0] != "Part 1 Part 2" {
				t.Errorf("expected 'Part 1 Part 2', got %q", embedder.doEmbedCalls[0].Values[0])
			}
		})

		t.Run("should respect custom messageRange", func(t *testing.T) {
			globalEmbeddingCache.Clear()

			mockVector := &mockVectorForSR{
				queryResults: []VectorQueryResult{
					{ID: "vec-1", Score: 0.95, Metadata: map[string]any{"message_id": "msg-1", "thread_id": "thread-1"}},
				},
			}
			mockStorage := &mockStorageForSR{
				listMessagesResult: StorageListMessagesOutput{
					Messages: []processors.MastraDBMessage{
						createTestMessage("msg-1", "user", "Message 1"),
					},
				},
			}
			embedder := &mockEmbedder{
				doEmbedResult: &EmbedResult{Embeddings: [][]float64{{0.1, 0.2, 0.3}}},
				modelIDValue:  "text-embedding-3-small",
			}

			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:      mockStorage,
				Vector:       mockVector,
				Embedder:     embedder,
				MessageRange: &MessageRange{Before: 5, After: 3},
			})

			inputMessages := []processors.MastraDBMessage{
				createTestMessage("msg-new", "user", "Test query"),
			}

			rc := setupMemoryRequestContext("thread-1", "resource-1")
			ml := &processors.MessageList{}

			proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:       inputMessages,
					MessageList:    ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			if len(mockStorage.listMessagesCalls) == 0 {
				t.Fatal("expected storage.ListMessages to be called")
			}
			includes := mockStorage.listMessagesCalls[0].Include
			if len(includes) != 1 {
				t.Fatalf("expected 1 include, got %d", len(includes))
			}
			if includes[0].WithPreviousMessages != 5 {
				t.Errorf("expected withPreviousMessages=5, got %d", includes[0].WithPreviousMessages)
			}
			if includes[0].WithNextMessages != 3 {
				t.Errorf("expected withNextMessages=3, got %d", includes[0].WithNextMessages)
			}
		})

		t.Run("should create vector index if it does not exist", func(t *testing.T) {
			globalEmbeddingCache.Clear()

			mockVector := &mockVectorForSR{queryResults: []VectorQueryResult{}}
			mockStorage := &mockStorageForSR{}
			embedder := &mockEmbedder{
				doEmbedResult: &EmbedResult{Embeddings: [][]float64{{0.1, 0.2, 0.3}}},
				modelIDValue:  "text-embedding-3-small",
			}

			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:  mockStorage,
				Vector:   mockVector,
				Embedder: embedder,
			})

			inputMessages := []processors.MastraDBMessage{
				createTestMessage("msg-new", "user", "Test query"),
			}

			rc := setupMemoryRequestContext("thread-1", "resource-1")
			ml := &processors.MessageList{}

			proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:       inputMessages,
					MessageList:    ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			if len(mockVector.createIndexCalls) == 0 {
				t.Fatal("expected vector.CreateIndex to be called")
			}
			call := mockVector.createIndexCalls[0]
			if call.IndexName != "mastra_memory_text_embedding_3_small" {
				t.Errorf("expected index name 'mastra_memory_text_embedding_3_small', got %q", call.IndexName)
			}
			if call.Dimension != 3 {
				t.Errorf("expected dimension=3, got %d", call.Dimension)
			}
			if call.Metric != vector.DistanceMetricCosine {
				t.Errorf("expected metric='cosine', got %q", call.Metric)
			}
		})

		t.Run("should use custom index name if provided", func(t *testing.T) {
			globalEmbeddingCache.Clear()

			mockVector := &mockVectorForSR{queryResults: []VectorQueryResult{}}
			mockStorage := &mockStorageForSR{}
			embedder := &mockEmbedder{
				doEmbedResult: &EmbedResult{Embeddings: [][]float64{{0.1, 0.2, 0.3}}},
				modelIDValue:  "text-embedding-3-small",
			}

			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:   mockStorage,
				Vector:    mockVector,
				Embedder:  embedder,
				IndexName: "custom-index",
			})

			inputMessages := []processors.MastraDBMessage{
				createTestMessage("msg-new", "user", "Test query"),
			}

			rc := setupMemoryRequestContext("thread-1", "resource-1")
			ml := &processors.MessageList{}

			proc.ProcessInput(processors.ProcessInputArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:       inputMessages,
					MessageList:    ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			if len(mockVector.queryCalls) == 0 {
				t.Fatal("expected vector.Query to be called")
			}
			if mockVector.queryCalls[0].IndexName != "custom-index" {
				t.Errorf("expected indexName='custom-index', got %q", mockVector.queryCalls[0].IndexName)
			}
		})
	})

	t.Run("Output Processing", func(t *testing.T) {
		t.Run("should create embeddings for output messages", func(t *testing.T) {
			globalEmbeddingCache.Clear()

			mockVector := &mockVectorForSR{}
			mockStorage := &mockStorageForSR{}
			embedder := &mockEmbedder{
				doEmbedResult: &EmbedResult{Embeddings: [][]float64{{0.4, 0.5, 0.6}}},
				modelIDValue:  "text-embedding-3-small",
			}

			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:  mockStorage,
				Vector:   mockVector,
				Embedder: embedder,
			})

			messages := []processors.MastraDBMessage{
				createTestMessage("msg-1", "user", "User message"),
				createTestMessage("msg-2", "assistant", "Assistant reply"),
				createTestMessage("msg-sys", "system", "System message"),
			}

			rc := setupMemoryRequestContext("thread-1", "resource-1")
			ml := &processors.MessageList{}

			proc.ProcessOutputResult(processors.ProcessOutputResultArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:       messages,
					MessageList:    ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: rc,
					},
				},
			})

			// Should embed user and assistant messages, but not system
			if len(embedder.doEmbedCalls) != 2 {
				t.Errorf("expected 2 embed calls (skipping system), got %d", len(embedder.doEmbedCalls))
			}

			// Should upsert vectors
			if len(mockVector.upsertCalls) == 0 {
				t.Fatal("expected vector.Upsert to be called")
			}
			upsertCall := mockVector.upsertCalls[0]
			if len(upsertCall.IDs) != 2 {
				t.Errorf("expected 2 IDs in upsert, got %d", len(upsertCall.IDs))
			}
		})

		t.Run("should skip when no vector or embedder", func(t *testing.T) {
			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage: &mockStorageForSR{},
				// No vector or embedder
			})

			rc := setupMemoryRequestContext("thread-1", "resource-1")
			ml := &processors.MessageList{}

			_, resultML, err := proc.ProcessOutputResult(processors.ProcessOutputResultArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:    []processors.MastraDBMessage{createTestMessage("msg-1", "user", "Hello")},
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
				t.Error("expected original MessageList")
			}
		})

		t.Run("should skip when no threadId", func(t *testing.T) {
			mockVector := &mockVectorForSR{}
			embedder := &mockEmbedder{modelIDValue: "test"}

			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:  &mockStorageForSR{},
				Vector:   mockVector,
				Embedder: embedder,
			})

			emptyRC := requestcontext.NewRequestContext()
			ml := &processors.MessageList{}

			proc.ProcessOutputResult(processors.ProcessOutputResultArgs{
				ProcessorMessageContext: processors.ProcessorMessageContext{
					Messages:    []processors.MastraDBMessage{createTestMessage("msg-1", "user", "Hello")},
					MessageList: ml,
					ProcessorContext: processors.ProcessorContext{
						RequestContext: emptyRC,
					},
				},
			})

			if len(embedder.doEmbedCalls) > 0 {
				t.Error("expected no embed calls when no threadId")
			}
		})
	})

	t.Run("Index Name Generation", func(t *testing.T) {
		t.Run("should generate default index name from model ID", func(t *testing.T) {
			embedder := &mockEmbedder{modelIDValue: "text-embedding-3-small"}
			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:  &mockStorageForSR{},
				Vector:   &mockVectorForSR{},
				Embedder: embedder,
			})

			indexName := proc.getDefaultIndexName()
			if indexName != "mastra_memory_text_embedding_3_small" {
				t.Errorf("expected 'mastra_memory_text_embedding_3_small', got %q", indexName)
			}
		})

		t.Run("should sanitize non-alphanumeric characters in index name", func(t *testing.T) {
			embedder := &mockEmbedder{modelIDValue: "my-model/v2.0"}
			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:  &mockStorageForSR{},
				Vector:   &mockVectorForSR{},
				Embedder: embedder,
			})

			indexName := proc.getDefaultIndexName()
			if indexName != "mastra_memory_my_model_v2_0" {
				t.Errorf("expected 'mastra_memory_my_model_v2_0', got %q", indexName)
			}
		})

		t.Run("should truncate long index names to 63 characters", func(t *testing.T) {
			embedder := &mockEmbedder{modelIDValue: "a-very-long-model-name-that-exceeds-the-maximum-allowed-length-for-index-names"}
			proc := NewSemanticRecall(SemanticRecallOptions{
				Storage:  &mockStorageForSR{},
				Vector:   &mockVectorForSR{},
				Embedder: embedder,
			})

			indexName := proc.getDefaultIndexName()
			if len(indexName) > 63 {
				t.Errorf("expected index name length <= 63, got %d", len(indexName))
			}
		})
	})
}
