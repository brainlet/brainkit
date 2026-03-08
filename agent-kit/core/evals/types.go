// Ported from: packages/core/src/evals/types.ts
package evals

import (
	"time"

	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
	"github.com/brainlet/brainkit/agent-kit/core/storage/domains"
)

// ============================================================================
// Sampling Config
// ============================================================================

// ScoringSamplingConfigType enumerates the sampling strategies for scoring.
type ScoringSamplingConfigType string

const (
	ScoringSamplingNone  ScoringSamplingConfigType = "none"
	ScoringSamplingRatio ScoringSamplingConfigType = "ratio"
)

// ScoringSamplingConfig defines how frequently a scorer should be executed.
// When Type is ScoringSamplingNone, all runs are scored.
// When Type is ScoringSamplingRatio, Rate defines the probability [0, 1].
type ScoringSamplingConfig struct {
	Type ScoringSamplingConfigType `json:"type"`
	Rate float64                  `json:"rate,omitempty"` // Only used when Type == ScoringSamplingRatio
}

// ============================================================================
// Scoring Source & Entity Type
// ============================================================================

// ScoringSource identifies the origin of a score.
type ScoringSource string

const (
	ScoringSourceLive ScoringSource = "LIVE"
	ScoringSourceTest ScoringSource = "TEST"
)

// ScoringEntityType identifies the type of entity being scored.
// Includes AGENT, WORKFLOW, and all SpanType values.
type ScoringEntityType string

const (
	ScoringEntityTypeAgent    ScoringEntityType = "AGENT"
	ScoringEntityTypeWorkflow ScoringEntityType = "WORKFLOW"
	// Additional entity types derived from SpanType values:
	ScoringEntityTypeAgentRun              ScoringEntityType = ScoringEntityType(obstypes.SpanTypeAgentRun)
	ScoringEntityTypeGeneric               ScoringEntityType = ScoringEntityType(obstypes.SpanTypeGeneric)
	ScoringEntityTypeModelGeneration       ScoringEntityType = ScoringEntityType(obstypes.SpanTypeModelGeneration)
	ScoringEntityTypeModelStep             ScoringEntityType = ScoringEntityType(obstypes.SpanTypeModelStep)
	ScoringEntityTypeModelChunk            ScoringEntityType = ScoringEntityType(obstypes.SpanTypeModelChunk)
	ScoringEntityTypeMCPToolCall           ScoringEntityType = ScoringEntityType(obstypes.SpanTypeMCPToolCall)
	ScoringEntityTypeProcessorRun          ScoringEntityType = ScoringEntityType(obstypes.SpanTypeProcessorRun)
	ScoringEntityTypeToolCall              ScoringEntityType = ScoringEntityType(obstypes.SpanTypeToolCall)
	ScoringEntityTypeWorkflowRun           ScoringEntityType = ScoringEntityType(obstypes.SpanTypeWorkflowRun)
	ScoringEntityTypeWorkflowStep          ScoringEntityType = ScoringEntityType(obstypes.SpanTypeWorkflowStep)
	ScoringEntityTypeWorkflowConditional   ScoringEntityType = ScoringEntityType(obstypes.SpanTypeWorkflowConditional)
	ScoringEntityTypeWorkflowConditionalEval ScoringEntityType = ScoringEntityType(obstypes.SpanTypeWorkflowConditionalEval)
	ScoringEntityTypeWorkflowParallel      ScoringEntityType = ScoringEntityType(obstypes.SpanTypeWorkflowParallel)
	ScoringEntityTypeWorkflowLoop          ScoringEntityType = ScoringEntityType(obstypes.SpanTypeWorkflowLoop)
	ScoringEntityTypeWorkflowSleep         ScoringEntityType = ScoringEntityType(obstypes.SpanTypeWorkflowSleep)
	ScoringEntityTypeWorkflowWaitEvent     ScoringEntityType = ScoringEntityType(obstypes.SpanTypeWorkflowWaitEvent)
)

// ============================================================================
// Scoring Prompts
// ============================================================================

// ScoringPrompts holds a description and prompt pair used in scorer steps.
type ScoringPrompts struct {
	Description string `json:"description"`
	Prompt      string `json:"prompt"`
}

// ============================================================================
// Base Scoring Input (used for scorer functions)
// ============================================================================

