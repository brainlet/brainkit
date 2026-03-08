// Ported from: packages/mistral/src/get-response-metadata.ts
package mistral

import (
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// getResponseMetadata extracts response metadata from a Mistral API response.
func getResponseMetadata(id *string, model *string, created *float64) languagemodel.ResponseMetadata {
	rm := languagemodel.ResponseMetadata{
		ID:      id,
		ModelID: model,
	}

	if created != nil {
		t := time.Unix(int64(*created), 0)
		rm.Timestamp = &t
	}

	return rm
}
