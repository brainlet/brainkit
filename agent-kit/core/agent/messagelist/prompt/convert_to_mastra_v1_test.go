// Ported from: packages/core/src/agent/message-list/prompt/convert-to-mastra-v1-reasoning.test.ts
// Ported from: packages/core/src/agent/message-list/prompt/convert-to-mastra-v1.test.ts (subset)
package prompt

import (
	"testing"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/agent/messagelist/state"
)

func TestConvertToV1Messages_ReasoningAndFileSupport(t *testing.T) {
	now := time.Now()

	t.Run("should handle file parts in messages", func(t *testing.T) {
		messages := []state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{
					ID:         "msg-with-file",
					Role:       "assistant",
					CreatedAt:  now,
					ThreadID:   "thread-1",
					ResourceID: "resource-1",
				},
				Content: state.MastraMessageContentV2{
					Format: 2,
					Parts: []state.MastraMessagePart{
						{Type: "text", Text: "Here is the image you requested:"},
						{Type: "file", Data: "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==", MimeType: "image/png"},
						{Type: "text", Text: "This is a 1x1 pixel image."},
					},
				},
			},
		}

		result := ConvertToV1Messages(messages)

		// File parts are extracted as attachments in the V1 converter, so the remaining
		// parts are text + text. The file becomes an attachment.
		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}
		if result[0].Role != "assistant" {
			t.Errorf("expected role assistant, got %s", result[0].Role)
		}
		if result[0].Type != "text" {
			t.Errorf("expected type text, got %s", result[0].Type)
		}

		contentArr, ok := result[0].Content.([]map[string]any)
		if !ok {
			t.Fatalf("expected content to be []map[string]any, got %T", result[0].Content)
		}
		// The convert function extracts file parts separately and keeps text + file + text
		if len(contentArr) < 2 {
			t.Fatalf("expected at least 2 content parts, got %d", len(contentArr))
		}
		if contentArr[0]["type"] != "text" {
			t.Errorf("expected first part type text, got %v", contentArr[0]["type"])
		}
	})

	t.Run("should handle reasoning parts in messages", func(t *testing.T) {
		messages := []state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{
					ID:         "msg-with-reasoning",
					Role:       "assistant",
					CreatedAt:  now,
					ThreadID:   "thread-1",
					ResourceID: "resource-1",
				},
				Content: state.MastraMessageContentV2{
					Format: 2,
					Parts: []state.MastraMessagePart{
						{Type: "text", Text: "Let me think about this problem:"},
						{
							Type: "reasoning",
							Details: []state.ReasoningDetail{
								{Type: "text", Text: "First, I need to analyze the requirements", Signature: "sig-123"},
								{Type: "text", Text: "Then, I will consider possible solutions", Signature: "sig-456"},
							},
						},
						{Type: "text", Text: "Based on my analysis, here is the solution:"},
					},
				},
			},
		}

		result := ConvertToV1Messages(messages)

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}
		if result[0].Role != "assistant" {
			t.Errorf("expected role assistant, got %s", result[0].Role)
		}
		if result[0].Type != "text" {
			t.Errorf("expected type text, got %s", result[0].Type)
		}

		contentArr, ok := result[0].Content.([]map[string]any)
		if !ok {
			t.Fatalf("expected content to be []map[string]any, got %T", result[0].Content)
		}
		// Text + 2 reasoning parts + text = 4 parts
		if len(contentArr) != 4 {
			t.Fatalf("expected 4 content parts, got %d", len(contentArr))
		}

		if contentArr[0]["type"] != "text" {
			t.Errorf("expected part 0 type text, got %v", contentArr[0]["type"])
		}
		if contentArr[1]["type"] != "reasoning" {
			t.Errorf("expected part 1 type reasoning, got %v", contentArr[1]["type"])
		}
		if contentArr[2]["type"] != "reasoning" {
			t.Errorf("expected part 2 type reasoning, got %v", contentArr[2]["type"])
		}
		if contentArr[3]["type"] != "text" {
			t.Errorf("expected part 3 type text, got %v", contentArr[3]["type"])
		}

		if contentArr[1]["text"] != "First, I need to analyze the requirements" {
			t.Errorf("expected reasoning 1 text, got %v", contentArr[1]["text"])
		}
		if contentArr[1]["signature"] != "sig-123" {
			t.Errorf("expected reasoning 1 signature sig-123, got %v", contentArr[1]["signature"])
		}
		if contentArr[2]["text"] != "Then, I will consider possible solutions" {
			t.Errorf("expected reasoning 2 text, got %v", contentArr[2]["text"])
		}
		if contentArr[2]["signature"] != "sig-456" {
			t.Errorf("expected reasoning 2 signature sig-456, got %v", contentArr[2]["signature"])
		}
	})

	t.Run("should handle redacted reasoning parts", func(t *testing.T) {
		messages := []state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{
					ID:         "msg-with-redacted",
					Role:       "assistant",
					CreatedAt:  now,
					ThreadID:   "thread-1",
					ResourceID: "resource-1",
				},
				Content: state.MastraMessageContentV2{
					Format: 2,
					Parts: []state.MastraMessagePart{
						{
							Type: "reasoning",
							Details: []state.ReasoningDetail{
								{Type: "text", Text: "Analyzing sensitive data", Signature: "sig-789"},
								{Type: "redacted", Data: "REDACTED_CONTENT_ID_123"},
							},
						},
						{Type: "text", Text: "The analysis is complete."},
					},
				},
			},
		}

		result := ConvertToV1Messages(messages)

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}

		contentArr, ok := result[0].Content.([]map[string]any)
		if !ok {
			t.Fatalf("expected content to be []map[string]any, got %T", result[0].Content)
		}
		if len(contentArr) != 3 {
			t.Fatalf("expected 3 content parts, got %d", len(contentArr))
		}

		if contentArr[0]["type"] != "reasoning" {
			t.Errorf("expected part 0 type reasoning, got %v", contentArr[0]["type"])
		}
		if contentArr[1]["type"] != "redacted-reasoning" {
			t.Errorf("expected part 1 type redacted-reasoning, got %v", contentArr[1]["type"])
		}
		if contentArr[2]["type"] != "text" {
			t.Errorf("expected part 2 type text, got %v", contentArr[2]["type"])
		}

		if contentArr[1]["data"] != "REDACTED_CONTENT_ID_123" {
			t.Errorf("expected redacted data REDACTED_CONTENT_ID_123, got %v", contentArr[1]["data"])
		}
	})

	t.Run("should handle mixed content with files, reasoning, and tool invocations", func(t *testing.T) {
		messages := []state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{
					ID:         "msg-mixed",
					Role:       "assistant",
					CreatedAt:  now,
					ThreadID:   "thread-1",
					ResourceID: "resource-1",
				},
				Content: state.MastraMessageContentV2{
					Format: 2,
					Parts: []state.MastraMessagePart{
						{Type: "text", Text: "Let me analyze this image:"},
						{Type: "file", Data: "data:image/png;base64,abc123", MimeType: "image/png"},
						{
							Type: "reasoning",
							Details: []state.ReasoningDetail{
								{Type: "text", Text: "I can see this is a chart showing data trends", Signature: "sig-abc"},
							},
						},
						{Type: "text", Text: "Now let me fetch the latest data:"},
						{
							Type: "tool-invocation",
							ToolInvocation: &state.ToolInvocation{
								State:      "result",
								ToolCallID: "call-789",
								ToolName:   "dataFetcher",
								Args:       map[string]any{"query": "latest_trends"},
								Result:     map[string]any{"data": []any{1, 2, 3}},
							},
						},
						{Type: "text", Text: "The data has been updated."},
					},
				},
			},
		}

		result := ConvertToV1Messages(messages)

		// Should split at tool invocation
		// File parts get extracted, so remaining non-tool parts flow through
		if len(result) < 3 {
			t.Fatalf("expected at least 3 messages (pre-tool, tool-call, tool-result, post-text), got %d", len(result))
		}

		// First message: combined content before tool
		if result[0].Role != "assistant" {
			t.Errorf("expected first message role assistant, got %s", result[0].Role)
		}
		if result[0].Type != "text" {
			t.Errorf("expected first message type text, got %s", result[0].Type)
		}

		// Should have a tool-call message
		foundToolCall := false
		foundToolResult := false
		for _, msg := range result {
			if msg.Type == "tool-call" {
				foundToolCall = true
			}
			if msg.Type == "tool-result" {
				foundToolResult = true
			}
		}
		if !foundToolCall {
			t.Error("expected to find a tool-call message")
		}
		if !foundToolResult {
			t.Error("expected to find a tool-result message")
		}
	})

	t.Run("should handle file attachments in experimental_attachments", func(t *testing.T) {
		messages := []state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{
					ID:         "msg-attachments",
					Role:       "user",
					CreatedAt:  now,
					ThreadID:   "thread-1",
					ResourceID: "resource-1",
				},
				Content: state.MastraMessageContentV2{
					Format:  2,
					Content: "Please analyze this image",
					Parts: []state.MastraMessagePart{
						{Type: "text", Text: "Please analyze this image"},
					},
					ExperimentalAttachments: []state.ExperimentalAttachment{
						{URL: "https://example.com/image.png", ContentType: "image/png"},
						{URL: "data:image/jpeg;base64,/9j/4AAQ...", ContentType: "image/jpeg"},
					},
				},
			},
		}

		result := ConvertToV1Messages(messages)

		if len(result) != 1 {
			t.Fatalf("expected 1 message, got %d", len(result))
		}
		if result[0].Role != "user" {
			t.Errorf("expected role user, got %s", result[0].Role)
		}

		contentArr, ok := result[0].Content.([]map[string]any)
		if !ok {
			t.Fatalf("expected content to be []map[string]any, got %T", result[0].Content)
		}
		// Text + 2 attachment parts = 3
		if len(contentArr) != 3 {
			t.Fatalf("expected 3 content parts, got %d", len(contentArr))
		}
		if contentArr[0]["type"] != "text" {
			t.Errorf("expected first part type text, got %v", contentArr[0]["type"])
		}
	})
}

