// Ported from: packages/core/src/processors/processors/processors-integration.test.ts
package concreteprocessors

import (
	"testing"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

func TestProcessorsIntegration(t *testing.T) {
	mockAbort := func(reason string, opts *processors.TripWireOptions) error {
		return nil
	}

	t.Run("should chain multiple processors in order (ToolCallFilter + TokenLimiter)", func(t *testing.T) {
		// Create messages with tool calls and text content.
		messages := []processors.MastraDBMessage{
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-1",
					Role: "user",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "What is the weather in NYC?",
					Parts:   []processors.MastraMessagePart{},
				},
			},
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-2",
					Role: "assistant",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "The weather in NYC is sunny and 72\u00b0F. It is a beautiful day outside with clear skies.",
					Parts: []processors.MastraMessagePart{
						{
							Type: "tool-invocation",
							ToolInvocation: &processors.ToolInvocation{
								State:      "call",
								ToolCallID: "call-1",
								ToolName:   "weather",
								Args:       map[string]any{"location": "NYC"},
							},
						},
						{
							Type: "tool-invocation",
							ToolInvocation: &processors.ToolInvocation{
								State:      "result",
								ToolCallID: "call-1",
								ToolName:   "weather",
								Result:     "Sunny, 72\u00b0F",
							},
						},
					},
				},
			},
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-5",
					Role: "user",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "What about San Francisco?",
					Parts:   []processors.MastraMessagePart{},
				},
			},
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-6",
					Role: "assistant",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "San Francisco is foggy with a temperature of 58\u00b0F.",
					Parts: []processors.MastraMessagePart{
						{
							Type: "tool-invocation",
							ToolInvocation: &processors.ToolInvocation{
								State:      "call",
								ToolCallID: "call-2",
								ToolName:   "time",
								Args:       map[string]any{"location": "SF"},
							},
						},
						{
							Type: "tool-invocation",
							ToolInvocation: &processors.ToolInvocation{
								State:      "result",
								ToolCallID: "call-2",
								ToolName:   "time",
								Result:     "3:45 PM",
							},
						},
					},
				},
			},
		}

		// Step 1: Apply ToolCallFilter to exclude weather tool calls.
		toolCallFilter := NewToolCallFilter(&ToolCallFilterOptions{Exclude: []string{"weather"}})

		filteredMessages, _, _, err := toolCallFilter.ProcessInput(processors.ProcessInputArgs{
			ProcessorMessageContext: processors.ProcessorMessageContext{
				Messages: messages,
				ProcessorContext: processors.ProcessorContext{
					Abort: mockAbort,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error from ToolCallFilter: %v", err)
		}

		// Verify ToolCallFilter removed weather tool call parts.
		// msg-2 has only weather tool invocation parts, so it should be removed entirely.
		// msg-6 has time tool invocation parts, which should be preserved.
		foundMsg2 := false
		foundMsg6 := false
		for _, msg := range filteredMessages {
			if msg.ID == "msg-2" {
				foundMsg2 = true
			}
			if msg.ID == "msg-6" {
				foundMsg6 = true
			}
		}

		if foundMsg2 {
			t.Fatal("expected msg-2 (weather tool) to be removed")
		}
		if !foundMsg6 {
			t.Fatal("expected msg-6 (time tool) to be preserved")
		}

		// Step 2: Apply TokenLimiter to limit message count.
		tokenLimiter := NewTokenLimiterProcessor(50, nil)

		limitedMessages, _, _, err := tokenLimiter.ProcessInput(processors.ProcessInputArgs{
			ProcessorMessageContext: processors.ProcessorMessageContext{
				Messages: filteredMessages,
				ProcessorContext: processors.ProcessorContext{
					Abort: mockAbort,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error from TokenLimiter: %v", err)
		}

		// Verify TokenLimiter further reduced messages.
		if len(limitedMessages) > len(filteredMessages) {
			t.Fatalf("expected limited messages (%d) <= filtered messages (%d)", len(limitedMessages), len(filteredMessages))
		}
		if len(limitedMessages) == 0 {
			t.Fatal("expected at least some messages after token limiting")
		}

		// Verify no message duplication.
		idSet := make(map[string]bool)
		for _, msg := range limitedMessages {
			if idSet[msg.ID] {
				t.Fatalf("found duplicate message id '%s'", msg.ID)
			}
			idSet[msg.ID] = true
		}

		// Verify final messages are a subset of filtered messages.
		filteredIDs := make(map[string]bool)
		for _, msg := range filteredMessages {
			filteredIDs[msg.ID] = true
		}
		for _, msg := range limitedMessages {
			if !filteredIDs[msg.ID] {
				t.Fatalf("limited message '%s' not found in filtered messages", msg.ID)
			}
		}
	})

	t.Run("should apply multiple processors without duplicating messages", func(t *testing.T) {
		messages := []processors.MastraDBMessage{
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-1",
					Role: "user",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "Hello",
					Parts:   []processors.MastraMessagePart{},
				},
			},
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-2",
					Role: "assistant",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "Weather is sunny",
					Parts: []processors.MastraMessagePart{
						{
							Type: "tool-invocation",
							ToolInvocation: &processors.ToolInvocation{
								State:      "call",
								ToolCallID: "tc-1",
								ToolName:   "weather",
								Args:       map[string]any{"location": "NYC"},
							},
						},
						{
							Type: "tool-invocation",
							ToolInvocation: &processors.ToolInvocation{
								State:      "result",
								ToolCallID: "tc-1",
								ToolName:   "weather",
								Result:     "Sunny",
							},
						},
					},
				},
			},
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-3",
					Role: "user",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "What time is it?",
					Parts:   []processors.MastraMessagePart{},
				},
			},
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-4",
					Role: "assistant",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "It is 3:45 PM",
					Parts: []processors.MastraMessagePart{
						{
							Type: "tool-invocation",
							ToolInvocation: &processors.ToolInvocation{
								State:      "call",
								ToolCallID: "tc-2",
								ToolName:   "time",
							},
						},
						{
							Type: "tool-invocation",
							ToolInvocation: &processors.ToolInvocation{
								State:      "result",
								ToolCallID: "tc-2",
								ToolName:   "time",
								Result:     "3:45 PM",
							},
						},
					},
				},
			},
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-5",
					Role: "user",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "Thanks",
					Parts:   []processors.MastraMessagePart{},
				},
			},
		}

		// Apply ToolCallFilter (exclude 'weather').
		toolCallFilter := NewToolCallFilter(&ToolCallFilterOptions{Exclude: []string{"weather"}})
		filteredMessages, _, _, err := toolCallFilter.ProcessInput(processors.ProcessInputArgs{
			ProcessorMessageContext: processors.ProcessorMessageContext{
				Messages: messages,
				ProcessorContext: processors.ProcessorContext{
					Abort: mockAbort,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error from ToolCallFilter: %v", err)
		}

		// Apply TokenLimiter.
		tokenLimiter := NewTokenLimiterProcessor(100, nil)
		limitedMessages, _, _, err := tokenLimiter.ProcessInput(processors.ProcessInputArgs{
			ProcessorMessageContext: processors.ProcessorMessageContext{
				Messages: filteredMessages,
				ProcessorContext: processors.ProcessorContext{
					Abort: mockAbort,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error from TokenLimiter: %v", err)
		}

		// Verify no duplicates by checking unique IDs.
		idSet := make(map[string]bool)
		for _, msg := range limitedMessages {
			if idSet[msg.ID] {
				t.Fatalf("found duplicate message id '%s'", msg.ID)
			}
			idSet[msg.ID] = true
		}

		// Verify final messages are subset of filtered messages.
		filteredIDs := make(map[string]bool)
		for _, msg := range filteredMessages {
			filteredIDs[msg.ID] = true
		}
		for _, msg := range limitedMessages {
			if !filteredIDs[msg.ID] {
				t.Fatalf("limited message '%s' not found in filtered messages", msg.ID)
			}
		}
	})

	t.Run("should integrate processors with ProcessorRunner", func(t *testing.T) {
		messages := []processors.MastraDBMessage{
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-1",
					Role: "user",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "What is the weather in Seattle?",
					Parts:   []processors.MastraMessagePart{},
				},
			},
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-2",
					Role: "assistant",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "The weather in Seattle is sunny and 70 degrees.",
					Parts: []processors.MastraMessagePart{
						{
							Type: "tool-invocation",
							ToolInvocation: &processors.ToolInvocation{
								State:      "call",
								ToolCallID: "call-weather-1",
								ToolName:   "get_weather",
								Args:       map[string]any{"location": "Seattle"},
							},
						},
						{
							Type: "tool-invocation",
							ToolInvocation: &processors.ToolInvocation{
								State:      "result",
								ToolCallID: "call-weather-1",
								ToolName:   "get_weather",
								Result:     "Sunny, 70\u00b0F",
							},
						},
					},
				},
			},
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-3",
					Role: "user",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "Calculate 123 * 456",
					Parts:   []processors.MastraMessagePart{},
				},
			},
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-4",
					Role: "assistant",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "The result of 123 * 456 is 56088.",
					Parts: []processors.MastraMessagePart{
						{
							Type: "tool-invocation",
							ToolInvocation: &processors.ToolInvocation{
								State:      "call",
								ToolCallID: "call-calc-1",
								ToolName:   "calculator",
								Args:       map[string]any{"expression": "123 * 456"},
							},
						},
						{
							Type: "tool-invocation",
							ToolInvocation: &processors.ToolInvocation{
								State:      "result",
								ToolCallID: "call-calc-1",
								ToolName:   "calculator",
								Result:     "56088",
							},
						},
					},
				},
			},
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-5",
					Role: "user",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "Tell me something interesting about space",
					Parts:   []processors.MastraMessagePart{},
				},
			},
			{
				MastraMessageShared: processors.MastraMessageShared{
					ID:   "msg-6",
					Role: "assistant",
				},
				Content: processors.MastraMessageContentV2{
					Format:  2,
					Content: "Space is vast and contains billions of galaxies.",
					Parts:   []processors.MastraMessagePart{},
				},
			},
		}

		// Test 1: Filter weather tool calls.
		weatherFilter := NewToolCallFilter(&ToolCallFilterOptions{Exclude: []string{"get_weather"}})
		weatherFiltered, _, _, err := weatherFilter.ProcessInput(processors.ProcessInputArgs{
			ProcessorMessageContext: processors.ProcessorMessageContext{
				Messages: messages,
				ProcessorContext: processors.ProcessorContext{
					Abort: mockAbort,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have fewer messages (msg-2 with weather tool removed).
		if len(weatherFiltered) != 5 {
			t.Fatalf("expected 5 messages after weather filter, got %d", len(weatherFiltered))
		}

		foundMsg2 := false
		foundMsg4 := false
		for _, msg := range weatherFiltered {
			if msg.ID == "msg-2" {
				foundMsg2 = true
			}
			if msg.ID == "msg-4" {
				foundMsg4 = true
			}
		}
		if foundMsg2 {
			t.Fatal("expected msg-2 (weather tool) to be removed")
		}
		if !foundMsg4 {
			t.Fatal("expected msg-4 (calculator) to be preserved")
		}

		// Test 2: Apply token limiting with a low limit.
		tokenLimiter := NewTokenLimiterProcessor(50, nil)
		tokenLimited, _, _, err := tokenLimiter.ProcessInput(processors.ProcessInputArgs{
			ProcessorMessageContext: processors.ProcessorMessageContext{
				Messages: messages,
				ProcessorContext: processors.ProcessorContext{
					Abort: mockAbort,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(tokenLimited) >= len(messages) {
			t.Fatalf("expected fewer messages due to token limit, got %d", len(tokenLimited))
		}
		if len(tokenLimited) == 0 {
			t.Fatal("expected at least some messages after token limiting")
		}

		// Test 3: Combine both processors.
		combinedFilter := NewToolCallFilter(&ToolCallFilterOptions{Exclude: []string{"get_weather", "calculator"}})
		combinedFiltered, _, _, err := combinedFilter.ProcessInput(processors.ProcessInputArgs{
			ProcessorMessageContext: processors.ProcessorMessageContext{
				Messages: messages,
				ProcessorContext: processors.ProcessorContext{
					Abort: mockAbort,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Then apply token limiter.
		finalResult, _, _, err := tokenLimiter.ProcessInput(processors.ProcessInputArgs{
			ProcessorMessageContext: processors.ProcessorMessageContext{
				Messages: combinedFiltered,
				ProcessorContext: processors.ProcessorContext{
					Abort: mockAbort,
				},
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have no tool call messages.
		for _, msg := range combinedFiltered {
			if msg.ID == "msg-2" || msg.ID == "msg-4" {
				t.Fatalf("expected msg '%s' to be filtered out", msg.ID)
			}
		}

		// But should still have user messages and simple assistant response.
		foundMsg1 := false
		foundMsg6 := false
		for _, msg := range combinedFiltered {
			if msg.ID == "msg-1" {
				foundMsg1 = true
			}
			if msg.ID == "msg-6" {
				foundMsg6 = true
			}
		}
		if !foundMsg1 {
			t.Fatal("expected msg-1 to be preserved")
		}
		if !foundMsg6 {
			t.Fatal("expected msg-6 to be preserved")
		}

		// Final result should be further limited by tokens.
		if len(finalResult) == 0 {
			t.Fatal("expected at least some messages in final result")
		}
		if len(finalResult) > len(combinedFiltered) {
			t.Fatalf("expected final result (%d) <= combined filtered (%d)", len(finalResult), len(combinedFiltered))
		}
	})
}
