// Ported from: packages/anthropic/src/convert-anthropic-messages-usage.test.ts
package anthropic

import (
	"testing"
)

func intPtr(v int) *int {
	return &v
}

func TestConvertAnthropicMessagesUsage(t *testing.T) {
	t.Run("should use usage as raw when rawUsage is not provided", func(t *testing.T) {
		usage := AnthropicMessagesUsage{
			InputTokens:  10,
			OutputTokens: 20,
		}

		result := convertAnthropicMessagesUsage(usage, nil)

		if result.Raw == nil {
			t.Fatal("expected raw usage to be non-nil")
		}
	})

	t.Run("should use rawUsage as raw when provided", func(t *testing.T) {
		usage := AnthropicMessagesUsage{
			InputTokens:  10,
			OutputTokens: 20,
		}
		rawUsage := map[string]any{
			"input_tokens":  10,
			"output_tokens": 20,
			"service_tier":  "standard",
			"inference_geo": "not_available",
			"cache_creation": map[string]any{
				"ephemeral_5m_input_tokens": 0,
				"ephemeral_1h_input_tokens": 0,
			},
		}

		result := convertAnthropicMessagesUsage(usage, rawUsage)

		// The raw should be the rawUsage we passed
		if result.Raw["service_tier"] != "standard" {
			t.Errorf("expected raw to contain service_tier='standard', got %v", result.Raw["service_tier"])
		}
	})

	t.Run("should compute token totals correctly with cache tokens", func(t *testing.T) {
		cacheCreation := 5
		cacheRead := 3
		usage := AnthropicMessagesUsage{
			InputTokens:              10,
			OutputTokens:             20,
			CacheCreationInputTokens: &cacheCreation,
			CacheReadInputTokens:     &cacheRead,
		}

		result := convertAnthropicMessagesUsage(usage, nil)

		if result.InputTokens.Total == nil || *result.InputTokens.Total != 18 {
			t.Errorf("expected input total 18, got %v", result.InputTokens.Total)
		}
		if result.InputTokens.NoCache == nil || *result.InputTokens.NoCache != 10 {
			t.Errorf("expected input noCache 10, got %v", result.InputTokens.NoCache)
		}
		if result.InputTokens.CacheRead == nil || *result.InputTokens.CacheRead != 3 {
			t.Errorf("expected input cacheRead 3, got %v", result.InputTokens.CacheRead)
		}
		if result.InputTokens.CacheWrite == nil || *result.InputTokens.CacheWrite != 5 {
			t.Errorf("expected input cacheWrite 5, got %v", result.InputTokens.CacheWrite)
		}
		if result.OutputTokens.Total == nil || *result.OutputTokens.Total != 20 {
			t.Errorf("expected output total 20, got %v", result.OutputTokens.Total)
		}
		if result.OutputTokens.Text != nil {
			t.Errorf("expected output text to be nil, got %v", result.OutputTokens.Text)
		}
		if result.OutputTokens.Reasoning != nil {
			t.Errorf("expected output reasoning to be nil, got %v", result.OutputTokens.Reasoning)
		}
	})

	t.Run("should handle nil cache tokens", func(t *testing.T) {
		usage := AnthropicMessagesUsage{
			InputTokens:              100,
			OutputTokens:             50,
			CacheCreationInputTokens: nil,
			CacheReadInputTokens:     nil,
		}

		result := convertAnthropicMessagesUsage(usage, nil)

		if *result.InputTokens.Total != 100 {
			t.Errorf("expected input total 100, got %d", *result.InputTokens.Total)
		}
		if *result.InputTokens.CacheRead != 0 {
			t.Errorf("expected input cacheRead 0, got %d", *result.InputTokens.CacheRead)
		}
		if *result.InputTokens.CacheWrite != 0 {
			t.Errorf("expected input cacheWrite 0, got %d", *result.InputTokens.CacheWrite)
		}
	})

	t.Run("compaction usage with iterations", func(t *testing.T) {
		t.Run("should sum across all iterations when iterations array is present", func(t *testing.T) {
			usage := AnthropicMessagesUsage{
				InputTokens:  45000,
				OutputTokens: 1234,
				Iterations: []AnthropicMessagesUsageIteration{
					{Type: "compaction", InputTokens: 180000, OutputTokens: 3500},
					{Type: "message", InputTokens: 23000, OutputTokens: 1000},
				},
			}

			result := convertAnthropicMessagesUsage(usage, nil)

			if *result.InputTokens.Total != 203000 {
				t.Errorf("expected input total 203000, got %d", *result.InputTokens.Total)
			}
			if *result.InputTokens.NoCache != 203000 {
				t.Errorf("expected input noCache 203000, got %d", *result.InputTokens.NoCache)
			}
			if *result.OutputTokens.Total != 4500 {
				t.Errorf("expected output total 4500, got %d", *result.OutputTokens.Total)
			}
		})

		t.Run("should handle single iteration (message only, no compaction triggered)", func(t *testing.T) {
			usage := AnthropicMessagesUsage{
				InputTokens:  5000,
				OutputTokens: 500,
				Iterations: []AnthropicMessagesUsageIteration{
					{Type: "message", InputTokens: 5000, OutputTokens: 500},
				},
			}

			result := convertAnthropicMessagesUsage(usage, nil)

			if *result.InputTokens.Total != 5000 {
				t.Errorf("expected input total 5000, got %d", *result.InputTokens.Total)
			}
			if *result.OutputTokens.Total != 500 {
				t.Errorf("expected output total 500, got %d", *result.OutputTokens.Total)
			}
		})

		t.Run("should handle multiple compaction iterations (long-running task)", func(t *testing.T) {
			usage := AnthropicMessagesUsage{
				InputTokens:  10000,
				OutputTokens: 500,
				Iterations: []AnthropicMessagesUsageIteration{
					{Type: "compaction", InputTokens: 200000, OutputTokens: 4000},
					{Type: "message", InputTokens: 50000, OutputTokens: 2000},
					{Type: "compaction", InputTokens: 180000, OutputTokens: 3500},
					{Type: "message", InputTokens: 30000, OutputTokens: 1500},
				},
			}

			result := convertAnthropicMessagesUsage(usage, nil)

			if *result.InputTokens.Total != 460000 {
				t.Errorf("expected input total 460000, got %d", *result.InputTokens.Total)
			}
			if *result.OutputTokens.Total != 11000 {
				t.Errorf("expected output total 11000, got %d", *result.OutputTokens.Total)
			}
		})

		t.Run("should combine iterations with cache tokens", func(t *testing.T) {
			cacheCreation := 1000
			cacheRead := 500
			usage := AnthropicMessagesUsage{
				InputTokens:              45000,
				OutputTokens:             1234,
				CacheCreationInputTokens: &cacheCreation,
				CacheReadInputTokens:     &cacheRead,
				Iterations: []AnthropicMessagesUsageIteration{
					{Type: "compaction", InputTokens: 180000, OutputTokens: 3500},
					{Type: "message", InputTokens: 23000, OutputTokens: 1000},
				},
			}

			result := convertAnthropicMessagesUsage(usage, nil)

			if *result.InputTokens.NoCache != 203000 {
				t.Errorf("expected input noCache 203000, got %d", *result.InputTokens.NoCache)
			}
			if *result.InputTokens.CacheWrite != 1000 {
				t.Errorf("expected input cacheWrite 1000, got %d", *result.InputTokens.CacheWrite)
			}
			if *result.InputTokens.CacheRead != 500 {
				t.Errorf("expected input cacheRead 500, got %d", *result.InputTokens.CacheRead)
			}
			if *result.InputTokens.Total != 204500 {
				t.Errorf("expected input total 204500, got %d", *result.InputTokens.Total)
			}
			if *result.OutputTokens.Total != 4500 {
				t.Errorf("expected output total 4500, got %d", *result.OutputTokens.Total)
			}
		})

		t.Run("should use rawUsage as raw even when iterations are present", func(t *testing.T) {
			usage := AnthropicMessagesUsage{
				InputTokens:  45000,
				OutputTokens: 1234,
				Iterations: []AnthropicMessagesUsageIteration{
					{Type: "compaction", InputTokens: 180000, OutputTokens: 3500},
					{Type: "message", InputTokens: 23000, OutputTokens: 1000},
				},
			}
			rawUsage := map[string]any{
				"input_tokens":  45000,
				"output_tokens": 1234,
				"service_tier":  "standard",
			}

			result := convertAnthropicMessagesUsage(usage, rawUsage)

			if result.Raw["service_tier"] != "standard" {
				t.Errorf("expected raw to contain service_tier='standard', got %v", result.Raw["service_tier"])
			}
			if *result.InputTokens.Total != 203000 {
				t.Errorf("expected input total 203000, got %d", *result.InputTokens.Total)
			}
		})
	})

	t.Run("edge cases", func(t *testing.T) {
		t.Run("should use top-level values when iterations is nil", func(t *testing.T) {
			usage := AnthropicMessagesUsage{
				InputTokens:  100,
				OutputTokens: 50,
				Iterations:   nil,
			}

			result := convertAnthropicMessagesUsage(usage, nil)

			if *result.InputTokens.Total != 100 {
				t.Errorf("expected input total 100, got %d", *result.InputTokens.Total)
			}
			if *result.OutputTokens.Total != 50 {
				t.Errorf("expected output total 50, got %d", *result.OutputTokens.Total)
			}
		})

		t.Run("should use top-level values when iterations array is empty", func(t *testing.T) {
			usage := AnthropicMessagesUsage{
				InputTokens:  100,
				OutputTokens: 50,
				Iterations:   []AnthropicMessagesUsageIteration{},
			}

			result := convertAnthropicMessagesUsage(usage, nil)

			if *result.InputTokens.Total != 100 {
				t.Errorf("expected input total 100, got %d", *result.InputTokens.Total)
			}
			if *result.OutputTokens.Total != 50 {
				t.Errorf("expected output total 50, got %d", *result.OutputTokens.Total)
			}
		})

		t.Run("should handle zero tokens in iterations", func(t *testing.T) {
			usage := AnthropicMessagesUsage{
				InputTokens:  0,
				OutputTokens: 0,
				Iterations: []AnthropicMessagesUsageIteration{
					{Type: "compaction", InputTokens: 0, OutputTokens: 0},
					{Type: "message", InputTokens: 0, OutputTokens: 0},
				},
			}

			result := convertAnthropicMessagesUsage(usage, nil)

			if *result.InputTokens.Total != 0 {
				t.Errorf("expected input total 0, got %d", *result.InputTokens.Total)
			}
			if *result.OutputTokens.Total != 0 {
				t.Errorf("expected output total 0, got %d", *result.OutputTokens.Total)
			}
		})
	})

	t.Run("real-world scenarios from documentation", func(t *testing.T) {
		t.Run("should match documentation example exactly", func(t *testing.T) {
			usage := AnthropicMessagesUsage{
				InputTokens:  45000,
				OutputTokens: 1234,
				Iterations: []AnthropicMessagesUsageIteration{
					{Type: "compaction", InputTokens: 180000, OutputTokens: 3500},
					{Type: "message", InputTokens: 23000, OutputTokens: 1000},
				},
			}

			result := convertAnthropicMessagesUsage(usage, nil)

			expectedTotalInput := 180000 + 23000  // 203000
			expectedTotalOutput := 3500 + 1000     // 4500

			if *result.InputTokens.Total != expectedTotalInput {
				t.Errorf("expected input total %d, got %d", expectedTotalInput, *result.InputTokens.Total)
			}
			if *result.OutputTokens.Total != expectedTotalOutput {
				t.Errorf("expected output total %d, got %d", expectedTotalOutput, *result.OutputTokens.Total)
			}

			// The top-level values (45000, 1234) are NOT the billed amounts when iterations is present
			if *result.InputTokens.Total == usage.InputTokens {
				t.Error("expected input total to differ from top-level input_tokens when iterations is present")
			}
			if *result.OutputTokens.Total == usage.OutputTokens {
				t.Error("expected output total to differ from top-level output_tokens when iterations is present")
			}
		})

		t.Run("should handle re-applying previous compaction block (no new compaction)", func(t *testing.T) {
			usage := AnthropicMessagesUsage{
				InputTokens:  15000,
				OutputTokens: 800,
			}

			result := convertAnthropicMessagesUsage(usage, nil)

			if *result.InputTokens.Total != 15000 {
				t.Errorf("expected input total 15000, got %d", *result.InputTokens.Total)
			}
			if *result.OutputTokens.Total != 800 {
				t.Errorf("expected output total 800, got %d", *result.OutputTokens.Total)
			}
		})
	})
}
