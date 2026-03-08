// Ported from: packages/openai/src/responses/openai-responses-options.ts
package openai

// TopLogprobsMax is the maximum number of most likely tokens to return at each
// token position, each with an associated log probability.
// See: https://platform.openai.com/docs/api-reference/responses/create#responses_create-top_logprobs
const TopLogprobsMax = 20

// OpenAIResponsesReasoningModelIDs is the list of known reasoning model IDs.
var OpenAIResponsesReasoningModelIDs = []string{
	"o1",
	"o1-2024-12-17",
	"o3",
	"o3-2025-04-16",
	"o3-mini",
	"o3-mini-2025-01-31",
	"o4-mini",
	"o4-mini-2025-04-16",
	"gpt-5",
	"gpt-5-2025-08-07",
	"gpt-5-codex",
	"gpt-5-mini",
	"gpt-5-mini-2025-08-07",
	"gpt-5-nano",
	"gpt-5-nano-2025-08-07",
	"gpt-5-pro",
	"gpt-5-pro-2025-10-06",
	"gpt-5.1",
	"gpt-5.1-chat-latest",
	"gpt-5.1-codex-mini",
	"gpt-5.1-codex",
	"gpt-5.1-codex-max",
	"gpt-5.2",
	"gpt-5.2-chat-latest",
	"gpt-5.2-pro",
	"gpt-5.2-codex",
	"gpt-5.4",
	"gpt-5.4-2026-03-05",
	"gpt-5.4-pro",
	"gpt-5.4-pro-2026-03-05",
	"gpt-5.3-codex",
}

// OpenAIResponsesModelIDs is the list of all known Responses API model IDs.
var OpenAIResponsesModelIDs = append([]string{
	"gpt-4.1",
	"gpt-4.1-2025-04-14",
	"gpt-4.1-mini",
	"gpt-4.1-mini-2025-04-14",
	"gpt-4.1-nano",
	"gpt-4.1-nano-2025-04-14",
	"gpt-4o",
	"gpt-4o-2024-05-13",
	"gpt-4o-2024-08-06",
	"gpt-4o-2024-11-20",
	"gpt-4o-audio-preview",
	"gpt-4o-audio-preview-2024-12-17",
	"gpt-4o-search-preview",
	"gpt-4o-search-preview-2025-03-11",
	"gpt-4o-mini-search-preview",
	"gpt-4o-mini-search-preview-2025-03-11",
	"gpt-4o-mini",
	"gpt-4o-mini-2024-07-18",
	"gpt-3.5-turbo-0125",
	"gpt-3.5-turbo",
	"gpt-3.5-turbo-1106",
	"gpt-5-chat-latest",
}, OpenAIResponsesReasoningModelIDs...)

// OpenAILanguageModelResponsesOptions are provider-specific options for
// OpenAI Responses API language model calls.
type OpenAILanguageModelResponsesOptions struct {
	// Conversation is the ID of the OpenAI Conversation to continue.
	// Cannot be used in conjunction with PreviousResponseID.
	Conversation *string `json:"conversation,omitempty"`

	// Include is the set of extra fields to include in the response.
	Include []string `json:"include,omitempty"`

	// Instructions for the model. They can be used to change the system or
	// developer message when continuing a conversation using PreviousResponseID.
	Instructions *string `json:"instructions,omitempty"`

	// Logprobs controls whether to return log probabilities of the tokens.
	// Can be a bool (true = max) or an int (1-20).
	Logprobs *LogprobsSetting `json:"logprobs,omitempty"`

	// MaxToolCalls is the maximum number of total calls to built-in tools
	// that can be processed in a response.
	MaxToolCalls *int `json:"maxToolCalls,omitempty"`

	// Metadata is additional metadata to store with the generation.
	Metadata any `json:"metadata,omitempty"`

	// ParallelToolCalls controls whether to use parallel tool calls. Defaults to true.
	ParallelToolCalls *bool `json:"parallelToolCalls,omitempty"`

	// PreviousResponseID is the ID of the previous response for conversation continuation.
	PreviousResponseID *string `json:"previousResponseId,omitempty"`

	// PromptCacheKey sets a cache key to tie this prompt to cached prefixes.
	PromptCacheKey *string `json:"promptCacheKey,omitempty"`

	// PromptCacheRetention is the retention policy for the prompt cache.
	// Valid values: "in_memory", "24h".
	PromptCacheRetention *string `json:"promptCacheRetention,omitempty"`

	// ReasoningEffort for reasoning models.
	// Valid values: "none", "minimal", "low", "medium", "high", "xhigh".
	ReasoningEffort *string `json:"reasoningEffort,omitempty"`

	// ReasoningSummary controls reasoning summary output from the model.
	// Valid values: "auto", "detailed".
	ReasoningSummary *string `json:"reasoningSummary,omitempty"`

	// SafetyIdentifier is the identifier for safety monitoring and tracking.
	SafetyIdentifier *string `json:"safetyIdentifier,omitempty"`

	// ServiceTier for the request.
	// Valid values: "auto", "flex", "priority", "default".
	ServiceTier *string `json:"serviceTier,omitempty"`

	// Store controls whether to store the generation. Defaults to true.
	Store *bool `json:"store,omitempty"`

	// StrictJSONSchema controls whether to use strict JSON schema validation. Defaults to true.
	StrictJSONSchema *bool `json:"strictJsonSchema,omitempty"`

	// TextVerbosity controls the verbosity of the model's responses.
	// Valid values: "low", "medium", "high".
	TextVerbosity *string `json:"textVerbosity,omitempty"`

	// Truncation controls output truncation. "auto" (default) or "disabled".
	Truncation *string `json:"truncation,omitempty"`

	// User is a unique identifier representing the end-user.
	User *string `json:"user,omitempty"`

	// SystemMessageMode overrides the system message mode for this model.
	// Valid values: "system", "developer", "remove".
	SystemMessageMode *string `json:"systemMessageMode,omitempty"`

	// ForceReasoning forces treating this model as a reasoning model.
	ForceReasoning *bool `json:"forceReasoning,omitempty"`
}

// LogprobsSetting represents a logprobs setting that can be either a bool or an int.
type LogprobsSetting struct {
	BoolValue *bool
	IntValue  *int
}

// IsEnabled returns true if logprobs are enabled.
func (l *LogprobsSetting) IsEnabled() bool {
	if l == nil {
		return false
	}
	if l.BoolValue != nil {
		return *l.BoolValue
	}
	if l.IntValue != nil {
		return *l.IntValue > 0
	}
	return false
}

// TopLogprobs returns the top_logprobs value for the API request.
// Returns nil if logprobs is not enabled.
func (l *LogprobsSetting) TopLogprobs() *int {
	if l == nil {
		return nil
	}
	if l.IntValue != nil {
		return l.IntValue
	}
	if l.BoolValue != nil && *l.BoolValue {
		max := TopLogprobsMax
		return &max
	}
	return nil
}
