// Ported from: packages/provider-utils/src/types/model-message.ts
package providerutils

// ModelMessage is a union type for messages that can be used in the messages field of a prompt.
// It can be a SystemModelMessage, UserModelMessage, AssistantModelMessage, or ToolModelMessage.
// In Go we represent this as interface{} since it's a union type.
type ModelMessage = interface{}
