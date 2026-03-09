// Ported from: packages/core/src/processors/memory/semantic-recall.ts
package memory

import (
	"context"
	"fmt"
	"regexp"

	"github.com/cespare/xxhash/v2"
	"strings"
	"sync"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/llm/model"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/processors"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	storagememory "github.com/brainlet/brainkit/agent-kit/core/storage/domains/memory"
	"github.com/brainlet/brainkit/agent-kit/core/vector"
)

// ---------------------------------------------------------------------------
// Type aliases from vector and llm/model packages
// ---------------------------------------------------------------------------

// MastraVector is the subset of vector.MastraVector used by SemanticRecall.
// Method signatures match the real interface (ctx as first param, real param types).
type MastraVector interface {
	Query(ctx context.Context, params vector.QueryVectorParams) ([]vector.QueryResult, error)
	CreateIndex(ctx context.Context, params vector.CreateIndexParams) error
	Upsert(ctx context.Context, params vector.UpsertVectorParams) ([]string, error)
}

// Type aliases for vector types used in this file.
type (
	VectorQueryResult = vector.QueryResult
)

// MastraEmbeddingModel is the subset of model.EmbeddingModelV2 used by SemanticRecall.
// Method signatures match the real interface.
type MastraEmbeddingModel interface {
	DoEmbed(args model.EmbedArgs) (*model.EmbedResult, error)
	ModelID() string
}

// Type aliases for embedding types used in this file and tests.
type (
	EmbedOptions = model.EmbedArgs
	EmbedResult  = model.EmbedResult
)

// MastraEmbeddingOptions aliases the real vector.EmbeddingOptions.
type MastraEmbeddingOptions = vector.EmbeddingOptions


// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	defaultTopK         = 4
	defaultMessageRange = 1
)

// ---------------------------------------------------------------------------
// MessageRange
// ---------------------------------------------------------------------------

// MessageRange defines the number of context messages to include before/after
// each semantic search match.
type MessageRange struct {
	Before int
	After  int
}

// ---------------------------------------------------------------------------
// SemanticRecallOptions
// ---------------------------------------------------------------------------

// SemanticRecallOptions configures the SemanticRecall processor.
type SemanticRecallOptions struct {
	// Storage instance for retrieving messages.
	Storage MemoryStorage

	// Vector store for semantic search.
	Vector MastraVector

	// Embedder for generating query embeddings.
	Embedder MastraEmbeddingModel

	// TopK is the number of most similar messages to retrieve. Default: 4.
	TopK int

	// MessageRange is the number of context messages to include before/after
	// each match. Default: 1 for both.
	MessageRange *MessageRange

	// Scope of semantic search: "thread" or "resource". Default: "resource".
	Scope string

	// Threshold is the minimum similarity score (0-1). Messages below this
	// threshold are filtered out. Zero means no threshold.
	Threshold float64

	// IndexName for the vector store. If empty, auto-generated from embedder model.
	IndexName string

	// Logger is an optional structured logger.
	Logger logger.IMastraLogger

	// EmbedderOptions are optional provider-specific embedding options.
	EmbedderOptions MastraEmbeddingOptions
}

// ---------------------------------------------------------------------------
// SemanticRecall
// ---------------------------------------------------------------------------

// SemanticRecall is both an input and output processor that:
//   - On input: performs semantic search on historical messages and adds relevant context.
//   - On output: creates embeddings for messages being saved to enable future semantic search.
type SemanticRecall struct {
	processors.BaseProcessor
	storage         MemoryStorage
	vector          MastraVector
	embedder        MastraEmbeddingModel
	topK            int
	messageRange    MessageRange
	scope           string
	threshold       float64
	indexName       string
	logger          logger.IMastraLogger
	embedderOptions MastraEmbeddingOptions

	// indexValidationCache prevents redundant API calls when index already validated.
	indexValidationCache sync.Map // map[string]int (indexName -> dimension)
}

// NewSemanticRecall creates a new SemanticRecall processor.
func NewSemanticRecall(opts SemanticRecallOptions) *SemanticRecall {
	topK := opts.TopK
	if topK <= 0 {
		topK = defaultTopK
	}

	scope := opts.Scope
	if scope == "" {
		scope = "resource"
	}

	mr := MessageRange{Before: defaultMessageRange, After: defaultMessageRange}
	if opts.MessageRange != nil {
		mr = *opts.MessageRange
	}

	return &SemanticRecall{
		BaseProcessor:   processors.NewBaseProcessor("semantic-recall", "SemanticRecall"),
		storage:         opts.Storage,
		vector:          opts.Vector,
		embedder:        opts.Embedder,
		topK:            topK,
		messageRange:    mr,
		scope:           scope,
		threshold:       opts.Threshold,
		indexName:       opts.IndexName,
		logger:          opts.Logger,
		embedderOptions: opts.EmbedderOptions,
	}
}

