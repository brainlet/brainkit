package aiembed

import "encoding/json"

// StreamObjectParams configures a streamObject call.
type StreamObjectParams struct {
	Model    Model `json:"model"`
	CallSettings
	Prompt            string                            `json:"prompt,omitempty"`
	System            string                            `json:"system,omitempty"`
	Messages          []Message                         `json:"messages,omitempty"`
	Schema            json.RawMessage                   `json:"schema"`
	SchemaName        string                            `json:"schemaName,omitempty"`
	SchemaDescription string                            `json:"schemaDescription,omitempty"`
	Mode              string                            `json:"mode,omitempty"`
	Output            string                            `json:"output,omitempty"`
	ProviderOptions   map[string]map[string]interface{} `json:"providerOptions,omitempty"`
	Middleware        []MiddlewareConfig                `json:"-"`

	// OnPartialObject is called with each partial object as the stream progresses.
	OnPartialObject func(partial json.RawMessage) `json:"-"`
}

// StreamObjectResult is returned by StreamObject after the stream completes.
type StreamObjectResult struct {
	Object       json.RawMessage `json:"object"`
	FinishReason FinishReason    `json:"finishReason"`
	Usage        Usage           `json:"usage"`
	Response     ResponseMeta    `json:"response"`
}
