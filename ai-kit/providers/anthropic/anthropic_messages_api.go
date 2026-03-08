// Ported from: packages/anthropic/src/anthropic-messages-api.ts
package anthropic

// AnthropicMessagesPrompt represents the prompt structure for the Anthropic Messages API.
type AnthropicMessagesPrompt struct {
	System   []AnthropicTextContent `json:"system,omitempty"`
	Messages []AnthropicMessage     `json:"messages"`
}

// AnthropicMessage is a union type for user or assistant messages.
// Use the Role field to discriminate.
type AnthropicMessage struct {
	Role    string `json:"role"` // "user" or "assistant"
	Content []any  `json:"content"`
}

// AnthropicCacheControl represents cache control settings.
type AnthropicCacheControl struct {
	Type string  `json:"type"` // "ephemeral"
	TTL  *string `json:"ttl,omitempty"` // "5m" or "1h"
}

// AnthropicCompactionContent represents a compaction content block.
type AnthropicCompactionContent struct {
	Type         string                `json:"type"` // "compaction"
	Content      string                `json:"content"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicTextContent represents a text content block.
type AnthropicTextContent struct {
	Type         string                `json:"type"` // "text"
	Text         string                `json:"text"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicThinkingContent represents a thinking content block.
type AnthropicThinkingContent struct {
	Type      string `json:"type"` // "thinking"
	Thinking  string `json:"thinking"`
	Signature string `json:"signature"`
}

// AnthropicRedactedThinkingContent represents a redacted thinking content block.
type AnthropicRedactedThinkingContent struct {
	Type string `json:"type"` // "redacted_thinking"
	Data string `json:"data"`
}

// AnthropicContentSource is a union type for content sources (base64, url, text).
type AnthropicContentSource struct {
	Type      string  `json:"type"` // "base64", "url", "text"
	MediaType *string `json:"media_type,omitempty"`
	Data      *string `json:"data,omitempty"`
	URL       *string `json:"url,omitempty"`
}

// AnthropicImageContent represents an image content block.
type AnthropicImageContent struct {
	Type         string                  `json:"type"` // "image"
	Source       AnthropicContentSource  `json:"source"`
	CacheControl *AnthropicCacheControl  `json:"cache_control,omitempty"`
}

// AnthropicDocumentContent represents a document content block.
type AnthropicDocumentContent struct {
	Type         string                  `json:"type"` // "document"
	Source       AnthropicContentSource  `json:"source"`
	Title        *string                 `json:"title,omitempty"`
	Context      *string                 `json:"context,omitempty"`
	Citations    *struct {
		Enabled bool `json:"enabled"`
	} `json:"citations,omitempty"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicToolCallCaller represents the caller information for programmatic tool calling.
type AnthropicToolCallCaller struct {
	Type   string  `json:"type"` // "code_execution_20250825", "code_execution_20260120", "direct"
	ToolID *string `json:"tool_id,omitempty"`
}

// AnthropicToolCallContent represents a tool_use content block.
type AnthropicToolCallContent struct {
	Type         string                   `json:"type"` // "tool_use"
	ID           string                   `json:"id"`
	Name         string                   `json:"name"`
	Input        any                      `json:"input"`
	Caller       *AnthropicToolCallCaller `json:"caller,omitempty"`
	CacheControl *AnthropicCacheControl   `json:"cache_control,omitempty"`
}

// AnthropicServerToolUseContent represents a server_tool_use content block.
type AnthropicServerToolUseContent struct {
	Type         string                   `json:"type"` // "server_tool_use"
	ID           string                   `json:"id"`
	Name         string                   `json:"name"`
	Input        any                      `json:"input"`
	Caller       *AnthropicToolCallCaller `json:"caller,omitempty"`
	CacheControl *AnthropicCacheControl   `json:"cache_control,omitempty"`
}

// AnthropicToolReferenceContent represents a tool_reference content block.
type AnthropicToolReferenceContent struct {
	Type     string `json:"type"` // "tool_reference"
	ToolName string `json:"tool_name"`
}

// AnthropicToolResultContent represents a tool_result content block.
type AnthropicToolResultContent struct {
	Type         string                `json:"type"` // "tool_result"
	ToolUseID    string                `json:"tool_use_id"`
	Content      any                   `json:"content"` // string or []any
	IsError      *bool                 `json:"is_error,omitempty"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicWebSearchToolResultContent represents a web_search_tool_result content block.
type AnthropicWebSearchToolResultContent struct {
	Type         string                `json:"type"` // "web_search_tool_result"
	ToolUseID    string                `json:"tool_use_id"`
	Content      []map[string]any      `json:"content"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicToolSearchToolResultContent represents a tool_search_tool_result content block.
type AnthropicToolSearchToolResultContent struct {
	Type         string                `json:"type"` // "tool_search_tool_result"
	ToolUseID    string                `json:"tool_use_id"`
	Content      map[string]any        `json:"content"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicCodeExecutionToolResultContent represents a code_execution_tool_result content block.
type AnthropicCodeExecutionToolResultContent struct {
	Type         string                `json:"type"` // "code_execution_tool_result"
	ToolUseID    string                `json:"tool_use_id"`
	Content      map[string]any        `json:"content"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicTextEditorCodeExecutionToolResultContent represents a text_editor_code_execution_tool_result content block.
type AnthropicTextEditorCodeExecutionToolResultContent struct {
	Type         string                `json:"type"` // "text_editor_code_execution_tool_result"
	ToolUseID    string                `json:"tool_use_id"`
	Content      map[string]any        `json:"content"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicBashCodeExecutionToolResultContent represents a bash_code_execution_tool_result content block.
type AnthropicBashCodeExecutionToolResultContent struct {
	Type         string                `json:"type"` // "bash_code_execution_tool_result"
	ToolUseID    string                `json:"tool_use_id"`
	Content      map[string]any        `json:"content"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicWebFetchToolResultContent represents a web_fetch_tool_result content block.
type AnthropicWebFetchToolResultContent struct {
	Type         string                `json:"type"` // "web_fetch_tool_result"
	ToolUseID    string                `json:"tool_use_id"`
	Content      map[string]any        `json:"content"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicMcpToolUseContent represents an mcp_tool_use content block.
type AnthropicMcpToolUseContent struct {
	Type         string                `json:"type"` // "mcp_tool_use"
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	ServerName   string                `json:"server_name"`
	Input        any                   `json:"input"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicMcpToolResultContent represents an mcp_tool_result content block.
type AnthropicMcpToolResultContent struct {
	Type         string                `json:"type"` // "mcp_tool_result"
	ToolUseID    string                `json:"tool_use_id"`
	IsError      bool                  `json:"is_error"`
	Content      any                   `json:"content"` // string or []map[string]any
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
}

// AnthropicTool represents a tool definition for the Anthropic API.
// This is a union type; use the Type field for built-in tools and check for InputSchema
// for function tools (where Type is empty or absent).
type AnthropicTool struct {
	// Common fields
	Name         string                `json:"name"`
	Type         string                `json:"type,omitempty"`
	CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`

	// Function tool fields
	Description        *string        `json:"description,omitempty"`
	InputSchema        map[string]any `json:"input_schema,omitempty"`
	EagerInputStreaming *bool          `json:"eager_input_streaming,omitempty"`
	Strict             *bool          `json:"strict,omitempty"`
	DeferLoading       *bool          `json:"defer_loading,omitempty"`
	AllowedCallers     []string       `json:"allowed_callers,omitempty"`
	InputExamples      []any          `json:"input_examples,omitempty"`

	// Computer tool fields
	DisplayWidthPx  *int  `json:"display_width_px,omitempty"`
	DisplayHeightPx *int  `json:"display_height_px,omitempty"`
	DisplayNumber   *int  `json:"display_number,omitempty"`
	EnableZoom      *bool `json:"enable_zoom,omitempty"`

	// Text editor 20250728 field
	MaxCharacters *int `json:"max_characters,omitempty"`

	// Web fetch fields
	MaxUses          *int              `json:"max_uses,omitempty"`
	AllowedDomains   []string          `json:"allowed_domains,omitempty"`
	BlockedDomains   []string          `json:"blocked_domains,omitempty"`
	Citations        *map[string]any   `json:"citations,omitempty"`
	MaxContentTokens *int              `json:"max_content_tokens,omitempty"`

	// Web search fields
	UserLocation *map[string]any `json:"user_location,omitempty"`
}

// AnthropicToolChoice represents tool choice configuration.
type AnthropicToolChoice struct {
	Type                   string `json:"type"` // "auto", "any", "tool"
	Name                   string `json:"name,omitempty"`
	DisableParallelToolUse *bool  `json:"disable_parallel_tool_use,omitempty"`
}

// AnthropicContainer represents container configuration for the API request.
type AnthropicContainer struct {
	ID     *string `json:"id,omitempty"`
	Skills []struct {
		Type    string  `json:"type"`
		SkillID string  `json:"skill_id"`
		Version *string `json:"version,omitempty"`
	} `json:"skills,omitempty"`
}

// AnthropicSpeed represents the speed setting.
type AnthropicSpeed = string

const (
	SpeedFast     AnthropicSpeed = "fast"
	SpeedStandard AnthropicSpeed = "standard"
)

// AnthropicMessagesResponse represents the response from the Anthropic Messages API.
type AnthropicMessagesResponse struct {
	Type         string           `json:"type"` // "message"
	ID           *string          `json:"id,omitempty"`
	Model        *string          `json:"model,omitempty"`
	Content      []map[string]any `json:"content"`
	StopReason   *string          `json:"stop_reason,omitempty"`
	StopSequence *string          `json:"stop_sequence,omitempty"`
	Usage        AnthropicMessagesUsage `json:"usage"`
	Container    *struct {
		ExpiresAt string `json:"expires_at"`
		ID        string `json:"id"`
		Skills    []struct {
			Type    string `json:"type"`
			SkillID string `json:"skill_id"`
			Version string `json:"version"`
		} `json:"skills"`
	} `json:"container,omitempty"`
	ContextManagement *struct {
		AppliedEdits []map[string]any `json:"applied_edits"`
	} `json:"context_management,omitempty"`
}

// anthropicMessagesResponseSchema is the schema for parsing Anthropic Messages API responses.
var anthropicMessagesResponseSchema = &AnthropicMessagesResponse{}

// AnthropicMessagesChunk represents a streaming chunk from the Anthropic Messages API.
type AnthropicMessagesChunk struct {
	Type  string         `json:"type"`
	Index *int           `json:"index,omitempty"`
	Delta map[string]any `json:"delta,omitempty"`

	// For message_start
	Message *AnthropicMessagesResponse `json:"message,omitempty"`

	// For content_block_start
	ContentBlock map[string]any `json:"content_block,omitempty"`

	// For usage events
	Usage *AnthropicMessagesUsage `json:"usage,omitempty"`
}

// AnthropicCitation represents a citation from the response.
type AnthropicCitation struct {
	Type           string  `json:"type"` // "web_search_result_location", "page_location", "char_location"
	CitedText      string  `json:"cited_text"`
	URL            *string `json:"url,omitempty"`
	Title          *string `json:"title,omitempty"`
	EncryptedIndex *string `json:"encrypted_index,omitempty"`
	DocumentIndex  *int    `json:"document_index,omitempty"`
	DocumentTitle  *string `json:"document_title,omitempty"`
	StartPageNumber *int   `json:"start_page_number,omitempty"`
	EndPageNumber   *int   `json:"end_page_number,omitempty"`
	StartCharIndex  *int   `json:"start_char_index,omitempty"`
	EndCharIndex    *int   `json:"end_char_index,omitempty"`
}

// AnthropicReasoningMetadata represents reasoning metadata in streaming responses.
type AnthropicReasoningMetadata struct {
	Type      string `json:"type"` // "summary"
	Summary   string `json:"summary"`
}

// AnthropicResponseContextManagement represents context management response data from the API.
type AnthropicResponseContextManagement struct {
	AppliedEdits []map[string]any `json:"applied_edits"`
}

// AnthropicContextManagementAPIEdit represents a context management edit in the API request.
type AnthropicContextManagementAPIEdit struct {
	Type string `json:"type"`

	// clear_tool_uses_20250919 fields
	Trigger        *map[string]any `json:"trigger,omitempty"`
	Keep           any             `json:"keep,omitempty"` // can be "all" or map
	ClearAtLeast   *map[string]any `json:"clear_at_least,omitempty"`
	ClearToolInputs *bool          `json:"clear_tool_inputs,omitempty"`
	ExcludeTools   []string        `json:"exclude_tools,omitempty"`

	// compact_20260112 fields
	PauseAfterCompaction *bool   `json:"pause_after_compaction,omitempty"`
	Instructions         *string `json:"instructions,omitempty"`
}