// ---------------------------------------------------------------------------
// ProcessInput
// ---------------------------------------------------------------------------

// ProcessInput performs semantic search on historical messages and adds
// relevant context to the message list.
func (sr *SemanticRecall) ProcessInput(args processors.ProcessInputArgs) (
	[]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error,
) {
	messageList := args.MessageList
	rc := args.RequestContext

	memCtx := ParseMemoryRequestContext(rc)
	if memCtx == nil {
		return nil, messageList, nil, nil
	}

	threadID := ""
	if memCtx.Thread != nil {
		threadID = memCtx.Thread.ID
	}
	if threadID == "" {
		return nil, messageList, nil, nil
	}

	resourceID := memCtx.ResourceID

	// Extract user query from the last user message.
	userQuery := sr.extractUserQuery(args.Messages)
	if userQuery == "" {
		return nil, messageList, nil, nil
	}

	ctx := context.Background()

	// Perform semantic search.
	similarMessages, err := sr.performSemanticSearch(ctx, userQuery, threadID, resourceID)
	if err != nil {
		if sr.logger != nil {
			sr.logger.Error("[SemanticRecall] Error during semantic search:", err)
		}
		return nil, messageList, nil, nil
	}

	if len(similarMessages) == 0 {
		return nil, messageList, nil, nil
	}

	// Filter out messages already in the MessageList (added by previous processors or current input).
	// Ported from TS: const existingMessages = messageList.get.all.db();
	//                  const existingIds = new Set(existingMessages.map(m => m.id).filter(Boolean));
	//                  const newMessages = similarMessages.filter(m => m.id && !existingIds.has(m.id));
	existingIDs := make(map[string]bool)
	if messageList != nil {
		for _, m := range messageList.GetAllDB() {
			if m.ID != "" {
				existingIDs[m.ID] = true
			}
		}
	}
	var newMessages []processors.MastraDBMessage
	for _, m := range similarMessages {
		if m.ID != "" && !existingIDs[m.ID] {
			newMessages = append(newMessages, m)
		}
	}

	if len(newMessages) == 0 {
		return nil, messageList, nil, nil
	}

	// Separate same-thread and cross-thread messages.
	// Ported from TS: const sameThreadMessages = newMessages.filter(m => !m.threadId || m.threadId === threadId);
	var sameThreadMessages []processors.MastraDBMessage
	for _, m := range newMessages {
		if m.ThreadID == "" || m.ThreadID == threadID {
			sameThreadMessages = append(sameThreadMessages, m)
		}
	}

	// Note: Cross-thread message formatting (formatCrossThreadMessages) is not yet
	// ported. When the scope is "resource", cross-thread messages in the TS source
	// are formatted as a tagged system message. This can be added in a follow-up
	// once MessageList V1 conversion is available.

	// Add all same-thread recalled messages with 'memory' source.
	// Ported from TS: messageList.add(sameThreadMessages, 'memory');
	if len(sameThreadMessages) > 0 && messageList != nil {
		for _, m := range sameThreadMessages {
			messageList.Add(m, "memory")
		}
	}

	return nil, messageList, nil, nil
}

// ProcessInputStep is a no-op for SemanticRecall.
func (sr *SemanticRecall) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	return nil, nil, nil
}

// ---------------------------------------------------------------------------
// ProcessOutputResult
// ---------------------------------------------------------------------------