// ScoringInput is the base input for scorer functions.
type ScoringInput struct {
	RunID             string         `json:"runId,omitempty"`
	Input             any            `json:"input,omitempty"`
	Output            any            `json:"output"`
	AdditionalContext map[string]any `json:"additionalContext,omitempty"`
	RequestContext    map[string]any `json:"requestContext,omitempty"`

	// ObservabilityContext is not serializable; added at runtime when needed.
	ObservabilityContext *obstypes.ObservabilityContext `json:"-"`
}

// ============================================================================
// Scoring Hook Input
// ============================================================================

// ScoringHookInput is the input provided to scoring hooks.
type ScoringHookInput struct {
	RunID             string            `json:"runId,omitempty"`
	Scorer            map[string]any    `json:"scorer"`
	Input             any               `json:"input"`
	Output            any               `json:"output"`
	Metadata          map[string]any    `json:"metadata,omitempty"`
	AdditionalContext map[string]any    `json:"additionalContext,omitempty"`
	Source            ScoringSource     `json:"source"`
	Entity            map[string]any    `json:"entity"`
	EntityType        ScoringEntityType `json:"entityType"`
	RequestContext    map[string]any    `json:"requestContext,omitempty"`
	StructuredOutput  *bool             `json:"structuredOutput,omitempty"`
	TraceID           string            `json:"traceId,omitempty"`
	SpanID            string            `json:"spanId,omitempty"`
	ResourceID        string            `json:"resourceId,omitempty"`
	ThreadID          string            `json:"threadId,omitempty"`

	// ObservabilityContext is not serializable; added at runtime when needed.
	ObservabilityContext *obstypes.ObservabilityContext `json:"-"`
}

// ============================================================================
// Extract Step Result
// ============================================================================

// ScoringExtractStepResult is the optional result of an extract step.
type ScoringExtractStepResult = map[string]any

// ============================================================================
// Analyze Step Result (Score Result)
// ============================================================================

// ScoringAnalyzeStepResult holds the result of an analyze step.
type ScoringAnalyzeStepResult struct {
	Result map[string]any `json:"result,omitempty"`
	Score  float64        `json:"score"`
	Prompt string         `json:"prompt,omitempty"`
}

// ============================================================================
// Composite Input Types (for scorer step functions)
// ============================================================================

// ScoringInputWithExtractStepResult extends ScoringInput with extract step results.
type ScoringInputWithExtractStepResult struct {
	RunID             string         `json:"runId"` // Required in this context
	Input             any            `json:"input,omitempty"`
	Output            any            `json:"output"`
	AdditionalContext map[string]any `json:"additionalContext,omitempty"`
	RequestContext    map[string]any `json:"requestContext,omitempty"`
	ExtractStepResult any            `json:"extractStepResult,omitempty"`
	ExtractPrompt     string         `json:"extractPrompt,omitempty"`

	ObservabilityContext *obstypes.ObservabilityContext `json:"-"`
}

// ScoringInputWithExtractStepResultAndAnalyzeStepResult extends with analyze results.
type ScoringInputWithExtractStepResultAndAnalyzeStepResult struct {
	RunID              string         `json:"runId"`
	Input              any            `json:"input,omitempty"`
	Output             any            `json:"output"`
	AdditionalContext  map[string]any `json:"additionalContext,omitempty"`
	RequestContext     map[string]any `json:"requestContext,omitempty"`
	ExtractStepResult  any            `json:"extractStepResult,omitempty"`
	ExtractPrompt      string         `json:"extractPrompt,omitempty"`
	Score              float64        `json:"score"`
	AnalyzeStepResult  map[string]any `json:"analyzeStepResult,omitempty"`
	AnalyzePrompt      string         `json:"analyzePrompt,omitempty"`

	ObservabilityContext *obstypes.ObservabilityContext `json:"-"`
}

// ScoringInputWithExtractStepResultAndScoreAndReason extends with reason.
type ScoringInputWithExtractStepResultAndScoreAndReason struct {
	RunID              string         `json:"runId"`
	Input              any            `json:"input,omitempty"`
	Output             any            `json:"output"`
	AdditionalContext  map[string]any `json:"additionalContext,omitempty"`
	RequestContext     map[string]any `json:"requestContext,omitempty"`
	ExtractStepResult  any            `json:"extractStepResult,omitempty"`
	ExtractPrompt      string         `json:"extractPrompt,omitempty"`
	Score              float64        `json:"score"`
	AnalyzeStepResult  map[string]any `json:"analyzeStepResult,omitempty"`
	AnalyzePrompt      string         `json:"analyzePrompt,omitempty"`
	Reason             string         `json:"reason,omitempty"`
	ReasonPrompt       string         `json:"reasonPrompt,omitempty"`

	ObservabilityContext *obstypes.ObservabilityContext `json:"-"`
}

