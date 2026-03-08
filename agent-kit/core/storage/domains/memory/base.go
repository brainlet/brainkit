// Ported from: packages/core/src/storage/domains/memory/base.ts
package memory

import (
	"context"
	"time"
)

// ---------------------------------------------------------------------------
// Domain Types
// ---------------------------------------------------------------------------

// StorageThreadType represents a thread record as a map.
// Fields: id, title, resourceId, createdAt, updatedAt, metadata.
type StorageThreadType = map[string]any

// MastraDBMessage represents a stored message as a map.
// Fields: id, threadId, content, role, type, createdAt, resourceId.
type MastraDBMessage = map[string]any

// StorageMessageType represents a message in storage format.
// Fields: id, thread_id, content (JSON string), role, type, createdAt, resourceId.
type StorageMessageType = map[string]any

// StorageResourceType represents a resource record as a map.
// Fields: id, workingMemory, metadata, createdAt, updatedAt.
type StorageResourceType = map[string]any

// ---------------------------------------------------------------------------
// Pagination & Ordering
// ---------------------------------------------------------------------------

// ThreadOrderBy is the field to order threads by.
type ThreadOrderBy = string

// ThreadSortDirection is the sort direction.
type ThreadSortDirection = string

// StorageOrderBy holds ordering configuration.
type StorageOrderBy struct {
	Field     ThreadOrderBy     `json:"field,omitempty"`
	Direction ThreadSortDirection `json:"direction,omitempty"`
}

// DateRangeFilter describes inclusive/exclusive date bounds for filtering.
type DateRangeFilter struct {
	Start          *time.Time `json:"start,omitempty"`
	End            *time.Time `json:"end,omitempty"`
	StartExclusive bool       `json:"startExclusive,omitempty"`
	EndExclusive   bool       `json:"endExclusive,omitempty"`
}

// MessageIncludeItem specifies a message to include with optional context.
type MessageIncludeItem struct {
	ID                   string `json:"id"`
	ThreadID             string `json:"threadId,omitempty"`
	WithPreviousMessages int    `json:"withPreviousMessages,omitempty"`
	WithNextMessages     int    `json:"withNextMessages,omitempty"`
}

// StorageListMessagesInput is the input for listing messages.
type StorageListMessagesInput struct {
	ThreadID   any                 `json:"threadId"` // string or []string
	ResourceID string              `json:"resourceId,omitempty"`
	Include    []MessageIncludeItem `json:"include,omitempty"`
	PerPage    *int                `json:"perPage,omitempty"` // nil = default (40)
	Page       int                 `json:"page,omitempty"`
	Filter     *MessagesFilter     `json:"filter,omitempty"`
	OrderBy    *StorageOrderBy     `json:"orderBy,omitempty"`
}

// MessagesFilter holds filter criteria for listing messages.
type MessagesFilter struct {
	DateRange *DateRangeFilter `json:"dateRange,omitempty"`
}

// StorageListMessagesByResourceIDInput is the input for listing messages by resource ID.
type StorageListMessagesByResourceIDInput struct {
	ResourceID string          `json:"resourceId"`
	PerPage    *int            `json:"perPage,omitempty"`
	Page       int             `json:"page,omitempty"`
	Filter     *MessagesFilter `json:"filter,omitempty"`
	OrderBy    *StorageOrderBy `json:"orderBy,omitempty"`
}

// StorageListMessagesOutput is the output for listing messages.
type StorageListMessagesOutput struct {
	Messages []MastraDBMessage `json:"messages"`
	Total    int               `json:"total"`
	Page     int               `json:"page"`
	PerPage  int               `json:"perPage"`
	HasMore  bool              `json:"hasMore"`
}

// StorageListThreadsInput is the input for listing threads.
type StorageListThreadsInput struct {
	PerPage *int            `json:"perPage,omitempty"` // nil = default (100)
	Page    int             `json:"page,omitempty"`
	OrderBy *StorageOrderBy `json:"orderBy,omitempty"`
	Filter  *ThreadsFilter  `json:"filter,omitempty"`
}

