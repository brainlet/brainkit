// Ported from: packages/openai/src/openai-language-model-capabilities.test.ts
package openai

import "testing"

func TestGetOpenAILanguageModelCapabilities_IsReasoningModel(t *testing.T) {
	tests := []struct {
		modelID  string
		expected bool
	}{
		{"gpt-4.1", false},
		{"gpt-4.1-2025-04-14", false},
		{"gpt-4.1-mini", false},
		{"gpt-4.1-mini-2025-04-14", false},
		{"gpt-4.1-nano", false},
		{"gpt-4.1-nano-2025-04-14", false},
		{"gpt-4o", false},
		{"gpt-4o-2024-05-13", false},
		{"gpt-4o-2024-08-06", false},
		{"gpt-4o-2024-11-20", false},
		{"gpt-4o-audio-preview", false},
		{"gpt-4o-audio-preview-2024-12-17", false},
		{"gpt-4o-search-preview", false},
		{"gpt-4o-search-preview-2025-03-11", false},
		{"gpt-4o-mini-search-preview", false},
		{"gpt-4o-mini-search-preview-2025-03-11", false},
		{"gpt-4o-mini", false},
		{"gpt-4o-mini-2024-07-18", false},
		{"gpt-3.5-turbo-0125", false},
		{"gpt-3.5-turbo", false},
		{"gpt-3.5-turbo-1106", false},
		{"gpt-5-chat-latest", false},
		{"o1", true},
		{"o1-2024-12-17", true},
		{"o3-mini", true},
		{"o3-mini-2025-01-31", true},
		{"o3", true},
		{"o3-2025-04-16", true},
		{"o4-mini", true},
		{"o4-mini-2025-04-16", true},
		{"gpt-5", true},
		{"gpt-5-2025-08-07", true},
		{"gpt-5-codex", true},
		{"gpt-5-mini", true},
		{"gpt-5-mini-2025-08-07", true},
		{"gpt-5-nano", true},
		{"gpt-5-nano-2025-08-07", true},
		{"gpt-5-pro", true},
		{"gpt-5-pro-2025-10-06", true},
		{"new-unknown-model", false},
		{"ft:gpt-4o-2024-08-06:org:custom:abc123", false},
		{"custom-model", false},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			caps := GetOpenAILanguageModelCapabilities(tt.modelID)
			if caps.IsReasoningModel != tt.expected {
				t.Errorf("IsReasoningModel(%q) = %v, want %v", tt.modelID, caps.IsReasoningModel, tt.expected)
			}
		})
	}
}

func TestGetOpenAILanguageModelCapabilities_SupportsNonReasoningParameters(t *testing.T) {
	tests := []struct {
		modelID  string
		expected bool
	}{
		{"gpt-5.1", true},
		{"gpt-5.1-chat-latest", true},
		{"gpt-5.1-codex-mini", true},
		{"gpt-5.1-codex", true},
		{"gpt-5.2", true},
		{"gpt-5.2-pro", true},
		{"gpt-5.2-chat-latest", true},
		{"gpt-5.4", true},
		{"gpt-5.4-pro", true},
		{"gpt-5.4-2026-03-05", true},
		{"gpt-5", false},
		{"gpt-5-mini", false},
		{"gpt-5-nano", false},
		{"gpt-5-pro", false},
		{"gpt-5-chat-latest", false},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			caps := GetOpenAILanguageModelCapabilities(tt.modelID)
			if caps.SupportsNonReasoningParameters != tt.expected {
				t.Errorf("SupportsNonReasoningParameters(%q) = %v, want %v", tt.modelID, caps.SupportsNonReasoningParameters, tt.expected)
			}
		})
	}
}
