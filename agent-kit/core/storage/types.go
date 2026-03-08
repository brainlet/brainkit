// Ported from: packages/core/src/storage/types.ts
package storage

import (
	"time"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
	"github.com/brainlet/brainkit/agent-kit/core/evals"
	"github.com/brainlet/brainkit/agent-kit/core/processorprovider"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/datasets"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/experiments"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains/workflows"
)

// Keep the compiler happy for the import.
var _ = mastraerror.SerializedError{}

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// AgentExecutionOptionsBase is a stub for ../agent/agent.types.
// TODO: import from agent package once ported.
type AgentExecutionOptionsBase = map[string]any

// ScoringSamplingConfig is imported from the evals package.
type ScoringSamplingConfig = evals.ScoringSamplingConfig

// ProcessorPhase is imported from the processorprovider package.
type ProcessorPhase = processorprovider.ProcessorPhase

// MastraDBMessage is a stub for ../memory/types.
// TODO: import from memory package once ported.
type MastraDBMessage = map[string]any

// StorageThreadType is a stub for ../memory/types.
// TODO: import from memory package once ported.
type StorageThreadType = map[string]any

// SerializedMemoryConfig is a stub for ../memory/types.
// TODO: import from memory package once ported.
type SerializedMemoryConfig = map[string]any

// StepResult is a stub for ../workflows.
// TODO: import from workflows package once ported.
type StepResult = map[string]any

// WorkflowRunState represents a workflow run's state snapshot.
// In TS this is imported from ../workflows. Stored as JSON or string.
type WorkflowRunState = map[string]any

// WorkflowRunStatus represents the status of a workflow run.
type WorkflowRunStatus = domains.WorkflowRunStatus

const (
	WorkflowRunStatusPending   WorkflowRunStatus = "pending"
	WorkflowRunStatusRunning   WorkflowRunStatus = "running"
	WorkflowRunStatusCompleted WorkflowRunStatus = "completed"
	WorkflowRunStatusFailed    WorkflowRunStatus = "failed"
	WorkflowRunStatusSuspended WorkflowRunStatus = "suspended"
	WorkflowRunStatusWaiting   WorkflowRunStatus = "waiting"
)

// ---------------------------------------------------------------------------
// Storage Basics
// ---------------------------------------------------------------------------

// StoragePagination specifies pagination parameters.
// PerPage can be 0 to mean "false" (fetch all records without limit).
type StoragePagination = domains.StoragePagination

// StorageColumnType, StorageColumn, StorageColumnRef, and StorageTableConfig
// are defined in constants.go (ported from constants.ts).

// ---------------------------------------------------------------------------
// Workflow Run Types (re-exported from storage/domains/workflows)
// ---------------------------------------------------------------------------

// WorkflowRuns wraps a list of runs with a total count.
type WorkflowRuns = workflows.WorkflowRuns

// StorageWorkflowRun is the storage-layer workflow run record.
type StorageWorkflowRun = workflows.StorageWorkflowRun

// WorkflowRun is the API-layer workflow run record.
type WorkflowRun = workflows.WorkflowRun

// PaginationInfo describes pagination metadata in responses.
// PerPage of 0 means all records were fetched without a limit.
type PaginationInfo = domains.PaginationInfo

// MastraMessageFormat discriminates message format versions.
type MastraMessageFormat string

const (
	MastraMessageFormatV1 MastraMessageFormat = "v1"
	MastraMessageFormatV2 MastraMessageFormat = "v2"
)

// ---------------------------------------------------------------------------
// Message Listing Types
// ---------------------------------------------------------------------------

// StorageListMessagesInclude specifies a message to include with context.
type StorageListMessagesInclude struct {
	ID                   string  `json:"id"`
	ThreadID             *string `json:"threadId,omitempty"`
	WithPreviousMessages *int    `json:"withPreviousMessages,omitempty"`
	WithNextMessages     *int    `json:"withNextMessages,omitempty"`
}

// StorageListMessagesDateRange filters messages by date range.
type StorageListMessagesDateRange struct {
	Start          *time.Time `json:"start,omitempty"`
	End            *time.Time `json:"end,omitempty"`
	StartExclusive *bool      `json:"startExclusive,omitempty"`
	EndExclusive   *bool      `json:"endExclusive,omitempty"`
}

// StorageListMessagesFilter contains filter options for listing messages.
type StorageListMessagesFilter struct {
	DateRange *StorageListMessagesDateRange `json:"dateRange,omitempty"`
}

// ThreadOrderBy specifies the field to order threads by.
type ThreadOrderBy string

const (
	ThreadOrderByCreatedAt ThreadOrderBy = "createdAt"
	ThreadOrderByUpdatedAt ThreadOrderBy = "updatedAt"
)

// ThreadSortDirection specifies sort direction.
type ThreadSortDirection string

const (
	ThreadSortASC  ThreadSortDirection = "ASC"
	ThreadSortDESC ThreadSortDirection = "DESC"
)

// StorageOrderBy describes an ordering clause.
type StorageOrderBy struct {
	Field     *ThreadOrderBy     `json:"field,omitempty"`
	Direction *ThreadSortDirection `json:"direction,omitempty"`
}

// ThreadSortOptions groups ordering options for threads.
type ThreadSortOptions struct {
	OrderBy       *ThreadOrderBy      `json:"orderBy,omitempty"`
	SortDirection *ThreadSortDirection `json:"sortDirection,omitempty"`
}

// StorageListMessagesOptions are common options for listing messages.
type StorageListMessagesOptions struct {
	Include []StorageListMessagesInclude `json:"include,omitempty"`
	PerPage *int                         `json:"perPage,omitempty"` // nil = default; 0 = all
	Page    *int                         `json:"page,omitempty"`
	Filter  *StorageListMessagesFilter   `json:"filter,omitempty"`
	OrderBy *StorageOrderBy              `json:"orderBy,omitempty"`
}

// StorageListMessagesInput is the input for listing messages by thread ID.
type StorageListMessagesInput struct {
	StorageListMessagesOptions
	ThreadID   any     `json:"threadId"`             // string | []string
	ResourceID *string `json:"resourceId,omitempty"`
}

// StorageListMessagesOutput is the paginated output for listing messages.
type StorageListMessagesOutput struct {
	PaginationInfo
	Messages []MastraDBMessage `json:"messages"`
}

// StorageListMessagesByResourceIdInput lists messages by resource ID across all threads.
type StorageListMessagesByResourceIdInput struct {
	StorageListMessagesOptions
	ResourceID string `json:"resourceId"`
}

// ---------------------------------------------------------------------------
// Workflow Run Listing Types (re-exported from storage/domains/workflows)
// ---------------------------------------------------------------------------

// StorageListWorkflowRunsInput specifies filters for listing workflow runs.
type StorageListWorkflowRunsInput = workflows.ListWorkflowRunsInput

// ---------------------------------------------------------------------------
// Thread Listing Types
// ---------------------------------------------------------------------------

