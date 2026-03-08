// Ported from: packages/openai/src/chat/openai-chat-prompt.ts
package openai

// OpenAIChatPrompt is a list of chat completion messages.
type OpenAIChatPrompt = []ChatCompletionMessage

// ChatCompletionMessage is a union type for all chat message types.
// In Go we use an interface with concrete implementations.
type ChatCompletionMessage interface {
	chatCompletionMessageRole() string
}

// ChatCompletionSystemMessage is a system message.
type ChatCompletionSystemMessage struct {
	Role    string `json:"role"` // "system"
	Content string `json:"content"`
}

func (ChatCompletionSystemMessage) chatCompletionMessageRole() string { return "system" }

// ChatCompletionDeveloperMessage is a developer message.
type ChatCompletionDeveloperMessage struct {
	Role    string `json:"role"` // "developer"
	Content string `json:"content"`
}

func (ChatCompletionDeveloperMessage) chatCompletionMessageRole() string { return "developer" }

// ChatCompletionUserMessage is a user message.
type ChatCompletionUserMessage struct {
	Role    string `json:"role"` // "user"
	Content any    `json:"content"` // string or []ChatCompletionContentPart
}

func (ChatCompletionUserMessage) chatCompletionMessageRole() string { return "user" }

// ChatCompletionAssistantMessage is an assistant message.
type ChatCompletionAssistantMessage struct {
	Role      string                          `json:"role"` // "assistant"
	Content   string                          `json:"content,omitempty"`
	ToolCalls []ChatCompletionMessageToolCall `json:"tool_calls,omitempty"`
}

func (ChatCompletionAssistantMessage) chatCompletionMessageRole() string { return "assistant" }

// ChatCompletionToolMessage is a tool message.
type ChatCompletionToolMessage struct {
	Role       string `json:"role"` // "tool"
	Content    string `json:"content"`
	ToolCallID string `json:"tool_call_id"`
}

func (ChatCompletionToolMessage) chatCompletionMessageRole() string { return "tool" }

// ChatCompletionContentPart is a union of content part types for user messages.
type ChatCompletionContentPart interface {
	chatCompletionContentPartType() string
}

// ChatCompletionContentPartText is a text content part.
type ChatCompletionContentPartText struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"`
}

func (ChatCompletionContentPartText) chatCompletionContentPartType() string { return "text" }

// ChatCompletionContentPartImage is an image content part.
type ChatCompletionContentPartImage struct {
	Type     string                              `json:"type"` // "image_url"
	ImageURL ChatCompletionContentPartImageURL   `json:"image_url"`
}

func (ChatCompletionContentPartImage) chatCompletionContentPartType() string { return "image_url" }

// ChatCompletionContentPartImageURL holds the URL and optional detail for an image.
type ChatCompletionContentPartImageURL struct {
	URL    string `json:"url"`
	Detail any    `json:"detail,omitempty"` // OpenAI specific extension
}

// ChatCompletionContentPartInputAudio is an audio input content part.
type ChatCompletionContentPartInputAudio struct {
	Type       string                                    `json:"type"` // "input_audio"
	InputAudio ChatCompletionContentPartInputAudioData   `json:"input_audio"`
}

func (ChatCompletionContentPartInputAudio) chatCompletionContentPartType() string {
	return "input_audio"
}

// ChatCompletionContentPartInputAudioData holds audio data and format.
type ChatCompletionContentPartInputAudioData struct {
	Data   string `json:"data"`
	Format string `json:"format"` // "wav" or "mp3"
}

// ChatCompletionContentPartFile is a file content part.
type ChatCompletionContentPartFile struct {
	Type string `json:"type"` // "file"
	File any    `json:"file"` // ChatCompletionFileByData or ChatCompletionFileByID
}

func (ChatCompletionContentPartFile) chatCompletionContentPartType() string { return "file" }

// ChatCompletionFileByData represents a file sent with inline data.
type ChatCompletionFileByData struct {
	Filename string `json:"filename"`
	FileData string `json:"file_data"`
}

// ChatCompletionFileByID represents a file referenced by ID.
type ChatCompletionFileByID struct {
	FileID string `json:"file_id"`
}

// ChatCompletionMessageToolCall represents a tool call in an assistant message.
type ChatCompletionMessageToolCall struct {
	Type     string                                `json:"type"` // "function"
	ID       string                                `json:"id"`
	Function ChatCompletionMessageToolCallFunction `json:"function"`
}

// ChatCompletionMessageToolCallFunction represents the function part of a tool call.
type ChatCompletionMessageToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
