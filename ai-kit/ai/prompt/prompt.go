// Ported from: packages/ai/src/prompt/prompt.ts
package prompt

// Prompt represents the prompt part of the AI function options.
// It contains a system message, a simple text prompt, or a list of messages.
//
// In TypeScript this uses a discriminated union with "prompt" and "messages" being
// mutually exclusive. In Go, we enforce this at runtime via StandardizePrompt.
type Prompt struct {
	// System is the system message. Can be used with Prompt or Messages.
	// It can be a string, a SystemModelMessage, or a slice of SystemModelMessage.
	System interface{} // string | SystemModelMessage | []SystemModelMessage

	// PromptStr is a text prompt string. Mutually exclusive with Messages.
	// If set and is a string, it becomes a single user message.
	// If set and is []ModelMessage, it is used directly.
	PromptValue interface{} // string | []ModelMessage

	// Messages is a list of messages. Mutually exclusive with PromptValue.
	Messages []ModelMessage
}
