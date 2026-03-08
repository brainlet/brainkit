// Ported from: packages/openai-compatible/src/chat/openai-compatible-api-types.ts
package openaicompatible

import "github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"

// ChatPrompt is a sequence of messages forming the chat prompt.
type ChatPrompt = []Message

// Message represents any message in the OpenAI-compatible chat format.
// It is a union type; use SystemMessage, UserMessage, AssistantMessage,
// or ToolMessage to construct values.
type Message struct {
	Role string `json:"role"`

	// Content is the message content. For user messages it may be a string or
	// an array of content parts. For assistant messages it is optional.
	Content interface{} `json:"content,omitempty"`

	// ReasoningContent is optional reasoning content (assistant messages).
	ReasoningContent *string `json:"reasoning_content,omitempty"`

	// ToolCalls is the list of tool calls (assistant messages).
	ToolCalls []MessageToolCall `json:"tool_calls,omitempty"`

	// ToolCallID identifies which tool call this response is for (tool messages).
	ToolCallID *string `json:"tool_call_id,omitempty"`

	// Extra holds arbitrary additional properties for provider-metadata-specific
	// extensibility.
	Extra map[string]jsonvalue.JSONValue `json:"-"`
}

// SystemMessage creates a system message.
func SystemMessage(content string) Message {
	return Message{Role: "system", Content: content}
}

// UserMessage creates a user message with a string content.
func UserMessage(content string) Message {
	return Message{Role: "user", Content: content}
}

// UserMessageParts creates a user message with content parts.
func UserMessageParts(parts []ContentPart) Message {
	return Message{Role: "user", Content: parts}
}

// NewAssistantMessage creates an assistant message.
func NewAssistantMessage(content *string, reasoningContent *string, toolCalls []MessageToolCall) Message {
	return Message{
		Role:             "assistant",
		Content:          content,
		ReasoningContent: reasoningContent,
		ToolCalls:        toolCalls,
	}
}

// NewToolMessage creates a tool message.
func NewToolMessage(content string, toolCallID string) Message {
	return Message{Role: "tool", Content: content, ToolCallID: &toolCallID}
}

// ContentPart is a union type for content parts within a user message.
// Use ContentPartText, ContentPartImage, ContentPartInputAudio, or
// ContentPartFile to construct values.
type ContentPart struct {
	Type string `json:"type"`

	// Text is set when Type == "text".
	Text *string `json:"text,omitempty"`

	// ImageURL is set when Type == "image_url".
	ImageURL *ImageURL `json:"image_url,omitempty"`

	// InputAudio is set when Type == "input_audio".
	InputAudio *InputAudio `json:"input_audio,omitempty"`

	// File is set when Type == "file".
	File *FileData `json:"file,omitempty"`

	// Extra holds arbitrary additional properties for provider-metadata-specific
	// extensibility.
	Extra map[string]jsonvalue.JSONValue `json:"-"`
}

// ImageURL contains the URL for an image content part.
type ImageURL struct {
	URL string `json:"url"`
}

// InputAudio contains the data for an audio content part (Google API).
type InputAudio struct {
	Data   string `json:"data"`
	Format string `json:"format"` // "wav" or "mp3"
}

// FileData contains the data for a file content part (Google API).
type FileData struct {
	Filename string `json:"filename"`
	FileData string `json:"file_data"`
}

// ContentPartText creates a text content part.
func ContentPartText(text string) ContentPart {
	return ContentPart{Type: "text", Text: &text}
}

// ContentPartImage creates an image content part.
func ContentPartImage(url string) ContentPart {
	return ContentPart{Type: "image_url", ImageURL: &ImageURL{URL: url}}
}

// ContentPartInputAudio creates an input audio content part.
func ContentPartInputAudio(data string, format string) ContentPart {
	return ContentPart{Type: "input_audio", InputAudio: &InputAudio{Data: data, Format: format}}
}

// ContentPartFile creates a file content part.
func ContentPartFile(filename string, fileData string) ContentPart {
	return ContentPart{Type: "file", File: &FileData{Filename: filename, FileData: fileData}}
}

// MessageToolCall represents a tool call in an assistant message.
type MessageToolCall struct {
	Type     string           `json:"type"` // Always "function"
	ID       string           `json:"id"`
	Function ToolCallFunction `json:"function"`

	// ExtraContent holds additional content for provider-specific features.
	// Used by Google Gemini for thought signatures via OpenAI compatibility.
	ExtraContent *ToolCallExtraContent `json:"extra_content,omitempty"`

	// Extra holds arbitrary additional properties for provider-metadata-specific
	// extensibility.
	Extra map[string]jsonvalue.JSONValue `json:"-"`
}

// ToolCallFunction contains the function name and arguments for a tool call.
type ToolCallFunction struct {
	Arguments string `json:"arguments"`
	Name      string `json:"name"`
}

// ToolCallExtraContent holds additional content for provider-specific features.
type ToolCallExtraContent struct {
	Google *GoogleExtraContent `json:"google,omitempty"`
}

// GoogleExtraContent holds Google-specific extra content.
type GoogleExtraContent struct {
	ThoughtSignature *string `json:"thought_signature,omitempty"`
}
