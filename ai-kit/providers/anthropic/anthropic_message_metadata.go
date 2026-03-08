// Ported from: packages/anthropic/src/anthropic-message-metadata.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"

// AnthropicUsageIteration represents a single iteration in the usage breakdown.
// When compaction occurs, the API returns an iterations array showing
// usage for each sampling iteration (compaction + message).
type AnthropicUsageIteration struct {
	// Type is the iteration type: "compaction" or "message".
	Type string `json:"type"`

	// InputTokens is the number of input tokens consumed in this iteration.
	InputTokens int `json:"inputTokens"`

	// OutputTokens is the number of output tokens generated in this iteration.
	OutputTokens int `json:"outputTokens"`
}

// AnthropicContainerSkill describes a skill loaded in a container.
type AnthropicContainerSkill struct {
	// Type is the skill type: either "anthropic" (built-in) or "custom" (user-defined).
	Type string `json:"type"`

	// SkillID is the skill ID (1-64 characters).
	SkillID string `json:"skillId"`

	// Version is the skill version or "latest" for most recent version (1-64 characters).
	Version string `json:"version"`
}

// AnthropicContainerInfo holds information about the container used in a request.
type AnthropicContainerInfo struct {
	// ExpiresAt is the time at which the container will expire (RFC3339 timestamp).
	ExpiresAt string `json:"expiresAt"`

	// ID is the identifier for the container used in this request.
	ID string `json:"id"`

	// Skills are the skills loaded in the container.
	Skills []AnthropicContainerSkill `json:"skills"`
}

// AnthropicContextManagementEditClearToolUses represents an edit where tool uses were cleared.
type AnthropicContextManagementEditClearToolUses struct {
	Type               string `json:"type"` // "clear_tool_uses_20250919"
	ClearedToolUses    int    `json:"clearedToolUses"`
	ClearedInputTokens int    `json:"clearedInputTokens"`
}

// AnthropicContextManagementEditClearThinking represents an edit where thinking turns were cleared.
type AnthropicContextManagementEditClearThinking struct {
	Type                string `json:"type"` // "clear_thinking_20251015"
	ClearedThinkingTurns int    `json:"clearedThinkingTurns"`
	ClearedInputTokens   int    `json:"clearedInputTokens"`
}

// AnthropicContextManagementEditCompact represents a compaction edit.
type AnthropicContextManagementEditCompact struct {
	Type string `json:"type"` // "compact_20260112"
}

// AnthropicContextManagementResponse holds context management response information.
type AnthropicContextManagementResponse struct {
	// AppliedEdits is the list of context management edits that were applied.
	AppliedEdits []map[string]any `json:"appliedEdits"`
}

// AnthropicMessageMetadata holds metadata about an Anthropic message response.
type AnthropicMessageMetadata struct {
	// Usage is the raw usage information.
	Usage jsonvalue.JSONObject `json:"usage"`

	// CacheCreationInputTokens is the number of cache creation input tokens.
	CacheCreationInputTokens *int `json:"cacheCreationInputTokens"`

	// StopSequence is the stop sequence that caused generation to stop, if any.
	StopSequence *string `json:"stopSequence"`

	// Iterations is the usage breakdown by iteration when compaction is triggered.
	Iterations []AnthropicUsageIteration `json:"iterations"`

	// Container is information about the container used in this request.
	Container *AnthropicContainerInfo `json:"container"`

	// ContextManagement holds context management response information.
	ContextManagement *AnthropicContextManagementResponse `json:"contextManagement"`
}