// ============================================================================
// Score Row Data (stored in DB)
// ============================================================================

// ScoreRowData represents a full score record as stored in the database.
type ScoreRowData struct {
	ID       string `json:"id"`
	ScorerID string `json:"scorerId"`
	EntityID string `json:"entityId"`

	// From ScoringInputWithExtractStepResultAndScoreAndReason
	RunID              string         `json:"runId"`
	Input              any            `json:"input,omitempty"`
	Output             any            `json:"output"`
	AdditionalContext  map[string]any `json:"additionalContext,omitempty"`
	RequestContext     map[string]any `json:"requestContext,omitempty"`
	ExtractStepResult  map[string]any `json:"extractStepResult,omitempty"`
	ExtractPrompt      string         `json:"extractPrompt,omitempty"`
	Score              float64        `json:"score"`
	AnalyzeStepResult  map[string]any `json:"analyzeStepResult,omitempty"`
	AnalyzePrompt      string         `json:"analyzePrompt,omitempty"`
	Reason             string         `json:"reason,omitempty"`
	ReasonPrompt       string         `json:"reasonPrompt,omitempty"`

	// From ScoringHookInput
	Scorer           map[string]any    `json:"scorer"`
	Metadata         map[string]any    `json:"metadata,omitempty"`
	Source           ScoringSource     `json:"source"`
	Entity           map[string]any    `json:"entity"`
	EntityType       ScoringEntityType `json:"entityType,omitempty"`
	StructuredOutput *bool             `json:"structuredOutput,omitempty"`
	TraceID          string            `json:"traceId,omitempty"`
	SpanID           string            `json:"spanId,omitempty"`
	ResourceID       string            `json:"resourceId,omitempty"`
	ThreadID         string            `json:"threadId,omitempty"`

	// Additional ScoreRowData fields
	PreprocessStepResult  map[string]any `json:"preprocessStepResult,omitempty"`
	PreprocessPrompt      string         `json:"preprocessPrompt,omitempty"`
	GenerateScorePrompt   string         `json:"generateScorePrompt,omitempty"`
	GenerateReasonPrompt  string         `json:"generateReasonPrompt,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ============================================================================
// Save Score Payload (for creating new scores)
// ============================================================================

// SaveScorePayload is the input for saving a score (ScoreRowData minus id and timestamps).
type SaveScorePayload struct {
	ScorerID string `json:"scorerId"`
	EntityID string `json:"entityId"`

	RunID              string         `json:"runId"`
	Input              any            `json:"input,omitempty"`
	Output             any            `json:"output"`
	AdditionalContext  map[string]any `json:"additionalContext,omitempty"`
	RequestContext     map[string]any `json:"requestContext,omitempty"`
	ExtractStepResult  map[string]any `json:"extractStepResult,omitempty"`
	ExtractPrompt      string         `json:"extractPrompt,omitempty"`
	Score              float64        `json:"score"`
	AnalyzeStepResult  map[string]any `json:"analyzeStepResult,omitempty"`
	AnalyzePrompt      string         `json:"analyzePrompt,omitempty"`
	Reason             string         `json:"reason,omitempty"`
	ReasonPrompt       string         `json:"reasonPrompt,omitempty"`

	Scorer           map[string]any    `json:"scorer"`
	Metadata         map[string]any    `json:"metadata,omitempty"`
	Source           ScoringSource     `json:"source"`
	Entity           map[string]any    `json:"entity"`
	EntityType       ScoringEntityType `json:"entityType,omitempty"`
	StructuredOutput *bool             `json:"structuredOutput,omitempty"`
	TraceID          string            `json:"traceId,omitempty"`
	SpanID           string            `json:"spanId,omitempty"`
	ResourceID       string            `json:"resourceId,omitempty"`
	ThreadID         string            `json:"threadId,omitempty"`

	PreprocessStepResult  map[string]any `json:"preprocessStepResult,omitempty"`
	PreprocessPrompt      string         `json:"preprocessPrompt,omitempty"`
	GenerateScorePrompt   string         `json:"generateScorePrompt,omitempty"`
	GenerateReasonPrompt  string         `json:"generateReasonPrompt,omitempty"`
}

// ============================================================================
// List Scores Response
// ============================================================================

// ListScoresResponse is the paginated response for listing scores.
type ListScoresResponse struct {
	Pagination domains.PaginationInfo `json:"pagination"`
	Scores     []ScoreRowData         `json:"scores"`
}

// ============================================================================
// Scorer Step Function Types
// ============================================================================

// ExtractionStepFn is a function that extracts data from scoring input.
type ExtractionStepFn func(input ScoringInput) (map[string]any, error)

// AnalyzeStepFn is a function that analyzes extracted data and produces a score.
type AnalyzeStepFn func(input ScoringInputWithExtractStepResult) (*ScoringAnalyzeStepResult, error)

// ReasonResult holds the output of a reason step.
type ReasonResult struct {
	Reason       string `json:"reason"`
	ReasonPrompt string `json:"reasonPrompt,omitempty"`
}

// ReasonStepFn is a function that generates a reason for the score.
// Returns nil when no reason is produced.
type ReasonStepFn func(input ScoringInputWithExtractStepResultAndAnalyzeStepResult) (*ReasonResult, error)

// ============================================================================
// Scorer Options (legacy scorer interface)
// ============================================================================

// ScorerOptions defines the configuration for a legacy scorer.
type ScorerOptions struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Extract     ExtractionStepFn
	Analyze     AnalyzeStepFn
	Reason      ReasonStepFn
	Metadata    map[string]any `json:"metadata,omitempty"`
	IsLLMScorer bool           `json:"isLLMScorer,omitempty"`
}

// ============================================================================
// Scorer Run Input/Output for Agent
// ============================================================================

// MastraDBMessage is a stub for the agent's MastraDBMessage type.
// STRUCTURAL MISMATCH (not a circular dep — evals CAN import
// agent/messagelist/state since state has zero agent-kit imports).
// The real state.MastraDBMessage uses embedded MastraMessageShared struct
// (ID, Role, CreatedAt, ThreadID, ResourceID, Type) while this stub uses
// flat fields. The real MastraMessageContentV2 uses typed slices
// ([]MastraMessagePart, []ToolInvocation) while this stub uses []any.
// Replacing would require refactoring all construction sites in
// scoretraces/utils.go that build parts as map[string]any values.
type MastraDBMessage struct {
	ID        string                 `json:"id"`
	Role      string                 `json:"role"`
	Content   MastraDBMessageContent `json:"content"`
	CreatedAt time.Time              `json:"createdAt"`
}

// MastraDBMessageContent holds the content of a MastraDBMessage.
// STRUCTURAL MISMATCH: The real state.MastraMessageContentV2 has additional
// fields (ExperimentalAttachments, Reasoning, Annotations, Metadata,
// ProviderMetadata) and uses typed slices ([]MastraMessagePart,
// []ToolInvocation) instead of []any. This stub uses simplified field set
// for compatibility with scoretraces code that constructs parts as
// map[string]any values.
type MastraDBMessageContent struct {
	Format          int    `json:"format"`
	Parts           []any  `json:"parts"`
	Content         string `json:"content"`
	ToolInvocations []any  `json:"toolInvocations,omitempty"`
}

// CoreMessage is a stub for the AI SDK CoreMessage type.
// NO REAL TYPE AVAILABLE: ai-kit only ported V3 (@ai-sdk/provider-v6).
// The V4/V5 CoreMessage types used by the Mastra scorer system have not
// been ported to Go. This stub provides the minimal fields needed.
type CoreMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// CoreSystemMessage is a stub for the AI SDK CoreSystemMessage type.
// NO REAL TYPE AVAILABLE: Same as CoreMessage — V4/V5 types not ported.
// The real CoreSystemMessage also has ExperimentalProviderMetadata and
// ProviderOptions fields (see agent/messagelist/state/message_state_manager.go).
type CoreSystemMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ScorerRunInputForAgent is the input format for agent-type scorers.
type ScorerRunInputForAgent struct {
	InputMessages       []MastraDBMessage              `json:"inputMessages"`
	RememberedMessages  []MastraDBMessage              `json:"rememberedMessages"`
	SystemMessages      []CoreMessage                  `json:"systemMessages"`
	TaggedSystemMessages map[string][]CoreSystemMessage `json:"taggedSystemMessages"`
}

// ScorerRunOutputForAgent is the output format for agent-type scorers.
type ScorerRunOutputForAgent = []MastraDBMessage
