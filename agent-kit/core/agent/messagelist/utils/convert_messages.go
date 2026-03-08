// Ported from: packages/core/src/agent/message-list/utils/convert-messages.ts
package utils

// OutputFormat represents the available output formats for message conversion.
// - "Mastra.V2" - Current database storage format
// - "AIV4.UI" - AI SDK v4 UIMessage format (for frontend components)
// - "AIV4.Core" - AI SDK v4 CoreMessage format (for LLM API calls)
// - "AIV5.UI" - AI SDK v5 UIMessage format (for frontend components)
// - "AIV5.Model" - AI SDK v5 ModelMessage format (for LLM API calls)
type OutputFormat string

const (
	OutputFormatMastraV2  OutputFormat = "Mastra.V2"
	OutputFormatAIV4UI    OutputFormat = "AIV4.UI"
	OutputFormatAIV4Core  OutputFormat = "AIV4.Core"
	OutputFormatAIV5UI    OutputFormat = "AIV5.UI"
	OutputFormatAIV5Model OutputFormat = "AIV5.Model"
)

// MessageConverter converts messages between supported formats.
// TODO: This depends on MessageList which creates a circular dependency.
// The full implementation should be done in the messagelist root package
// since it needs access to MessageList.add() and all the get accessors.
// See ConvertMessages in the messagelist package root for the public API.
type MessageConverter struct {
	// messageList would be a *MessageList, but we avoid the circular import here
}

// ConvertMessages converts messages from any supported format to another format.
// TODO: This is a stub. The real implementation lives in the messagelist package root
// because it requires access to the MessageList type.
// Usage: convertMessages(messages).To("AIV5.UI")
func ConvertMessages(messages any) *MessageConverter {
	return &MessageConverter{}
}
