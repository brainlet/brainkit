package aiembed

import "encoding/json"

// GenerateObjectParams configures a generateObject call.
type GenerateObjectParams struct {
	Model    Model `json:"model"`
	CallSettings
	Prompt            string                             `json:"prompt,omitempty"`
	System            string                             `json:"system,omitempty"`
	Messages          []Message                          `json:"messages,omitempty"`
	Schema            json.RawMessage                    `json:"schema"`
	SchemaName        string                             `json:"schemaName,omitempty"`
	SchemaDescription string                             `json:"schemaDescription,omitempty"`
	Mode              string                             `json:"mode,omitempty"`
	Output            string                             `json:"output,omitempty"`
	Enum              []string                           `json:"enum,omitempty"`
	ProviderOptions   map[string]map[string]interface{}  `json:"providerOptions,omitempty"`
	Middleware        []MiddlewareConfig                `json:"-"`
}

// GenerateObjectResult is returned by GenerateObject.
type GenerateObjectResult struct {
	Object       json.RawMessage `json:"object"`
	FinishReason FinishReason    `json:"finishReason"`
	Usage        Usage           `json:"usage"`
	Response     ResponseMeta    `json:"response"`
}