// ProcessOutputResult creates embeddings for messages being saved to enable
// future semantic search.
func (sr *SemanticRecall) ProcessOutputResult(args processors.ProcessOutputResultArgs) (
	[]processors.MastraDBMessage, *processors.MessageList, error,
) {
	messageList := args.MessageList
	rc := args.RequestContext

	if sr.vector == nil || sr.embedder == nil || sr.storage == nil {
		return nil, messageList, nil
	}

	memCtx := ParseMemoryRequestContext(rc)
	if memCtx == nil {
		return nil, messageList, nil
	}

	threadID := ""
	if memCtx.Thread != nil {
		threadID = memCtx.Thread.ID
	}
	if threadID == "" {
		return nil, messageList, nil
	}

	ctx := context.Background()
	resourceID := memCtx.ResourceID
	indexName := sr.getDefaultIndexName()
	if sr.indexName != "" {
		indexName = sr.indexName
	}

	// Collect embeddings for messages.
	var vectors [][]float64
	var ids []string
	var metadataList []map[string]any
	vectorDimension := 0

	// Get all new messages that need embeddings (both user and response messages).
	// The 'messages' argument only contains response messages, so we also need
	// to get user messages from the messageList for embedding.
	// Ported from TS: let messagesToEmbed = [...messages];
	//                  if (messageList) {
	//                    const newUserMessages = messageList.get.input.db().filter(m => messageList.isNewMessage(m));
	//                    for (const userMsg of newUserMessages) {
	//                      if (!existingIds.has(userMsg.id)) messagesToEmbed.push(userMsg);
	//                    }
	//                  }
	messagesToEmbed := make([]processors.MastraDBMessage, len(args.Messages))
	copy(messagesToEmbed, args.Messages)
	if messageList != nil {
		existingIDs := make(map[string]bool)
		for _, m := range messagesToEmbed {
			if m.ID != "" {
				existingIDs[m.ID] = true
			}
		}
		for _, userMsg := range messageList.GetInputDB() {
			if messageList.IsNewMessage(userMsg) && !existingIDs[userMsg.ID] {
				messagesToEmbed = append(messagesToEmbed, userMsg)
			}
		}
	}

	for _, message := range messagesToEmbed {
		if message.Role == "system" {
			continue
		}
		if message.ID == "" {
			continue
		}

		// Only embed new user messages and new response messages.
		// Skip context messages and memory messages.
		// Ported from TS: if (messageList) { const isNewMessage = messageList.isNewMessage(message); if (!isNewMessage) continue; }
		if messageList != nil {
			if !messageList.IsNewMessage(message) {
				continue
			}
		}

		textContent := sr.extractTextContent(message)
		if textContent == "" {
			continue
		}

		embeddings, dimension, err := sr.embedMessageContent(textContent, indexName)
		if err != nil {
			if sr.logger != nil {
				sr.logger.Error(fmt.Sprintf("[SemanticRecall] Error creating embedding for message %s:", message.ID), err)
			}
			continue
		}

		if len(embeddings) == 0 {
			continue
		}

		embedding := embeddings[0]
		vectors = append(vectors, embedding)
		ids = append(ids, message.ID)
		metadataList = append(metadataList, map[string]any{
			"message_id":  message.ID,
			"thread_id":   threadID,
			"resource_id": resourceID,
			"role":        message.Role,
			"content":     textContent,
			"created_at":  message.CreatedAt.Format(time.RFC3339),
		})
		vectorDimension = dimension
	}

	if len(vectors) > 0 {
		if err := sr.ensureVectorIndex(indexName, vectorDimension); err != nil {
			if sr.logger != nil {
				sr.logger.Error("[SemanticRecall] Error ensuring vector index:", err)
			}
			return nil, messageList, nil
		}
		if _, err := sr.vector.Upsert(ctx, vector.UpsertVectorParams{
			IndexName: indexName,
			Vectors:   vectors,
			IDs:       ids,
			Metadata:  metadataList,
		}); err != nil {
			if sr.logger != nil {
				sr.logger.Error("[SemanticRecall] Error upserting vectors:", err)
			}
		}
	}

	return nil, messageList, nil
}

// ProcessOutputStream is a no-op for SemanticRecall.
func (sr *SemanticRecall) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	return &args.Part, nil
}

// ProcessOutputStep is a no-op for SemanticRecall.
func (sr *SemanticRecall) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, args.MessageList, nil
}

// ---------------------------------------------------------------------------
// Private helpers
// ---------------------------------------------------------------------------

// extractUserQuery extracts the user query from the last user message.
func (sr *SemanticRecall) extractUserQuery(messages []processors.MastraDBMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != "user" {
			continue
		}

		// Check content string first.
		if msg.Content.Content != "" {
			return msg.Content.Content
		}

		// Extract from parts.
		var textParts []string
		for _, part := range msg.Content.Parts {
			if part.Type == "text" && part.Text != "" {
				textParts = append(textParts, part.Text)
			}
		}
		if text := strings.Join(textParts, " "); text != "" {
			return text
		}
	}
	return ""
}

