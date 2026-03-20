package aiembed

// Message represents a conversation message.
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// SystemMessage creates a system message.
func SystemMessage(content string) Message {
	return Message{Role: "system", Content: content}
}

// UserMessage creates a user message with text content.
func UserMessage(content string) Message {
	return Message{Role: "user", Content: content}
}

// AssistantMessage creates an assistant message with text content.
func AssistantMessage(content string) Message {
	return Message{Role: "assistant", Content: content}
}

// ToolResultMessage creates a tool result message.
func ToolResultMessage(toolCallID, toolName string, result interface{}) Message {
	return Message{
		Role: "tool",
		Content: []ToolResultPart{{
			Type:       "tool-result",
			ToolCallID: toolCallID,
			ToolName:   toolName,
			Result:     result,
		}},
	}
}

// TextPart is a text content part.
type TextPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ImagePart is an image content part.
type ImagePart struct {
	Type     string `json:"type"`
	Image    string `json:"image"`
	MimeType string `json:"mimeType,omitempty"`
}

// FilePart is a file content part.
type FilePart struct {
	Type     string `json:"type"`
	Data     string `json:"data"`
	MimeType string `json:"mimeType"`
	Filename string `json:"filename,omitempty"`
}

// ToolCallPart represents a tool call in an assistant message.
type ToolCallPart struct {
	Type       string      `json:"type"`
	ToolCallID string      `json:"toolCallId"`
	ToolName   string      `json:"toolName"`
	Args       interface{} `json:"args"`
}

// ToolResultPart represents a tool result in a tool message.
type ToolResultPart struct {
	Type       string      `json:"type"`
	ToolCallID string      `json:"toolCallId"`
	ToolName   string      `json:"toolName"`
	Result     interface{} `json:"result"`
	IsError    bool        `json:"isError,omitempty"`
}
