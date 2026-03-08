// Ported from: packages/xai/src/get-response-metadata.ts
package xai

import (
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// getResponseMetadataInput holds the fields used to extract response metadata.
type getResponseMetadataInput struct {
	ID        *string
	Model     *string
	Created   *int64
	CreatedAt *int64
}

// getResponseMetadata extracts response metadata from API response fields.
func getResponseMetadata(input getResponseMetadataInput) languagemodel.ResponseMetadata {
	var unixTime *int64
	if input.Created != nil {
		unixTime = input.Created
	} else if input.CreatedAt != nil {
		unixTime = input.CreatedAt
	}

	var ts *time.Time
	if unixTime != nil {
		t := time.Unix(*unixTime, 0)
		ts = &t
	}

	return languagemodel.ResponseMetadata{
		ID:        input.ID,
		ModelID:   input.Model,
		Timestamp: ts,
	}
}
