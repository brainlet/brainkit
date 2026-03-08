// Ported from: packages/openai/src/chat/openai-chat-api.ts
package openai

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// --- Function tool types ---

// OpenAIChatFunctionTool represents a function tool in OpenAI chat format.
type OpenAIChatFunctionTool struct {
	Type     string                         `json:"type"` // "function"
	Function OpenAIChatFunctionToolFunction `json:"function"`
}

// OpenAIChatFunctionToolFunction holds the function definition.
type OpenAIChatFunctionToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Strict      *bool          `json:"strict,omitempty"`
}

// OpenAIChatToolChoice represents tool choice. Can be a string ("auto", "none", "required")
// or a specific function selection.
type OpenAIChatToolChoice = any // string or OpenAIChatToolChoiceFunction

// OpenAIChatToolChoiceFunction selects a specific function tool.
type OpenAIChatToolChoiceFunction struct {
	Type     string                                `json:"type"` // "function"
	Function OpenAIChatToolChoiceFunctionReference `json:"function"`
}

// OpenAIChatToolChoiceFunctionReference holds the function name.
type OpenAIChatToolChoiceFunctionReference struct {
	Name string `json:"name"`
}

// --- Response types ---

// openaiChatResponse represents the non-streaming chat API response.
type openaiChatResponse struct {
	ID      *string              `json:"id,omitempty"`
	Created *float64             `json:"created,omitempty"`
	Model   *string              `json:"model,omitempty"`
	Choices []openaiChatChoice   `json:"choices"`
	Usage   *OpenAIChatUsage     `json:"usage,omitempty"`
}

// openaiChatChoice represents a choice in the chat completion response.
type openaiChatChoice struct {
	Message      openaiChatMessage `json:"message"`
	Index        int               `json:"index"`
	Logprobs     *openaiLogprobs   `json:"logprobs,omitempty"`
	FinishReason *string           `json:"finish_reason,omitempty"`
}

// openaiChatMessage represents a message in a chat completion choice.
type openaiChatMessage struct {
	Role        *string                        `json:"role,omitempty"`
	Content     *string                        `json:"content,omitempty"`
	ToolCalls   []openaiChatMessageToolCall    `json:"tool_calls,omitempty"`
	Annotations []openaiChatAnnotation         `json:"annotations,omitempty"`
}

// openaiChatMessageToolCall represents a tool call in the response message.
type openaiChatMessageToolCall struct {
	ID       *string                               `json:"id,omitempty"`
	Type     string                                `json:"type"` // "function"
	Function openaiChatMessageToolCallFunction     `json:"function"`
}

// openaiChatMessageToolCallFunction holds the function details.
type openaiChatMessageToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// openaiChatAnnotation represents a URL citation annotation.
type openaiChatAnnotation struct {
	Type        string                     `json:"type"` // "url_citation"
	URLCitation openaiChatAnnotationCitation `json:"url_citation"`
}

// openaiChatAnnotationCitation holds citation details.
type openaiChatAnnotationCitation struct {
	StartIndex int    `json:"start_index"`
	EndIndex   int    `json:"end_index"`
	URL        string `json:"url"`
	Title      string `json:"title"`
}

// openaiLogprobs holds log probability data.
type openaiLogprobs struct {
	Content []openaiLogprobContent `json:"content,omitempty"`
}

// openaiLogprobContent holds individual token log probability data.
type openaiLogprobContent struct {
	Token       string                  `json:"token"`
	Logprob     float64                 `json:"logprob"`
	TopLogprobs []openaiTopLogprob      `json:"top_logprobs"`
}

// openaiTopLogprob holds top log probability for a token.
type openaiTopLogprob struct {
	Token   string  `json:"token"`
	Logprob float64 `json:"logprob"`
}

// openaiChatResponseSchema is the schema for non-streaming responses.
var openaiChatResponseSchema = &providerutils.Schema[openaiChatResponse]{}

// --- Streaming chunk types ---

// openaiChatChunk represents a streaming chunk. Since it can be either a regular
// chunk or an error, we parse into map[string]any and handle both cases.
// This mirrors the TS z.union([chunkSchema, errorSchema]).

// openaiChatChunkSchema is the schema for streaming chunks.
// Uses map[string]any to handle the union of regular chunks and error chunks.
var openaiChatChunkSchema = &providerutils.Schema[any]{}
