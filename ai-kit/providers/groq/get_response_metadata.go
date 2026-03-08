// Ported from: packages/groq/src/get-response-metadata.ts
package groq

import (
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// getResponseMetadata extracts response metadata from a Groq API response.
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

// getResponseMetadataFromMap extracts response metadata from a streaming chunk map.
func getResponseMetadataFromMap(m map[string]any) languagemodel.ResponseMetadata {
	rm := languagemodel.ResponseMetadata{}
	if id, ok := m["id"].(string); ok {
		rm.ID = &id
	}
	if model, ok := m["model"].(string); ok {
		rm.ModelID = &model
	}
	if created, ok := m["created"].(float64); ok {
		t := time.Unix(int64(created), 0)
		rm.Timestamp = &t
	}
	return rm
}