// StorageListThreadsFilter contains filter options for listing threads.
type StorageListThreadsFilter struct {
	ResourceID *string        `json:"resourceId,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// StorageListThreadsInput specifies filters for listing threads.
type StorageListThreadsInput struct {
	PerPage *int                      `json:"perPage,omitempty"`
	Page    *int                      `json:"page,omitempty"`
	OrderBy *StorageOrderBy           `json:"orderBy,omitempty"`
	Filter  *StorageListThreadsFilter `json:"filter,omitempty"`
}

// StorageListThreadsOutput is the paginated output for listing threads.
type StorageListThreadsOutput struct {
	PaginationInfo
	Threads []StorageThreadType `json:"threads"`
}

// ---------------------------------------------------------------------------
// Thread Clone Types
// ---------------------------------------------------------------------------

// ThreadCloneMetadata is metadata stored on cloned threads to track their origin.
type ThreadCloneMetadata struct {
	SourceThreadID string    `json:"sourceThreadId"`
	ClonedAt       time.Time `json:"clonedAt"`
	LastMessageID  *string   `json:"lastMessageId,omitempty"`
}

// StorageCloneThreadMessageFilter filters which messages to include in a clone.
type StorageCloneThreadMessageFilter struct {
	StartDate  *time.Time `json:"startDate,omitempty"`
	EndDate    *time.Time `json:"endDate,omitempty"`
	MessageIDs []string   `json:"messageIds,omitempty"`
}

// StorageCloneThreadOptions specifies options for cloning thread messages.
type StorageCloneThreadOptions struct {
	MessageLimit  *int                             `json:"messageLimit,omitempty"`
	MessageFilter *StorageCloneThreadMessageFilter `json:"messageFilter,omitempty"`
}

// StorageCloneThreadInput is the input for cloning a thread.
type StorageCloneThreadInput struct {
	SourceThreadID string                     `json:"sourceThreadId"`
	NewThreadID    *string                    `json:"newThreadId,omitempty"`
	ResourceID     *string                    `json:"resourceId,omitempty"`
	Title          *string                    `json:"title,omitempty"`
	Metadata       map[string]any             `json:"metadata,omitempty"`
	Options        *StorageCloneThreadOptions `json:"options,omitempty"`
}

// StorageCloneThreadOutput is the output from cloning a thread.
type StorageCloneThreadOutput struct {
	Thread         StorageThreadType  `json:"thread"`
	ClonedMessages []MastraDBMessage  `json:"clonedMessages"`
	MessageIDMap   map[string]string  `json:"messageIdMap,omitempty"`
}

// ---------------------------------------------------------------------------
// Resource Type
// ---------------------------------------------------------------------------

// StorageResourceType represents a resource record.
type StorageResourceType struct {
	ID            string         `json:"id"`
	WorkingMemory *string        `json:"workingMemory,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
}

// StorageMessageType represents a message record.
type StorageMessageType struct {
	ID         string    `json:"id"`
	ThreadID   string    `json:"thread_id"`
	Content    string    `json:"content"`
	Role       string    `json:"role"`
	Type       string    `json:"type"`
	CreatedAt  time.Time `json:"createdAt"`
	ResourceID *string   `json:"resourceId"` // string | null
}

// ---------------------------------------------------------------------------
// Agent Storage Types
// ---------------------------------------------------------------------------

// StorageToolConfig is per-tool configuration stored in agent snapshots.
type StorageToolConfig struct {
	Description *string    `json:"description,omitempty"`
	Rules       *RuleGroup `json:"rules,omitempty"`
}

// StorageMCPClientToolsConfig specifies which tools from an MCP client are enabled.
// When Tools is nil, all tools from the source are included.
type StorageMCPClientToolsConfig struct {
	Tools map[string]StorageToolConfig `json:"tools,omitempty"`
}

// StorageScorerConfig is a scorer reference with optional sampling configuration.
type StorageScorerConfig struct {
	Description *string              `json:"description,omitempty"`
	Sampling    *ScoringSamplingConfig `json:"sampling,omitempty"`
	Rules       *RuleGroup           `json:"rules,omitempty"`
}

// StorageModelConfig is model configuration stored in agent snapshots.
type StorageModelConfig struct {
	Provider            string  `json:"provider"`
	Name                string  `json:"name"`
	Temperature         *float64 `json:"temperature,omitempty"`
	TopP                *float64 `json:"topP,omitempty"`
	FrequencyPenalty    *float64 `json:"frequencyPenalty,omitempty"`
	PresencePenalty     *float64 `json:"presencePenalty,omitempty"`
	MaxCompletionTokens *int     `json:"maxCompletionTokens,omitempty"`
	// Extra holds additional provider-specific options ([key: string]: unknown in TS).
	Extra map[string]any `json:"extra,omitempty"`
}

// StorageDefaultOptions represents serializable default options for agent execution.
// In TS this is Omit<AgentExecutionOptionsBase, ...non-serializable fields>.
// Since AgentExecutionOptionsBase is not yet ported, this is a map stub.
type StorageDefaultOptions = map[string]any

// ---------------------------------------------------------------------------
// Conditional Variant Types
// ---------------------------------------------------------------------------

// StorageConditionalVariant pairs a value with an optional rule group.
type StorageConditionalVariant[T any] struct {
	Value T          `json:"value"`
	Rules *RuleGroup `json:"rules,omitempty"`
}

// StorageConditionalField is either a static value T or a slice of conditional variants.
// In Go, since union types are not native, callers should use any and type-assert,
// or use the typed wrapper functions. For struct fields we use any.
// Example: the field is either T or []StorageConditionalVariant[T].
type StorageConditionalField = any

// ---------------------------------------------------------------------------
// Agent Instruction Blocks
// ---------------------------------------------------------------------------

// AgentInstructionBlockType discriminates instruction block types.
type AgentInstructionBlockType string

const (
	AgentInstructionBlockTypeText           AgentInstructionBlockType = "text"
	AgentInstructionBlockTypePromptBlockRef AgentInstructionBlockType = "prompt_block_ref"
	AgentInstructionBlockTypePromptBlock    AgentInstructionBlockType = "prompt_block"
)

// AgentInstructionBlock is a discriminated union for agent instructions.
type AgentInstructionBlock struct {
	Type    AgentInstructionBlockType `json:"type"`
	Content *string                   `json:"content,omitempty"` // present for "text" and "prompt_block"
	ID      *string                   `json:"id,omitempty"`      // present for "prompt_block_ref"
	Rules   *RuleGroup                `json:"rules,omitempty"`   // present for "prompt_block"
}

// ---------------------------------------------------------------------------
// Agent Snapshot Type
// ---------------------------------------------------------------------------

// StorageAgentSnapshotType contains ALL agent configuration fields.
// These fields live exclusively in version snapshot rows.
type StorageAgentSnapshotType struct {
	Name                 string                    `json:"name"`
	Description          *string                   `json:"description,omitempty"`
	Instructions         any                       `json:"instructions"`          // string | []AgentInstructionBlock
	Model                any                       `json:"model"`                 // StorageConditionalField<StorageModelConfig>
	Tools                any                       `json:"tools,omitempty"`       // StorageConditionalField<map[string]StorageToolConfig>
	DefaultOptions       any                       `json:"defaultOptions,omitempty"` // StorageConditionalField<StorageDefaultOptions>
	Workflows            any                       `json:"workflows,omitempty"`   // StorageConditionalField<map[string]StorageToolConfig>
	Agents               any                       `json:"agents,omitempty"`      // StorageConditionalField<map[string]StorageToolConfig>
	IntegrationTools     any                       `json:"integrationTools,omitempty"` // StorageConditionalField<map[string]StorageMCPClientToolsConfig>
	InputProcessors      any                       `json:"inputProcessors,omitempty"`  // StorageConditionalField<StoredProcessorGraph>
	OutputProcessors     any                       `json:"outputProcessors,omitempty"` // StorageConditionalField<StoredProcessorGraph>
	Memory               any                       `json:"memory,omitempty"`           // StorageConditionalField<SerializedMemoryConfig>
	Scorers              any                       `json:"scorers,omitempty"`          // StorageConditionalField<map[string]StorageScorerConfig>
	MCPClients           any                       `json:"mcpClients,omitempty"`       // StorageConditionalField<map[string]StorageMCPClientToolsConfig>
	Workspace            any                       `json:"workspace,omitempty"`        // StorageConditionalField<StorageWorkspaceRef>
	Skills               any                       `json:"skills,omitempty"`           // StorageConditionalField<map[string]StorageSkillConfig>
	SkillsFormat         *string                   `json:"skillsFormat,omitempty"`     // "xml" | "json" | "markdown"
	RequestContextSchema map[string]any            `json:"requestContextSchema,omitempty"`
}

// SkillsFormatType enumerates the supported skills format types.
type SkillsFormatType string

const (
	SkillsFormatXML      SkillsFormatType = "xml"
	SkillsFormatJSON     SkillsFormatType = "json"
	SkillsFormatMarkdown SkillsFormatType = "markdown"
)

// ---------------------------------------------------------------------------
// Agent Record Types
// ---------------------------------------------------------------------------

// EntityStatus represents the publication status of a versioned entity.
type EntityStatus = domains.EntityStatus

const (
	EntityStatusDraft     EntityStatus = "draft"
	EntityStatusPublished EntityStatus = "published"
	EntityStatusArchived  EntityStatus = "archived"
)

