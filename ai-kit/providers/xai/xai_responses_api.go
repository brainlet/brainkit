// Ported from: packages/xai/src/responses/xai-responses-api.ts
package xai

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// --- Input types ---

// XaiResponsesInput is the input to the xAI responses API.
type XaiResponsesInput = []XaiResponsesInputItem

// XaiResponsesInputItem is a single input item.
type XaiResponsesInputItem interface {
	xaiResponsesInputItemType() string
}

// XaiResponsesSystemMessage is a system/developer message.
type XaiResponsesSystemMessage struct {
	Role    string `json:"role"` // "system" or "developer"
	Content string `json:"content"`
}

func (XaiResponsesSystemMessage) xaiResponsesInputItemType() string { return "system" }

// XaiResponsesUserMessageContentPart is content within a user message.
type XaiResponsesUserMessageContentPart struct {
	Type     string `json:"type"` // "input_text" or "input_image"
	Text     string `json:"text,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
}

// XaiResponsesUserMessage is a user message.
type XaiResponsesUserMessage struct {
	Role    string                                `json:"role"`
	Content []XaiResponsesUserMessageContentPart `json:"content"`
}

func (XaiResponsesUserMessage) xaiResponsesInputItemType() string { return "user" }

// XaiResponsesAssistantMessage is an assistant message.
type XaiResponsesAssistantMessage struct {
	Role    string  `json:"role"`
	Content string  `json:"content"`
	ID      *string `json:"id,omitempty"`
}

func (XaiResponsesAssistantMessage) xaiResponsesInputItemType() string { return "assistant" }

// XaiResponsesFunctionCallOutput is a function call output.
type XaiResponsesFunctionCallOutput struct {
	Type   string `json:"type"` // "function_call_output"
	CallID string `json:"call_id"`
	Output string `json:"output"`
}

func (XaiResponsesFunctionCallOutput) xaiResponsesInputItemType() string { return "function_call_output" }

// XaiResponsesReasoning is a reasoning input item.
type XaiResponsesReasoning struct {
	Type             string                          `json:"type"` // "reasoning"
	ID               string                          `json:"id"`
	Summary          []XaiResponsesReasoningSummary  `json:"summary"`
	Status           string                          `json:"status"`
	EncryptedContent *string                         `json:"encrypted_content,omitempty"`
}

func (XaiResponsesReasoning) xaiResponsesInputItemType() string { return "reasoning" }

// XaiResponsesReasoningSummary is a summary entry.
type XaiResponsesReasoningSummary struct {
	Type string `json:"type"` // "summary_text"
	Text string `json:"text"`
}

// XaiResponsesToolCallInput is a tool call input item.
type XaiResponsesToolCallInput struct {
	Type      string  `json:"type"` // "function_call", "web_search_call", etc.
	ID        string  `json:"id"`
	CallID    *string `json:"call_id,omitempty"`
	Name      *string `json:"name,omitempty"`
	Arguments *string `json:"arguments,omitempty"`
	Input     *string `json:"input,omitempty"`
	Status    string  `json:"status"`
	Action    any     `json:"action,omitempty"`
}

func (XaiResponsesToolCallInput) xaiResponsesInputItemType() string { return "tool_call" }

// --- Tool types ---

// XaiResponsesTool represents a tool for the responses API.
// Represented as a map for flexibility since it's a union type.
type XaiResponsesTool = map[string]interface{}

// --- Response types ---

// XaiResponsesUsage is usage information from the responses API.
type XaiResponsesUsage struct {
	InputTokens   int                          `json:"input_tokens"`
	OutputTokens  int                          `json:"output_tokens"`
	TotalTokens   *int                         `json:"total_tokens,omitempty"`
	InputTokensDetails  *XaiResponsesInputTokensDetails  `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *XaiResponsesOutputTokensDetails `json:"output_tokens_details,omitempty"`
	NumSourcesUsed         *int `json:"num_sources_used,omitempty"`
	NumServerSideToolsUsed *int `json:"num_server_side_tools_used,omitempty"`
}

// XaiResponsesInputTokensDetails contains details about input tokens.
type XaiResponsesInputTokensDetails struct {
	CachedTokens *int `json:"cached_tokens,omitempty"`
}

// XaiResponsesOutputTokensDetails contains details about output tokens.
type XaiResponsesOutputTokensDetails struct {
	ReasoningTokens *int `json:"reasoning_tokens,omitempty"`
}

// XaiResponsesAnnotation is an annotation in a response.
type XaiResponsesAnnotation struct {
	Type  string  `json:"type"`
	URL   string  `json:"url,omitempty"`
	Title *string `json:"title,omitempty"`
}

