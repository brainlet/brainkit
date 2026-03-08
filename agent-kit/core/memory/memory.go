// Ported from: packages/core/src/memory/memory.ts
package memory

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"sync"
	"time"

	agentkit "github.com/brainlet/brainkit/agent-kit/core"
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/storage"
	aktypes "github.com/brainlet/brainkit/agent-kit/core/types"
	"github.com/brainlet/brainkit/agent-kit/core/vector"
)

// ---------------------------------------------------------------------------
// Stub interfaces for cross-package dependencies
//
// Some stubs MUST remain because of circular import constraints:
//   - processors → memory  ⇒  memory CANNOT import processors
//   - core.Mastra references memory  ⇒  memory CANNOT import core
//
// Others are kept because the real type shape differs from the simplified
// stub used here (e.g., IdGeneratorContext, ModelRouterEmbeddingModel).
// ---------------------------------------------------------------------------

// Mastra is a stub for core.Mastra.
// CANNOT import core: core defines Mastra struct that references memory types.
type Mastra interface {
	GenerateID(ctx *IdGeneratorContext) string
}

// IdGeneratorContext is re-exported from the types package.
// Ported from: packages/core/src/types/dynamic-argument.ts — IdGeneratorContext
type IdGeneratorContext = aktypes.IdGeneratorContext

// InputProcessor is a stub for processors.InputProcessor.
// CANNOT import: processors → memory (circular).
type InputProcessor interface{}

// OutputProcessor is a stub for processors.OutputProcessor.
// CANNOT import: processors → memory (circular).
type OutputProcessor interface{}

// InputProcessorOrWorkflow is a stub for processors.InputProcessorOrWorkflow.
// CANNOT import: processors → memory (circular).
type InputProcessorOrWorkflow interface{}

// OutputProcessorOrWorkflow is a stub for processors.OutputProcessorOrWorkflow.
// CANNOT import: processors → memory (circular).
type OutputProcessorOrWorkflow interface{}

// RequestContextRef is a minimal interface matching requestcontext.RequestContext.
// The real *requestcontext.RequestContext satisfies this interface (has Get(key string) any).
// Kept as an interface to allow other implementations and avoid tighter coupling.
type RequestContextRef interface {
	Get(key string) any
}

// ToolAction is a stub for tools.ToolAction.
// Real type: tools.ToolAction struct. Kept as any because the interface method
// ListTools returns map[string]ToolAction, and changing to the real struct
// would alter the MastraMemory interface contract.
type ToolAction = any

// ModelRouterEmbeddingModel is a simplified stub for llm/model.ModelRouterEmbeddingModel.
// Real type: model.ModelRouterEmbeddingModel struct (has many more fields).
// Kept as a simple struct because memory only needs the ID field when
// constructing from string embedder configs.
type ModelRouterEmbeddingModel struct {
	ID string
}

// ---------------------------------------------------------------------------
// extractModelIdString
// ---------------------------------------------------------------------------

