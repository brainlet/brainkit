// Ported from: packages/perplexity/src/map-perplexity-finish-reason.ts
package perplexity

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// MapPerplexityFinishReason maps a Perplexity finish reason string
// to a unified finish reason.
func MapPerplexityFinishReason(finishReason *string) languagemodel.UnifiedFinishReason {
	if finishReason == nil {
		return languagemodel.FinishReasonOther
	}
	switch *finishReason {
	case "stop":
		return languagemodel.FinishReasonStop
	case "length":
		return languagemodel.FinishReasonLength
	default:
		return languagemodel.FinishReasonOther
	}
}
