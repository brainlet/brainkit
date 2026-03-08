// Ported from: packages/xai/src/convert-xai-chat-usage.ts
package xai

import (
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// convertXaiChatUsage converts xAI chat usage data to the standard Usage format.
func convertXaiChatUsage(usage XaiChatUsage) languagemodel.Usage {
	cacheReadTokens := 0
	if usage.PromptTokensDetails != nil && usage.PromptTokensDetails.CachedTokens != nil {
		cacheReadTokens = *usage.PromptTokensDetails.CachedTokens
	}

	reasoningTokens := 0
	if usage.CompletionTokensDetails != nil && usage.CompletionTokensDetails.ReasoningTokens != nil {
		reasoningTokens = *usage.CompletionTokensDetails.ReasoningTokens
	}

	promptTokensIncludesCached := cacheReadTokens <= usage.PromptTokens

	var inputTotal, inputNoCache int
	if promptTokensIncludesCached {
		inputTotal = usage.PromptTokens
		inputNoCache = usage.PromptTokens - cacheReadTokens
	} else {
		inputTotal = usage.PromptTokens + cacheReadTokens
		inputNoCache = usage.PromptTokens
	}

	outputTotal := usage.CompletionTokens + reasoningTokens
	outputText := usage.CompletionTokens

	return languagemodel.Usage{
		InputTokens: languagemodel.InputTokenUsage{
			Total:    &inputTotal,
			NoCache:  &inputNoCache,
			CacheRead: &cacheReadTokens,
			CacheWrite: nil,
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total:     &outputTotal,
			Text:      &outputText,
			Reasoning: &reasoningTokens,
		},
		Raw: mapToJSONObject(usage),
	}
}

// mapToJSONObject converts a struct to a generic map for the Raw field.
func mapToJSONObject(v interface{}) map[string]interface{} {
	// Simple conversion: return nil if not needed, the raw usage is stored for debugging.
	if v == nil {
		return nil
	}
	result := make(map[string]interface{})
	switch u := v.(type) {
	case XaiChatUsage:
		result["prompt_tokens"] = u.PromptTokens
		result["completion_tokens"] = u.CompletionTokens
		result["total_tokens"] = u.TotalTokens
		if u.PromptTokensDetails != nil {
			details := make(map[string]interface{})
			if u.PromptTokensDetails.TextTokens != nil {
				details["text_tokens"] = *u.PromptTokensDetails.TextTokens
			}
			if u.PromptTokensDetails.AudioTokens != nil {
				details["audio_tokens"] = *u.PromptTokensDetails.AudioTokens
			}
			if u.PromptTokensDetails.ImageTokens != nil {
				details["image_tokens"] = *u.PromptTokensDetails.ImageTokens
			}
			if u.PromptTokensDetails.CachedTokens != nil {
				details["cached_tokens"] = *u.PromptTokensDetails.CachedTokens
			}
			result["prompt_tokens_details"] = details
		}
		if u.CompletionTokensDetails != nil {
			details := make(map[string]interface{})
			if u.CompletionTokensDetails.ReasoningTokens != nil {
				details["reasoning_tokens"] = *u.CompletionTokensDetails.ReasoningTokens
			}
			if u.CompletionTokensDetails.AudioTokens != nil {
				details["audio_tokens"] = *u.CompletionTokensDetails.AudioTokens
			}
			if u.CompletionTokensDetails.AcceptedPredictionTokens != nil {
				details["accepted_prediction_tokens"] = *u.CompletionTokensDetails.AcceptedPredictionTokens
			}
			if u.CompletionTokensDetails.RejectedPredictionTokens != nil {
				details["rejected_prediction_tokens"] = *u.CompletionTokensDetails.RejectedPredictionTokens
			}
			result["completion_tokens_details"] = details
		}
	}
	return result
}