// ---------------------------------------------------------------------------
// Tool invocation splitting (Issue #6087)
// ---------------------------------------------------------------------------

func TestConvertToV1Messages_ToolInvocationSplitting(t *testing.T) {
	now := time.Now()

	t.Run("should preserve toolInvocations when text follows tool invocations (issue #6087)", func(t *testing.T) {
		messages := []state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{
					ID: "msg-1", Role: "assistant", CreatedAt: now,
					ThreadID: "thread-1", ResourceID: "resource-1",
				},
				Content: state.MastraMessageContentV2{
					Format: 2,
					Parts: []state.MastraMessagePart{
						{Type: "text", Text: "I'll use the weather tool for Paris now:"},
						{
							Type: "tool-invocation",
							ToolInvocation: &state.ToolInvocation{
								State: "result", ToolCallID: "toolu_01Y9o5yfKq",
								ToolName: "weatherTool",
								Args:     map[string]any{"location": "Paris"},
								Result:   map[string]any{"temperature": 24.3, "conditions": "Partly cloudy"},
							},
						},
						{Type: "text", Text: "Ok, I just checked the weather."},
					},
				},
			},
		}

		result := ConvertToV1Messages(messages)

		toolCallMsgs := filterByType(result, "tool-call")
		if len(toolCallMsgs) == 0 {
			t.Fatal("expected at least one tool-call message")
		}

		toolResultMsgs := filterByType(result, "tool-result")
		if len(toolResultMsgs) == 0 {
			t.Fatal("expected at least one tool-result message")
		}

		// Check that tool call info is preserved
		hasWeatherToolCall := false
		for _, msg := range result {
			if arr, ok := msg.Content.([]map[string]any); ok {
				for _, part := range arr {
					if part["type"] == "tool-call" && part["toolName"] == "weatherTool" {
						hasWeatherToolCall = true
					}
				}
			}
		}
		if !hasWeatherToolCall {
			t.Error("expected to find weatherTool tool-call")
		}
	})

	t.Run("should handle mixed content with text, tool invocation, and more text", func(t *testing.T) {
		messages := []state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{
					ID: "test-mixed-1", Role: "assistant", CreatedAt: now,
					ThreadID: "thread-1", ResourceID: "resource-1",
				},
				Content: state.MastraMessageContentV2{
					Format: 2,
					Parts: []state.MastraMessagePart{
						{Type: "text", Text: "Let me check the weather for you..."},
						{
							Type: "tool-invocation",
							ToolInvocation: &state.ToolInvocation{
								State: "result", ToolCallID: "call-123",
								ToolName: "weatherTool",
								Args:     map[string]any{"location": "New York"},
								Result:   map[string]any{"temperature": 72, "conditions": "Sunny", "humidity": 45},
							},
						},
						{Type: "text", Text: "The weather in New York is currently sunny with a temperature of 72F."},
					},
				},
			},
		}

		result := ConvertToV1Messages(messages)

		// Should have 4 messages: text before, tool call, tool result, text after
		if len(result) != 4 {
			t.Fatalf("expected 4 messages, got %d", len(result))
		}

		if result[0].Role != "assistant" || result[0].Type != "text" {
			t.Errorf("msg 0: expected assistant/text, got %s/%s", result[0].Role, result[0].Type)
		}
		if result[1].Role != "assistant" || result[1].Type != "tool-call" {
			t.Errorf("msg 1: expected assistant/tool-call, got %s/%s", result[1].Role, result[1].Type)
		}
		if result[2].Role != "tool" || result[2].Type != "tool-result" {
			t.Errorf("msg 2: expected tool/tool-result, got %s/%s", result[2].Role, result[2].Type)
		}
		if result[3].Role != "assistant" || result[3].Type != "text" {
			t.Errorf("msg 3: expected assistant/text, got %s/%s", result[3].Role, result[3].Type)
		}

		// Verify tool result data preserved
		toolResultMsg := filterByType(result, "tool-result")
		if len(toolResultMsg) == 0 {
			t.Fatal("expected tool-result message")
		}
		if arr, ok := toolResultMsg[0].Content.([]map[string]any); ok {
			for _, part := range arr {
				if part["type"] == "tool-result" {
					res, ok := part["result"].(map[string]any)
					if !ok {
						t.Fatalf("expected result to be map, got %T", part["result"])
					}
					if res["temperature"] != 72 {
						t.Errorf("expected temperature 72, got %v", res["temperature"])
					}
				}
			}
		}
	})

	t.Run("should handle multiple tool calls in a single message", func(t *testing.T) {
		messages := []state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{
					ID: "test-multiple-tools", Role: "assistant", CreatedAt: now,
					ThreadID: "thread-1", ResourceID: "resource-1",
				},
				Content: state.MastraMessageContentV2{
					Format: 2,
					Parts: []state.MastraMessagePart{
						{Type: "text", Text: "I'll check the weather in multiple cities."},
						{
							Type: "tool-invocation",
							ToolInvocation: &state.ToolInvocation{
								State: "result", ToolCallID: "tool-call-1",
								ToolName: "weatherTool",
								Args:     map[string]any{"location": "Paris"},
								Result:   map[string]any{"temperature": 24.3, "conditions": "Partly cloudy", "location": "Paris"},
							},
						},
						{
							Type: "tool-invocation",
							ToolInvocation: &state.ToolInvocation{
								State: "result", ToolCallID: "tool-call-2",
								ToolName: "weatherTool",
								Args:     map[string]any{"location": "London"},
								Result:   map[string]any{"temperature": 18.5, "conditions": "Rainy", "location": "London"},
							},
						},
						{Type: "text", Text: "Now let me search for flights."},
						{
							Type: "tool-invocation",
							ToolInvocation: &state.ToolInvocation{
								State: "result", ToolCallID: "tool-call-3",
								ToolName: "flightSearchTool",
								Args:     map[string]any{"from": "Paris", "to": "London"},
								Result:   map[string]any{"flights": []any{"AF123", "BA456"}},
							},
						},
						{Type: "text", Text: "Paris has better weather."},
					},
				},
			},
		}

		result := ConvertToV1Messages(messages)

		// Should produce 9 messages:
		// 1. text, 2. tool-call (Paris), 3. tool-result (Paris),
		// 4. tool-call (London), 5. tool-result (London),
		// 6. text, 7. tool-call (flight), 8. tool-result (flight), 9. text
		if len(result) != 9 {
			t.Fatalf("expected 9 messages, got %d", len(result))
		}

		if result[0].Type != "text" {
			t.Errorf("msg 0: expected text, got %s", result[0].Type)
		}
		if result[1].Type != "tool-call" {
			t.Errorf("msg 1: expected tool-call, got %s", result[1].Type)
		}
		if result[2].Type != "tool-result" {
			t.Errorf("msg 2: expected tool-result, got %s", result[2].Type)
		}
		if result[3].Type != "tool-call" {
			t.Errorf("msg 3: expected tool-call, got %s", result[3].Type)
		}
		if result[4].Type != "tool-result" {
			t.Errorf("msg 4: expected tool-result, got %s", result[4].Type)
		}
		if result[5].Type != "text" {
			t.Errorf("msg 5: expected text, got %s", result[5].Type)
		}
		if result[6].Type != "tool-call" {
			t.Errorf("msg 6: expected tool-call, got %s", result[6].Type)
		}
		if result[7].Type != "tool-result" {
			t.Errorf("msg 7: expected tool-result, got %s", result[7].Type)
		}
		if result[8].Type != "text" {
			t.Errorf("msg 8: expected text, got %s", result[8].Type)
		}

		toolCallMsgs := filterByType(result, "tool-call")
		if len(toolCallMsgs) != 3 {
			t.Errorf("expected 3 tool-call messages, got %d", len(toolCallMsgs))
		}

		toolResultMsgs := filterByType(result, "tool-result")
		if len(toolResultMsgs) != 3 {
			t.Errorf("expected 3 tool-result messages, got %d", len(toolResultMsgs))
		}

		// Verify specific tool results
		weatherResults := 0
		flightResults := 0
		for _, msg := range toolResultMsgs {
			if arr, ok := msg.Content.([]map[string]any); ok {
				for _, part := range arr {
					if part["toolName"] == "weatherTool" {
						weatherResults++
					}
					if part["toolName"] == "flightSearchTool" {
						flightResults++
					}
				}
			}
		}
		if weatherResults != 2 {
			t.Errorf("expected 2 weather tool results, got %d", weatherResults)
		}
		if flightResults != 1 {
			t.Errorf("expected 1 flight tool result, got %d", flightResults)
		}
	})

	t.Run("should handle user messages without tool invocations", func(t *testing.T) {
		messages := []state.MastraDBMessage{
			{
				MastraMessageShared: state.MastraMessageShared{
					ID: "user-1", Role: "user", CreatedAt: now,
					ThreadID: "thread-1", ResourceID: "resource-1",
				},
				Content: state.MastraMessageContentV2{
					Format:  2,
					Content: "Hello, how are you?",
					Parts:   []state.MastraMessagePart{{Type: "text", Text: "Hello, how are you?"}},
				},
			},
			{
				MastraMessageShared: state.MastraMessageShared{
					ID: "asst-1", Role: "assistant", CreatedAt: now.Add(time.Second),
					ThreadID: "thread-1", ResourceID: "resource-1",
				},
				Content: state.MastraMessageContentV2{
					Format:  2,
					Content: "I'm doing well, thanks for asking!",
					Parts:   []state.MastraMessagePart{{Type: "text", Text: "I'm doing well, thanks for asking!"}},
				},
			},
		}

		result := ConvertToV1Messages(messages)

		if len(result) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(result))
		}
		if result[0].Role != "user" {
			t.Errorf("expected user, got %s", result[0].Role)
		}
		if result[1].Role != "assistant" {
			t.Errorf("expected assistant, got %s", result[1].Role)
		}
	})
}

func filterByType(msgs []state.MastraMessageV1, typ string) []state.MastraMessageV1 {
	var result []state.MastraMessageV1
	for _, m := range msgs {
		if m.Type == typ {
			result = append(result, m)
		}
	}
	return result
}
