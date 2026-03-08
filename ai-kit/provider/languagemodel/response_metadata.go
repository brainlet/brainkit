// Ported from: packages/provider/src/language-model/v3/language-model-v3-response-metadata.ts
package languagemodel

import "time"

// ResponseMetadata contains metadata about a language model response.
type ResponseMetadata struct {
	// ID is the generated response ID, if the provider sends one.
	ID *string

	// Timestamp is the start timestamp of the generated response, if the provider sends one.
	Timestamp *time.Time

	// ModelID is the ID of the response model that was used to generate the response.
	ModelID *string
}
