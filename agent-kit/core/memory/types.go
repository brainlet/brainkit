// Ported from: packages/core/src/memory/types.ts
package memory

import (
	"errors"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/llm/model"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
	"github.com/brainlet/brainkit/agent-kit/core/storage"
	"github.com/brainlet/brainkit/agent-kit/core/vector"
)

// ---------------------------------------------------------------------------
// Cross-package type stubs and imports
//
// Circular import constraints:
//   - processors → memory  ⇒  memory CANNOT import processors
//   - agent → memory (indirectly)  ⇒  memory CANNOT import agent
//
// AI SDK types (CoreMessage, UserContent, etc.) remain local stubs because
// only V3 (@ai-sdk/provider-v6) has been ported. V4/V5 types are not yet available.
// ---------------------------------------------------------------------------

// CoreMessage is a stub for @internal/ai-sdk-v4.CoreMessage.
// AI SDK V4/V5 types have not been ported; only V3 is available.
type CoreMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content,omitempty"`
}

// UserContent is a stub for @internal/ai-sdk-v4.UserContent.
type UserContent = any

// AssistantContent is a stub for @internal/ai-sdk-v4.AssistantContent.
type AssistantContent = any

// ToolContent is a stub for @internal/ai-sdk-v4.ToolContent.
type ToolContent = any

// MastraDBMessage is a stub for agent/message-list.MastraDBMessage.
// CANNOT import agent: agent imports memory (indirectly via processors).
// Kept as map[string]any to match the untyped message format used across packages.
type MastraDBMessage = map[string]any

// AgentConfig is a stub for agent/types.AgentConfig.
// CANNOT import agent: would create circular dependency.
type AgentConfig = map[string]any

// AgentExecutionOptions is a stub for agent/agent.types.AgentExecutionOptions.
// CANNOT import agent: would create circular dependency.
type AgentExecutionOptions = map[string]any

// EmbeddingModelId is imported from the llm/model package.
type EmbeddingModelId = model.EmbeddingModelID

// ModelRouterModelId is imported from the llm/model package.
type ModelRouterModelId = model.ModelRouterModelID

// MastraLanguageModel is imported from the llm/model package.
type MastraLanguageModel = model.MastraLanguageModel

// MastraModelConfig is imported from the llm/model package.
type MastraModelConfig = model.MastraModelConfig

// MastraCompositeStore is imported from the storage package.
type MastraCompositeStore = storage.MastraCompositeStore

// MastraEmbeddingModel is a stub for vector.EmbeddingModel.
// Real type: vector.EmbeddingModel (interface{}). Effectively the same as any.
// Kept as any for compatibility with usage patterns.
type MastraEmbeddingModel = any

// MastraEmbeddingOptions is a stub for vector.EmbeddingOptions.
// Real type: vector.EmbeddingOptions struct (has MaxRetries, Headers, etc.).
// Kept as map[string]any because memory.go iterates over it with range.
type MastraEmbeddingOptions = map[string]any

// MastraVector is the vector store interface used by the memory system for
// semantic recall embedding storage and retrieval.
// Wired to the real vector.MastraVector interface.
type MastraVector = vector.MastraVector

// DynamicArgument is a stub for types.DynamicArgument[T].
// Real type: types.DynamicArgument[T any] (generic struct with resolver support).
// Kept as any because the generic type parameter cannot be inferred here.
type DynamicArgument = any

// JSONSchema7 is a stub for json-schema.JSONSchema7.
// No Go equivalent library chosen yet.
type JSONSchema7 = map[string]any

// ZodObject is a stub for zod.ZodObject.
// No Go equivalent library chosen yet (schema validation is handled differently).
type ZodObject = any

// MemoryProcessorRef is a forward declaration for the MemoryProcessor interface
// defined in memory.go. Used here to break the circular reference with SharedMemoryConfig.
type MemoryProcessorRef = any

// ---------------------------------------------------------------------------
// Message types
// ---------------------------------------------------------------------------

// MastraMessageV1 represents a v1 message in the memory system.
type MastraMessageV1 struct {
	ID           string    `json:"id"`
	Content      any       `json:"content"` // string | UserContent | AssistantContent | ToolContent
	Role         string    `json:"role"`     // "system" | "user" | "assistant" | "tool"
	CreatedAt    time.Time `json:"createdAt"`
	ThreadID     string    `json:"threadId,omitempty"`
	ResourceID   string    `json:"resourceId,omitempty"`
	ToolCallIDs  []string  `json:"toolCallIds,omitempty"`
	ToolCallArgs []map[string]any `json:"toolCallArgs,omitempty"`
	ToolNames    []string  `json:"toolNames,omitempty"`
	Type         string    `json:"type"` // "text" | "tool-call" | "tool-result"
}

