// Ported from: packages/openai/src/chat/get-response-metadata.ts
package openai

import (
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// getChatResponseMetadata extracts response metadata from an OpenAI chat response.
func getChatResponseMetadata(id *string, model *string, created *float64) languagemodel.ResponseMetadata {
	meta := languagemodel.ResponseMetadata{}

	if id != nil {
		meta.ID = id
	}
	if model != nil {
		meta.ModelID = model
	}
	if created != nil {
		t := time.Unix(int64(*created), 0)
		meta.Timestamp = &t
	}

	return meta
}