// performSemanticSearch uses vector embeddings to find similar messages.
func (sr *SemanticRecall) performSemanticSearch(ctx context.Context, query, threadID, resourceID string) ([]processors.MastraDBMessage, error) {
	indexName := sr.getDefaultIndexName()
	if sr.indexName != "" {
		indexName = sr.indexName
	}

	embeddings, dimension, err := sr.embedMessageContent(query, indexName)
	if err != nil {
		return nil, err
	}
	if err := sr.ensureVectorIndex(indexName, dimension); err != nil {
		return nil, err
	}

	var vectorResults []VectorQueryResult
	for _, embedding := range embeddings {
		filter := map[string]any{"thread_id": threadID}
		if sr.scope == "resource" && resourceID != "" {
			filter = map[string]any{"resource_id": resourceID}
		}

		results, err := sr.vector.Query(ctx, vector.QueryVectorParams{
			IndexName:   indexName,
			QueryVector: embedding,
			TopK:        sr.topK,
			Filter:      filter,
		})
		if err != nil {
			return nil, err
		}
		vectorResults = append(vectorResults, results...)
	}

	// Filter by threshold if specified.
	var filteredResults []VectorQueryResult
	for _, r := range vectorResults {
		if sr.threshold > 0 && r.Score < sr.threshold {
			continue
		}
		filteredResults = append(filteredResults, r)
	}

	if len(filteredResults) == 0 {
		return nil, nil
	}

	// Retrieve messages with context.
	var includes []MessageIncludeItem
	for _, r := range filteredResults {
		messageID, _ := r.Metadata["message_id"].(string)
		rThreadID, _ := r.Metadata["thread_id"].(string)
		includes = append(includes, MessageIncludeItem{
			ID:                   messageID,
			ThreadID:             rThreadID,
			WithNextMessages:     sr.messageRange.After,
			WithPreviousMessages: sr.messageRange.Before,
		})
	}

	result, err := sr.storage.ListMessages(ctx, storagememory.StorageListMessagesInput{
		ThreadID:   threadID,
		ResourceID: resourceID,
		Include:    includes,
	})
	if err != nil {
		return nil, err
	}

	return result.Messages, nil
}

// hashContent creates a cache key for embedding lookup.
// Uses xxhash (same algorithm as xxhash-wasm in the TS source).
// TS source: packages/core/src/processors/memory/semantic-recall.ts
func (sr *SemanticRecall) hashContent(content, indexName string) string {
	combined := indexName + ":" + content
	h := xxhash.Sum64String(combined)
	return fmt.Sprintf("%016x", h)
}

// embedMessageContent generates embeddings for message content.
func (sr *SemanticRecall) embedMessageContent(content, indexName string) ([][]float64, int, error) {
	// Check global cache first.
	contentHash := sr.hashContent(content, indexName)
	if cachedEmbedding, ok := globalEmbeddingCache.Get(contentHash); ok {
		return [][]float64{cachedEmbedding}, len(cachedEmbedding), nil
	}

	result, err := sr.embedder.DoEmbed(EmbedOptions{Values: []string{content}})
	if err != nil {
		return nil, 0, err
	}

	if len(result.Embeddings) > 0 && len(result.Embeddings[0]) > 0 {
		globalEmbeddingCache.Set(contentHash, result.Embeddings[0])
	}

	dimension := 0
	if len(result.Embeddings) > 0 {
		dimension = len(result.Embeddings[0])
	}

	return result.Embeddings, dimension, nil
}

// sanitizeModelRe replaces non-alphanumeric/underscore characters.
var sanitizeModelRe = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// getDefaultIndexName generates a default index name from the embedder model ID.
func (sr *SemanticRecall) getDefaultIndexName() string {
	model := "default"
	if sr.embedder != nil {
		if mid := sr.embedder.ModelID(); mid != "" {
			model = mid
		}
	}
	sanitized := sanitizeModelRe.ReplaceAllString(model, "_")
	indexName := "mastra_memory_" + sanitized
	if len(indexName) > 63 {
		indexName = indexName[:63]
	}
	return indexName
}

// ensureVectorIndex ensures the vector index exists with correct dimensions.
func (sr *SemanticRecall) ensureVectorIndex(indexName string, dimension int) error {
	if cached, ok := sr.indexValidationCache.Load(indexName); ok {
		if cached.(int) == dimension {
			return nil
		}
	}

	if err := sr.vector.CreateIndex(context.Background(), vector.CreateIndexParams{
		IndexName: indexName,
		Dimension: dimension,
		Metric:    vector.DistanceMetricCosine,
	}); err != nil {
		return err
	}

	sr.indexValidationCache.Store(indexName, dimension)
	return nil
}

// extractTextContent extracts text content from a MastraDBMessage.
func (sr *SemanticRecall) extractTextContent(message processors.MastraDBMessage) string {
	if message.Content.Content != "" {
		return message.Content.Content
	}

	if len(message.Content.Parts) > 0 {
		var parts []string
		for _, part := range message.Content.Parts {
			if part.Type == "text" && part.Text != "" {
				parts = append(parts, part.Text)
			}
		}
		return strings.Join(parts, "\n")
	}

	return ""
}

// Ensure *SemanticRecall satisfies the expected interfaces at compile time.
var (
	_ processors.InputProcessor  = (*SemanticRecall)(nil)
	_ processors.OutputProcessor = (*SemanticRecall)(nil)
)

// Ensure *SemanticRecall also has the no-op request context helper.
var _ interface {
	ProcessInput(processors.ProcessInputArgs) ([]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error)
} = (*SemanticRecall)(nil)

// parseMemoryRequestContextLocal is a convenience alias.
func parseMemoryRequestContextLocal(rc *requestcontext.RequestContext) *MemoryRequestContext {
	return ParseMemoryRequestContext(rc)
}
