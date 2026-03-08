// Ported from: packages/huggingface/src/responses/huggingface-responses-settings.ts
package huggingface

// ResponsesModelID is the type for HuggingFace model identifiers.
// In TS this is a union of string literals with a catch-all (string & {}).
// In Go we use a plain string type.
type ResponsesModelID = string

// ResponsesSettings holds provider-specific settings for HuggingFace responses.
type ResponsesSettings struct {
	Metadata        map[string]string `json:"metadata,omitempty"`
	Instructions    *string           `json:"instructions,omitempty"`
	StrictJSONSchema *bool            `json:"strictJsonSchema,omitempty"`
}
