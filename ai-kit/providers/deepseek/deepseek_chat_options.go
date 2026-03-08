// Ported from: packages/deepseek/src/chat/deepseek-chat-options.ts
package deepseek

// DeepSeekChatModelId represents a DeepSeek chat model identifier.
// Known values: "deepseek-chat", "deepseek-reasoner", or any custom string.
type DeepSeekChatModelId = string

// DeepSeekLanguageModelOptions holds provider-specific options for DeepSeek language models.
type DeepSeekLanguageModelOptions struct {
	// Thinking configures the thinking/reasoning mode.
	Thinking *DeepSeekThinkingOption `json:"thinking,omitempty"`
}

// DeepSeekThinkingOption configures thinking mode.
type DeepSeekThinkingOption struct {
	// Type is the thinking type. Valid values: "enabled", "disabled".
	Type *string `json:"type,omitempty"`
}

// DeepSeekLanguageModelOptionsSchema is the schema for parsing DeepSeek provider options.
// Uses the providerutils.Schema pattern for type-safe parsing.
// Since Go doesn't have Zod, we define the struct and use the schema for validation.
