// Ported from: packages/openai-compatible/src/chat/get-response-metadata.ts
// NOTE: chat/ and completion/ versions are identical; ported once here.
package openaicompatible

import "time"

// ResponseMetadataInput holds the optional fields extracted from an API response
// that are used to construct ResponseMetadata.
type ResponseMetadataInput struct {
	ID      *string
	Model   *string
	Created *int64
}

// ResponseMetadata contains the extracted metadata from an API response.
type ResponseMetadata struct {
	ID        *string
	ModelID   *string
	Timestamp *time.Time
}

// GetResponseMetadata extracts id, modelId, and timestamp from an API response.
func GetResponseMetadata(input ResponseMetadataInput) ResponseMetadata {
	var ts *time.Time
	if input.Created != nil {
		t := time.Unix(*input.Created, 0)
		ts = &t
	}
	return ResponseMetadata{
		ID:        input.ID,
		ModelID:   input.Model,
		Timestamp: ts,
	}
}
