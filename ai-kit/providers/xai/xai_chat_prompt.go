// Ported from: packages/xai/src/xai-chat-prompt.ts
package xai

// XaiChatPrompt is a list of xAI chat messages.
type XaiChatPrompt = []XaiChatMessage

// XaiChatMessage is a sealed interface for xAI chat messages.
type XaiChatMessage interface {
	xaiChatMessageRole() string
}

// XaiSystemMessage is a system message.
type XaiSystemMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (XaiSystemMessage) xaiChatMessageRole() string { return "system" }

// XaiUserMessage is a user message with text or multipart content.
type XaiUserMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []XaiUserMessageContent
}

func (XaiUserMessage) xaiChatMessageRole() string { return "user" }

// XaiUserMessageContent is content within a user message.
type XaiUserMessageContent interface {
	xaiUserMessageContentType() string
}

// XaiUserTextContent is text content in a user message.
type XaiUserTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (XaiUserTextContent) xaiUserMessageContentType() string { return "text" }

// XaiUserImageURLContent is image URL content in a user message.
type XaiUserImageURLContent struct {
	Type     string            `json:"type"`
	ImageURL XaiImageURLDetail `json:"image_url"`
}

func (XaiUserImageURLContent) xaiUserMessageContentType() string { return "image_url" }

// XaiImageURLDetail contains the URL for an image.
type XaiImageURLDetail struct {
	URL string `json:"url"`
}

// XaiAssistantMessage is an assistant message.
type XaiAssistantMessage struct {
	Role      string             `json:"role"`
	Content   string             `json:"content"`
	ToolCalls []XaiToolCallEntry `json:"tool_calls,omitempty"`
}

func (XaiAssistantMessage) xaiChatMessageRole() string { return "assistant" }

// XaiToolCallEntry is a tool call entry in an assistant message.
type XaiToolCallEntry struct {
	ID       string               `json:"id"`
	Type     string               `json:"type"`
	Function XaiToolCallFunction  `json:"function"`
}

// XaiToolCallFunction contains the function details of a tool call.
type XaiToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// XaiToolMessage is a tool response message.
type XaiToolMessage struct {
	Role       string `json:"role"`
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
}

func (XaiToolMessage) xaiChatMessageRole() string { return "tool" }

// XaiToolChoice represents the tool choice for a chat completion request.
// It can be a string ("auto", "none", "required") or a specific tool selection.
type XaiToolChoice interface {
	xaiToolChoiceValue()
}

// XaiToolChoiceString is a string tool choice ("auto", "none", "required").
type XaiToolChoiceString string

func (XaiToolChoiceString) xaiToolChoiceValue() {}

// XaiToolChoiceFunction is a specific function tool choice.
type XaiToolChoiceFunction struct {
	Type     string                       `json:"type"`
	Function XaiToolChoiceFunctionDetail  `json:"function"`
}

func (XaiToolChoiceFunction) xaiToolChoiceValue() {}

// XaiToolChoiceFunctionDetail contains the function name for tool choice.
type XaiToolChoiceFunctionDetail struct {
	Name string `json:"name"`
}
