// Ported from: packages/deepseek/src/chat/deepseek-chat-api-types.ts
package deepseek

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// DeepSeekChatPrompt is a list of DeepSeek messages.
type DeepSeekChatPrompt = []DeepSeekMessage

// DeepSeekMessage is a sealed interface for DeepSeek messages.
type DeepSeekMessage interface {
	deepSeekMessageRole() string
}

// DeepSeekSystemMessage represents a system message.
type DeepSeekSystemMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (DeepSeekSystemMessage) deepSeekMessageRole() string { return "system" }

// DeepSeekUserMessage represents a user message.
type DeepSeekUserMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (DeepSeekUserMessage) deepSeekMessageRole() string { return "user" }

// DeepSeekAssistantMessage represents an assistant message.
type DeepSeekAssistantMessage struct {
	Role             string                   `json:"role"`
	Content          *string                  `json:"content,omitempty"`
	ReasoningContent *string                  `json:"reasoning_content,omitempty"`
	ToolCalls        []DeepSeekMessageToolCall `json:"tool_calls,omitempty"`
}

func (DeepSeekAssistantMessage) deepSeekMessageRole() string { return "assistant" }

// DeepSeekMessageToolCall represents a tool call in an assistant message.
type DeepSeekMessageToolCall struct {
	Type     string                          `json:"type"`
	ID       string                          `json:"id"`
	Function DeepSeekMessageToolCallFunction `json:"function"`
}

// DeepSeekMessageToolCallFunction represents the function part of a tool call.
type DeepSeekMessageToolCallFunction struct {
	Arguments string `json:"arguments"`
	Name      string `json:"name"`
}

// DeepSeekToolMessage represents a tool result message.
type DeepSeekToolMessage struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolCallID string `json:"tool_call_id"`
}

func (DeepSeekToolMessage) deepSeekMessageRole() string { return "tool" }

// DeepSeekFunctionTool represents a function tool definition.
type DeepSeekFunctionTool struct {
	Type     string                       `json:"type"`
	Function DeepSeekFunctionToolFunction `json:"function"`
}

// DeepSeekFunctionToolFunction is the function definition inside a tool.
type DeepSeekFunctionToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters,omitempty"`
	Strict      *bool  `json:"strict,omitempty"`
}

// DeepSeekToolChoice represents the tool choice for a request.
// Can be nil, a string ("auto", "none", "required"), or a DeepSeekToolChoiceFunction.
type DeepSeekToolChoice = any

// DeepSeekToolChoiceFunction selects a specific function tool.
type DeepSeekToolChoiceFunction struct {
	Type     string                              `json:"type"`
	Function DeepSeekToolChoiceFunctionReference `json:"function"`
}

// DeepSeekToolChoiceFunctionReference holds the name of a specific function.
type DeepSeekToolChoiceFunctionReference struct {
	Name string `json:"name"`
}

// DeepSeekChatTokenUsage represents token usage in a DeepSeek API response.
type DeepSeekChatTokenUsage struct {
	PromptTokens         *int `json:"prompt_tokens,omitempty"`
	CompletionTokens     *int `json:"completion_tokens,omitempty"`
	PromptCacheHitTokens *int `json:"prompt_cache_hit_tokens,omitempty"`
	PromptCacheMissTokens *int `json:"prompt_cache_miss_tokens,omitempty"`
	TotalTokens          *int `json:"total_tokens,omitempty"`

	CompletionTokensDetails *struct {
		ReasoningTokens *int `json:"reasoning_tokens,omitempty"`
	} `json:"completion_tokens_details,omitempty"`
}

// DeepSeekErrorData represents a DeepSeek API error response.
type DeepSeekErrorData struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type,omitempty"`
		Param   any    `json:"param,omitempty"`
		Code    any    `json:"code,omitempty"`
	} `json:"error"`
}

// deepSeekChatCompletionResponse represents the non-streaming API response.
type deepSeekChatCompletionResponse struct {
	ID      *string                          `json:"id,omitempty"`
	Created *float64                         `json:"created,omitempty"`
	Model   *string                          `json:"model,omitempty"`
	Choices []deepSeekChatCompletionChoice   `json:"choices"`
	Usage   *DeepSeekChatTokenUsage          `json:"usage,omitempty"`
}

// deepSeekChatCompletionChoice represents a choice in the response.
type deepSeekChatCompletionChoice struct {
	Message      deepSeekChatCompletionMessage `json:"message"`
	FinishReason *string                       `json:"finish_reason,omitempty"`
}

// deepSeekChatCompletionMessage represents a message in a choice.
type deepSeekChatCompletionMessage struct {
	Role             *string                                `json:"role,omitempty"`
	Content          *string                                `json:"content,omitempty"`
	ReasoningContent *string                                `json:"reasoning_content,omitempty"`
	ToolCalls        []deepSeekChatCompletionMessageToolCall `json:"tool_calls,omitempty"`
}

// deepSeekChatCompletionMessageToolCall represents a tool call in a response message.
type deepSeekChatCompletionMessageToolCall struct {
	ID       *string `json:"id,omitempty"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// deepSeekChatCompletionResponseSchema is the schema for non-streaming responses.
var deepSeekChatCompletionResponseSchema = &providerutils.Schema[deepSeekChatCompletionResponse]{}

// deepSeekErrorSchema is the schema for DeepSeek error responses.
var deepSeekErrorSchema = &providerutils.Schema[DeepSeekErrorData]{}

// deepSeekChatChunkSchema is the schema for streaming chunks.
// In Go, we parse into map[string]any since the chunk can be either a regular chunk or an error.
var deepSeekChatChunkSchema = &providerutils.Schema[any]{}