// StorageAgentType is the thin agent record containing only metadata fields.
type StorageAgentType struct {
	ID              string         `json:"id"`
	Status          EntityStatus   `json:"status"`
	ActiveVersionID *string        `json:"activeVersionId,omitempty"`
	AuthorID        *string        `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// StorageResolvedAgentType combines the thin agent record with version snapshot config.
type StorageResolvedAgentType struct {
	StorageAgentType
	StorageAgentSnapshotType
	ResolvedVersionID *string `json:"resolvedVersionId,omitempty"`
}

// StorageCreateAgentInput is the input for creating a new agent.
type StorageCreateAgentInput struct {
	ID       string         `json:"id"`
	AuthorID *string        `json:"authorId,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	StorageAgentSnapshotType
}

// StorageUpdateAgentInput is the input for updating an agent.
type StorageUpdateAgentInput struct {
	ID              string         `json:"id"`
	AuthorID        *string        `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	ActiveVersionID *string        `json:"activeVersionId,omitempty"`
	Status          *EntityStatus  `json:"status,omitempty"`
	// Snapshot fields (all optional for update).
	Name                 *string `json:"name,omitempty"`
	Description          *string `json:"description,omitempty"`
	Instructions         any     `json:"instructions,omitempty"`
	Model                any     `json:"model,omitempty"`
	Tools                any     `json:"tools,omitempty"`
	DefaultOptions       any     `json:"defaultOptions,omitempty"`
	Workflows            any     `json:"workflows,omitempty"`
	Agents               any     `json:"agents,omitempty"`
	IntegrationTools     any     `json:"integrationTools,omitempty"`
	InputProcessors      any     `json:"inputProcessors,omitempty"`
	OutputProcessors     any     `json:"outputProcessors,omitempty"`
	Memory               any     `json:"memory,omitempty"` // can be nil to disable
	Scorers              any     `json:"scorers,omitempty"`
	MCPClients           any     `json:"mcpClients,omitempty"`
	Workspace            any     `json:"workspace,omitempty"`
	Skills               any     `json:"skills,omitempty"`
	SkillsFormat         *string `json:"skillsFormat,omitempty"`
	RequestContextSchema map[string]any `json:"requestContextSchema,omitempty"`
}

// StorageListAgentsInput specifies filters for listing agents.
type StorageListAgentsInput struct {
	PerPage  *int            `json:"perPage,omitempty"`
	Page     *int            `json:"page,omitempty"`
	OrderBy  *StorageOrderBy `json:"orderBy,omitempty"`
	AuthorID *string         `json:"authorId,omitempty"`
	Metadata map[string]any  `json:"metadata,omitempty"`
	Status   *EntityStatus   `json:"status,omitempty"`
}

// StorageListAgentsOutput is the paginated output for listing thin agent records.
type StorageListAgentsOutput struct {
	PaginationInfo
	Agents []StorageAgentType `json:"agents"`
}

// StorageListAgentsResolvedOutput is the paginated output for listing resolved agents.
type StorageListAgentsResolvedOutput struct {
	PaginationInfo
	Agents []StorageResolvedAgentType `json:"agents"`
}

// ---------------------------------------------------------------------------
// Condition / Rule Types
// ---------------------------------------------------------------------------

// ConditionOperator enumerates condition operators for rule evaluation.
type ConditionOperator string

const (
	ConditionEquals             ConditionOperator = "equals"
	ConditionNotEquals         ConditionOperator = "not_equals"
	ConditionContains          ConditionOperator = "contains"
	ConditionNotContains       ConditionOperator = "not_contains"
	ConditionGreaterThan       ConditionOperator = "greater_than"
	ConditionLessThan          ConditionOperator = "less_than"
	ConditionGreaterThanOrEqual ConditionOperator = "greater_than_or_equal"
	ConditionLessThanOrEqual   ConditionOperator = "less_than_or_equal"
	ConditionIn                ConditionOperator = "in"
	ConditionNotIn             ConditionOperator = "not_in"
	ConditionExists            ConditionOperator = "exists"
	ConditionNotExists         ConditionOperator = "not_exists"
)

// Rule is a leaf rule: evaluates a single condition against a context field.
type Rule struct {
	Field    string            `json:"field"`
	Operator ConditionOperator `json:"operator"`
	Value    any               `json:"value,omitempty"`
}

// RuleGroupOperator is the logical operator for a rule group.
type RuleGroupOperator string

const (
	RuleGroupAND RuleGroupOperator = "AND"
	RuleGroupOR  RuleGroupOperator = "OR"
)

// RuleGroupDepth2 is the innermost rule group (depth 2): may only contain leaf Rules.
type RuleGroupDepth2 struct {
	Operator   RuleGroupOperator `json:"operator"`
	Conditions []Rule            `json:"conditions"`
}

// RuleGroupDepth1Condition is a condition element in a depth-1 group: Rule or RuleGroupDepth2.
// Use the Type field to discriminate. If Group is non-nil, it's a group; otherwise it's a leaf Rule.
type RuleGroupDepth1Condition struct {
	// Leaf rule fields (used when Group is nil).
	Rule *Rule `json:"rule,omitempty"`
	// Nested group (used when non-nil).
	Group *RuleGroupDepth2 `json:"group,omitempty"`
}

// RuleGroupDepth1 is a mid-level group (depth 1): may contain Rules or depth-2 groups.
type RuleGroupDepth1 struct {
	Operator   RuleGroupOperator          `json:"operator"`
	Conditions []RuleGroupDepth1Condition `json:"conditions"`
}

// RuleGroupCondition is a condition element in the top-level rule group.
type RuleGroupCondition struct {
	// Leaf rule (used when Group is nil).
	Rule *Rule `json:"rule,omitempty"`
	// Nested group (used when non-nil).
	Group *RuleGroupDepth1 `json:"group,omitempty"`
}

// RuleGroup is the top-level rule group (depth 0).
type RuleGroup struct {
	Operator   RuleGroupOperator    `json:"operator"`
	Conditions []RuleGroupCondition `json:"conditions"`
}

// ---------------------------------------------------------------------------
// Stored Processor Graph Types
// ---------------------------------------------------------------------------

// ProcessorGraphStep is a single processor step in a stored processor graph.
type ProcessorGraphStep struct {
	ID            string         `json:"id"`
	ProviderID    string         `json:"providerId"`
	Config        map[string]any `json:"config"`
	EnabledPhases []ProcessorPhase `json:"enabledPhases"`
}

// ProcessorGraphEntryDepth3 is a leaf entry (depth 3): only step entries.
type ProcessorGraphEntryDepth3 struct {
	Type string              `json:"type"` // always "step"
	Step *ProcessorGraphStep `json:"step"`
}

// ProcessorGraphConditionDepth2 is a condition at depth 2.
type ProcessorGraphConditionDepth2 struct {
	Steps []ProcessorGraphEntryDepth3 `json:"steps"`
	Rules *RuleGroup                  `json:"rules,omitempty"`
}

// ProcessorGraphEntryDepth2 is a depth-2 entry: step, parallel, or conditional.
type ProcessorGraphEntryDepth2 struct {
	Type       string                           `json:"type"` // "step" | "parallel" | "conditional"
	Step       *ProcessorGraphStep              `json:"step,omitempty"`
	Branches   [][]ProcessorGraphEntryDepth3    `json:"branches,omitempty"`
	Conditions []ProcessorGraphConditionDepth2  `json:"conditions,omitempty"`
}

// ProcessorGraphCondition is a condition at depth 1.
type ProcessorGraphCondition struct {
	Steps []ProcessorGraphEntryDepth2 `json:"steps"`
	Rules *RuleGroup                  `json:"rules,omitempty"`
}

// ProcessorGraphEntry is a top-level entry (depth 1): step, parallel, or conditional.
type ProcessorGraphEntry struct {
	Type       string                          `json:"type"` // "step" | "parallel" | "conditional"
	Step       *ProcessorGraphStep             `json:"step,omitempty"`
	Branches   [][]ProcessorGraphEntryDepth2   `json:"branches,omitempty"`
	Conditions []ProcessorGraphCondition       `json:"conditions,omitempty"`
}

// StoredProcessorGraph represents a stored processor pipeline.
type StoredProcessorGraph struct {
	Steps []ProcessorGraphEntry `json:"steps"`
}

// ---------------------------------------------------------------------------
// Prompt Block Storage Types
// ---------------------------------------------------------------------------

// StoragePromptBlockType is the thin prompt block record (metadata only).
type StoragePromptBlockType struct {
	ID              string         `json:"id"`
	Status          EntityStatus   `json:"status"`
	ActiveVersionID *string        `json:"activeVersionId,omitempty"`
	AuthorID        *string        `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// StoragePromptBlockSnapshotType is the prompt block version snapshot.
type StoragePromptBlockSnapshotType struct {
	Name                 string         `json:"name"`
	Description          *string        `json:"description,omitempty"`
	Content              string         `json:"content"`
	Rules                *RuleGroup     `json:"rules,omitempty"`
	RequestContextSchema map[string]any `json:"requestContextSchema,omitempty"`
}

// StorageResolvedPromptBlockType is the resolved prompt block (record + snapshot).
type StorageResolvedPromptBlockType struct {
	StoragePromptBlockType
	StoragePromptBlockSnapshotType
	ResolvedVersionID *string `json:"resolvedVersionId,omitempty"`
}

// StorageCreatePromptBlockInput is the input for creating a new prompt block.
type StorageCreatePromptBlockInput struct {
	ID       string         `json:"id"`
	AuthorID *string        `json:"authorId,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	StoragePromptBlockSnapshotType
}

// StorageUpdatePromptBlockInput is the input for updating a prompt block.
type StorageUpdatePromptBlockInput struct {
	ID              string         `json:"id"`
	AuthorID        *string        `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	ActiveVersionID *string        `json:"activeVersionId,omitempty"`
	Status          *EntityStatus  `json:"status,omitempty"`
	// Snapshot fields (all optional for update).
	Name                 *string        `json:"name,omitempty"`
	Description          *string        `json:"description,omitempty"`
	Content              *string        `json:"content,omitempty"`
	Rules                *RuleGroup     `json:"rules,omitempty"`
	RequestContextSchema map[string]any `json:"requestContextSchema,omitempty"`
}

// StorageListPromptBlocksInput specifies filters for listing prompt blocks.
type StorageListPromptBlocksInput struct {
	PerPage  *int            `json:"perPage,omitempty"`
	Page     *int            `json:"page,omitempty"`
	OrderBy  *StorageOrderBy `json:"orderBy,omitempty"`
	AuthorID *string         `json:"authorId,omitempty"`
	Metadata map[string]any  `json:"metadata,omitempty"`
	Status   *EntityStatus   `json:"status,omitempty"`
}

// StorageListPromptBlocksOutput is the paginated output for thin prompt block records.
type StorageListPromptBlocksOutput struct {
	PaginationInfo
	PromptBlocks []StoragePromptBlockType `json:"promptBlocks"`
}

// StorageListPromptBlocksResolvedOutput is the paginated output for resolved prompt blocks.
type StorageListPromptBlocksResolvedOutput struct {
	PaginationInfo
	PromptBlocks []StorageResolvedPromptBlockType `json:"promptBlocks"`
}

// ---------------------------------------------------------------------------
// Stored Scorer Types
// ---------------------------------------------------------------------------

// StoredScorerType is the scorer type discriminator.
type StoredScorerType string

const (
	StoredScorerLLMJudge          StoredScorerType = "llm-judge"
	StoredScorerAnswerRelevancy   StoredScorerType = "answer-relevancy"
	StoredScorerAnswerSimilarity  StoredScorerType = "answer-similarity"
	StoredScorerBias              StoredScorerType = "bias"
	StoredScorerContextPrecision  StoredScorerType = "context-precision"
	StoredScorerContextRelevance  StoredScorerType = "context-relevance"
	StoredScorerFaithfulness      StoredScorerType = "faithfulness"
	StoredScorerHallucination     StoredScorerType = "hallucination"
	StoredScorerNoiseSensitivity  StoredScorerType = "noise-sensitivity"
	StoredScorerPromptAlignment   StoredScorerType = "prompt-alignment"
	StoredScorerToolCallAccuracy  StoredScorerType = "tool-call-accuracy"
	StoredScorerToxicity          StoredScorerType = "toxicity"
)

// ScorerScoreRange defines the min/max score range for a scorer.
type ScorerScoreRange struct {
	Min *float64 `json:"min,omitempty"`
	Max *float64 `json:"max,omitempty"`
}

// StorageScorerDefinitionSnapshotType is the scorer version snapshot.
type StorageScorerDefinitionSnapshotType struct {
	Name            string                `json:"name"`
	Description     *string               `json:"description,omitempty"`
	Type            StoredScorerType      `json:"type"`
	Model           *StorageModelConfig   `json:"model,omitempty"`
	Instructions    *string               `json:"instructions,omitempty"`
	ScoreRange      *ScorerScoreRange     `json:"scoreRange,omitempty"`
	PresetConfig    map[string]any        `json:"presetConfig,omitempty"`
	DefaultSampling *ScoringSamplingConfig `json:"defaultSampling,omitempty"`
}

// StorageScorerDefinitionType is the thin scorer record.
type StorageScorerDefinitionType struct {
	ID              string         `json:"id"`
	Status          EntityStatus   `json:"status"`
	ActiveVersionID *string        `json:"activeVersionId,omitempty"`
	AuthorID        *string        `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// StorageResolvedScorerDefinitionType is the resolved scorer (record + snapshot).
type StorageResolvedScorerDefinitionType struct {
	StorageScorerDefinitionType
	StorageScorerDefinitionSnapshotType
	ResolvedVersionID *string `json:"resolvedVersionId,omitempty"`
}

// StorageCreateScorerDefinitionInput is the input for creating a new scorer.
type StorageCreateScorerDefinitionInput struct {
	ID       string         `json:"id"`
	AuthorID *string        `json:"authorId,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	StorageScorerDefinitionSnapshotType
}

// StorageUpdateScorerDefinitionInput is the input for updating a scorer.
type StorageUpdateScorerDefinitionInput struct {
	ID              string         `json:"id"`
	AuthorID        *string        `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	ActiveVersionID *string        `json:"activeVersionId,omitempty"`
	Status          *EntityStatus  `json:"status,omitempty"`
	// Snapshot fields (all optional for update).
	Name            *string               `json:"name,omitempty"`
	Description     *string               `json:"description,omitempty"`
	Type            *StoredScorerType     `json:"type,omitempty"`
	Model           *StorageModelConfig   `json:"model,omitempty"`
	Instructions    *string               `json:"instructions,omitempty"`
	ScoreRange      *ScorerScoreRange     `json:"scoreRange,omitempty"`
	PresetConfig    map[string]any        `json:"presetConfig,omitempty"`
	DefaultSampling *ScoringSamplingConfig `json:"defaultSampling,omitempty"`
}

// StorageListScorerDefinitionsInput specifies filters for listing scorers.
type StorageListScorerDefinitionsInput struct {
	PerPage  *int            `json:"perPage,omitempty"`
	Page     *int            `json:"page,omitempty"`
	OrderBy  *StorageOrderBy `json:"orderBy,omitempty"`
	AuthorID *string         `json:"authorId,omitempty"`
	Metadata map[string]any  `json:"metadata,omitempty"`
	Status   *EntityStatus   `json:"status,omitempty"`
}

// StorageListScorerDefinitionsOutput is the paginated output for thin scorer records.
type StorageListScorerDefinitionsOutput struct {
	PaginationInfo
	ScorerDefinitions []StorageScorerDefinitionType `json:"scorerDefinitions"`
}

// StorageListScorerDefinitionsResolvedOutput is the paginated output for resolved scorers.
type StorageListScorerDefinitionsResolvedOutput struct {
	PaginationInfo
	ScorerDefinitions []StorageResolvedScorerDefinitionType `json:"scorerDefinitions"`
}

// ---------------------------------------------------------------------------
// Index Management Types
// ---------------------------------------------------------------------------

// IndexMethod enumerates supported index methods.
type IndexMethod = domains.IndexMethod

const (
	IndexMethodBtree  IndexMethod = "btree"
	IndexMethodHash   IndexMethod = "hash"
	IndexMethodGIN    IndexMethod = "gin"
	IndexMethodGIST   IndexMethod = "gist"
	IndexMethodSPGIST IndexMethod = "spgist"
	IndexMethodBRIN   IndexMethod = "brin"
)

// CreateIndexOptions specifies options for creating a database index.
type CreateIndexOptions = domains.CreateIndexOptions

// IndexInfo describes an existing database index.
type IndexInfo = domains.IndexInfo

// StorageIndexStats extends IndexInfo with usage statistics.
type StorageIndexStats = domains.StorageIndexStats

// ---------------------------------------------------------------------------
// Observational Memory Types
// ---------------------------------------------------------------------------

// ObservationalMemoryScope is the scope of observational memory.
type ObservationalMemoryScope string

const (
	ObservationalMemoryScopeThread   ObservationalMemoryScope = "thread"
	ObservationalMemoryScopeResource ObservationalMemoryScope = "resource"
)

// ObservationalMemoryOriginType describes how the record was created.
type ObservationalMemoryOriginType string

const (
	ObservationalMemoryOriginInitial    ObservationalMemoryOriginType = "initial"
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
	SuggestedContinuation *string   `json:"suggestedContinuation,omitempty"`
	CurrentTask           *string   `json:"currentTask,omitempty"`
}

// BufferedObservationChunkInput is the input for creating a new buffered chunk.
type BufferedObservationChunkInput struct {
	CycleID               string    `json:"cycleId"`
	Observations          string    `json:"observations"`
	TokenCount            int       `json:"tokenCount"`
	MessageIDs            []string  `json:"messageIds"`
	MessageTokens         int       `json:"messageTokens"`
	LastObservedAt        time.Time `json:"lastObservedAt"`
	SuggestedContinuation *string   `json:"suggestedContinuation,omitempty"`
	CurrentTask           *string   `json:"currentTask,omitempty"`
}

// ObservationalMemoryRecord is the core database record for observational memory.
type ObservationalMemoryRecord struct {
	// Identity
	ID         string                   `json:"id"`
	Scope      ObservationalMemoryScope `json:"scope"`
	ThreadID   *string                  `json:"threadId"`   // null for resource scope
	ResourceID string                   `json:"resourceId"`

	// Timestamps
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
	LastObservedAt *time.Time `json:"lastObservedAt,omitempty"`

	// Generation tracking
	OriginType      ObservationalMemoryOriginType `json:"originType"`
	GenerationCount int                           `json:"generationCount"`

	// Observation content
	ActiveObservations        string                      `json:"activeObservations"`
	BufferedObservationChunks []BufferedObservationChunk   `json:"bufferedObservationChunks,omitempty"`
	// Deprecated: Use BufferedObservationChunks instead.
	BufferedObservations      *string  `json:"bufferedObservations,omitempty"`
	// Deprecated: Use BufferedObservationChunks instead.
	BufferedObservationTokens *int     `json:"bufferedObservationTokens,omitempty"`
	// Deprecated: Use BufferedObservationChunks instead.
	BufferedMessageIDs        []string `json:"bufferedMessageIds,omitempty"`
	BufferedReflection        *string  `json:"bufferedReflection,omitempty"`
	BufferedReflectionTokens  *int     `json:"bufferedReflectionTokens,omitempty"`
	BufferedReflectionInputTokens *int `json:"bufferedReflectionInputTokens,omitempty"`
	ReflectedObservationLineCount *int `json:"reflectedObservationLineCount,omitempty"`

	ObservedMessageIDs []string `json:"observedMessageIds,omitempty"`
	ObservedTimezone   *string  `json:"observedTimezone,omitempty"`

	// Token tracking
	TotalTokensObserved   int `json:"totalTokensObserved"`
	ObservationTokenCount int `json:"observationTokenCount"`
	PendingMessageTokens  int `json:"pendingMessageTokens"`

	// State flags
	IsReflecting           bool       `json:"isReflecting"`
	IsObserving            bool       `json:"isObserving"`
	IsBufferingObservation bool       `json:"isBufferingObservation"`
	IsBufferingReflection  bool       `json:"isBufferingReflection"`
	LastBufferedAtTokens   int        `json:"lastBufferedAtTokens"`
	LastBufferedAtTime     *time.Time `json:"lastBufferedAtTime"` // null when not set

	// Configuration
	Config map[string]any `json:"config"`

	// Extensible metadata
	Metadata map[string]any `json:"metadata,omitempty"`
}

// CreateObservationalMemoryInput is the input for creating a new observational memory record.
type CreateObservationalMemoryInput struct {
	ThreadID         *string                  `json:"threadId"`
	ResourceID       string                   `json:"resourceId"`
	Scope            ObservationalMemoryScope `json:"scope"`
	Config           map[string]any           `json:"config"`
	ObservedTimezone *string                  `json:"observedTimezone,omitempty"`
}

// UpdateActiveObservationsInput is the input for updating active observations.
type UpdateActiveObservationsInput struct {
	ID                 string    `json:"id"`
	Observations       string    `json:"observations"`
	TokenCount         int       `json:"tokenCount"`
	LastObservedAt     time.Time `json:"lastObservedAt"`
	ObservedMessageIDs []string  `json:"observedMessageIds,omitempty"`
	ObservedTimezone   *string   `json:"observedTimezone,omitempty"`
}

// UpdateBufferedObservationsInput is the input for updating buffered observations.
type UpdateBufferedObservationsInput struct {
	ID                 string                        `json:"id"`
	Chunk              BufferedObservationChunkInput  `json:"chunk"`
	LastBufferedAtTime *time.Time                    `json:"lastBufferedAtTime,omitempty"`
}

// SwapBufferedToActiveInput is the input for swapping buffered observations to active.
type SwapBufferedToActiveInput struct {
	ID                    string     `json:"id"`
	ActivationRatio       float64    `json:"activationRatio"`
	MessageTokensThreshold int       `json:"messageTokensThreshold"`
	CurrentPendingTokens  int        `json:"currentPendingTokens"`
	ForceMaxActivation    *bool      `json:"forceMaxActivation,omitempty"`
	LastObservedAt        *time.Time `json:"lastObservedAt,omitempty"`
}

// SwapBufferedToActivePerChunk describes per-chunk activation details.
type SwapBufferedToActivePerChunk struct {
	CycleID           string `json:"cycleId"`
	MessageTokens     int    `json:"messageTokens"`
	ObservationTokens int    `json:"observationTokens"`
	MessageCount      int    `json:"messageCount"`
	Observations      string `json:"observations"`
}

// SwapBufferedToActiveResult is the result from swapping buffered observations to active.
type SwapBufferedToActiveResult struct {
	ChunksActivated           int                            `json:"chunksActivated"`
	MessageTokensActivated    int                            `json:"messageTokensActivated"`
	ObservationTokensActivated int                           `json:"observationTokensActivated"`
	MessagesActivated         int                            `json:"messagesActivated"`
	ActivatedCycleIDs         []string                       `json:"activatedCycleIds"`
	ActivatedMessageIDs       []string                       `json:"activatedMessageIds"`
	Observations              *string                        `json:"observations,omitempty"`
	PerChunk                  []SwapBufferedToActivePerChunk  `json:"perChunk,omitempty"`
	SuggestedContinuation     *string                        `json:"suggestedContinuation,omitempty"`
	CurrentTask               *string                        `json:"currentTask,omitempty"`
}

// UpdateBufferedReflectionInput is the input for updating buffered reflection.
type UpdateBufferedReflectionInput struct {
	ID                            string `json:"id"`
	Reflection                    string `json:"reflection"`
	TokenCount                    int    `json:"tokenCount"`
	InputTokenCount               int    `json:"inputTokenCount"`
	ReflectedObservationLineCount int    `json:"reflectedObservationLineCount"`
}

// SwapBufferedReflectionToActiveInput is the input for swapping buffered reflection to active.
type SwapBufferedReflectionToActiveInput struct {
	CurrentRecord ObservationalMemoryRecord `json:"currentRecord"`
	TokenCount    int                       `json:"tokenCount"`
}

// CreateReflectionGenerationInput is the input for creating a reflection generation.
type CreateReflectionGenerationInput struct {
	CurrentRecord ObservationalMemoryRecord `json:"currentRecord"`
	Reflection    string                    `json:"reflection"`
	TokenCount    int                       `json:"tokenCount"`
}

// ---------------------------------------------------------------------------
// MCP Client Storage Types
// ---------------------------------------------------------------------------

// StorageMCPTransportType is the transport type for MCP server configs.
type StorageMCPTransportType string

const (
	StorageMCPTransportStdio StorageMCPTransportType = "stdio"
	StorageMCPTransportHTTP  StorageMCPTransportType = "http"
)

// StorageMCPServerConfig is a serializable MCP server transport definition.
type StorageMCPServerConfig struct {
	Type    StorageMCPTransportType      `json:"type"`
	Command *string                      `json:"command,omitempty"`
	Args    []string                     `json:"args,omitempty"`
	Env     map[string]string            `json:"env,omitempty"`
	URL     *string                      `json:"url,omitempty"`
	Timeout *int                         `json:"timeout,omitempty"`
	Tools   map[string]StorageToolConfig `json:"tools,omitempty"`
}

// StorageMCPClientSnapshotType is the MCP client version snapshot.
type StorageMCPClientSnapshotType struct {
	Name        string                            `json:"name"`
	Description *string                           `json:"description,omitempty"`
	Servers     map[string]StorageMCPServerConfig `json:"servers"`
}

// StorageMCPClientType is the thin MCP client record.
type StorageMCPClientType struct {
	ID              string         `json:"id"`
	Status          EntityStatus   `json:"status"`
	ActiveVersionID *string        `json:"activeVersionId,omitempty"`
	AuthorID        *string        `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// StorageResolvedMCPClientType is the resolved MCP client (record + snapshot).
type StorageResolvedMCPClientType struct {
	StorageMCPClientType
	StorageMCPClientSnapshotType
	ResolvedVersionID *string `json:"resolvedVersionId,omitempty"`
}

// StorageCreateMCPClientInput is the input for creating a new MCP client.
type StorageCreateMCPClientInput struct {
	ID       string         `json:"id"`
	AuthorID *string        `json:"authorId,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	StorageMCPClientSnapshotType
}

// StorageUpdateMCPClientInput is the input for updating an MCP client.
type StorageUpdateMCPClientInput struct {
	ID              string         `json:"id"`
	AuthorID        *string        `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	ActiveVersionID *string        `json:"activeVersionId,omitempty"`
	Status          *EntityStatus  `json:"status,omitempty"`
	// Snapshot fields (all optional for update).
	Name        *string                           `json:"name,omitempty"`
	Description *string                           `json:"description,omitempty"`
	Servers     map[string]StorageMCPServerConfig `json:"servers,omitempty"`
}

// StorageListMCPClientsInput specifies filters for listing MCP clients.
type StorageListMCPClientsInput struct {
	PerPage  *int            `json:"perPage,omitempty"`
	Page     *int            `json:"page,omitempty"`
	OrderBy  *StorageOrderBy `json:"orderBy,omitempty"`
	AuthorID *string         `json:"authorId,omitempty"`
	Metadata map[string]any  `json:"metadata,omitempty"`
	Status   *EntityStatus   `json:"status,omitempty"`
}

// StorageListMCPClientsOutput is the paginated output for thin MCP client records.
type StorageListMCPClientsOutput struct {
	PaginationInfo
	MCPClients []StorageMCPClientType `json:"mcpClients"`
}

// StorageListMCPClientsResolvedOutput is the paginated output for resolved MCP clients.
type StorageListMCPClientsResolvedOutput struct {
	PaginationInfo
	MCPClients []StorageResolvedMCPClientType `json:"mcpClients"`
}

// ---------------------------------------------------------------------------
// MCP Server Storage Types
// ---------------------------------------------------------------------------

// StorageMCPServerRepository describes repository information.
type StorageMCPServerRepository struct {
	URL       string  `json:"url"`
	Type      *string `json:"type,omitempty"`
	Directory *string `json:"directory,omitempty"`
}

// StorageMCPServerSnapshotType is the MCP server version snapshot.
type StorageMCPServerSnapshotType struct {
	Name             string                       `json:"name"`
	Version          string                       `json:"version"`
	Description      *string                      `json:"description,omitempty"`
	Instructions     *string                      `json:"instructions,omitempty"`
	Repository       *StorageMCPServerRepository  `json:"repository,omitempty"`
	ReleaseDate      *string                      `json:"releaseDate,omitempty"`
	IsLatest         *bool                        `json:"isLatest,omitempty"`
	PackageCanonical *string                      `json:"packageCanonical,omitempty"`
	Tools            map[string]StorageToolConfig  `json:"tools,omitempty"`
	Agents           map[string]StorageToolConfig  `json:"agents,omitempty"`
	Workflows        map[string]StorageToolConfig  `json:"workflows,omitempty"`
}

// StorageMCPServerType is the thin MCP server record.
type StorageMCPServerType struct {
	ID              string         `json:"id"`
	Status          EntityStatus   `json:"status"`
	ActiveVersionID *string        `json:"activeVersionId,omitempty"`
	AuthorID        *string        `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// StorageResolvedMCPServerType is the resolved MCP server (record + snapshot).
type StorageResolvedMCPServerType struct {
	StorageMCPServerType
	StorageMCPServerSnapshotType
	ResolvedVersionID *string `json:"resolvedVersionId,omitempty"`
}

// StorageCreateMCPServerInput is the input for creating a new MCP server.
type StorageCreateMCPServerInput struct {
	ID       string         `json:"id"`
	AuthorID *string        `json:"authorId,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	StorageMCPServerSnapshotType
}

// StorageUpdateMCPServerInput is the input for updating an MCP server.
type StorageUpdateMCPServerInput struct {
	ID              string         `json:"id"`
	AuthorID        *string        `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	ActiveVersionID *string        `json:"activeVersionId,omitempty"`
	Status          *EntityStatus  `json:"status,omitempty"`
	// Snapshot fields (all optional for update).
	Name             *string                      `json:"name,omitempty"`
	Version          *string                      `json:"version,omitempty"`
	Description      *string                      `json:"description,omitempty"`
	Instructions     *string                      `json:"instructions,omitempty"`
	Repository       *StorageMCPServerRepository  `json:"repository,omitempty"`
	ReleaseDate      *string                      `json:"releaseDate,omitempty"`
	IsLatest         *bool                        `json:"isLatest,omitempty"`
	PackageCanonical *string                      `json:"packageCanonical,omitempty"`
	Tools            map[string]StorageToolConfig  `json:"tools,omitempty"`
	Agents           map[string]StorageToolConfig  `json:"agents,omitempty"`
	Workflows        map[string]StorageToolConfig  `json:"workflows,omitempty"`
}

// StorageListMCPServersInput specifies filters for listing MCP servers.
type StorageListMCPServersInput struct {
	PerPage  *int            `json:"perPage,omitempty"`
	Page     *int            `json:"page,omitempty"`
	OrderBy  *StorageOrderBy `json:"orderBy,omitempty"`
	AuthorID *string         `json:"authorId,omitempty"`
	Metadata map[string]any  `json:"metadata,omitempty"`
	Status   *EntityStatus   `json:"status,omitempty"`
}

// StorageListMCPServersOutput is the paginated output for thin MCP server records.
type StorageListMCPServersOutput struct {
	PaginationInfo
	MCPServers []StorageMCPServerType `json:"mcpServers"`
}

// StorageListMCPServersResolvedOutput is the paginated output for resolved MCP servers.
type StorageListMCPServersResolvedOutput struct {
	PaginationInfo
	MCPServers []StorageResolvedMCPServerType `json:"mcpServers"`
}

// ---------------------------------------------------------------------------
// Workspace Storage Types
// ---------------------------------------------------------------------------

// StorageFilesystemConfig is a serializable filesystem configuration.
type StorageFilesystemConfig struct {
	Provider string         `json:"provider"`
	Config   map[string]any `json:"config"`
	ReadOnly *bool          `json:"readOnly,omitempty"`
}

// StorageSandboxConfig is a serializable sandbox configuration.
type StorageSandboxConfig struct {
	Provider string         `json:"provider"`
	Config   map[string]any `json:"config"`
}

// StorageSearchBM25Config holds BM25 parameters when not just a boolean.
type StorageSearchBM25Config struct {
	K1 *float64 `json:"k1,omitempty"`
	B  *float64 `json:"b,omitempty"`
}

// StorageSearchConfig is a serializable search configuration.
type StorageSearchConfig struct {
	VectorProvider  *string         `json:"vectorProvider,omitempty"`
	VectorConfig    map[string]any  `json:"vectorConfig,omitempty"`
	EmbedderProvider *string        `json:"embedderProvider,omitempty"`
	EmbedderModel   *string         `json:"embedderModel,omitempty"`
	EmbedderConfig  map[string]any  `json:"embedderConfig,omitempty"`
	BM25            any             `json:"bm25,omitempty"` // bool | StorageSearchBM25Config
	SearchIndexName *string         `json:"searchIndexName,omitempty"`
	AutoIndexPaths  []string        `json:"autoIndexPaths,omitempty"`
}

// StorageWorkspaceToolConfig is per-tool configuration for workspace tools.
type StorageWorkspaceToolConfig struct {
	Enabled               *bool `json:"enabled,omitempty"`
	RequireApproval       *bool `json:"requireApproval,omitempty"`
	RequireReadBeforeWrite *bool `json:"requireReadBeforeWrite,omitempty"`
}

// StorageWorkspaceToolsConfig is the workspace tools configuration.
type StorageWorkspaceToolsConfig struct {
	Enabled         *bool                                 `json:"enabled,omitempty"`
	RequireApproval *bool                                 `json:"requireApproval,omitempty"`
	Tools           map[string]StorageWorkspaceToolConfig `json:"tools,omitempty"`
}

// StorageWorkspaceSnapshotType is the workspace version snapshot.
type StorageWorkspaceSnapshotType struct {
	Name             string                              `json:"name"`
	Description      *string                             `json:"description,omitempty"`
	Filesystem       *StorageFilesystemConfig            `json:"filesystem,omitempty"`
	Sandbox          *StorageSandboxConfig               `json:"sandbox,omitempty"`
	Mounts           map[string]StorageFilesystemConfig  `json:"mounts,omitempty"`
	Search           *StorageSearchConfig                `json:"search,omitempty"`
	Skills           []string                            `json:"skills,omitempty"`
	Tools            *StorageWorkspaceToolsConfig        `json:"tools,omitempty"`
	AutoSync         *bool                               `json:"autoSync,omitempty"`
	OperationTimeout *int                                `json:"operationTimeout,omitempty"`
}

// StorageWorkspaceType is the thin workspace record.
type StorageWorkspaceType struct {
	ID              string         `json:"id"`
	Status          EntityStatus   `json:"status"`
	ActiveVersionID *string        `json:"activeVersionId,omitempty"`
	AuthorID        *string        `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// StorageResolvedWorkspaceType is the resolved workspace (record + snapshot).
type StorageResolvedWorkspaceType struct {
	StorageWorkspaceType
	StorageWorkspaceSnapshotType
	ResolvedVersionID *string `json:"resolvedVersionId,omitempty"`
}

// StorageCreateWorkspaceInput is the input for creating a new workspace.
type StorageCreateWorkspaceInput struct {
	ID       string         `json:"id"`
	AuthorID *string        `json:"authorId,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
	StorageWorkspaceSnapshotType
}

// StorageUpdateWorkspaceInput is the input for updating a workspace.
type StorageUpdateWorkspaceInput struct {
	ID              string         `json:"id"`
	AuthorID        *string        `json:"authorId,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	ActiveVersionID *string        `json:"activeVersionId,omitempty"`
	Status          *EntityStatus  `json:"status,omitempty"`
	// Snapshot fields (all optional for update).
	Name             *string                             `json:"name,omitempty"`
	Description      *string                             `json:"description,omitempty"`
	Filesystem       *StorageFilesystemConfig            `json:"filesystem,omitempty"`
	Sandbox          *StorageSandboxConfig               `json:"sandbox,omitempty"`
	Mounts           map[string]StorageFilesystemConfig  `json:"mounts,omitempty"`
	Search           *StorageSearchConfig                `json:"search,omitempty"`
	Skills           []string                            `json:"skills,omitempty"`
	Tools            *StorageWorkspaceToolsConfig        `json:"tools,omitempty"`
	AutoSync         *bool                               `json:"autoSync,omitempty"`
	OperationTimeout *int                                `json:"operationTimeout,omitempty"`
}

// StorageListWorkspacesInput specifies filters for listing workspaces.
type StorageListWorkspacesInput struct {
	PerPage  *int            `json:"perPage,omitempty"`
	Page     *int            `json:"page,omitempty"`
	OrderBy  *StorageOrderBy `json:"orderBy,omitempty"`
	AuthorID *string         `json:"authorId,omitempty"`
	Metadata map[string]any  `json:"metadata,omitempty"`
}

// StorageListWorkspacesOutput is the paginated output for thin workspace records.
type StorageListWorkspacesOutput struct {
	PaginationInfo
	Workspaces []StorageWorkspaceType `json:"workspaces"`
}

// StorageListWorkspacesResolvedOutput is the paginated output for resolved workspaces.
type StorageListWorkspacesResolvedOutput struct {
	PaginationInfo
	Workspaces []StorageResolvedWorkspaceType `json:"workspaces"`
}

// ---------------------------------------------------------------------------
// Skill Storage Types
// ---------------------------------------------------------------------------

// StorageContentSourceType discriminates content source types.
type StorageContentSourceType string

const (
	StorageContentSourceExternal StorageContentSourceType = "external"
	StorageContentSourceLocal    StorageContentSourceType = "local"
	StorageContentSourceManaged  StorageContentSourceType = "managed"
)

// StorageContentSource is a serializable content source for skills.
type StorageContentSource struct {
	Type        StorageContentSourceType `json:"type"`
	PackagePath *string                  `json:"packagePath,omitempty"` // for "external"
	ProjectPath *string                  `json:"projectPath,omitempty"` // for "local"
	MastraPath  *string                  `json:"mastraPath,omitempty"`  // for "managed"
}

// StorageSkillSnapshotType is the skill version snapshot.
type StorageSkillSnapshotType struct {
	Name          string                `json:"name"`
	Description   string                `json:"description"`
	Instructions  string                `json:"instructions"`
	License       *string               `json:"license,omitempty"`
	Compatibility any                   `json:"compatibility,omitempty"`
	Source        *StorageContentSource `json:"source,omitempty"`
	References    []string              `json:"references,omitempty"`
	Scripts       []string              `json:"scripts,omitempty"`
	Assets        []string              `json:"assets,omitempty"`
	Metadata      map[string]any        `json:"metadata,omitempty"`
	Tree          *SkillVersionTree     `json:"tree,omitempty"`
}

// StorageSkillType is the thin skill record.
type StorageSkillType struct {
	ID              string       `json:"id"`
	Status          EntityStatus `json:"status"`
	ActiveVersionID *string      `json:"activeVersionId,omitempty"`
	AuthorID        *string      `json:"authorId,omitempty"`
	CreatedAt       time.Time    `json:"createdAt"`
	UpdatedAt       time.Time    `json:"updatedAt"`
}

// StorageResolvedSkillType is the resolved skill (record + snapshot).
type StorageResolvedSkillType struct {
	StorageSkillType
	StorageSkillSnapshotType
	ResolvedVersionID *string `json:"resolvedVersionId,omitempty"`
}

// StorageCreateSkillInput is the input for creating a new skill.
type StorageCreateSkillInput struct {
	ID       string  `json:"id"`
	AuthorID *string `json:"authorId,omitempty"`
	StorageSkillSnapshotType
}

// StorageUpdateSkillInput is the input for updating a skill.
type StorageUpdateSkillInput struct {
	ID              string        `json:"id"`
	AuthorID        *string       `json:"authorId,omitempty"`
	ActiveVersionID *string       `json:"activeVersionId,omitempty"`
	Status          *EntityStatus `json:"status,omitempty"`
	// Snapshot fields (all optional for update).
	Name          *string               `json:"name,omitempty"`
	Description   *string               `json:"description,omitempty"`
	Instructions  *string               `json:"instructions,omitempty"`
	License       *string               `json:"license,omitempty"`
	Compatibility any                   `json:"compatibility,omitempty"`
	Source        *StorageContentSource `json:"source,omitempty"`
	References    []string              `json:"references,omitempty"`
	Scripts       []string              `json:"scripts,omitempty"`
	Assets        []string              `json:"assets,omitempty"`
	Metadata      map[string]any        `json:"metadata,omitempty"`
	Tree          *SkillVersionTree     `json:"tree,omitempty"`
}

// StorageListSkillsInput specifies filters for listing skills.
type StorageListSkillsInput struct {
	PerPage  *int            `json:"perPage,omitempty"`
	Page     *int            `json:"page,omitempty"`
	OrderBy  *StorageOrderBy `json:"orderBy,omitempty"`
	AuthorID *string         `json:"authorId,omitempty"`
	Metadata map[string]any  `json:"metadata,omitempty"`
}

// StorageListSkillsOutput is the paginated output for thin skill records.
type StorageListSkillsOutput struct {
	PaginationInfo
	Skills []StorageSkillType `json:"skills"`
}

// StorageListSkillsResolvedOutput is the paginated output for resolved skills.
type StorageListSkillsResolvedOutput struct {
	PaginationInfo
	Skills []StorageResolvedSkillType `json:"skills"`
}

// StorageSkillConfig is per-skill configuration stored in agent snapshots.
type StorageSkillConfig struct {
	Description  *string `json:"description,omitempty"`
	Instructions *string `json:"instructions,omitempty"`
	Pin          *string `json:"pin,omitempty"`
	Strategy     *string `json:"strategy,omitempty"` // "latest" | "live"
}

// SkillResolutionStrategy enumerates skill resolution strategies.
type SkillResolutionStrategy string

const (
	SkillResolutionLatest SkillResolutionStrategy = "latest"
	SkillResolutionLive   SkillResolutionStrategy = "live"
)

// SkillVersionTreeEntry is a single entry in a skill version's file tree manifest.
type SkillVersionTreeEntry struct {
	BlobHash string  `json:"blobHash"`
	Size     int     `json:"size"`
	MimeType *string `json:"mimeType,omitempty"`
	Encoding *string `json:"encoding,omitempty"` // "utf-8" | "base64"
}

// SkillVersionTree is the complete file tree manifest for a skill version.
type SkillVersionTree struct {
	Entries map[string]SkillVersionTreeEntry `json:"entries"`
}

// StorageBlobEntry is a stored blob in the content-addressable blob store.
type StorageBlobEntry struct {
	Hash      string    `json:"hash"`
	Content   string    `json:"content"`
	Size      int       `json:"size"`
	MimeType  *string   `json:"mimeType,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// ---------------------------------------------------------------------------
// Workspace Reference Types
// ---------------------------------------------------------------------------

// StorageWorkspaceRefType discriminates workspace reference types.
type StorageWorkspaceRefType string

const (
	StorageWorkspaceRefByID     StorageWorkspaceRefType = "id"
	StorageWorkspaceRefInline   StorageWorkspaceRefType = "inline"
)

// StorageWorkspaceRef is a workspace reference: either by ID or inline config.
type StorageWorkspaceRef struct {
	Type        StorageWorkspaceRefType       `json:"type"`
	WorkspaceID *string                       `json:"workspaceId,omitempty"` // for "id"
	Config      *StorageWorkspaceSnapshotType `json:"config,omitempty"`     // for "inline"
}

// ---------------------------------------------------------------------------
// Workflow Storage Types (re-exported from storage/domains and workflows)
// ---------------------------------------------------------------------------

// UpdateWorkflowStateOptions specifies options for updating workflow state.
type UpdateWorkflowStateOptions = workflows.UpdateWorkflowStateOptions

// UpdateWorkflowResumeLabel describes a resume label in workflow state.
type UpdateWorkflowResumeLabel = workflows.UpdateWorkflowResumeLabel

// ---------------------------------------------------------------------------
// Dataset Types (re-exported from storage/domains and storage/domains/datasets)
// ---------------------------------------------------------------------------

// TargetType is the type of entity a dataset experiment targets.
type TargetType = domains.TargetType

const (
	TargetTypeAgent     TargetType = "agent"
	TargetTypeWorkflow  TargetType = "workflow"
	TargetTypeScorer    TargetType = "scorer"
	TargetTypeProcessor TargetType = "processor"
)

// DatasetRecord is a dataset record.
type DatasetRecord = datasets.DatasetRecord

// DatasetItem is an item within a dataset.
type DatasetItem = datasets.DatasetItem

// DatasetItemRow is the raw database row for a dataset item (includes versioning fields).
type DatasetItemRow = datasets.DatasetItemRow

// DatasetVersion represents a dataset version record.
type DatasetVersion = datasets.DatasetVersion

// CreateDatasetInput is the input for creating a new dataset.
type CreateDatasetInput = datasets.CreateDatasetInput

// UpdateDatasetInput is the input for updating a dataset.
type UpdateDatasetInput = datasets.UpdateDatasetInput

// AddDatasetItemInput is the input for adding an item to a dataset.
type AddDatasetItemInput = datasets.AddDatasetItemInput

// UpdateDatasetItemInput is the input for updating a dataset item.
type UpdateDatasetItemInput = datasets.UpdateDatasetItemInput

// ListDatasetsInput is the input for listing datasets.
type ListDatasetsInput = datasets.ListDatasetsInput

// ListDatasetsOutput is the paginated output for listing datasets.
type ListDatasetsOutput = datasets.ListDatasetsOutput

// ListDatasetItemsInput is the input for listing dataset items.
type ListDatasetItemsInput = datasets.ListDatasetItemsInput

// ListDatasetItemsOutput is the paginated output for listing dataset items.
type ListDatasetItemsOutput = datasets.ListDatasetItemsOutput

// ListDatasetVersionsInput is the input for listing dataset versions.
type ListDatasetVersionsInput = datasets.ListDatasetVersionsInput

// ListDatasetVersionsOutput is the paginated output for listing dataset versions.
type ListDatasetVersionsOutput = datasets.ListDatasetVersionsOutput

// BatchInsertItemInput is a single item in a batch insert operation.
type BatchInsertItemInput = domains.BatchInsertItemInput

// BatchInsertItemsInput is the input for batch inserting dataset items.
type BatchInsertItemsInput = datasets.BatchInsertItemsInput

// BatchDeleteItemsInput is the input for batch deleting dataset items.
type BatchDeleteItemsInput = datasets.BatchDeleteItemsInput

// ---------------------------------------------------------------------------
// Experiment Types (re-exported from storage/domains and storage/domains/experiments)
// ---------------------------------------------------------------------------

// ExperimentStatus represents the status of an experiment.
type ExperimentStatus = domains.ExperimentStatus

// Experiment represents an experiment record.
type Experiment = experiments.Experiment

// ExperimentResultError is the error structure in an experiment result.
type ExperimentResultError = domains.ExperimentResultError

// ExperimentResult is a single result within an experiment.
type ExperimentResult = experiments.ExperimentResult

// CreateExperimentInput is the input for creating a new experiment.
type CreateExperimentInput = experiments.CreateExperimentInput

// UpdateExperimentInput is the input for updating an experiment.
type UpdateExperimentInput = experiments.UpdateExperimentInput

// AddExperimentResultInput is the input for adding an experiment result.
type AddExperimentResultInput = experiments.AddExperimentResultInput

// ListExperimentsInput is the input for listing experiments.
type ListExperimentsInput = experiments.ListExperimentsInput

// ListExperimentsOutput is the paginated output for listing experiments.
type ListExperimentsOutput = experiments.ListExperimentsOutput

// ListExperimentResultsInput is the input for listing experiment results.
type ListExperimentResultsInput = experiments.ListExperimentResultsInput

// ListExperimentResultsOutput is the paginated output for listing experiment results.
type ListExperimentResultsOutput = experiments.ListExperimentResultsOutput
