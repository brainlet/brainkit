// Ported from: packages/core/src/stream/aisdk/v5/transform.test.ts
package v5

import (
	"testing"

	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

func TestStreamPartFromRaw(t *testing.T) {
	t.Run("should extract type", func(t *testing.T) {
		raw := stream.LanguageModelV2StreamPart{Type: "text-delta"}
		sp := StreamPartFromRaw(raw)
		if sp.Type != "text-delta" {
			t.Errorf("expected type 'text-delta', got %q", sp.Type)
		}
	})

	t.Run("should handle nil data", func(t *testing.T) {
		raw := stream.LanguageModelV2StreamPart{Type: "text-delta", Data: nil}
		sp := StreamPartFromRaw(raw)
		if sp.Type != "text-delta" {
			t.Errorf("expected type 'text-delta', got %q", sp.Type)
		}
		if sp.ID != "" {
			t.Errorf("expected empty ID, got %q", sp.ID)
		}
	})

	t.Run("should extract common fields", func(t *testing.T) {
		raw := stream.LanguageModelV2StreamPart{
			Type: "text-delta",
			Data: map[string]any{
				"id":               "txt-1",
				"delta":            "Hello",
				"providerMetadata": map[string]any{"key": "value"},
			},
		}
		sp := StreamPartFromRaw(raw)
		if sp.ID != "txt-1" {
			t.Errorf("expected ID 'txt-1', got %q", sp.ID)
		}
		if sp.Delta != "Hello" {
			t.Errorf("expected delta 'Hello', got %q", sp.Delta)
		}
		if sp.ProviderMetadata["key"] != "value" {
			t.Errorf("expected providerMetadata key 'value', got %v", sp.ProviderMetadata["key"])
		}
	})

	t.Run("should extract tool fields", func(t *testing.T) {
		provExec := true
		raw := stream.LanguageModelV2StreamPart{
			Type: "tool-call",
			Data: map[string]any{
				"toolCallId":       "call-1",
				"toolName":         "search",
				"input":            `{"query":"test"}`,
				"providerExecuted": provExec,
			},
		}
		sp := StreamPartFromRaw(raw)
		if sp.ToolCallID != "call-1" {
			t.Errorf("expected toolCallId 'call-1', got %q", sp.ToolCallID)
		}
		if sp.ToolName != "search" {
			t.Errorf("expected toolName 'search', got %q", sp.ToolName)
		}
		if sp.ProviderExecuted == nil || *sp.ProviderExecuted != true {
			t.Error("expected providerExecuted to be true")
		}
	})

	t.Run("should extract finish fields", func(t *testing.T) {
		raw := stream.LanguageModelV2StreamPart{
			Type: "finish",
			Data: map[string]any{
				"finishReason": "stop",
				"usage":        map[string]any{"inputTokens": 5, "outputTokens": 10},
			},
		}
		sp := StreamPartFromRaw(raw)
		if sp.FinishReason != "stop" {
			t.Errorf("expected finishReason 'stop', got %q", sp.FinishReason)
		}
		if sp.Usage == nil {
			t.Error("expected usage to be non-nil")
		}
	})

	t.Run("should extract response-metadata fields", func(t *testing.T) {
		raw := stream.LanguageModelV2StreamPart{
			Type: "response-metadata",
			Data: map[string]any{
				"modelId":   "gpt-4",
				"timestamp": "2024-01-01",
			},
		}
		sp := StreamPartFromRaw(raw)
		if sp.ModelID != "gpt-4" {
			t.Errorf("expected modelId 'gpt-4', got %q", sp.ModelID)
		}
	})

	t.Run("should extract source fields", func(t *testing.T) {
		raw := stream.LanguageModelV2StreamPart{
			Type: "source",
			Data: map[string]any{
				"sourceType": "url",
				"url":        "https://example.com",
				"title":      "Example",
			},
		}
		sp := StreamPartFromRaw(raw)
		if sp.SourceType != "url" {
			t.Errorf("expected sourceType 'url', got %q", sp.SourceType)
		}
		if sp.URL != "https://example.com" {
			t.Errorf("expected URL 'https://example.com', got %q", sp.URL)
		}
	})

	t.Run("should extract file fields", func(t *testing.T) {
		raw := stream.LanguageModelV2StreamPart{
			Type: "file",
			Data: map[string]any{
				"mediaType": "image/png",
				"data":      "iVBOR",
			},
		}
		sp := StreamPartFromRaw(raw)
		if sp.MediaType != "image/png" {
			t.Errorf("expected mediaType 'image/png', got %q", sp.MediaType)
		}
	})

	t.Run("should extract error fields", func(t *testing.T) {
		raw := stream.LanguageModelV2StreamPart{
			Type: "error",
			Data: map[string]any{
				"error": "something went wrong",
			},
		}
		sp := StreamPartFromRaw(raw)
		if sp.Error != "something went wrong" {
			t.Errorf("expected error value, got %v", sp.Error)
		}
	})

	t.Run("should extract raw fields", func(t *testing.T) {
		raw := stream.LanguageModelV2StreamPart{
			Type: "raw",
			Data: map[string]any{
				"rawValue": map[string]any{"custom": "data"},
			},
		}
		sp := StreamPartFromRaw(raw)
		if sp.RawValue == nil {
			t.Error("expected rawValue to be non-nil")
		}
	})
}

func TestConvertFullStreamChunkToMastra(t *testing.T) {
	ctx := TransformContext{RunID: "run-1"}

	t.Run("should convert text-delta", func(t *testing.T) {
		sp := StreamPart{Type: "text-delta", ID: "t1", Delta: "Hello"}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "text-delta" {
			t.Errorf("expected type 'text-delta', got %q", result.Type)
		}
		payload := result.Payload.(map[string]any)
		if payload["text"] != "Hello" {
			t.Errorf("expected text 'Hello', got %v", payload["text"])
		}
		if result.RunID != "run-1" {
			t.Errorf("expected runID 'run-1', got %q", result.RunID)
		}
	})

	t.Run("should skip empty text-delta", func(t *testing.T) {
		sp := StreamPart{Type: "text-delta", Delta: ""}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result != nil {
			t.Error("expected nil for empty text-delta")
		}
	})

	t.Run("should convert text-start", func(t *testing.T) {
		sp := StreamPart{Type: "text-start", ID: "t1"}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "text-start" {
			t.Errorf("expected type 'text-start', got %q", result.Type)
		}
	})

	t.Run("should convert text-end", func(t *testing.T) {
		sp := StreamPart{Type: "text-end", ID: "t1"}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "text-end" {
			t.Errorf("expected type 'text-end', got %q", result.Type)
		}
	})

	t.Run("should convert reasoning-start", func(t *testing.T) {
		sp := StreamPart{Type: "reasoning-start", ID: "r1"}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "reasoning-start" {
			t.Errorf("expected type 'reasoning-start', got %q", result.Type)
		}
	})

	t.Run("should convert reasoning-delta", func(t *testing.T) {
		sp := StreamPart{Type: "reasoning-delta", ID: "r1", Delta: "thinking..."}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "reasoning-delta" {
			t.Errorf("expected type 'reasoning-delta', got %q", result.Type)
		}
		payload := result.Payload.(map[string]any)
		if payload["text"] != "thinking..." {
			t.Errorf("expected text 'thinking...', got %v", payload["text"])
		}
	})

	t.Run("should convert reasoning-end", func(t *testing.T) {
		sp := StreamPart{Type: "reasoning-end", ID: "r1"}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "reasoning-end" {
			t.Errorf("expected type 'reasoning-end', got %q", result.Type)
		}
	})

	t.Run("should convert source with url type", func(t *testing.T) {
		sp := StreamPart{
			Type:       "source",
			ID:         "s1",
			SourceType: "url",
			URL:        "https://example.com",
			Title:      "Example",
		}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "source" {
			t.Errorf("expected type 'source', got %q", result.Type)
		}
		payload := result.Payload.(map[string]any)
		if payload["url"] != "https://example.com" {
			t.Errorf("expected url, got %v", payload["url"])
		}
	})

	t.Run("should convert source with document type", func(t *testing.T) {
		sp := StreamPart{
			Type:       "source",
			ID:         "d1",
			SourceType: "document",
			MediaType:  "application/pdf",
			Filename:   "doc.pdf",
			Title:      "Document",
		}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		payload := result.Payload.(map[string]any)
		if payload["mimeType"] != "application/pdf" {
			t.Errorf("expected mimeType, got %v", payload["mimeType"])
		}
		if payload["filename"] != "doc.pdf" {
			t.Errorf("expected filename, got %v", payload["filename"])
		}
	})

	t.Run("should convert file", func(t *testing.T) {
		sp := StreamPart{Type: "file", MediaType: "image/png", Data: "iVBOR"}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "file" {
			t.Errorf("expected type 'file', got %q", result.Type)
		}
		payload := result.Payload.(map[string]any)
		if payload["base64"] != "iVBOR" {
			t.Errorf("expected base64 'iVBOR', got %v", payload["base64"])
		}
	})

	t.Run("should convert tool-call with JSON string input", func(t *testing.T) {
		sp := StreamPart{
			Type:       "tool-call",
			ToolCallID: "tc-1",
			ToolName:   "search",
			Input:      `{"query":"test"}`,
		}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "tool-call" {
			t.Errorf("expected type 'tool-call', got %q", result.Type)
		}
		payload := result.Payload.(map[string]any)
		args, ok := payload["args"].(map[string]any)
		if !ok {
			t.Fatal("expected args to be map")
		}
		if args["query"] != "test" {
			t.Errorf("expected query 'test', got %v", args["query"])
		}
	})

	t.Run("should convert tool-call with map input", func(t *testing.T) {
		sp := StreamPart{
			Type:       "tool-call",
			ToolCallID: "tc-2",
			ToolName:   "calc",
			Input:      map[string]any{"x": float64(5)},
		}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		payload := result.Payload.(map[string]any)
		args, ok := payload["args"].(map[string]any)
		if !ok {
			t.Fatal("expected args to be map")
		}
		if args["x"] != float64(5) {
			t.Errorf("expected x=5, got %v", args["x"])
		}
	})

	t.Run("should convert tool-result", func(t *testing.T) {
		isErr := false
		sp := StreamPart{
			Type:       "tool-result",
			ToolCallID: "tc-1",
			ToolName:   "search",
			Result:     map[string]any{"data": "found"},
			IsError:    &isErr,
		}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "tool-result" {
			t.Errorf("expected type 'tool-result', got %q", result.Type)
		}
	})

	t.Run("should convert tool-input-start", func(t *testing.T) {
		sp := StreamPart{Type: "tool-input-start", ID: "inp-1", ToolName: "calc"}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "tool-call-input-streaming-start" {
			t.Errorf("expected type 'tool-call-input-streaming-start', got %q", result.Type)
		}
	})

	t.Run("should convert tool-input-delta", func(t *testing.T) {
		sp := StreamPart{Type: "tool-input-delta", ID: "inp-1", Delta: `{"a":5`}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "tool-call-delta" {
			t.Errorf("expected type 'tool-call-delta', got %q", result.Type)
		}
	})

	t.Run("should skip empty tool-input-delta", func(t *testing.T) {
		sp := StreamPart{Type: "tool-input-delta", Delta: ""}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result != nil {
			t.Error("expected nil for empty tool-input-delta")
		}
	})

	t.Run("should convert tool-input-end", func(t *testing.T) {
		sp := StreamPart{Type: "tool-input-end", ID: "inp-1"}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "tool-call-input-streaming-end" {
			t.Errorf("expected type 'tool-call-input-streaming-end', got %q", result.Type)
		}
	})

	t.Run("should convert finish with V2 usage", func(t *testing.T) {
		sp := StreamPart{
			Type:         "finish",
			FinishReason: "stop",
			Usage: map[string]any{
				"inputTokens":  float64(5),
				"outputTokens": float64(10),
			},
		}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "finish" {
			t.Errorf("expected type 'finish', got %q", result.Type)
		}
		payload := result.Payload.(map[string]any)
		stepResult := payload["stepResult"].(map[string]any)
		if stepResult["reason"] != "stop" {
			t.Errorf("expected reason 'stop', got %v", stepResult["reason"])
		}
		output := payload["output"].(map[string]any)
		usage, ok := output["usage"].(stream.LanguageModelUsage)
		if !ok {
			t.Fatal("expected usage to be LanguageModelUsage")
		}
		if usage.InputTokens != 5 {
			t.Errorf("expected inputTokens 5, got %d", usage.InputTokens)
		}
	})

	t.Run("should convert error", func(t *testing.T) {
		sp := StreamPart{Type: "error", Error: "failed"}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "error" {
			t.Errorf("expected type 'error', got %q", result.Type)
		}
	})

	t.Run("should convert raw", func(t *testing.T) {
		sp := StreamPart{Type: "raw", RawValue: map[string]any{"custom": "data"}}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Type != "raw" {
			t.Errorf("expected type 'raw', got %q", result.Type)
		}
	})

	t.Run("should return nil for unknown type", func(t *testing.T) {
		sp := StreamPart{Type: "unknown-type"}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result != nil {
			t.Error("expected nil for unknown type")
		}
	})

	t.Run("should set From to agent", func(t *testing.T) {
		sp := StreamPart{Type: "text-delta", Delta: "hi"}
		result := ConvertFullStreamChunkToMastra(sp, ctx)
		if result.From != stream.ChunkFromAgent {
			t.Errorf("expected From to be agent, got %q", result.From)
		}
	})
}

func TestConvertMastraChunkToAISDKv5(t *testing.T) {
	t.Run("should convert text-delta", func(t *testing.T) {
		chunk := stream.ChunkType{
			Type:    "text-delta",
			Payload: map[string]any{"id": "t1", "text": "Hello", "providerMetadata": nil},
		}
		result := ConvertMastraChunkToAISDKv5(chunk, "stream")
		if result["type"] != "text-delta" {
			t.Errorf("expected type 'text-delta', got %v", result["type"])
		}
		if result["text"] != "Hello" {
			t.Errorf("expected text 'Hello', got %v", result["text"])
		}
	})

	t.Run("should convert start", func(t *testing.T) {
		chunk := stream.ChunkType{Type: "start"}
		result := ConvertMastraChunkToAISDKv5(chunk, "")
		if result["type"] != "start" {
			t.Errorf("expected type 'start', got %v", result["type"])
		}
	})

	t.Run("should convert finish", func(t *testing.T) {
		chunk := stream.ChunkType{
			Type: "finish",
			Payload: map[string]any{
				"stepResult": map[string]any{"reason": "stop"},
				"output":     map[string]any{"usage": map[string]any{"inputTokens": 5}},
			},
		}
		result := ConvertMastraChunkToAISDKv5(chunk, "stream")
		if result["type"] != "finish" {
			t.Errorf("expected type 'finish', got %v", result["type"])
		}
		if result["finishReason"] != "stop" {
			t.Errorf("expected finishReason 'stop', got %v", result["finishReason"])
		}
	})

	t.Run("should convert tool-call", func(t *testing.T) {
		chunk := stream.ChunkType{
			Type: "tool-call",
			Payload: map[string]any{
				"toolCallId": "tc-1",
				"toolName":   "search",
				"args":       map[string]any{"query": "test"},
			},
		}
		result := ConvertMastraChunkToAISDKv5(chunk, "stream")
		if result["type"] != "tool-call" {
			t.Errorf("expected type 'tool-call', got %v", result["type"])
		}
		if result["toolCallId"] != "tc-1" {
			t.Errorf("expected toolCallId 'tc-1', got %v", result["toolCallId"])
		}
		if result["input"] == nil {
			t.Error("expected input field")
		}
	})

	t.Run("should convert tool-result", func(t *testing.T) {
		chunk := stream.ChunkType{
			Type: "tool-result",
			Payload: map[string]any{
				"toolCallId": "tc-1",
				"toolName":   "search",
				"result":     map[string]any{"data": "found"},
			},
		}
		result := ConvertMastraChunkToAISDKv5(chunk, "stream")
		if result["type"] != "tool-result" {
			t.Errorf("expected type 'tool-result', got %v", result["type"])
		}
		if result["output"] == nil {
			t.Error("expected output field")
		}
	})

	t.Run("should convert error", func(t *testing.T) {
		chunk := stream.ChunkType{
			Type:    "error",
			Payload: map[string]any{"error": "failed"},
		}
		result := ConvertMastraChunkToAISDKv5(chunk, "stream")
		if result["type"] != "error" {
			t.Errorf("expected type 'error', got %v", result["type"])
		}
	})

	t.Run("should convert abort", func(t *testing.T) {
		chunk := stream.ChunkType{Type: "abort"}
		result := ConvertMastraChunkToAISDKv5(chunk, "stream")
		if result["type"] != "abort" {
			t.Errorf("expected type 'abort', got %v", result["type"])
		}
	})

	t.Run("should convert object", func(t *testing.T) {
		chunk := stream.ChunkType{
			Type:   "object",
			Object: map[string]any{"name": "test"},
		}
		result := ConvertMastraChunkToAISDKv5(chunk, "stream")
		if result["type"] != "object" {
			t.Errorf("expected type 'object', got %v", result["type"])
		}
		if result["object"] == nil {
			t.Error("expected object field")
		}
	})

	t.Run("should return nil for unknown type with nil payload", func(t *testing.T) {
		chunk := stream.ChunkType{Type: "completely-unknown"}
		result := ConvertMastraChunkToAISDKv5(chunk, "stream")
		if result != nil {
			t.Error("expected nil for unknown type with nil payload")
		}
	})

	t.Run("should pass through unknown type with payload", func(t *testing.T) {
		chunk := stream.ChunkType{
			Type:    "custom-type",
			Payload: map[string]any{"data": "value"},
		}
		result := ConvertMastraChunkToAISDKv5(chunk, "stream")
		if result == nil {
			t.Fatal("expected non-nil result for unknown type with payload")
		}
		if result["type"] != "custom-type" {
			t.Errorf("expected type 'custom-type', got %v", result["type"])
		}
	})

	t.Run("should convert reasoning-delta", func(t *testing.T) {
		chunk := stream.ChunkType{
			Type:    "reasoning-delta",
			Payload: map[string]any{"id": "r1", "text": "thinking", "providerMetadata": nil},
		}
		result := ConvertMastraChunkToAISDKv5(chunk, "stream")
		if result["type"] != "reasoning-delta" {
			t.Errorf("expected type 'reasoning-delta', got %v", result["type"])
		}
		if result["text"] != "thinking" {
			t.Errorf("expected text 'thinking', got %v", result["text"])
		}
	})

	t.Run("should convert source", func(t *testing.T) {
		chunk := stream.ChunkType{
			Type: "source",
			Payload: map[string]any{
				"id":         "s1",
				"sourceType": "url",
				"url":        "https://example.com",
				"title":      "Example",
			},
		}
		result := ConvertMastraChunkToAISDKv5(chunk, "stream")
		if result["type"] != "source" {
			t.Errorf("expected type 'source', got %v", result["type"])
		}
		if result["sourceType"] != "url" {
			t.Errorf("expected sourceType 'url', got %v", result["sourceType"])
		}
	})

	t.Run("should convert tool-call-delta", func(t *testing.T) {
		chunk := stream.ChunkType{
			Type: "tool-call-delta",
			Payload: map[string]any{
				"toolCallId":    "tc-1",
				"argsTextDelta": `{"a":`,
			},
		}
		result := ConvertMastraChunkToAISDKv5(chunk, "stream")
		if result["type"] != "tool-input-delta" {
			t.Errorf("expected type 'tool-input-delta', got %v", result["type"])
		}
		if result["delta"] != `{"a":` {
			t.Errorf("expected delta, got %v", result["delta"])
		}
	})

	t.Run("should default mode to stream", func(t *testing.T) {
		chunk := stream.ChunkType{
			Type:    "file",
			Payload: map[string]any{"data": "iVBOR", "mimeType": "image/png"},
		}
		result := ConvertMastraChunkToAISDKv5(chunk, "")
		if result["type"] != "file" {
			t.Errorf("expected type 'file', got %v", result["type"])
		}
	})
}

func TestNormalizeUsage(t *testing.T) {
	t.Run("should handle nil usage", func(t *testing.T) {
		result := normalizeUsage(nil)
		if result.InputTokens != 0 || result.OutputTokens != 0 {
			t.Errorf("expected zero usage, got input=%d output=%d", result.InputTokens, result.OutputTokens)
		}
	})

	t.Run("should handle non-map usage", func(t *testing.T) {
		result := normalizeUsage("not a map")
		if result.InputTokens != 0 {
			t.Errorf("expected zero usage, got input=%d", result.InputTokens)
		}
	})

	t.Run("should normalize V2 flat usage", func(t *testing.T) {
		usage := map[string]any{
			"inputTokens":  float64(10),
			"outputTokens": float64(20),
			"totalTokens":  float64(30),
		}
		result := normalizeUsage(usage)
		if result.InputTokens != 10 {
			t.Errorf("expected inputTokens 10, got %d", result.InputTokens)
		}
		if result.OutputTokens != 20 {
			t.Errorf("expected outputTokens 20, got %d", result.OutputTokens)
		}
		if result.TotalTokens != 30 {
			t.Errorf("expected totalTokens 30, got %d", result.TotalTokens)
		}
	})

	t.Run("should compute totalTokens if not provided in V2", func(t *testing.T) {
		usage := map[string]any{
			"inputTokens":  float64(10),
			"outputTokens": float64(20),
		}
		result := normalizeUsage(usage)
		if result.TotalTokens != 30 {
			t.Errorf("expected totalTokens 30, got %d", result.TotalTokens)
		}
	})

	t.Run("should normalize V3 nested usage", func(t *testing.T) {
		usage := map[string]any{
			"inputTokens": map[string]any{
				"total":      float64(15),
				"cacheRead":  float64(5),
				"cacheWrite": float64(2),
			},
			"outputTokens": map[string]any{
				"total":     float64(25),
				"reasoning": float64(10),
			},
		}
		result := normalizeUsage(usage)
		if result.InputTokens != 15 {
			t.Errorf("expected inputTokens 15, got %d", result.InputTokens)
		}
		if result.OutputTokens != 25 {
			t.Errorf("expected outputTokens 25, got %d", result.OutputTokens)
		}
		if result.TotalTokens != 40 {
			t.Errorf("expected totalTokens 40, got %d", result.TotalTokens)
		}
		if result.ReasoningTokens != 10 {
			t.Errorf("expected reasoningTokens 10, got %d", result.ReasoningTokens)
		}
		if result.CachedInputTokens != 5 {
			t.Errorf("expected cachedInputTokens 5, got %d", result.CachedInputTokens)
		}
	})
}

func TestNormalizeFinishReason(t *testing.T) {
	t.Run("should return other for nil", func(t *testing.T) {
		result := normalizeFinishReason(nil)
		if result != "other" {
			t.Errorf("expected 'other', got %q", result)
		}
	})

	t.Run("should pass through standard reasons", func(t *testing.T) {
		result := normalizeFinishReason("stop")
		if result != "stop" {
			t.Errorf("expected 'stop', got %q", result)
		}
	})

	t.Run("should pass through tripwire reason", func(t *testing.T) {
		result := normalizeFinishReason("tripwire")
		if result != "tripwire" {
			t.Errorf("expected 'tripwire', got %q", result)
		}
	})

	t.Run("should pass through retry reason", func(t *testing.T) {
		result := normalizeFinishReason("retry")
		if result != "retry" {
			t.Errorf("expected 'retry', got %q", result)
		}
	})

	t.Run("should map unknown to other", func(t *testing.T) {
		result := normalizeFinishReason("unknown")
		if result != "other" {
			t.Errorf("expected 'other', got %q", result)
		}
	})

	t.Run("should extract unified from V3 object format", func(t *testing.T) {
		result := normalizeFinishReason(map[string]any{
			"unified": "stop",
			"raw":     "end_turn",
		})
		if result != "stop" {
			t.Errorf("expected 'stop', got %q", result)
		}
	})

	t.Run("should return other for V3 object without unified", func(t *testing.T) {
		result := normalizeFinishReason(map[string]any{"raw": "end_turn"})
		if result != "other" {
			t.Errorf("expected 'other', got %q", result)
		}
	})

	t.Run("should return other for non-string non-map", func(t *testing.T) {
		result := normalizeFinishReason(42)
		if result != "other" {
			t.Errorf("expected 'other', got %q", result)
		}
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("payloadMap should handle nil", func(t *testing.T) {
		result := payloadMap(nil)
		if result == nil || len(result) != 0 {
			t.Error("expected empty map for nil")
		}
	})

	t.Run("payloadMap should handle map", func(t *testing.T) {
		m := map[string]any{"key": "value"}
		result := payloadMap(m)
		if result["key"] != "value" {
			t.Errorf("expected key 'value', got %v", result["key"])
		}
	})

	t.Run("payloadMap should handle non-map", func(t *testing.T) {
		result := payloadMap("not a map")
		if result == nil || len(result) != 0 {
			t.Error("expected empty map for non-map")
		}
	})

	t.Run("intFromAnyV should handle float64", func(t *testing.T) {
		result := intFromAnyV(float64(42))
		if result != 42 {
			t.Errorf("expected 42, got %d", result)
		}
	})

	t.Run("intFromAnyV should handle int", func(t *testing.T) {
		result := intFromAnyV(42)
		if result != 42 {
			t.Errorf("expected 42, got %d", result)
		}
	})

	t.Run("intFromAnyV should handle nil", func(t *testing.T) {
		result := intFromAnyV(nil)
		if result != 0 {
			t.Errorf("expected 0, got %d", result)
		}
	})

	t.Run("orDefault should return value when non-nil", func(t *testing.T) {
		result := orDefault("hello", "default")
		if result != "hello" {
			t.Errorf("expected 'hello', got %v", result)
		}
	})

	t.Run("orDefault should return default when nil", func(t *testing.T) {
		result := orDefault(nil, "default")
		if result != "default" {
			t.Errorf("expected 'default', got %v", result)
		}
	})

	t.Run("boolFromAny should handle bool", func(t *testing.T) {
		if !boolFromAny(true) {
			t.Error("expected true")
		}
		if boolFromAny(false) {
			t.Error("expected false")
		}
	})

	t.Run("boolFromAny should return false for non-bool", func(t *testing.T) {
		if boolFromAny("true") {
			t.Error("expected false for string")
		}
		if boolFromAny(nil) {
			t.Error("expected false for nil")
		}
	})
}