// ThreadsFilter holds filter criteria for listing threads.
type ThreadsFilter struct {
	ResourceID string         `json:"resourceId,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// StorageListThreadsOutput is the output for listing threads.
type StorageListThreadsOutput struct {
	Threads []StorageThreadType `json:"threads"`
	Total   int                 `json:"total"`
	Page    int                 `json:"page"`
	PerPage int                 `json:"perPage"`
	HasMore bool                `json:"hasMore"`
}

// ThreadCloneMetadata holds metadata about a cloned thread's origin.
type ThreadCloneMetadata struct {
	SourceThreadID string    `json:"sourceThreadId"`
	ClonedAt       time.Time `json:"clonedAt"`
	LastMessageID  string    `json:"lastMessageId,omitempty"`
}

// StorageCloneThreadInput is the input for cloning a thread.
type StorageCloneThreadInput struct {
	SourceThreadID string              `json:"sourceThreadId"`
	NewThreadID    string              `json:"newThreadId,omitempty"`
	ResourceID     string              `json:"resourceId,omitempty"`
	Title          string              `json:"title,omitempty"`
	Metadata       map[string]any      `json:"metadata,omitempty"`
	Options        *CloneThreadOptions `json:"options,omitempty"`
}

// CloneThreadOptions holds options for filtering cloned messages.
type CloneThreadOptions struct {
	MessageLimit  int                  `json:"messageLimit,omitempty"`
	MessageFilter *CloneMessageFilter  `json:"messageFilter,omitempty"`
}

// CloneMessageFilter holds filter criteria for clone message selection.
type CloneMessageFilter struct {
	StartDate  *time.Time `json:"startDate,omitempty"`
	EndDate    *time.Time `json:"endDate,omitempty"`
	MessageIDs []string   `json:"messageIds,omitempty"`
}

// StorageCloneThreadOutput is the output from cloning a thread.
type StorageCloneThreadOutput struct {
	Thread         StorageThreadType  `json:"thread"`
	ClonedMessages []MastraDBMessage  `json:"clonedMessages"`
	MessageIDMap   map[string]string  `json:"messageIdMap,omitempty"`
}

// ---------------------------------------------------------------------------
// Observational Memory Types
// ---------------------------------------------------------------------------

// ObservationalMemoryScope defines the scope for observational memory.
type ObservationalMemoryScope = string

const (
	// ObservationalMemoryScopeThread limits observations to a single thread.
	ObservationalMemoryScopeThread ObservationalMemoryScope = "thread"
	// ObservationalMemoryScopeResource observations span all threads for a resource.
	ObservationalMemoryScopeResource ObservationalMemoryScope = "resource"
)

// ObservationalMemoryOriginType describes how an OM record was created.
type ObservationalMemoryOriginType = string

const (
	// ObservationalMemoryOriginInitial is the initial observation record.
	ObservationalMemoryOriginInitial ObservationalMemoryOriginType = "initial"
	// ObservationalMemoryOriginReflection is a record created via reflection.
	ObservationalMemoryOriginReflection ObservationalMemoryOriginType = "reflection"
)

// BufferedObservationChunk is a chunk of buffered observations from a single cycle.
type BufferedObservationChunk struct {
	ID                    string    `json:"id"`
	CycleID               string    `json:"cycleId"`
	Observations          string    `json:"observations"`
	TokenCount            int       `json:"tokenCount"`
	MessageIDs            []string  `json:"messageIds"`
	MessageTokens         int       `json:"messageTokens"`
	LastObservedAt        time.Time `json:"lastObservedAt"`
	CreatedAt             time.Time `json:"createdAt"`
	SuggestedContinuation string    `json:"suggestedContinuation,omitempty"`
	CurrentTask           string    `json:"currentTask,omitempty"`
}

// BufferedObservationChunkInput is the input for creating a buffered observation chunk.
type BufferedObservationChunkInput struct {
	CycleID               string    `json:"cycleId"`
	Observations          string    `json:"observations"`
	TokenCount            int       `json:"tokenCount"`
	MessageIDs            []string  `json:"messageIds"`
	MessageTokens         int       `json:"messageTokens"`
	LastObservedAt        time.Time `json:"lastObservedAt"`
	SuggestedContinuation string    `json:"suggestedContinuation,omitempty"`
	CurrentTask           string    `json:"currentTask,omitempty"`
}

// ObservationalMemoryRecord is the core database record for observational memory.
type ObservationalMemoryRecord struct {
	// Identity
	ID         string                        `json:"id"`
	Scope      ObservationalMemoryScope      `json:"scope"`
	ThreadID   string                        `json:"threadId"`   // empty for resource scope
	ResourceID string                        `json:"resourceId"`

	// Timestamps
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	LastObservedAt *time.Time `json:"lastObservedAt,omitempty"`

	// Generation tracking
	OriginType      ObservationalMemoryOriginType `json:"originType"`
	GenerationCount int                           `json:"generationCount"`

	// Observation content
	ActiveObservations       string                     `json:"activeObservations"`
	BufferedObservationChunks []BufferedObservationChunk `json:"bufferedObservationChunks,omitempty"`
	// Deprecated fields (legacy compatibility)
	BufferedObservations     string   `json:"bufferedObservations,omitempty"`
	BufferedObservationTokens *int    `json:"bufferedObservationTokens,omitempty"`
	BufferedMessageIDs       []string `json:"bufferedMessageIds,omitempty"`
	// Reflection buffering
	BufferedReflection           string `json:"bufferedReflection,omitempty"`
	BufferedReflectionTokens     *int   `json:"bufferedReflectionTokens,omitempty"`
	BufferedReflectionInputTokens *int  `json:"bufferedReflectionInputTokens,omitempty"`
	ReflectedObservationLineCount *int  `json:"reflectedObservationLineCount,omitempty"`

	// Message tracking
	ObservedMessageIDs []string `json:"observedMessageIds,omitempty"`

	// Timezone
	ObservedTimezone string `json:"observedTimezone,omitempty"`

	// Token tracking
	TotalTokensObserved   int `json:"totalTokensObserved"`
	ObservationTokenCount int `json:"observationTokenCount"`
	PendingMessageTokens  int `json:"pendingMessageTokens"`

	// State flags
	IsReflecting           bool `json:"isReflecting"`
	IsObserving            bool `json:"isObserving"`
	IsBufferingObservation bool `json:"isBufferingObservation"`
	IsBufferingReflection  bool `json:"isBufferingReflection"`
	LastBufferedAtTokens   int  `json:"lastBufferedAtTokens"`
	LastBufferedAtTime     *time.Time `json:"lastBufferedAtTime"`

	// Configuration
	Config map[string]any `json:"config"`

	// Extensible metadata
	Metadata map[string]any `json:"metadata,omitempty"`
}

// CreateObservationalMemoryInput is the input for creating observational memory.
type CreateObservationalMemoryInput struct {
	ThreadID         string                   `json:"threadId"`
	ResourceID       string                   `json:"resourceId"`
	Scope            ObservationalMemoryScope `json:"scope"`
	Config           map[string]any           `json:"config"`
	ObservedTimezone string                   `json:"observedTimezone,omitempty"`
}

// UpdateActiveObservationsInput is the input for updating active observations.
type UpdateActiveObservationsInput struct {
	ID                 string    `json:"id"`
	Observations       string    `json:"observations"`
	TokenCount         int       `json:"tokenCount"`
	LastObservedAt     time.Time `json:"lastObservedAt"`
	ObservedMessageIDs []string  `json:"observedMessageIds,omitempty"`
	ObservedTimezone   string    `json:"observedTimezone,omitempty"`
}

// UpdateBufferedObservationsInput is the input for updating buffered observations.
type UpdateBufferedObservationsInput struct {
	ID                string                        `json:"id"`
	Chunk             BufferedObservationChunkInput  `json:"chunk"`
	LastBufferedAtTime *time.Time                   `json:"lastBufferedAtTime,omitempty"`
}

// SwapBufferedToActiveInput is the input for swapping buffered observations to active.
type SwapBufferedToActiveInput struct {
	ID                     string     `json:"id"`
	ActivationRatio        float64    `json:"activationRatio"`
	MessageTokensThreshold int        `json:"messageTokensThreshold"`
	CurrentPendingTokens   int        `json:"currentPendingTokens"`
	ForceMaxActivation     bool       `json:"forceMaxActivation,omitempty"`
	LastObservedAt         *time.Time `json:"lastObservedAt,omitempty"`
}

// SwapBufferedToActivePerChunk holds per-chunk activation breakdown.
type SwapBufferedToActivePerChunk struct {
	CycleID           string `json:"cycleId"`
	MessageTokens     int    `json:"messageTokens"`
	ObservationTokens int    `json:"observationTokens"`
	MessageCount      int    `json:"messageCount"`
	Observations      string `json:"observations"`
}

// SwapBufferedToActiveResult is the result of swapping buffered observations to active.
type SwapBufferedToActiveResult struct {
	ChunksActivated            int                            `json:"chunksActivated"`
	MessageTokensActivated     int                            `json:"messageTokensActivated"`
	ObservationTokensActivated int                            `json:"observationTokensActivated"`
	MessagesActivated          int                            `json:"messagesActivated"`
	ActivatedCycleIDs          []string                       `json:"activatedCycleIds"`
	ActivatedMessageIDs        []string                       `json:"activatedMessageIds"`
	Observations               string                         `json:"observations,omitempty"`
	PerChunk                   []SwapBufferedToActivePerChunk  `json:"perChunk,omitempty"`
	SuggestedContinuation      string                         `json:"suggestedContinuation,omitempty"`
	CurrentTask                string                         `json:"currentTask,omitempty"`
}

// UpdateBufferedReflectionInput is the input for updating buffered reflections.
type UpdateBufferedReflectionInput struct {
	ID                            string `json:"id"`
	Reflection                    string `json:"reflection"`
	TokenCount                    int    `json:"tokenCount"`
	InputTokenCount               int    `json:"inputTokenCount"`
	ReflectedObservationLineCount int    `json:"reflectedObservationLineCount"`
}

// SwapBufferedReflectionToActiveInput is the input for swapping buffered reflections.
type SwapBufferedReflectionToActiveInput struct {
	CurrentRecord *ObservationalMemoryRecord `json:"currentRecord"`
	TokenCount    int                        `json:"tokenCount"`
}

// CreateReflectionGenerationInput is the input for creating a reflection generation.
type CreateReflectionGenerationInput struct {
	CurrentRecord *ObservationalMemoryRecord `json:"currentRecord"`
	Reflection    string                     `json:"reflection"`
	TokenCount    int                        `json:"tokenCount"`
}

// UpdateThreadInput holds the fields for updating a thread.
type UpdateThreadInput struct {
	ID       string         `json:"id"`
	Title    string         `json:"title"`
	Metadata map[string]any `json:"metadata"`
}

// UpdateMessagesInput holds the fields for updating messages.
type UpdateMessagesInput struct {
	Messages []any `json:"messages"`
}

// UpdateResourceInput holds the fields for updating a resource.
type UpdateResourceInput struct {
	ResourceID    string         `json:"resourceId"`
	WorkingMemory *string        `json:"workingMemory,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// MemoryStorage Interface
// ---------------------------------------------------------------------------

// MemoryStorage is the storage interface for the memory domain.
// This is the largest domain with 30+ methods covering threads, messages,
// resources, and observational memory.
type MemoryStorage interface {
	// Init initializes the storage domain (creates tables, etc).
	Init(ctx context.Context) error

	// DangerouslyClearAll clears all data. Primarily used for testing.
	DangerouslyClearAll(ctx context.Context) error

	// SupportsObservationalMemory returns whether this storage adapter
	// supports Observational Memory. Defaults to false for backwards
	// compatibility with custom adapters.
	SupportsObservationalMemory() bool

	// --- Thread Methods ---

	// GetThreadByID retrieves a thread by its ID.
	GetThreadByID(ctx context.Context, threadID string) (StorageThreadType, error)

	// SaveThread creates or saves a thread.
	SaveThread(ctx context.Context, thread StorageThreadType) (StorageThreadType, error)

	// UpdateThread updates an existing thread.
	UpdateThread(ctx context.Context, input UpdateThreadInput) (StorageThreadType, error)

	// DeleteThread removes a thread by ID.
	DeleteThread(ctx context.Context, threadID string) error

	// ListThreads lists threads with optional filtering by resourceId and metadata.
	ListThreads(ctx context.Context, args StorageListThreadsInput) (StorageListThreadsOutput, error)

	// CloneThread clones a thread and its messages to create a new independent thread.
	// The cloned thread will have clone metadata stored in its metadata field.
	CloneThread(ctx context.Context, args StorageCloneThreadInput) (StorageCloneThreadOutput, error)

	// --- Message Methods ---

	// ListMessages lists messages with optional filtering.
	ListMessages(ctx context.Context, args StorageListMessagesInput) (StorageListMessagesOutput, error)

	// ListMessagesByResourceID lists messages by resource ID only (across all threads).
	// Used by Observational Memory and LongMemEval for resource-scoped queries.
	ListMessagesByResourceID(ctx context.Context, args StorageListMessagesByResourceIDInput) (StorageListMessagesOutput, error)

	// ListMessagesByID retrieves messages by their IDs.
	ListMessagesByID(ctx context.Context, messageIDs []string) ([]MastraDBMessage, error)

	// SaveMessages saves multiple messages.
	SaveMessages(ctx context.Context, messages []MastraDBMessage) ([]MastraDBMessage, error)

	// UpdateMessages updates multiple messages with partial data.
	UpdateMessages(ctx context.Context, input UpdateMessagesInput) ([]MastraDBMessage, error)

	// DeleteMessages deletes messages by their IDs.
	DeleteMessages(ctx context.Context, messageIDs []string) error

	// --- Resource Methods ---

	// GetResourceByID retrieves a resource by its ID.
	GetResourceByID(ctx context.Context, resourceID string) (StorageResourceType, error)

	// SaveResource creates or saves a resource.
	SaveResource(ctx context.Context, resource StorageResourceType) (StorageResourceType, error)

	// UpdateResource updates an existing resource.
	UpdateResource(ctx context.Context, input UpdateResourceInput) (StorageResourceType, error)

	// --- Observational Memory Methods ---

	// GetObservationalMemory gets the current observational memory record for
	// a thread/resource. threadID may be empty. Returns the most recent active record.
	GetObservationalMemory(ctx context.Context, threadID string, resourceID string) (*ObservationalMemoryRecord, error)

	// GetObservationalMemoryHistory gets observational memory history (previous generations).
	// Returns records in reverse chronological order (newest first).
	GetObservationalMemoryHistory(ctx context.Context, threadID string, resourceID string, limit int) ([]ObservationalMemoryRecord, error)

	// InitializeObservationalMemory creates a new observational memory record.
	// Called when starting observations for a new thread/resource.
	InitializeObservationalMemory(ctx context.Context, input CreateObservationalMemoryInput) (*ObservationalMemoryRecord, error)

	// UpdateActiveObservations updates active observations.
	// Called when observations are created and immediately activated (no buffering).
	UpdateActiveObservations(ctx context.Context, input UpdateActiveObservationsInput) error

	// --- Buffering Methods (for async observation/reflection) ---

	// UpdateBufferedObservations updates buffered observations.
	// Called when observations are created asynchronously via bufferTokens.
	UpdateBufferedObservations(ctx context.Context, input UpdateBufferedObservationsInput) error

	// SwapBufferedToActive atomically swaps buffered observations to active.
	// 1. Appends bufferedObservations -> activeObservations (based on activationRatio)
	// 2. Moves activated bufferedMessageIds -> observedMessageIds
	// 3. Keeps remaining buffered content if activationRatio < 100
	// 4. Updates lastObservedAt
	SwapBufferedToActive(ctx context.Context, input SwapBufferedToActiveInput) (*SwapBufferedToActiveResult, error)

	// CreateReflectionGeneration creates a new generation from a reflection.
	// Creates a new record with originType: 'reflection', activeObservations
	// containing the reflection, and generationCount incremented.
	CreateReflectionGeneration(ctx context.Context, input CreateReflectionGenerationInput) (*ObservationalMemoryRecord, error)

	// UpdateBufferedReflection updates the buffered reflection (async reflection in progress).
	UpdateBufferedReflection(ctx context.Context, input UpdateBufferedReflectionInput) error

	// SwapBufferedReflectionToActive swaps buffered reflection to active observations.
	// Creates a new generation where activeObservations = bufferedReflection + unreflected observations.
	SwapBufferedReflectionToActive(ctx context.Context, input SwapBufferedReflectionToActiveInput) (*ObservationalMemoryRecord, error)

	// --- Observational Memory Flag Methods ---

	// SetReflectingFlag sets the isReflecting flag.
	SetReflectingFlag(ctx context.Context, id string, isReflecting bool) error

	// SetObservingFlag sets the isObserving flag.
	SetObservingFlag(ctx context.Context, id string, isObserving bool) error

	// SetBufferingObservationFlag sets the isBufferingObservation flag
	// and updates lastBufferedAtTokens.
	SetBufferingObservationFlag(ctx context.Context, id string, isBuffering bool, lastBufferedAtTokens *int) error

	// SetBufferingReflectionFlag sets the isBufferingReflection flag.
	SetBufferingReflectionFlag(ctx context.Context, id string, isBuffering bool) error

	// InsertObservationalMemoryRecord inserts a fully-formed observational
	// memory record. Used by thread cloning to copy OM state with remapped IDs.
	InsertObservationalMemoryRecord(ctx context.Context, record ObservationalMemoryRecord) error

	// ClearObservationalMemory clears all observational memory for a
	// thread/resource. Removes all records and history.
	ClearObservationalMemory(ctx context.Context, threadID string, resourceID string) error

	// SetPendingMessageTokens sets the pending message token count.
	// Called at the end of each OM processing step to persist the current
	// context window token count.
	SetPendingMessageTokens(ctx context.Context, id string, tokenCount int) error
}
