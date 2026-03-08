// Ported from: packages/google/src/map-google-generative-ai-finish-reason.ts
package google

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// MapGoogleFinishReasonOptions are the options for MapGoogleFinishReason.
type MapGoogleFinishReasonOptions struct {
	FinishReason *string
	HasToolCalls bool
}

// MapGoogleFinishReason maps a Google Generative AI finish reason string to a
// unified LanguageModel finish reason.
func MapGoogleFinishReason(opts MapGoogleFinishReasonOptions) languagemodel.UnifiedFinishReason {
	if opts.FinishReason == nil {
		return languagemodel.FinishReasonOther
	}

	switch *opts.FinishReason {
	case "STOP":
		if opts.HasToolCalls {
			return languagemodel.FinishReasonToolCalls
		}
		return languagemodel.FinishReasonStop
	case "MAX_TOKENS":
		return languagemodel.FinishReasonLength
	case "IMAGE_SAFETY", "RECITATION", "SAFETY", "BLOCKLIST", "PROHIBITED_CONTENT", "SPII":
		return languagemodel.FinishReasonContentFilter
	case "MALFORMED_FUNCTION_CALL":
		return languagemodel.FinishReasonError
	case "FINISH_REASON_UNSPECIFIED", "OTHER":
		return languagemodel.FinishReasonOther
	default:
		return languagemodel.FinishReasonOther
	}
}