// XaiResponsesMessageContentPart is a content part in a response message.
type XaiResponsesMessageContentPart struct {
	Type        string                     `json:"type"`
	Text        *string                    `json:"text,omitempty"`
	Logprobs    []interface{}              `json:"logprobs,omitempty"`
	Annotations []XaiResponsesAnnotation   `json:"annotations,omitempty"`
}

// XaiResponsesReasoningSummaryPart is a reasoning summary part.
type XaiResponsesReasoningSummaryPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// XaiResponsesOutputItem represents an output item in a response.
type XaiResponsesOutputItem struct {
	Type             string                            `json:"type"`
	ID               string                            `json:"id"`
	Status           string                            `json:"status,omitempty"`
	Role             string                            `json:"role,omitempty"`
	Content          []XaiResponsesMessageContentPart  `json:"content,omitempty"`
	Name             *string                           `json:"name,omitempty"`
	Arguments        *string                           `json:"arguments,omitempty"`
	Input            *string                           `json:"input,omitempty"`
	CallID           string                            `json:"call_id,omitempty"`
	Action           any                               `json:"action,omitempty"`
	Summary          []XaiResponsesReasoningSummaryPart `json:"summary,omitempty"`
	EncryptedContent *string                           `json:"encrypted_content,omitempty"`
	Queries          []string                          `json:"queries,omitempty"`
	Results          []XaiResponsesFileSearchResult    `json:"results,omitempty"`
	// MCP-specific fields
	ServerLabel string  `json:"server_label,omitempty"`
	Output      *string `json:"output,omitempty"`
	Error       *string `json:"error,omitempty"`
}

// XaiResponsesFileSearchResult is a file search result.
type XaiResponsesFileSearchResult struct {
	FileID   string  `json:"file_id"`
	Filename string  `json:"filename"`
	Score    float64 `json:"score"`
	Text     string  `json:"text"`
}

// XaiResponsesResponse is the full response from the responses API.
type XaiResponsesResponse struct {
	ID        *string                   `json:"id,omitempty"`
	CreatedAt *int64                    `json:"created_at,omitempty"`
	Model     *string                   `json:"model,omitempty"`
	Object    string                    `json:"object"`
	Output    []XaiResponsesOutputItem  `json:"output"`
	Usage     *XaiResponsesUsage        `json:"usage,omitempty"`
	Status    *string                   `json:"status,omitempty"`
}

// xaiResponsesResponseSchema is the schema for response validation.
var xaiResponsesResponseSchema = &providerutils.Schema[XaiResponsesResponse]{}

// --- Streaming event types ---

// XaiResponsesChunk represents a streaming event from the responses API.
type XaiResponsesChunk struct {
	Type            string                             `json:"type"`
	Response        *XaiResponsesResponse              `json:"response,omitempty"`
	Item            *XaiResponsesOutputItem            `json:"item,omitempty"`
	OutputIndex     int                                `json:"output_index,omitempty"`
	ContentIndex    int                                `json:"content_index,omitempty"`
	SummaryIndex    int                                `json:"summary_index,omitempty"`
	AnnotationIndex int                                `json:"annotation_index,omitempty"`
	ItemID          string                             `json:"item_id,omitempty"`
	Delta           string                             `json:"delta,omitempty"`
	Text            string                             `json:"text,omitempty"`
	Part            *XaiResponsesMessageContentPart    `json:"part,omitempty"`
	Annotation      *XaiResponsesAnnotation            `json:"annotation,omitempty"`
	Arguments       string                             `json:"arguments,omitempty"`
	Input           string                             `json:"input,omitempty"`
	Code            string                             `json:"code,omitempty"`
	Output          string                             `json:"output,omitempty"`
	Logprobs        []interface{}                      `json:"logprobs,omitempty"`
	Annotations     []XaiResponsesAnnotation           `json:"annotations,omitempty"`
}

// xaiResponsesChunkSchema is the schema for streaming chunk validation.
var xaiResponsesChunkSchema = &providerutils.Schema[XaiResponsesChunk]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[XaiResponsesChunk], error) {
		b, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		var chunk XaiResponsesChunk
		if err := json.Unmarshal(b, &chunk); err != nil {
			return nil, err
		}
		return &providerutils.ValidationResult[XaiResponsesChunk]{
			Success: true,
			Value:   chunk,
		}, nil
	},
}

// XaiResponsesIncludeValue represents values for the include parameter.
type XaiResponsesIncludeValue = string
