// Ported from: packages/anthropic/src/convert-anthropic-messages-usage.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"

// AnthropicMessagesUsageIteration represents a single iteration in the usage
// breakdown from the API response (snake_case field names).
type AnthropicMessagesUsageIteration struct {
	Type         string `json:"type"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
}

// AnthropicMessagesUsage represents the usage information from the Anthropic API response.
type AnthropicMessagesUsage struct {
	InputTokens              int                               `json:"input_tokens"`
	OutputTokens             int                               `json:"output_tokens"`
	CacheCreationInputTokens *int                              `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     *int                              `json:"cache_read_input_tokens,omitempty"`
	Iterations               []AnthropicMessagesUsageIteration `json:"iterations,omitempty"`
}

// convertAnthropicMessagesUsage converts Anthropic API usage to the unified usage format.
func convertAnthropicMessagesUsage(usage AnthropicMessagesUsage, rawUsage map[string]any) languagemodel.Usage {
	cacheCreationTokens := 0
	if usage.CacheCreationInputTokens != nil {
		cacheCreationTokens = *usage.CacheCreationInputTokens
	}
	cacheReadTokens := 0
	if usage.CacheReadInputTokens != nil {
		cacheReadTokens = *usage.CacheReadInputTokens
	}

	var inputTokens, outputTokens int

	// When iterations is present (compaction occurred), sum across all iterations
	// to get the true total tokens consumed/billed. The top-level input_tokens
	// and output_tokens exclude compaction iteration usage.
	if len(usage.Iterations) > 0 {
		for _, iter := range usage.Iterations {
			inputTokens += iter.InputTokens
			outputTokens += iter.OutputTokens
		}
	} else {
		inputTokens = usage.InputTokens
		outputTokens = usage.OutputTokens
	}

	totalInput := inputTokens + cacheCreationTokens + cacheReadTokens
	totalOutput := outputTokens

	raw := rawUsage
	if raw == nil {
		raw = map[string]any{
			"input_tokens":                inputTokens,
			"output_tokens":               outputTokens,
			"cache_creation_input_tokens": cacheCreationTokens,
			"cache_read_input_tokens":     cacheReadTokens,
		}
	}

	return languagemodel.Usage{
		InputTokens: languagemodel.InputTokenUsage{
			Total:      &totalInput,
			NoCache:    &inputTokens,
			CacheRead:  &cacheReadTokens,
			CacheWrite: &cacheCreationTokens,
		},
		OutputTokens: languagemodel.OutputTokenUsage{
			Total:     &totalOutput,
			Text:      nil,
			Reasoning: nil,
		},
		Raw: raw,
	}
}
