// Ported from: packages/deepseek/src/chat/get-response-metadata.ts
package deepseek

import (
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// getResponseMetadata converts response fields into a languagemodel.ResponseMetadata.
func getResponseMetadata(id *string, model *string, created *float64) languagemodel.ResponseMetadata {
	var ts *time.Time
	if created != nil {
		t := time.Unix(int64(*created), 0)
		ts = &t
	}
	return languagemodel.ResponseMetadata{
		ID:        id,
		ModelID:   model,
		Timestamp: ts,
	}
}

// getResponseMetadataFromResponse extracts metadata from a chatCompletionResponse.
func getResponseMetadataFromResponse(resp deepSeekChatCompletionResponse) languagemodel.ResponseMetadata {
	return getResponseMetadata(resp.ID, resp.Model, resp.Created)
}

// getResponseMetadataFromMap extracts metadata from a streaming chunk map.
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
