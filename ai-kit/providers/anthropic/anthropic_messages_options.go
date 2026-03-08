// Ported from: packages/anthropic/src/anthropic-messages-options.ts
package anthropic

// AnthropicMessagesModelId represents the Anthropic model ID type.
// In Go this is simply a string; all known model IDs are listed as constants.
type AnthropicMessagesModelId = string

// Known Anthropic model IDs.
const (
	ModelClaude3Haiku20240307     AnthropicMessagesModelId = "claude-3-haiku-20240307"
	ModelClaudeHaiku45_20251001   AnthropicMessagesModelId = "claude-haiku-4-5-20251001"
	ModelClaudeHaiku45            AnthropicMessagesModelId = "claude-haiku-4-5"
	ModelClaudeOpus40             AnthropicMessagesModelId = "claude-opus-4-0"
	ModelClaudeOpus4_20250514     AnthropicMessagesModelId = "claude-opus-4-20250514"
	ModelClaudeOpus41_20250805    AnthropicMessagesModelId = "claude-opus-4-1-20250805"
	ModelClaudeOpus41             AnthropicMessagesModelId = "claude-opus-4-1"
	ModelClaudeOpus45             AnthropicMessagesModelId = "claude-opus-4-5"
	ModelClaudeOpus45_20251101    AnthropicMessagesModelId = "claude-opus-4-5-20251101"
	ModelClaudeSonnet40           AnthropicMessagesModelId = "claude-sonnet-4-0"
	ModelClaudeSonnet4_20250514   AnthropicMessagesModelId = "claude-sonnet-4-20250514"
	ModelClaudeSonnet45_20250929  AnthropicMessagesModelId = "claude-sonnet-4-5-20250929"
	ModelClaudeSonnet45           AnthropicMessagesModelId = "claude-sonnet-4-5"
	ModelClaudeSonnet46           AnthropicMessagesModelId = "claude-sonnet-4-6"
	ModelClaudeOpus46             AnthropicMessagesModelId = "claude-opus-4-6"
)

// AnthropicFilePartProviderOptions contains Anthropic file part provider options
// for document-specific features. These options apply to individual file parts (documents).
type AnthropicFilePartProviderOptions struct {
	// Citations configuration for this document.
	Citations *struct {
		Enabled bool `json:"enabled"`
	} `json:"citations,omitempty"`

	// Title is a custom title for the document.
	Title *string `json:"title,omitempty"`

	// Context is context about the document that will be passed to the model.
	Context *string `json:"context,omitempty"`
}

// AnthropicThinkingConfig represents the thinking configuration.
type AnthropicThinkingConfig struct {
	// Type is the thinking type: "adaptive", "enabled", or "disabled".
	Type string `json:"type"`

	// BudgetTokens is the budget for thinking tokens (only for "enabled" type).
	BudgetTokens *int `json:"budgetTokens,omitempty"`
}

// AnthropicMCPServerConfig represents an MCP server configuration.
type AnthropicMCPServerConfig struct {
	Type               string                           `json:"type"` // "url"
	Name               string                           `json:"name"`
	URL                string                           `json:"url"`
	AuthorizationToken *string                          `json:"authorizationToken,omitempty"`
	ToolConfiguration  *AnthropicMCPToolConfiguration   `json:"toolConfiguration,omitempty"`
}

// AnthropicMCPToolConfiguration configures tool access for an MCP server.
type AnthropicMCPToolConfiguration struct {
	Enabled      *bool    `json:"enabled,omitempty"`
	AllowedTools []string `json:"allowedTools,omitempty"`
}

// AnthropicContainerConfig represents container configuration for skills.
type AnthropicContainerConfig struct {
	ID     *string                       `json:"id,omitempty"`
	Skills []AnthropicContainerSkillConfig `json:"skills,omitempty"`
}

// AnthropicContainerSkillConfig represents a skill configuration entry.
type AnthropicContainerSkillConfig struct {
	Type    string  `json:"type"` // "anthropic" or "custom"
	SkillID string  `json:"skillId"`
	Version *string `json:"version,omitempty"`
}