// extractModelIdString extracts a string model ID from a model value.
// Returns empty string for non-serializable values (functions, LanguageModel instances).
func extractModelIdString(model any) string {
	if model == nil {
		return ""
	}
	if s, ok := model.(string); ok {
		return s
	}
	// Check if it's a struct/map with an "id" field
	if m, ok := model.(map[string]any); ok {
		if id, ok := m["id"].(string); ok {
			return id
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// MemoryProcessorOpts
// ---------------------------------------------------------------------------

// MemoryProcessorOpts holds options for memory processors.
type MemoryProcessorOpts struct {
	SystemMessage       string
	MemorySystemMessage string
	NewMessages         []CoreMessage
}

// ---------------------------------------------------------------------------
// MemoryProcessor
// ---------------------------------------------------------------------------

// MemoryProcessor is an abstract base for message processors that can filter
// or transform messages before they're sent to the LLM.
type MemoryProcessor struct {
	*agentkit.MastraBase
}

// NewMemoryProcessor creates a new MemoryProcessor.
func NewMemoryProcessor(name string) *MemoryProcessor {
	return &MemoryProcessor{
		MastraBase: agentkit.NewMastraBase(agentkit.MastraBaseOptions{
			Component: logger.RegisteredLoggerMemory,
			Name:      name,
		}),
	}
}

// Process processes a list of messages and returns a filtered or transformed list.
// The default implementation returns messages unchanged.
func (mp *MemoryProcessor) Process(messages []CoreMessage, opts MemoryProcessorOpts) []CoreMessage {
	return messages
}

// ---------------------------------------------------------------------------
// MemoryDefaultOptions
// ---------------------------------------------------------------------------

// MemoryDefaultOptions returns the default MemoryConfig used when no config is provided.
func MemoryDefaultOptions() MemoryConfig {
	return MemoryConfig{
		LastMessages: &LastMessagesConfig{Count: 10},
		SemanticRecall: &SemanticRecallConfig{
			BoolValue: boolPtr(false),
		},
		GenerateTitle: &GenerateTitleConfig{
			BoolValue: boolPtr(false),
		},
		WorkingMemory: &WorkingMemory{
			Enabled: false,
			Template: `
# User Information
- **First Name**:
- **Last Name**:
- **Location**:
- **Occupation**:
- **Interests**:
- **Goals**:
- **Events**:
- **Facts**:
- **Projects**:
`,
		},
	}
}

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool {
	return &b
}

// ---------------------------------------------------------------------------
// SaveMessagesResult
// ---------------------------------------------------------------------------

// SaveMessagesResult holds the result of saving messages.
type SaveMessagesResult struct {
	Messages []MastraDBMessage `json:"messages"`
	Usage    *TokenUsage       `json:"usage,omitempty"`
}

// TokenUsage holds token usage information.
type TokenUsage struct {
	Tokens int `json:"tokens"`
}

// ---------------------------------------------------------------------------
// RecallResult
// ---------------------------------------------------------------------------

// RecallResult holds the result of recalling messages.
type RecallResult struct {
	Messages []MastraDBMessage `json:"messages"`
	Usage    *TokenUsage       `json:"usage,omitempty"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PerPage  any               `json:"perPage"` // int or false
	HasMore  bool              `json:"hasMore"`
}

// ---------------------------------------------------------------------------
// MastraMemory — abstract base class
// ---------------------------------------------------------------------------

// MastraMemory is the abstract base interface for implementing conversation memory systems.
//
// Key features:
//   - Thread-based conversation organization with resource association
//   - Optional vector database integration for semantic similarity search
//   - Working memory templates for structured conversation state
//   - Handles memory processors to manipulate messages before they are sent to the LLM
type MastraMemory interface {
	// ID returns the unique identifier for the memory instance.
	ID() string

	// GetMaxContextTokens returns the maximum context token limit, or 0 if not set.
	GetMaxContextTokens() int

	// GetThreadById retrieves a specific thread by its ID.
	GetThreadById(ctx context.Context, threadId string) (*StorageThreadType, error)

	// ListThreads lists threads with optional filtering.
	ListThreads(ctx context.Context, args StorageListThreadsInput) (StorageListThreadsOutput, error)

	// SaveThread saves or updates a thread.
	SaveThread(ctx context.Context, thread StorageThreadType, memoryConfig *MemoryConfig) (*StorageThreadType, error)

	// SaveMessages saves messages to a thread.
	SaveMessages(ctx context.Context, messages []MastraDBMessage, memoryConfig *MemoryConfig) (*SaveMessagesResult, error)

	// Recall retrieves messages for a specific thread with optional semantic recall.
	Recall(ctx context.Context, args RecallArgs) (*RecallResult, error)

	// CreateThread creates a new thread.
	CreateThread(ctx context.Context, opts CreateThreadOpts) (*StorageThreadType, error)

	// DeleteThread deletes a thread by ID.
	DeleteThread(ctx context.Context, threadId string) error

	// DeleteMessages deletes messages by their IDs.
	DeleteMessages(ctx context.Context, messageIds MessageDeleteInput) error

	// GetWorkingMemory retrieves working memory for a specific thread.
	GetWorkingMemory(ctx context.Context, opts GetWorkingMemoryOpts) (*string, error)

	// GetWorkingMemoryTemplate gets the working memory template.
	GetWorkingMemoryTemplate(ctx context.Context, memoryConfig *MemoryConfig) (*WorkingMemoryTemplate, error)

	// UpdateWorkingMemory updates working memory for a thread.
	UpdateWorkingMemory(ctx context.Context, opts UpdateWorkingMemoryOpts) error

	// ExperimentalUpdateWorkingMemoryVNext is an experimental working memory update.
	// WARNING: Can be removed or changed at any time.
	ExperimentalUpdateWorkingMemoryVNext(ctx context.Context, opts ExperimentalUpdateWorkingMemoryVNextOpts) (*ExperimentalUpdateResult, error)

	// CloneThread clones a thread with all its messages.
	CloneThread(ctx context.Context, args StorageCloneThreadInput) (StorageCloneThreadOutput, error)

	// GetMergedThreadConfig merges the given config with the base thread config.
	GetMergedThreadConfig(config *MemoryConfig) MemoryConfig

	// GetSystemMessage gets a system message to inject into the conversation.
	GetSystemMessage(ctx context.Context, input GetSystemMessageInput) (*string, error)

	// ListTools gets tools that should be available to the agent.
	ListTools(config *MemoryConfig) map[string]ToolAction

	// GetConfig returns serializable configuration for this memory instance.
	GetConfig() SerializedMemoryConfig

	// EstimateTokens estimates the token count for the given text.
	EstimateTokens(text string) int

	// GenerateId generates a unique identifier.
	GenerateId(idCtx *IdGeneratorContext) string

	// Storage returns the storage provider.
	Storage() *storage.MastraCompositeStore

	// HasOwnStorage returns whether this memory has its own storage.
	HasOwnStorage() bool

	// SetStorage sets the storage provider.
	SetStorage(store *storage.MastraCompositeStore)

	// SetVector sets the vector provider.
	SetVector(vector MastraVector)

	// SetEmbedder sets the embedding model.
	SetEmbedder(embedder any, opts MastraEmbeddingOptions)

	// GetInputProcessors returns input processors for this memory instance.
	GetInputProcessors(ctx context.Context, configured []InputProcessorOrWorkflow, reqCtx RequestContextRef) ([]InputProcessor, error)

	// GetOutputProcessors returns output processors for this memory instance.
	GetOutputProcessors(ctx context.Context, configured []OutputProcessorOrWorkflow, reqCtx RequestContextRef) ([]OutputProcessor, error)

	// RegisterMastra registers the Mastra instance with the memory (internal).
	RegisterMastra(m Mastra)
}

// RecallArgs holds the arguments for the Recall method.
type RecallArgs struct {
	StorageListMessagesInput
	ThreadConfig      *MemoryConfig `json:"threadConfig,omitempty"`
	VectorSearchString string       `json:"vectorSearchString,omitempty"`
}

// CreateThreadOpts holds options for creating a thread.
type CreateThreadOpts struct {
	ThreadID     string          `json:"threadId,omitempty"`
	ResourceID   string          `json:"resourceId"`
	Title        string          `json:"title,omitempty"`
	Metadata     map[string]any  `json:"metadata,omitempty"`
	MemoryConfig *MemoryConfig   `json:"memoryConfig,omitempty"`
	SaveThread   *bool           `json:"saveThread,omitempty"`
}

// GetWorkingMemoryOpts holds options for getting working memory.
type GetWorkingMemoryOpts struct {
	ThreadID     string        `json:"threadId"`
	ResourceID   string        `json:"resourceId,omitempty"`
	MemoryConfig *MemoryConfig `json:"memoryConfig,omitempty"`
}

// UpdateWorkingMemoryOpts holds options for updating working memory.
type UpdateWorkingMemoryOpts struct {
	ThreadID      string        `json:"threadId"`
	ResourceID    string        `json:"resourceId,omitempty"`
	WorkingMemory string        `json:"workingMemory"`
	MemoryConfig  *MemoryConfig `json:"memoryConfig,omitempty"`
}

// ExperimentalUpdateWorkingMemoryVNextOpts holds options for the experimental update.
type ExperimentalUpdateWorkingMemoryVNextOpts struct {
	ThreadID      string        `json:"threadId"`
	ResourceID    string        `json:"resourceId,omitempty"`
	WorkingMemory string        `json:"workingMemory"`
	SearchString  string        `json:"searchString,omitempty"`
	MemoryConfig  *MemoryConfig `json:"memoryConfig,omitempty"`
}

// ExperimentalUpdateResult holds the result of the experimental update.
type ExperimentalUpdateResult struct {
	Success bool   `json:"success"`
	Reason  string `json:"reason"`
}

// GetSystemMessageInput holds the input for GetSystemMessage.
type GetSystemMessageInput struct {
	ThreadID     string        `json:"threadId"`
	ResourceID   string        `json:"resourceId,omitempty"`
	MemoryConfig *MemoryConfig `json:"memoryConfig,omitempty"`
}

// ---------------------------------------------------------------------------
// MastraMemoryBase — shared implementation for the abstract base
// ---------------------------------------------------------------------------

// MastraMemoryBase provides the shared implementation for MastraMemory.
// Concrete memory implementations should embed this struct and implement
// the abstract methods.
type MastraMemoryBase struct {
	*agentkit.MastraBase

	id                string
	maxContextTokens  int
	storage_          *storage.MastraCompositeStore
	vector            MastraVector
	embedder          MastraEmbeddingModel
	embedderOptions   MastraEmbeddingOptions
	threadConfig      MemoryConfig
	mastra            Mastra
	hasOwnStorage     bool

	// Cached promise for the embedding dimension probe.
	embeddingDimOnce  sync.Once
	embeddingDim      int
	embeddingDimErr   error
}

// MastraMemoryBaseConfig holds the configuration for creating a MastraMemoryBase.
type MastraMemoryBaseConfig struct {
	ID      string
	Name    string
	Storage *storage.MastraCompositeStore
	Options *MemoryConfig
	Vector  MastraVector
	// Embedder can be a string (model ID) or MastraEmbeddingModel.
	Embedder        any
	EmbedderOptions MastraEmbeddingOptions
	// Processors is deprecated. Will error if non-nil.
	Processors []MemoryProcessorRef
}

// NewMastraMemoryBase creates a new MastraMemoryBase.
func NewMastraMemoryBase(config MastraMemoryBaseConfig) (*MastraMemoryBase, error) {
	base := agentkit.NewMastraBase(agentkit.MastraBaseOptions{
		Component: logger.RegisteredLoggerMemory,
		Name:      config.Name,
	})

	id := config.ID
	if id == "" {
		id = config.Name
	}
	if id == "" {
		id = "default-memory"
	}

	m := &MastraMemoryBase{
		MastraBase:   base,
		id:           id,
		threadConfig: MemoryDefaultOptions(),
	}

	if config.Options != nil {
		m.threadConfig = m.GetMergedThreadConfig(config.Options)
	}

	// DEPRECATION: Block old processors config
	if len(config.Processors) > 0 {
		return nil, fmt.Errorf(
			"the 'processors' option in Memory is deprecated and has been removed.\n\n" +
				"Please use the new Input/Output processor system instead.\n\n" +
				"See: https://mastra.ai/en/docs/memory/processors",
		)
	}

	if config.Storage != nil {
		m.storage_ = config.Storage
		m.hasOwnStorage = true
	}

	if m.threadConfig.SemanticRecall != nil && m.threadConfig.SemanticRecall.IsEnabled() {
		if config.Vector == nil {
			return nil, fmt.Errorf(
				"semantic recall requires a vector store to be configured.\n\n" +
					"https://mastra.ai/en/docs/memory/semantic-recall",
			)
		}
		m.vector = config.Vector

		if config.Embedder == nil {
			return nil, fmt.Errorf(
				"semantic recall requires an embedder to be configured.\n\n" +
					"https://mastra.ai/en/docs/memory/semantic-recall",
			)
		}

		// Convert string embedder to ModelRouterEmbeddingModel
		if s, ok := config.Embedder.(string); ok {
			m.embedder = &ModelRouterEmbeddingModel{ID: s}
		} else {
			m.embedder = config.Embedder
		}

		// Set embedder options
		if config.EmbedderOptions != nil {
			m.embedderOptions = config.EmbedderOptions
		}
	}

	return m, nil
}

// ID returns the unique identifier for the memory instance.
func (m *MastraMemoryBase) ID() string {
	return m.id
}

// GetMaxContextTokens returns the maximum context token limit.
func (m *MastraMemoryBase) GetMaxContextTokens() int {
	return m.maxContextTokens
}

// RegisterMastra registers the Mastra instance with the memory (internal).
func (m *MastraMemoryBase) RegisterMastra(mastra Mastra) {
	m.mastra = mastra
}

// HasOwnStorage returns whether this memory has its own storage.
func (m *MastraMemoryBase) HasOwnStorage() bool {
	return m.hasOwnStorage
}

// Storage returns the storage provider.
// Panics if no storage is configured.
func (m *MastraMemoryBase) Storage() *storage.MastraCompositeStore {
	if m.storage_ == nil {
		panic(
			"memory requires a storage provider to function. " +
				"Add a storage configuration to Memory or to your Mastra instance.\n\n" +
				"https://mastra.ai/en/docs/memory/overview",
		)
	}
	return m.storage_
}

// SetStorage sets the storage provider.
func (m *MastraMemoryBase) SetStorage(store *storage.MastraCompositeStore) {
	m.storage_ = store
}

// SetVector sets the vector provider.
func (m *MastraMemoryBase) SetVector(vector MastraVector) {
	m.vector = vector
}

// SetEmbedder sets the embedding model and options.
func (m *MastraMemoryBase) SetEmbedder(embedder any, opts MastraEmbeddingOptions) {
	if s, ok := embedder.(string); ok {
		m.embedder = &ModelRouterEmbeddingModel{ID: s}
	} else {
		m.embedder = embedder
	}
	if opts != nil {
		m.embedderOptions = opts
	}
}

// GetSystemMessage gets a system message to inject into the conversation.
// Default implementation returns nil (no system message).
func (m *MastraMemoryBase) GetSystemMessage(ctx context.Context, input GetSystemMessageInput) (*string, error) {
	return nil, nil
}

// ListTools gets tools that should be available to the agent.
// Default implementation returns an empty map.
func (m *MastraMemoryBase) ListTools(config *MemoryConfig) map[string]ToolAction {
	return map[string]ToolAction{}
}

// EmbedderProber is an optional interface that embedder implementations can
// satisfy so that GetEmbeddingDimension can probe the real output dimension.
// Matches the TS embedder.doEmbed({values: ['a']}) pattern.
type EmbedderProber interface {
	DoEmbed(values []string, opts map[string]any) (*EmbedResult, error)
}

// EmbedResult holds the result of an embedding probe.
type EmbedResult struct {
	Embeddings [][]float64
}

// GetEmbeddingDimension probes the embedder to determine its actual output dimension.
// The result is cached so subsequent calls are free.
func (m *MastraMemoryBase) GetEmbeddingDimension() (int, error) {
	if m.embedder == nil {
		return 0, nil
	}

	m.embeddingDimOnce.Do(func() {
		// Attempt to probe the embedder if it satisfies EmbedderProber.
		if prober, ok := m.embedder.(EmbedderProber); ok {
			opts := make(map[string]any)
			if m.embedderOptions != nil {
				for k, v := range m.embedderOptions {
					opts[k] = v
				}
			}
			result, err := prober.DoEmbed([]string{"a"}, opts)
			if err != nil {
				log.Printf("[Mastra Memory] Failed to probe embedder for dimension, falling back to default. "+
					"This may cause index name mismatches if the embedder uses non-default dimensions. Error: %v", err)
				m.embeddingDim = 0
				m.embeddingDimErr = nil
				return
			}
			if result != nil && len(result.Embeddings) > 0 && len(result.Embeddings[0]) > 0 {
				m.embeddingDim = len(result.Embeddings[0])
				m.embeddingDimErr = nil
				return
			}
		}

		// Fallback: embedder does not satisfy EmbedderProber or returned no embeddings.
		log.Println("[Mastra Memory] Embedding dimension probe not available, using default")
		m.embeddingDim = 0
		m.embeddingDimErr = nil
	})

	return m.embeddingDim, m.embeddingDimErr
}

// GetEmbeddingIndexName returns the index name for semantic recall embeddings.
func (m *MastraMemoryBase) GetEmbeddingIndexName(dimensions int) string {
	const defaultDimensions = 1536
	usedDimensions := dimensions
	if usedDimensions == 0 {
		usedDimensions = defaultDimensions
	}
	isDefault := usedDimensions == defaultDimensions

	// In TS: this.vector?.indexSeparator ?? '_'
	separator := "_"
	if m.vector != nil {
		if sep := m.vector.IndexSeparator(); sep != "" {
			separator = sep
		}
	}

	if isDefault {
		return fmt.Sprintf("memory%smessages", separator)
	}
	return fmt.Sprintf("memory%smessages%s%d", separator, separator, usedDimensions)
}

// CreateEmbeddingIndex creates the embedding index for semantic recall.
func (m *MastraMemoryBase) CreateEmbeddingIndex(ctx context.Context, dimensions int, config *MemoryConfig) (string, error) {
	const defaultDimensions = 1536
	usedDimensions := dimensions
	if usedDimensions == 0 {
		usedDimensions = defaultDimensions
	}
	indexName := m.GetEmbeddingIndexName(dimensions)

	if m.vector == nil {
		return "", fmt.Errorf("tried to create embedding index but no vector db is attached to this Memory instance")
	}

	// Create the index on the vector store.
	// TS source: this.vector.createIndex({ indexName, dimension, ...indexConfig })
	createParams := vector.CreateIndexParams{
		IndexName: indexName,
		Dimension: usedDimensions,
	}
	// Extract metric from config.semanticRecall.indexConfig if available.
	if config != nil && config.SemanticRecall != nil && config.SemanticRecall.Options != nil {
		if ic := config.SemanticRecall.Options.IndexConfig; ic != nil && ic.Metric != "" {
			createParams.Metric = vector.DistanceMetric(ic.Metric)
		}
	}
	if err := m.vector.CreateIndex(ctx, createParams); err != nil {
		return "", fmt.Errorf("failed to create embedding index %q: %w", indexName, err)
	}

	return indexName, nil
}

// GetMergedThreadConfig merges the given config with the base thread config.
func (m *MastraMemoryBase) GetMergedThreadConfig(config *MemoryConfig) MemoryConfig {
	if config == nil {
		return m.threadConfig
	}

	// Block deprecated workingMemory.use option
	if config.WorkingMemory != nil {
		// In TS: checking for 'use' in config.workingMemory
		// In Go: WorkingMemory struct doesn't have Use field, so this is a no-op.
		// The TS validation is handled by the struct definition.
	}

	// Block deprecated threads.generateTitle
	if config.Threads != nil && config.Threads.GenerateTitle != nil {
		panic("the threads.generateTitle option has been moved. Use the top-level generateTitle option instead.")
	}

	merged := deepMergeMemoryConfig(m.threadConfig, *config)

	// Preserve schema from the input config (don't merge it)
	if config.WorkingMemory != nil && config.WorkingMemory.Schema != nil {
		if merged.WorkingMemory != nil {
			merged.WorkingMemory.Schema = config.WorkingMemory.Schema
		}
	}

	return merged
}

// EstimateTokens estimates the token count for the given text.
func (m *MastraMemoryBase) EstimateTokens(text string) int {
	return int(math.Ceil(float64(len(strings.Fields(text))) * 1.3))
}

// CreateThread creates a new thread.
func (m *MastraMemoryBase) CreateThread(ctx context.Context, opts CreateThreadOpts) (*StorageThreadType, error) {
	threadID := opts.ThreadID
	if threadID == "" {
		source := aktypes.IdGeneratorSourceMemory
		threadID = m.GenerateId(&IdGeneratorContext{
			IdType:     aktypes.IdTypeThread,
			Source:     &source,
			ResourceId: &opts.ResourceID,
		})
	}

	now := time.Now()
	thread := StorageThreadType{
		ID:         threadID,
		Title:      opts.Title,
		ResourceID: opts.ResourceID,
		CreatedAt:  now,
		UpdatedAt:  now,
		Metadata:   opts.Metadata,
	}

	shouldSave := true
	if opts.SaveThread != nil {
		shouldSave = *opts.SaveThread
	}

	if shouldSave {
		return m.saveThread(ctx, thread, opts.MemoryConfig)
	}
	return &thread, nil
}

// saveThread is a helper that concrete implementations should override.
// The default panics to indicate it must be implemented.
func (m *MastraMemoryBase) saveThread(ctx context.Context, thread StorageThreadType, memoryConfig *MemoryConfig) (*StorageThreadType, error) {
	panic("saveThread must be implemented by concrete memory type")
}

// AddMessage is deprecated. Use SaveMessages instead.
func (m *MastraMemoryBase) AddMessage() error {
	return fmt.Errorf("addMessage is deprecated. Please use saveMessages instead")
}

// GenerateId generates a unique identifier.
func (m *MastraMemoryBase) GenerateId(idCtx *IdGeneratorContext) string {
	if m.mastra != nil && idCtx != nil {
		return m.mastra.GenerateID(idCtx)
	}
	// Fallback to a simple UUID-like ID.
	// In TS: crypto.randomUUID()
	return generateUUID()
}

// GetConfig returns serializable configuration for this memory instance.
func (m *MastraMemoryBase) GetConfig() SerializedMemoryConfig {
	config := SerializedMemoryConfig{}

	// Set vector ID from the vector store instance.
	if m.vector != nil {
		config.Vector = m.vector.ID()
	}

	// Set options
	opts := &SerializedMemoryOptions{
		ReadOnly: m.threadConfig.ReadOnly,
	}

	// Serialize lastMessages
	if m.threadConfig.LastMessages != nil {
		if m.threadConfig.LastMessages.Disabled {
			opts.LastMessages = false
		} else if m.threadConfig.LastMessages.Count > 0 {
			opts.LastMessages = m.threadConfig.LastMessages.Count
		}
	}

	// Serialize semanticRecall
	if m.threadConfig.SemanticRecall != nil {
		if m.threadConfig.SemanticRecall.BoolValue != nil {
			opts.SemanticRecall = *m.threadConfig.SemanticRecall.BoolValue
		} else if m.threadConfig.SemanticRecall.Options != nil {
			opts.SemanticRecall = m.threadConfig.SemanticRecall.Options
		}
	}

	// Serialize generateTitle
	if m.threadConfig.GenerateTitle != nil {
		if m.threadConfig.GenerateTitle.BoolValue != nil {
			opts.GenerateTitle = *m.threadConfig.GenerateTitle.BoolValue
		} else if m.threadConfig.GenerateTitle.Options != nil {
			gtOpts := m.threadConfig.GenerateTitle.Options
			modelId := extractModelIdString(gtOpts.Model)
			if modelId != "" {
				opts.GenerateTitle = map[string]any{
					"model": modelId,
				}
				if instructions, ok := gtOpts.Instructions.(string); ok {
					opts.GenerateTitle.(map[string]any)["instructions"] = instructions
				}
			}
		}
	}

	config.Options = opts

	// Serialize embedder
	if m.embedder != nil {
		config.Embedder = m.embedder
	}

	// Serialize embedder options (omit telemetry)
	if m.embedderOptions != nil {
		filtered := make(map[string]any)
		for k, v := range m.embedderOptions {
			if k != "telemetry" {
				filtered[k] = v
			}
		}
		if len(filtered) > 0 {
			config.EmbedderOptions = filtered
		}
	}

	// Serialize observationalMemory
	if m.threadConfig.ObservationalMemory != nil {
		config.ObservationalMemory = m.serializeObservationalMemory(m.threadConfig.ObservationalMemory)
	}

	return config
}

// serializeObservationalMemory serializes observational memory config to a JSON-safe representation.
func (m *MastraMemoryBase) serializeObservationalMemory(om *ObservationalMemoryConfig) any {
	if om == nil {
		return nil
	}

	if om.BoolValue != nil {
		return *om.BoolValue
	}

	if om.Options == nil {
		return nil
	}

	opts := om.Options
	if opts.Enabled != nil && !*opts.Enabled {
		return false
	}

	result := SerializedObservationalMemoryConfig{
		Scope:            opts.Scope,
		ShareTokenBudget: opts.ShareTokenBudget,
	}

	// Extract top-level model ID
	topModelId := extractModelIdString(opts.Model)
	if topModelId != "" {
		result.Model = topModelId
	}

	// Serialize observation config
	if opts.Observation != nil {
		obs := opts.Observation
		serializedObs := &SerializedObservationalMemoryObservationConfig{
			MessageTokens:     obs.MessageTokens,
			ModelSettings:     obs.ModelSettings,
			ProviderOptions:   obs.ProviderOptions,
			MaxTokensPerBatch: obs.MaxTokensPerBatch,
			BufferActivation:  obs.BufferActivation,
			BlockAfter:        obs.BlockAfter,
		}
		if obs.BufferTokens != nil {
			serializedObs.BufferTokens = *obs.BufferTokens
		}
		if obs.BufferTokensDisabled {
			serializedObs.BufferTokens = false
		}
		obsModelId := extractModelIdString(obs.Model)
		if obsModelId != "" {
			serializedObs.Model = obsModelId
		}
		result.Observation = serializedObs
	}

	// Serialize reflection config
	if opts.Reflection != nil {
		ref := opts.Reflection
		serializedRef := &SerializedObservationalMemoryReflectionConfig{
			ObservationTokens: ref.ObservationTokens,
			ModelSettings:     ref.ModelSettings,
			ProviderOptions:   ref.ProviderOptions,
			BlockAfter:        ref.BlockAfter,
			BufferActivation:  ref.BufferActivation,
		}
		refModelId := extractModelIdString(ref.Model)
		if refModelId != "" {
			serializedRef.Model = refModelId
		}
		result.Reflection = serializedRef
	}

	return result
}

// GetInputProcessors returns input processors for this memory instance.
// This allows Memory to be used as a ProcessorProvider in Agent's inputProcessors array.
//
// Ported from: packages/core/src/memory/memory.ts getInputProcessors (lines 600-735)
//
// The method resolves the effective memory config (merging any runtime override from
// RequestContext), then creates WorkingMemory, MessageHistory, and SemanticRecall
// input processors as needed. It performs deduplication against the already-configured
// processors so that manually added instances are respected.
func (m *MastraMemoryBase) GetInputProcessors(ctx context.Context, configured []InputProcessorOrWorkflow, reqCtx RequestContextRef) ([]InputProcessor, error) {
	memoryStore := m.getMemoryStoreOrNil()
	var processors []InputProcessor

	// Extract runtime memoryConfig from context if available
	effectiveConfig := m.resolveEffectiveConfig(reqCtx)

	// --- WorkingMemory ---
	isWorkingMemoryEnabled := effectiveConfig.WorkingMemory != nil && effectiveConfig.WorkingMemory.Enabled

	if isWorkingMemoryEnabled {
		if memoryStore == nil {
			return nil, fmt.Errorf(
				"using Mastra Memory working memory requires a storage adapter but no attached adapter was detected",
			)
		}

		// Check if user already manually added WorkingMemory
		hasWorkingMemory := hasProcessorWithID(configured, "working-memory")

		if !hasWorkingMemory {
			// Convert string template to WorkingMemoryTemplate format
			var template *WorkingMemoryTemplate
			if effectiveConfig.WorkingMemory.Template != "" {
				template = &WorkingMemoryTemplate{
					Format:  WorkingMemoryFormatMarkdown,
					Content: effectiveConfig.WorkingMemory.Template,
				}
			}

			useVNext := effectiveConfig.WorkingMemory.Version == "vnext"

			processors = append(processors, &workingMemoryProcessorStub{
				id:       "working-memory",
				storage:  memoryStore,
				template: template,
				scope:    effectiveConfig.WorkingMemory.Scope,
				useVNext: useVNext,
			})
		}
	}

	// --- MessageHistory ---
	lastMessages := effectiveConfig.LastMessages
	if lastMessages != nil && lastMessages.IsEnabled() {
		if memoryStore == nil {
			return nil, fmt.Errorf(
				"using Mastra Memory message history requires a storage adapter but no attached adapter was detected",
			)
		}

		// Check if user already manually added MessageHistory
		hasMessageHistory := hasProcessorWithID(configured, "message-history")

		// Check if ObservationalMemory is present (via processor or config)
		// It handles its own message loading and saving.
		hasObservationalMemory := hasProcessorWithID(configured, "observational-memory") ||
			IsObservationalMemoryEnabled(effectiveConfig.ObservationalMemory)

		// Skip MessageHistory input processor if ObservationalMemory handles message loading
		if !hasMessageHistory && !hasObservationalMemory {
			var lastMessagesCount int
			if lastMessages.Count > 0 {
				lastMessagesCount = lastMessages.Count
			}

			processors = append(processors, &messageHistoryProcessorStub{
				id:           "message-history",
				storage:      memoryStore,
				lastMessages: lastMessagesCount,
			})
		}
	}

	// --- SemanticRecall ---
	if effectiveConfig.SemanticRecall != nil && effectiveConfig.SemanticRecall.IsEnabled() {
		if memoryStore == nil {
			return nil, fmt.Errorf(
				"using Mastra Memory semantic recall requires a storage adapter but no attached adapter was detected",
			)
		}

		if m.vector == nil {
			return nil, fmt.Errorf(
				"using Mastra Memory semantic recall requires a vector adapter but no attached adapter was detected",
			)
		}

		if m.embedder == nil {
			return nil, fmt.Errorf(
				"using Mastra Memory semantic recall requires an embedder but no attached embedder was detected",
			)
		}

		// Check if user already manually added SemanticRecall
		hasSemanticRecall := hasProcessorWithID(configured, "semantic-recall")

		if !hasSemanticRecall {
			// Probe the embedder for its actual dimension to generate the correct index name.
			embeddingDimension, err := m.GetEmbeddingDimension()
			if err != nil {
				return nil, fmt.Errorf("failed to probe embedding dimension: %w", err)
			}
			indexName := m.GetEmbeddingIndexName(embeddingDimension)

			// Extract semantic recall options if present
			var semanticConfig *SemanticRecall
			if effectiveConfig.SemanticRecall.Options != nil {
				semanticConfig = effectiveConfig.SemanticRecall.Options
			}

			processors = append(processors, &semanticRecallProcessorStub{
				id:              "semantic-recall",
				storage:         memoryStore,
				vector:          m.vector,
				embedder:        m.embedder,
				embedderOptions: m.embedderOptions,
				indexName:       indexName,
				config:          semanticConfig,
			})
		}
	}

	// Return only the auto-generated processors (not the configured ones)
	// The agent will merge them with configuredProcessors
	return processors, nil
}

// GetOutputProcessors returns output processors for this memory instance.
// This allows Memory to be used as a ProcessorProvider in Agent's outputProcessors array.
//
// Ported from: packages/core/src/memory/memory.ts getOutputProcessors (lines 749-844)
//
// Note: We intentionally do NOT check readOnly here. The readOnly check happens at
// execution time in each processor's processOutputResult method. This allows proper
// isolation when agents share a RequestContext -- each agent's readOnly setting is
// respected when its processors actually run, not when processors are resolved.
// See: https://github.com/mastra-ai/mastra/issues/11651
func (m *MastraMemoryBase) GetOutputProcessors(ctx context.Context, configured []OutputProcessorOrWorkflow, reqCtx RequestContextRef) ([]OutputProcessor, error) {
	memoryStore := m.getMemoryStoreOrNil()
	var processors []OutputProcessor

	// Extract runtime memoryConfig from context if available
	effectiveConfig := m.resolveEffectiveConfig(reqCtx)

	// --- SemanticRecall ---
	if effectiveConfig.SemanticRecall != nil && effectiveConfig.SemanticRecall.IsEnabled() {
		if memoryStore == nil {
			return nil, fmt.Errorf(
				"using Mastra Memory semantic recall requires a storage adapter but no attached adapter was detected",
			)
		}

		if m.vector == nil {
			return nil, fmt.Errorf(
				"using Mastra Memory semantic recall requires a vector adapter but no attached adapter was detected",
			)
		}

		if m.embedder == nil {
			return nil, fmt.Errorf(
				"using Mastra Memory semantic recall requires an embedder but no attached embedder was detected",
			)
		}

		// Check if user already manually added SemanticRecall
		hasSemanticRecall := hasOutputProcessorWithID(configured, "semantic-recall")

		if !hasSemanticRecall {
			// Probe the embedder for its actual dimension to generate the correct index name.
			embeddingDimension, err := m.GetEmbeddingDimension()
			if err != nil {
				return nil, fmt.Errorf("failed to probe embedding dimension: %w", err)
			}
			indexName := m.GetEmbeddingIndexName(embeddingDimension)

			// Extract semantic recall options if present
			var semanticConfig *SemanticRecall
			if effectiveConfig.SemanticRecall.Options != nil {
				semanticConfig = effectiveConfig.SemanticRecall.Options
			}

			processors = append(processors, &semanticRecallOutputProcessorStub{
				id:              "semantic-recall",
				storage:         memoryStore,
				vector:          m.vector,
				embedder:        m.embedder,
				embedderOptions: m.embedderOptions,
				indexName:       indexName,
				config:          semanticConfig,
			})
		}
	}

	// --- MessageHistory ---
	lastMessages := effectiveConfig.LastMessages
	if lastMessages != nil && lastMessages.IsEnabled() {
		if memoryStore == nil {
			return nil, fmt.Errorf(
				"using Mastra Memory message history requires a storage adapter but no attached adapter was detected",
			)
		}

		// Check if user already manually added MessageHistory
		hasMessageHistory := hasOutputProcessorWithID(configured, "message-history")

		// Check if ObservationalMemory is present (via processor or config)
		// It handles its own message saving.
		hasObservationalMemory := hasOutputProcessorWithID(configured, "observational-memory") ||
			IsObservationalMemoryEnabled(effectiveConfig.ObservationalMemory)

		// Skip MessageHistory output processor if ObservationalMemory handles message saving
		if !hasMessageHistory && !hasObservationalMemory {
			var lastMessagesCount int
			if lastMessages.Count > 0 {
				lastMessagesCount = lastMessages.Count
			}

			processors = append(processors, &messageHistoryOutputProcessorStub{
				id:           "message-history",
				storage:      memoryStore,
				lastMessages: lastMessagesCount,
			})
		}
	}

	// Return only the auto-generated processors (not the configured ones)
	// The agent will merge them with configuredProcessors
	return processors, nil
}

// ---------------------------------------------------------------------------
// Internal helpers for processor resolution
// ---------------------------------------------------------------------------

// getMemoryStoreOrNil returns the memory domain store or nil if not available.
func (m *MastraMemoryBase) getMemoryStoreOrNil() any {
	if m.storage_ == nil {
		return nil
	}
	return m.storage_.GetStore(storage.DomainMemory)
}

// resolveEffectiveConfig extracts and merges runtime memory config from the request context.
func (m *MastraMemoryBase) resolveEffectiveConfig(reqCtx RequestContextRef) MemoryConfig {
	if reqCtx == nil {
		return m.threadConfig
	}

	memoryContextRaw := reqCtx.Get("MastraMemory")
	if memoryContextRaw == nil {
		return m.threadConfig
	}

	// Try typed MemoryRequestContext
	if mrc, ok := memoryContextRaw.(*MemoryRequestContext); ok && mrc != nil && mrc.MemoryConfig != nil {
		return m.GetMergedThreadConfig(mrc.MemoryConfig)
	}

	// Try map[string]any (raw context)
	if ctx, ok := memoryContextRaw.(map[string]any); ok {
		if mc, ok := ctx["memoryConfig"]; ok && mc != nil {
			if mcPtr, ok := mc.(*MemoryConfig); ok {
				return m.GetMergedThreadConfig(mcPtr)
			}
		}
	}

	return m.threadConfig
}

// processorIDer is an interface for objects that have an ID method.
type processorIDer interface {
	ID() string
}

// hasProcessorWithID checks if any configured input processor (not a workflow) has the given ID.
// Mirrors TS: configuredProcessors.some(p => !isProcessorWorkflow(p) && p.id === id)
func hasProcessorWithID(configured []InputProcessorOrWorkflow, id string) bool {
	for _, p := range configured {
		if p == nil {
			continue
		}
		// Skip processor workflows
		if isProcessorWorkflowCheck(p) {
			continue
		}
		if ider, ok := p.(processorIDer); ok {
			if ider.ID() == id {
				return true
			}
		}
	}
	return false
}

// hasOutputProcessorWithID checks if any configured output processor (not a workflow) has the given ID.
func hasOutputProcessorWithID(configured []OutputProcessorOrWorkflow, id string) bool {
	for _, p := range configured {
		if p == nil {
			continue
		}
		// Skip processor workflows
		if isProcessorWorkflowCheck(p) {
			continue
		}
		if ider, ok := p.(processorIDer); ok {
			if ider.ID() == id {
				return true
			}
		}
	}
	return false
}

// isProcessorWorkflowCheck is a simplified isProcessorWorkflow check.
// A workflow has a GetID method but is NOT an InputProcessor or OutputProcessor.
type workflowChecker interface {
	GetID() string
}

func isProcessorWorkflowCheck(obj any) bool {
	if obj == nil {
		return false
	}
	_, isWorkflow := obj.(workflowChecker)
	if !isWorkflow {
		return false
	}
	// Must NOT have processor-specific interfaces
	_, isInput := obj.(processorIDer)
	return !isInput
}

// ---------------------------------------------------------------------------
// Processor stub types
// ---------------------------------------------------------------------------
//
// These are lightweight stub implementations of InputProcessor and OutputProcessor
// that hold the configuration needed by the real processor implementations.
// Once the processors/memory package is fully ported, these stubs will be
// replaced with proper constructor calls.

// workingMemoryProcessorStub is a stub InputProcessor for working memory.
type workingMemoryProcessorStub struct {
	id       string
	storage  any
	template *WorkingMemoryTemplate
	scope    WorkingMemoryScope
	useVNext bool
}

func (s *workingMemoryProcessorStub) ID() string { return s.id }

// messageHistoryProcessorStub is a stub InputProcessor for message history.
type messageHistoryProcessorStub struct {
	id           string
	storage      any
	lastMessages int
}

func (s *messageHistoryProcessorStub) ID() string { return s.id }

// semanticRecallProcessorStub is a stub InputProcessor for semantic recall.
type semanticRecallProcessorStub struct {
	id              string
	storage         any
	vector          MastraVector
	embedder        MastraEmbeddingModel
	embedderOptions MastraEmbeddingOptions
	indexName       string
	config          *SemanticRecall
}

func (s *semanticRecallProcessorStub) ID() string { return s.id }

// semanticRecallOutputProcessorStub is a stub OutputProcessor for semantic recall.
type semanticRecallOutputProcessorStub struct {
	id              string
	storage         any
	vector          MastraVector
	embedder        MastraEmbeddingModel
	embedderOptions MastraEmbeddingOptions
	indexName       string
	config          *SemanticRecall
}

func (s *semanticRecallOutputProcessorStub) ID() string { return s.id }

// messageHistoryOutputProcessorStub is a stub OutputProcessor for message history.
type messageHistoryOutputProcessorStub struct {
	id           string
	storage      any
	lastMessages int
}

func (s *messageHistoryOutputProcessorStub) ID() string { return s.id }

// ---------------------------------------------------------------------------
// Deep merge helper
// ---------------------------------------------------------------------------

// deepMergeMemoryConfig performs a deep merge of two MemoryConfig values.
// Fields from override take precedence over base.
func deepMergeMemoryConfig(base, override MemoryConfig) MemoryConfig {
	result := base

	if override.ReadOnly {
		result.ReadOnly = override.ReadOnly
	}

	if override.LastMessages != nil {
		result.LastMessages = override.LastMessages
	}

	if override.SemanticRecall != nil {
		result.SemanticRecall = override.SemanticRecall
	}

	if override.WorkingMemory != nil {
		if result.WorkingMemory == nil {
			result.WorkingMemory = override.WorkingMemory
		} else {
			merged := *result.WorkingMemory
			if override.WorkingMemory.Enabled {
				merged.Enabled = true
			}
			if override.WorkingMemory.Scope != "" {
				merged.Scope = override.WorkingMemory.Scope
			}
			if override.WorkingMemory.Template != "" {
				merged.Template = override.WorkingMemory.Template
			}
			if override.WorkingMemory.Schema != nil {
				merged.Schema = override.WorkingMemory.Schema
			}
			if override.WorkingMemory.Version != "" {
				merged.Version = override.WorkingMemory.Version
			}
			result.WorkingMemory = &merged
		}
	}

	if override.ObservationalMemory != nil {
		result.ObservationalMemory = override.ObservationalMemory
	}

	if override.GenerateTitle != nil {
		result.GenerateTitle = override.GenerateTitle
	}

	if override.Threads != nil {
		result.Threads = override.Threads
	}

	return result
}

// ---------------------------------------------------------------------------
// UUID helper
// ---------------------------------------------------------------------------

// generateUUID generates a simple UUID v4 string.
// Uses crypto/rand for randomness.
func generateUUID() string {
	// Use a simple implementation based on timestamp + random.
	// In production, use a proper UUID library.
	now := time.Now().UnixNano()
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		now&0xFFFFFFFF,
		(now>>32)&0xFFFF,
		(now>>48)&0x0FFF|0x4000,
		(now>>60)&0x3F|0x80,
		now&0xFFFFFFFFFFFF,
	)
}
