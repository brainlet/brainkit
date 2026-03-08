// Ported from: packages/groq/src/groq-api-types.ts
package groq

// GroqChatPrompt is a slice of GroqMessage representing a chat prompt.
type GroqChatPrompt = []GroqMessage

// GroqMessage is a sealed interface for Groq chat messages.
type GroqMessage interface {
	groqMessageRole() string
}

// GroqSystemMessage represents a system message.
type GroqSystemMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (GroqSystemMessage) groqMessageRole() string { return "system" }

// GroqUserMessage represents a user message with text or multipart content.
type GroqUserMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []GroqContentPart
}

func (GroqUserMessage) groqMessageRole() string { return "user" }

// GroqContentPart is a sealed interface for content parts in a user message.
type GroqContentPart interface {
	groqContentPartType() string
}

// GroqContentPartText represents a text content part.
type GroqContentPartText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (GroqContentPartText) groqContentPartType() string { return "text" }

// GroqContentPartImage represents an image content part.
type GroqContentPartImage struct {
	Type     string              `json:"type"`
	ImageURL GroqContentImageURL `json:"image_url"`
}

func (GroqContentPartImage) groqContentPartType() string { return "image_url" }

// GroqContentImageURL holds the URL for an image content part.
type GroqContentImageURL struct {
	URL string `json:"url"`
}

// GroqAssistantMessage represents an assistant message.
type GroqAssistantMessage struct {
	Role      string                `json:"role"`
	Content   *string               `json:"content,omitempty"`
	Reasoning *string               `json:"reasoning,omitempty"`
	ToolCalls []GroqMessageToolCall `json:"tool_calls,omitempty"`
}

func (GroqAssistantMessage) groqMessageRole() string { return "assistant" }

// GroqMessageToolCall represents a tool call in an assistant message.
type GroqMessageToolCall struct {
	Type     string                      `json:"type"`
	ID       string                      `json:"id"`
	Function GroqMessageToolCallFunction `json:"function"`
}

// GroqMessageToolCallFunction represents the function part of a tool call.
type GroqMessageToolCallFunction struct {
	Arguments string `json:"arguments"`
	Name      string `json:"name"`
}

// GroqToolMessage represents a tool response message.
type GroqToolMessage struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolCallID string `json:"tool_call_id"`
}

func (GroqToolMessage) groqMessageRole() string { return "tool" }

// GroqTranscriptionAPITypes represents the Groq transcription API request fields.
type GroqTranscriptionAPITypes struct {
	// File is the audio file object for direct upload.
	File *string `json:"file,omitempty"`

	// URL is the audio URL to translate/transcribe.
	URL *string `json:"url,omitempty"`

	// Language is the input audio language in ISO-639-1 format.
	Language *string `json:"language,omitempty"`

	// Model is the ID of the model to use.
	Model string `json:"model"`

	// Prompt to guide the model's style (limited to 224 tokens).
	Prompt *string `json:"prompt,omitempty"`

	// ResponseFormat defines the output response format.
	ResponseFormat *string `json:"response_format,omitempty"`

	// Temperature between 0 and 1.
	Temperature *float64 `json:"temperature,omitempty"`

	// TimestampGranularities for transcription (word and/or segment).
	TimestampGranularities []string `json:"timestamp_granularities,omitempty"`
}
