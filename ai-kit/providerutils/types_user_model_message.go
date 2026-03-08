// Ported from: packages/provider-utils/src/types/user-model-message.ts
package providerutils

// UserModelMessage represents a user message. It can contain text or a
// combination of text and images.
type UserModelMessage struct {
	Role            string          `json:"role"` // "user"
	Content         UserContent     `json:"content"`
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`
}

// UserContent is the content of a user message.
// It can be a plain string or an array of content parts (TextPart, ImagePart, FilePart).
// In Go we represent this as interface{} since it's a union type.
type UserContent = interface{}
