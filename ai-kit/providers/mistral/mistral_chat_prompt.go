// Ported from: packages/mistral/src/mistral-chat-prompt.ts
package mistral

// MistralPrompt is a list of Mistral messages.
type MistralPrompt = []MistralMessage

// MistralMessage is a sealed interface representing a Mistral chat message.
type MistralMessage interface {
	mistralMessageRole() string
}

// MistralSystemMessage is a system message.
type MistralSystemMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func (MistralSystemMessage) mistralMessageRole() string { return "system" }

// MistralUserMessage is a user message with multi-part content.
type MistralUserMessage struct {
	Role    string                     `json:"role"`
	Content []MistralUserMessageContent `json:"content"`
}

func (MistralUserMessage) mistralMessageRole() string { return "user" }

// MistralUserMessageContent is a sealed interface for user message content parts.
type MistralUserMessageContent interface {
	mistralUserMessageContentType() string
}

// MistralUserContentText is a text content part.
type MistralUserContentText struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (MistralUserContentText) mistralUserMessageContentType() string { return "text" }

// MistralUserContentImageURL is an image URL content part.
type MistralUserContentImageURL struct {
	Type     string `json:"type"`
	ImageURL string `json:"image_url"`
}

func (MistralUserContentImageURL) mistralUserMessageContentType() string { return "image_url" }

// MistralUserContentDocumentURL is a document URL content part.
type MistralUserContentDocumentURL struct {
	Type        string `json:"type"`
	DocumentURL string `json:"document_url"`
}

func (MistralUserContentDocumentURL) mistralUserMessageContentType() string { return "document_url" }

// MistralAssistantMessage is an assistant message.
type MistralAssistantMessage struct {
	Role      string                      `json:"role"`
	Content   string                      `json:"content"`
	Prefix    *bool                       `json:"prefix,omitempty"`
	ToolCalls []MistralAssistantToolCall  `json:"tool_calls,omitempty"`
}

func (MistralAssistantMessage) mistralMessageRole() string { return "assistant" }

// MistralAssistantToolCall represents a tool call in an assistant message.
type MistralAssistantToolCall struct {
	ID       string                            `json:"id"`
	Type     string                            `json:"type"`
	Function MistralAssistantToolCallFunction  `json:"function"`
}

// MistralAssistantToolCallFunction represents the function part of a tool call.
type MistralAssistantToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// MistralToolMessage is a tool result message.
type MistralToolMessage struct {
	Role       string `json:"role"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	ToolCallID string `json:"tool_call_id"`
}

func (MistralToolMessage) mistralMessageRole() string { return "tool" }

// MistralToolChoice represents a Mistral tool choice configuration.
// It can be a string ("auto", "none", "any") or a structured tool choice.
type MistralToolChoice interface {
	isMistralToolChoice()
}

// MistralToolChoiceString is a string tool choice: "auto", "none", or "any".
type MistralToolChoiceString string

func (MistralToolChoiceString) isMistralToolChoice() {}

const (
	MistralToolChoiceAuto MistralToolChoiceString = "auto"
	MistralToolChoiceNone MistralToolChoiceString = "none"
	MistralToolChoiceAny  MistralToolChoiceString = "any"
)

// MistralToolChoiceFunction is a structured tool choice specifying a specific function.
type MistralToolChoiceFunction struct {
	Type     string                              `json:"type"`
	Function MistralToolChoiceFunctionSpec       `json:"function"`
}

func (MistralToolChoiceFunction) isMistralToolChoice() {}

// MistralToolChoiceFunctionSpec specifies the function name in a tool choice.
type MistralToolChoiceFunctionSpec struct {
	Name string `json:"name"`
}
