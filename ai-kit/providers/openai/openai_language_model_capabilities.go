// Ported from: packages/openai/src/openai-language-model-capabilities.ts
package openai

import "strings"

// OpenAILanguageModelCapabilities describes the capabilities of an OpenAI language model.
type OpenAILanguageModelCapabilities struct {
	IsReasoningModel bool
	SystemMessageMode string // "remove" | "system" | "developer"
	SupportsFlexProcessing bool
	SupportsPriorityProcessing bool

	// SupportsNonReasoningParameters allows temperature, topP, logProbs when reasoningEffort is none.
	SupportsNonReasoningParameters bool
}

// GetOpenAILanguageModelCapabilities returns the capabilities for a given model ID.
func GetOpenAILanguageModelCapabilities(modelID string) OpenAILanguageModelCapabilities {
	supportsFlexProcessing :=
		strings.HasPrefix(modelID, "o3") ||
			strings.HasPrefix(modelID, "o4-mini") ||
			(strings.HasPrefix(modelID, "gpt-5") && !strings.HasPrefix(modelID, "gpt-5-chat"))

	supportsPriorityProcessing :=
		strings.HasPrefix(modelID, "gpt-4") ||
			strings.HasPrefix(modelID, "gpt-5-mini") ||
			(strings.HasPrefix(modelID, "gpt-5") &&
				!strings.HasPrefix(modelID, "gpt-5-nano") &&
				!strings.HasPrefix(modelID, "gpt-5-chat")) ||
			strings.HasPrefix(modelID, "o3") ||
			strings.HasPrefix(modelID, "o4-mini")

	// Use allowlist approach: only known reasoning models should use 'developer' role
	isReasoningModel :=
		strings.HasPrefix(modelID, "o1") ||
			strings.HasPrefix(modelID, "o3") ||
			strings.HasPrefix(modelID, "o4-mini") ||
			(strings.HasPrefix(modelID, "gpt-5") && !strings.HasPrefix(modelID, "gpt-5-chat"))

	// https://platform.openai.com/docs/guides/latest-model#gpt-5-1-parameter-compatibility
	supportsNonReasoningParameters :=
		strings.HasPrefix(modelID, "gpt-5.1") ||
			strings.HasPrefix(modelID, "gpt-5.2") ||
			strings.HasPrefix(modelID, "gpt-5.4")

	systemMessageMode := "system"
	if isReasoningModel {
		systemMessageMode = "developer"
	}

	return OpenAILanguageModelCapabilities{
		SupportsFlexProcessing:         supportsFlexProcessing,
		SupportsPriorityProcessing:     supportsPriorityProcessing,
		IsReasoningModel:               isReasoningModel,
		SystemMessageMode:              systemMessageMode,
		SupportsNonReasoningParameters: supportsNonReasoningParameters,
	}
}