// MessageType is a deprecated alias for MastraMessageV1.
// Deprecated: use MastraMessageV1 or MastraDBMessage.
type MessageType = MastraMessageV1

// AiMessageType is re-exported from @internal/ai-sdk-v4.Message.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type AiMessageType = any

// ---------------------------------------------------------------------------
// Thread types
// ---------------------------------------------------------------------------

// StorageThreadType represents a conversation thread record.
type StorageThreadType struct {
	ID         string          `json:"id"`
	Title      string          `json:"title,omitempty"`
	ResourceID string          `json:"resourceId"`
	CreatedAt  time.Time       `json:"createdAt"`
	UpdatedAt  time.Time       `json:"updatedAt"`
	Metadata   map[string]any  `json:"metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// Thread OM Metadata
// ---------------------------------------------------------------------------

// ThreadOMMetadata holds thread-specific Observational Memory metadata.
// Stored on thread.metadata.mastra.om to keep thread-specific data
// separate from the shared resource-level OM record.
type ThreadOMMetadata struct {
	// CurrentTask is the current task being worked on in this thread.
	CurrentTask string `json:"currentTask,omitempty"`
	// SuggestedResponse is the suggested response for continuing this thread's conversation.
	SuggestedResponse string `json:"suggestedResponse,omitempty"`
	// LastObservedAt is the timestamp of the last observed message (ISO string for JSON serialization).
	LastObservedAt string `json:"lastObservedAt,omitempty"`
}

// ThreadMastraMetadata holds Mastra-specific thread metadata.
// Stored on thread.metadata.mastra.
type ThreadMastraMetadata struct {
	OM *ThreadOMMetadata `json:"om,omitempty"`
}

// isPlainObject checks if a value is a non-nil map[string]any.
func isPlainObject(value any) bool {
	if value == nil {
		return false
	}
	_, ok := value.(map[string]any)
	return ok
}

// GetThreadOMMetadata extracts OM metadata from a thread's metadata object.
// Returns nil if not present or if the structure is invalid.
func GetThreadOMMetadata(threadMetadata map[string]any) *ThreadOMMetadata {
	if threadMetadata == nil {
		return nil
	}
	mastra, ok := threadMetadata["mastra"]
	if !ok || !isPlainObject(mastra) {
		return nil
	}
	mastraMap := mastra.(map[string]any)
	om, ok := mastraMap["om"]
	if !ok || !isPlainObject(om) {
		return nil
	}
	omMap := om.(map[string]any)

	result := &ThreadOMMetadata{}
	if v, ok := omMap["currentTask"].(string); ok {
		result.CurrentTask = v
	}
	if v, ok := omMap["suggestedResponse"].(string); ok {
		result.SuggestedResponse = v
	}
	if v, ok := omMap["lastObservedAt"].(string); ok {
		result.LastObservedAt = v
	}
	return result
}

// SetThreadOMMetadata sets OM metadata on a thread's metadata object.
// Creates the nested structure if it doesn't exist.
// Returns a new metadata object (does not mutate the original).
// Safely handles cases where existing mastra/om values are not objects.
func SetThreadOMMetadata(threadMetadata map[string]any, omMetadata ThreadOMMetadata) map[string]any {
	existing := threadMetadata
	if existing == nil {
		existing = map[string]any{}
	}

	// Clone existing
	result := make(map[string]any, len(existing))
	for k, v := range existing {
		result[k] = v
	}

	var existingMastra map[string]any
	if mastra, ok := existing["mastra"]; ok && isPlainObject(mastra) {
		existingMastra = mastra.(map[string]any)
	} else {
		existingMastra = map[string]any{}
	}

	var existingOM map[string]any
	if om, ok := existingMastra["om"]; ok && isPlainObject(om) {
		existingOM = om.(map[string]any)
	} else {
		existingOM = map[string]any{}
	}

	// Clone existingMastra
	newMastra := make(map[string]any, len(existingMastra))
	for k, v := range existingMastra {
		newMastra[k] = v
	}

	// Clone existingOM and merge omMetadata
	newOM := make(map[string]any, len(existingOM)+3)
	for k, v := range existingOM {
		newOM[k] = v
	}
	if omMetadata.CurrentTask != "" {
		newOM["currentTask"] = omMetadata.CurrentTask
	}
	if omMetadata.SuggestedResponse != "" {
		newOM["suggestedResponse"] = omMetadata.SuggestedResponse
	}
	if omMetadata.LastObservedAt != "" {
		newOM["lastObservedAt"] = omMetadata.LastObservedAt
	}

	newMastra["om"] = newOM
	result["mastra"] = newMastra

	return result
}

// ---------------------------------------------------------------------------
// Memory Request Context
// ---------------------------------------------------------------------------

// MemoryRequestContext holds memory-specific context passed via RequestContext
// under the 'MastraMemory' key.
// This provides processors with access to memory-related execution context.
type MemoryRequestContext struct {
	Thread       *MemoryRequestThread `json:"thread,omitempty"`
	ResourceID   string               `json:"resourceId,omitempty"`
	MemoryConfig *MemoryConfig        `json:"memoryConfig,omitempty"`
}

// MemoryRequestThread represents a partial thread with a required ID.
type MemoryRequestThread struct {
	ID         string          `json:"id"`
	Title      string          `json:"title,omitempty"`
	ResourceID string          `json:"resourceId,omitempty"`
	CreatedAt  *time.Time      `json:"createdAt,omitempty"`
	UpdatedAt  *time.Time      `json:"updatedAt,omitempty"`
	Metadata   map[string]any  `json:"metadata,omitempty"`
}

// ParseMemoryRequestContext parses and validates memory runtime context from RequestContext.
// Returns the validated MemoryRequestContext or nil if not available.
// Returns an error if the context exists but is malformed.
func ParseMemoryRequestContext(rc *requestcontext.RequestContext) (*MemoryRequestContext, error) {
	if rc == nil {
		return nil, nil
	}

	memoryContext := rc.Get("MastraMemory")
	if memoryContext == nil {
		return nil, nil
	}

	// Validate the structure
	ctx, ok := memoryContext.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid MemoryRequestContext: expected object, got %T", memoryContext)
	}

	// If it's already a *MemoryRequestContext, return directly
	if mrc, ok := memoryContext.(*MemoryRequestContext); ok {
		return mrc, nil
	}

	// Validate thread if present
	if threadVal, ok := ctx["thread"]; ok && threadVal != nil {
		threadMap, ok := threadVal.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid MemoryRequestContext.thread: expected object, got %T", threadVal)
		}
		id, ok := threadMap["id"].(string)
		if !ok {
			return nil, errors.New("invalid MemoryRequestContext.thread.id: expected string")
		}
		_ = id
	}

	// Validate resourceId if present
	if resourceIDVal, ok := ctx["resourceId"]; ok && resourceIDVal != nil {
		if _, ok := resourceIDVal.(string); !ok {
			return nil, fmt.Errorf("invalid MemoryRequestContext.resourceId: expected string, got %T", resourceIDVal)
		}
	}

	// Return as the raw context (callers typically use type assertion anyway)
	// For a faithful port we return the typed struct when possible.
	result := &MemoryRequestContext{}
	if resourceID, ok := ctx["resourceId"].(string); ok {
		result.ResourceID = resourceID
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// MessageResponse
// ---------------------------------------------------------------------------

// MessageResponse represents the result type for message retrieval.
// The format parameter distinguishes between raw (MastraMessageV1) and
// core_message (CoreMessage) formats.
type MessageResponse struct {
	Raw         []MastraMessageV1 `json:"raw,omitempty"`
	CoreMessage []CoreMessage     `json:"core_message,omitempty"`
}

// ---------------------------------------------------------------------------
// WorkingMemory types
// ---------------------------------------------------------------------------

// WorkingMemoryScope defines the scope for working memory storage.
type WorkingMemoryScope string

const (
	// WorkingMemoryScopeThread means memory is isolated per conversation thread.
	WorkingMemoryScopeThread WorkingMemoryScope = "thread"
	// WorkingMemoryScopeResource means memory persists across all threads for the same resource/user (default).
	WorkingMemoryScopeResource WorkingMemoryScope = "resource"
)

// WorkingMemory holds configuration for working memory.
// In TypeScript this is a discriminated union (template vs schema vs none).
// In Go we use a single struct with mutually exclusive fields.
type WorkingMemory struct {
	Enabled  bool               `json:"enabled"`
	// Scope controls where working memory is stored.
	// 'resource': persists across all threads for the same resource/user (default).
	// 'thread': isolated per conversation thread.
	Scope    WorkingMemoryScope `json:"scope,omitempty"`
	// Template is a markdown template for working memory. Mutually exclusive with Schema.
	Template string             `json:"template,omitempty"`
	// Schema is a JSON schema or Zod schema for working memory. Mutually exclusive with Template.
	Schema   any                `json:"schema,omitempty"`
	// Version controls working memory behavior. "stable" or "vnext".
	Version  string             `json:"version,omitempty"`
}

// ---------------------------------------------------------------------------
// Vector Index Config
// ---------------------------------------------------------------------------

// VectorIndexConfig holds vector index configuration for optimizing semantic recall performance.
// These settings are primarily supported by PostgreSQL with pgvector extension.
type VectorIndexConfig struct {
	// Type of vector index to create (PostgreSQL/pgvector only).
	// 'ivfflat', 'hnsw', or 'flat'.
	Type string `json:"type,omitempty"`
	// Metric is the distance metric for similarity calculations.
	// 'cosine', 'euclidean', or 'dotproduct'.
	Metric string `json:"metric,omitempty"`
	// IVF holds configuration for IVFFlat index (PostgreSQL only).
	IVF *IVFConfig `json:"ivf,omitempty"`
	// HNSW holds configuration for HNSW index (PostgreSQL only).
	HNSW *HNSWConfig `json:"hnsw,omitempty"`
}

// IVFConfig holds configuration for IVFFlat index.
type IVFConfig struct {
	// Lists is the number of inverted lists (clusters) to create.
	Lists int `json:"lists,omitempty"`
}

// HNSWConfig holds configuration for HNSW index.
type HNSWConfig struct {
	// M is the maximum number of bi-directional links per node.
	M int `json:"m,omitempty"`
	// EfConstruction is the size of the dynamic candidate list during index construction.
	EfConstruction int `json:"efConstruction,omitempty"`
}

// ---------------------------------------------------------------------------
// SemanticRecall
// ---------------------------------------------------------------------------

// SemanticRecallScope defines the scope for semantic search queries.
type SemanticRecallScope string

const (
	// SemanticRecallScopeThread limits recall to current thread only.
	SemanticRecallScopeThread SemanticRecallScope = "thread"
	// SemanticRecallScopeResource searches across all threads for the same resource/user (default).
	SemanticRecallScopeResource SemanticRecallScope = "resource"
)

// SemanticRecallMessageRange can be a single number (same before/after)
// or separate before/after values.
type SemanticRecallMessageRange struct {
	// Symmetric is used when a single number applies to both before and after.
	// Only used when Before and After are both zero.
	Symmetric int `json:"symmetric,omitempty"`
	// Before is the number of messages to include before the match.
	Before int `json:"before,omitempty"`
	// After is the number of messages to include after the match.
	After int `json:"after,omitempty"`
}

// SemanticRecall holds configuration for semantic recall using RAG-based retrieval.
type SemanticRecall struct {
	// TopK is the number of semantically similar messages to retrieve.
	TopK int `json:"topK"`
	// MessageRange controls how much surrounding context to include with each retrieved message.
	MessageRange SemanticRecallMessageRange `json:"messageRange"`
	// Scope for semantic search queries. 'resource' (default) or 'thread'.
	Scope SemanticRecallScope `json:"scope,omitempty"`
	// IndexConfig holds vector index configuration (PostgreSQL/pgvector specific).
	IndexConfig *VectorIndexConfig `json:"indexConfig,omitempty"`
	// Threshold is the minimum similarity score threshold (0-1).
	Threshold *float64 `json:"threshold,omitempty"`
	// IndexName is the custom index name for the vector store.
	IndexName string `json:"indexName,omitempty"`
}

// ---------------------------------------------------------------------------
// Observational Memory
// ---------------------------------------------------------------------------

// ObservationalMemoryModelSettings holds model settings for Observer/Reflector agents.
// Uses the same settings as Agent.generate() modelSettings.
type ObservationalMemoryModelSettings = map[string]any

// ObservationalMemoryObservationConfig holds configuration for the observation step.
type ObservationalMemoryObservationConfig struct {
	// Model for the Observer agent.
	Model any `json:"model,omitempty"`
	// MessageTokens is the token count of unobserved messages that triggers observation.
	// Default: 30000.
	MessageTokens *int `json:"messageTokens,omitempty"`
	// ModelSettings for the Observer agent.
	ModelSettings ObservationalMemoryModelSettings `json:"modelSettings,omitempty"`
	// ProviderOptions are provider-specific options passed to the Observer model.
	ProviderOptions map[string]map[string]any `json:"providerOptions,omitempty"`
	// MaxTokensPerBatch is the maximum tokens per batch when observing multiple threads.
	// Default: 10000.
	MaxTokensPerBatch *int `json:"maxTokensPerBatch,omitempty"`
	// BufferTokens is the token interval for async background observation buffering.
	// Can be a number or false (nil pointer) to disable.
	// Default: 0.2 (20% of messageTokens).
	BufferTokens *float64 `json:"bufferTokens,omitempty"`
	// BufferTokensDisabled explicitly disables async buffering when true.
	BufferTokensDisabled bool `json:"bufferTokensDisabled,omitempty"`
	// BufferActivation is the ratio (0-1) of buffered observations to activate.
	// Default: 0.8.
	BufferActivation *float64 `json:"bufferActivation,omitempty"`
	// BlockAfter is the token threshold above which synchronous observation is forced.
	BlockAfter *float64 `json:"blockAfter,omitempty"`
	// Instruction is custom instructions appended to the Observer agent's system prompt.
	Instruction string `json:"instruction,omitempty"`
}

// ObservationalMemoryReflectionConfig holds configuration for the reflection step.
type ObservationalMemoryReflectionConfig struct {
	// Model for the Reflector agent.
	Model any `json:"model,omitempty"`
	// ObservationTokens is the token count that triggers reflection.
	// Default: 40000.
	ObservationTokens *int `json:"observationTokens,omitempty"`
	// ModelSettings for the Reflector agent.
	ModelSettings ObservationalMemoryModelSettings `json:"modelSettings,omitempty"`
	// ProviderOptions are provider-specific options passed to the Reflector model.
	ProviderOptions map[string]map[string]any `json:"providerOptions,omitempty"`
	// BlockAfter is the token threshold above which synchronous reflection is forced.
	BlockAfter *float64 `json:"blockAfter,omitempty"`
	// BufferActivation is the ratio (0-1) controlling when async reflection buffering starts.
	BufferActivation *float64 `json:"bufferActivation,omitempty"`
	// Instruction is custom instructions appended to the Reflector agent's system prompt.
	Instruction string `json:"instruction,omitempty"`
}

// ObservationalMemoryOptions holds configuration for Observational Memory.
type ObservationalMemoryOptions struct {
	// Enabled controls whether Observational Memory is active. Default: true.
	Enabled *bool `json:"enabled,omitempty"`
	// Model for both Observer and Reflector agents.
	Model any `json:"model,omitempty"`
	// Observation holds observation step configuration.
	Observation *ObservationalMemoryObservationConfig `json:"observation,omitempty"`
	// Reflection holds reflection step configuration.
	Reflection *ObservationalMemoryReflectionConfig `json:"reflection,omitempty"`
	// Scope for observations. 'resource' or 'thread' (default).
	Scope string `json:"scope,omitempty"`
	// ShareTokenBudget shares the token budget between messages and observations.
	ShareTokenBudget bool `json:"shareTokenBudget,omitempty"`
}

// IsObservationalMemoryEnabled checks if observational memory is enabled.
//
// Semantics:
//   - observationalMemory == nil → disabled
//   - observationalMemory is a bool: true → enabled, false → disabled
//   - observationalMemory is *ObservationalMemoryOptions: enabled unless Enabled == false
//
// In Go, since we can't have a union type of bool | ObservationalMemoryOptions,
// the caller should use the typed helpers or check the MemoryConfig fields directly.
func IsObservationalMemoryEnabled(config *ObservationalMemoryConfig) bool {
	if config == nil {
		return false
	}
	if config.BoolValue != nil {
		return *config.BoolValue
	}
	if config.Options != nil {
		return config.Options.Enabled == nil || *config.Options.Enabled
	}
	return false
}

// ObservationalMemoryConfig wraps the TypeScript union type
// `boolean | ObservationalMemoryOptions`. In Go we use a struct
// with mutually exclusive fields.
type ObservationalMemoryConfig struct {
	// BoolValue is set when the config is a simple boolean.
	BoolValue *bool `json:"boolValue,omitempty"`
	// Options is set when the config is an ObservationalMemoryOptions object.
	Options *ObservationalMemoryOptions `json:"options,omitempty"`
}

// NewObservationalMemoryConfigBool creates an ObservationalMemoryConfig from a boolean.
func NewObservationalMemoryConfigBool(enabled bool) *ObservationalMemoryConfig {
	return &ObservationalMemoryConfig{BoolValue: &enabled}
}

// NewObservationalMemoryConfigOptions creates an ObservationalMemoryConfig from options.
func NewObservationalMemoryConfigOptions(opts ObservationalMemoryOptions) *ObservationalMemoryConfig {
	return &ObservationalMemoryConfig{Options: &opts}
}

// ---------------------------------------------------------------------------
// MemoryConfig
// ---------------------------------------------------------------------------

// SemanticRecallConfig wraps the TypeScript union type `boolean | SemanticRecall`.
// In Go we use a struct with mutually exclusive fields.
type SemanticRecallConfig struct {
	// BoolValue is set when the config is a simple boolean.
	BoolValue *bool `json:"boolValue,omitempty"`
	// Options is set when the config is a SemanticRecall object.
	Options *SemanticRecall `json:"options,omitempty"`
}

// IsEnabled returns whether semantic recall is enabled.
func (s *SemanticRecallConfig) IsEnabled() bool {
	if s == nil {
		return false
	}
	if s.BoolValue != nil {
		return *s.BoolValue
	}
	return s.Options != nil
}

// NewSemanticRecallConfigBool creates a SemanticRecallConfig from a boolean.
func NewSemanticRecallConfigBool(enabled bool) *SemanticRecallConfig {
	return &SemanticRecallConfig{BoolValue: &enabled}
}

// NewSemanticRecallConfigOptions creates a SemanticRecallConfig from options.
func NewSemanticRecallConfigOptions(opts SemanticRecall) *SemanticRecallConfig {
	return &SemanticRecallConfig{Options: &opts}
}

// LastMessagesConfig wraps the TypeScript union type `number | false`.
// In Go we use a struct.
type LastMessagesConfig struct {
	// Count is the number of recent messages to include. Zero means use default.
	Count int `json:"count,omitempty"`
	// Disabled is true when lastMessages is explicitly set to false.
	Disabled bool `json:"disabled,omitempty"`
}

// IsEnabled returns whether last messages retrieval is enabled.
func (l *LastMessagesConfig) IsEnabled() bool {
	if l == nil {
		return true // default is enabled
	}
	return !l.Disabled
}

// GetCount returns the configured count, or the default if not set.
func (l *LastMessagesConfig) GetCount(defaultCount int) int {
	if l == nil || l.Count == 0 {
		return defaultCount
	}
	return l.Count
}

// GenerateTitleConfig wraps the TypeScript union type
// `boolean | { model: DynamicArgument<MastraModelConfig>; instructions?: DynamicArgument<string> }`.
type GenerateTitleConfig struct {
	// BoolValue is set when the config is a simple boolean.
	BoolValue *bool `json:"boolValue,omitempty"`
	// Options is set when the config is an object with model and instructions.
	Options *GenerateTitleOptions `json:"options,omitempty"`
}

// GenerateTitleOptions holds configuration for title generation.
type GenerateTitleOptions struct {
	// Model is the language model for title generation.
	Model any `json:"model,omitempty"`
	// Instructions are custom instructions for title generation.
	Instructions any `json:"instructions,omitempty"`
}

// IsEnabled returns whether title generation is enabled.
func (g *GenerateTitleConfig) IsEnabled() bool {
	if g == nil {
		return false
	}
	if g.BoolValue != nil {
		return *g.BoolValue
	}
	return g.Options != nil
}

// ThreadsConfig holds deprecated thread management configuration.
// Deprecated: Use top-level GenerateTitle instead of Threads.GenerateTitle.
type ThreadsConfig struct {
	// GenerateTitle is deprecated. Use top-level GenerateTitle.
	GenerateTitle *GenerateTitleConfig `json:"generateTitle,omitempty"`
}

// MemoryConfig holds configuration for memory behaviors and retrieval strategies.
// Controls three types of memory: conversation history (recent messages), semantic recall
// (RAG-based retrieval of relevant past messages), and working memory (persistent user data).
type MemoryConfig struct {
	// ReadOnly prevents memory from saving new messages when true.
	ReadOnly bool `json:"readOnly,omitempty"`
	// LastMessages controls how many recent messages to include.
	// nil means use default (10). Use LastMessagesConfig{Disabled: true} to disable.
	LastMessages *LastMessagesConfig `json:"lastMessages,omitempty"`
	// SemanticRecall holds semantic recall configuration.
	SemanticRecall *SemanticRecallConfig `json:"semanticRecall,omitempty"`
	// WorkingMemory holds working memory configuration.
	WorkingMemory *WorkingMemory `json:"workingMemory,omitempty"`
	// ObservationalMemory holds observational memory configuration.
	ObservationalMemory *ObservationalMemoryConfig `json:"observationalMemory,omitempty"`
	// GenerateTitle controls automatic thread title generation.
	GenerateTitle *GenerateTitleConfig `json:"generateTitle,omitempty"`
	// Threads holds deprecated thread management configuration.
	// Deprecated: Use top-level GenerateTitle instead.
	Threads *ThreadsConfig `json:"threads,omitempty"`
}

// ---------------------------------------------------------------------------
// SharedMemoryConfig
// ---------------------------------------------------------------------------

// SharedMemoryConfig holds configuration for Mastra's memory system.
// Enables agents to persist and recall information across conversations.
type SharedMemoryConfig struct {
	// Storage adapter for persisting conversation threads, messages, and working memory.
	Storage MastraCompositeStore `json:"storage,omitempty"`
	// Options holds memory behavior configuration.
	Options *MemoryConfig `json:"options,omitempty"`
	// Vector database for semantic recall capabilities.
	Vector MastraVector `json:"vector,omitempty"`
	// Embedder is the embedding model for vector representations.
	Embedder any `json:"embedder,omitempty"` // EmbeddingModelId | MastraEmbeddingModel | string
	// EmbedderOptions holds options for the embedder.
	EmbedderOptions MastraEmbeddingOptions `json:"embedderOptions,omitempty"`
	// Processors is deprecated. Use the new Input/Output processor system instead.
	// Deprecated: Will throw an error if used.
	Processors []MemoryProcessorRef `json:"processors,omitempty"`
}

// ---------------------------------------------------------------------------
// WorkingMemory format and template
// ---------------------------------------------------------------------------

// WorkingMemoryFormat represents the format of a working memory template.
type WorkingMemoryFormat string

const (
	// WorkingMemoryFormatJSON indicates JSON format.
	WorkingMemoryFormatJSON WorkingMemoryFormat = "json"
	// WorkingMemoryFormatMarkdown indicates Markdown format.
	WorkingMemoryFormatMarkdown WorkingMemoryFormat = "markdown"
)

// WorkingMemoryTemplate holds a working memory template with its format.
type WorkingMemoryTemplate struct {
	Format  WorkingMemoryFormat `json:"format"`
	Content string              `json:"content"`
}

// ---------------------------------------------------------------------------
// MessageDeleteInput
// ---------------------------------------------------------------------------

// MessageDeleteInput represents the input for deleting messages.
// Can be a slice of string IDs or a slice of objects with an ID field.
// In Go we use a slice of strings; callers should extract IDs from objects.
type MessageDeleteInput = []string

// ---------------------------------------------------------------------------
// Storage List types (re-exported stubs)
// ---------------------------------------------------------------------------

// StorageListMessagesInput is the input for listing messages.
// STUB REASON: The real storage.StorageListMessagesInput is a struct with ThreadID,
// ResourceID, Include fields and embeds StorageListMessagesOptions. This stub uses
// map[string]any because the memory package methods pass these as generic maps.
type StorageListMessagesInput = map[string]any

// StorageListThreadsInput is the input for listing threads.
// STUB REASON: The real storage.StorageListThreadsInput is a struct with PerPage,
// Page, OrderBy, Filter fields. This stub uses map[string]any for generic pass-through.
type StorageListThreadsInput = map[string]any

// StorageListThreadsOutput is the output from listing threads.
// STUB REASON: The real storage.StorageListThreadsOutput is a struct with PaginationInfo
// embedded and Threads []StorageThreadType. This stub uses map[string]any.
type StorageListThreadsOutput = map[string]any

// StorageCloneThreadInput is the input for cloning a thread.
// STUB REASON: The real storage.StorageCloneThreadInput is a struct with SourceThreadID,
// NewThreadID, ResourceID, etc. This stub uses map[string]any.
type StorageCloneThreadInput = map[string]any

// StorageCloneThreadOutput is the output from cloning a thread.
// STUB REASON: The real storage.StorageCloneThreadOutput is a struct with Thread,
// ClonedMessages, MessageIDMap fields. This stub uses map[string]any.
type StorageCloneThreadOutput = map[string]any

// ---------------------------------------------------------------------------
// Serialized Memory Config
// ---------------------------------------------------------------------------

// SerializedMemoryConfig holds a serializable memory configuration
// that can be stored in the database.
type SerializedMemoryConfig struct {
	// Vector is the vector database identifier.
	Vector any `json:"vector,omitempty"` // string | false
	// Options holds serializable memory behavior configuration.
	Options *SerializedMemoryOptions `json:"options,omitempty"`
	// Embedder is the embedding model ID.
	Embedder any `json:"embedder,omitempty"` // EmbeddingModelId | string
	// EmbedderOptions holds options for the embedder (without telemetry).
	EmbedderOptions map[string]any `json:"embedderOptions,omitempty"`
	// ObservationalMemory holds serialized observational memory configuration.
	ObservationalMemory any `json:"observationalMemory,omitempty"` // bool | SerializedObservationalMemoryConfig
}

// SerializedMemoryOptions holds serializable memory behavior configuration.
type SerializedMemoryOptions struct {
	// ReadOnly prevents memory from saving new messages.
	ReadOnly bool `json:"readOnly,omitempty"`
	// LastMessages controls how many recent messages to include.
	LastMessages any `json:"lastMessages,omitempty"` // number | false
	// SemanticRecall holds semantic recall configuration.
	SemanticRecall any `json:"semanticRecall,omitempty"` // bool | SemanticRecall
	// GenerateTitle holds title generation configuration.
	GenerateTitle any `json:"generateTitle,omitempty"` // bool | { model, instructions }
}

// SerializedObservationalMemoryConfig holds JSON-serializable observational memory config.
type SerializedObservationalMemoryConfig struct {
	// Model is the model ID for both Observer and Reflector.
	Model string `json:"model,omitempty"`
	// Scope is the memory scope: 'resource' or 'thread'.
	Scope string `json:"scope,omitempty"`
	// ShareTokenBudget shares the token budget between messages and observations.
	ShareTokenBudget bool `json:"shareTokenBudget,omitempty"`
	// Observation holds observation step configuration.
	Observation *SerializedObservationalMemoryObservationConfig `json:"observation,omitempty"`
	// Reflection holds reflection step configuration.
	Reflection *SerializedObservationalMemoryReflectionConfig `json:"reflection,omitempty"`
}

// SerializedObservationalMemoryObservationConfig holds serializable observation config.
type SerializedObservationalMemoryObservationConfig struct {
	Model             string                         `json:"model,omitempty"`
	MessageTokens     *int                           `json:"messageTokens,omitempty"`
	ModelSettings     map[string]any                 `json:"modelSettings,omitempty"`
	ProviderOptions   map[string]map[string]any      `json:"providerOptions,omitempty"`
	MaxTokensPerBatch *int                           `json:"maxTokensPerBatch,omitempty"`
	BufferTokens      any                            `json:"bufferTokens,omitempty"` // number | false
	BufferActivation  *float64                       `json:"bufferActivation,omitempty"`
	BlockAfter        *float64                       `json:"blockAfter,omitempty"`
}

// SerializedObservationalMemoryReflectionConfig holds serializable reflection config.
type SerializedObservationalMemoryReflectionConfig struct {
	Model             string                         `json:"model,omitempty"`
	ObservationTokens *int                           `json:"observationTokens,omitempty"`
	ModelSettings     map[string]any                 `json:"modelSettings,omitempty"`
	ProviderOptions   map[string]map[string]any      `json:"providerOptions,omitempty"`
	BlockAfter        *float64                       `json:"blockAfter,omitempty"`
	BufferActivation  *float64                       `json:"bufferActivation,omitempty"`
}