// AnthropicContextManagementConfig represents context management configuration.
type AnthropicContextManagementConfig struct {
	Edits []AnthropicContextManagementEdit `json:"edits"`
}

// AnthropicContextManagementEdit represents a context management edit configuration.
// This is a union type; the Type field discriminates between the variants.
type AnthropicContextManagementEdit struct {
	// Type discriminates the edit kind:
	// "clear_tool_uses_20250919", "clear_thinking_20251015", "compact_20260112"
	Type string `json:"type"`

	// For clear_tool_uses_20250919:
	Trigger       *AnthropicContextManagementTrigger `json:"trigger,omitempty"`
	Keep          *AnthropicContextManagementKeep    `json:"keep,omitempty"`
	ClearAtLeast  *AnthropicContextManagementClearAtLeast `json:"clearAtLeast,omitempty"`
	ClearToolInputs *bool                            `json:"clearToolInputs,omitempty"`
	ExcludeTools  []string                           `json:"excludeTools,omitempty"`

	// For clear_thinking_20251015:
	// Keep is reused; when Type is "clear_thinking_20251015", Keep may be nil
	// and KeepAll may be true (represented as the string "all").
	KeepAll *bool `json:"-"` // internal; serialized specially

	// For compact_20260112:
	PauseAfterCompaction *bool   `json:"pauseAfterCompaction,omitempty"`
	Instructions         *string `json:"instructions,omitempty"`
}

// AnthropicContextManagementTrigger represents a trigger for context management edits.
type AnthropicContextManagementTrigger struct {
	Type  string `json:"type"` // "input_tokens" or "tool_uses"
	Value int    `json:"value"`
}

// AnthropicContextManagementKeep represents the keep configuration for context management edits.
type AnthropicContextManagementKeep struct {
	Type  string `json:"type"` // "tool_uses" or "thinking_turns"
	Value int    `json:"value"`
}

// AnthropicContextManagementClearAtLeast represents the clearAtLeast configuration.
type AnthropicContextManagementClearAtLeast struct {
	Type  string `json:"type"` // "input_tokens"
	Value int    `json:"value"`
}

// AnthropicLanguageModelOptions represents provider-specific options for Anthropic language models.
type AnthropicLanguageModelOptions struct {
	// SendReasoning indicates whether to send reasoning to the model.
	SendReasoning *bool `json:"sendReasoning,omitempty"`

	// StructuredOutputMode determines how structured outputs are generated.
	// Values: "outputFormat", "jsonTool", "auto"
	StructuredOutputMode *string `json:"structuredOutputMode,omitempty"`

	// Thinking is the configuration for enabling Claude's extended thinking.
	Thinking *AnthropicThinkingConfig `json:"thinking,omitempty"`

	// DisableParallelToolUse indicates whether to disable parallel function calling.
	DisableParallelToolUse *bool `json:"disableParallelToolUse,omitempty"`

	// CacheControl is cache control settings for this message.
	CacheControl *AnthropicCacheControl `json:"cacheControl,omitempty"`

	// MCPServers are MCP servers to be utilized in this request.
	MCPServers []AnthropicMCPServerConfig `json:"mcpServers,omitempty"`

	// Container is agent skills configuration.
	Container *AnthropicContainerConfig `json:"container,omitempty"`

	// ToolStreaming indicates whether to enable tool streaming.
	ToolStreaming *bool `json:"toolStreaming,omitempty"`

	// Effort is the effort level: "low", "medium", "high", "max".
	Effort *string `json:"effort,omitempty"`

	// Speed enables fast mode for faster inference. Values: "fast", "standard".
	Speed *string `json:"speed,omitempty"`

	// AnthropicBeta is a set of beta features to enable.
	AnthropicBeta []string `json:"anthropicBeta,omitempty"`

	// ContextManagement is the context management configuration.
	ContextManagement *AnthropicContextManagementConfig `json:"contextManagement,omitempty"`
}
