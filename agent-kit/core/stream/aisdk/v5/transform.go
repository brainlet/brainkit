// Ported from: packages/core/src/stream/aisdk/v5/transform.ts
package v5

import (
	"encoding/json"
	"fmt"

	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// LanguageModelV2StreamPart mirrors @ai-sdk/provider-v5 LanguageModelV2StreamPart.
// This is a discriminated union in TS; in Go we use a struct with a Type field.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 provider types remain local stubs.
type LanguageModelV2StreamPart = stream.LanguageModelV2StreamPart

// LanguageModelV3FinishReason mirrors @ai-sdk/provider-v6 LanguageModelV3FinishReason.
// In V6, finish reason is an object with unified and raw properties.
// The canonical V3 type is languagemodel.FinishReason in
// brainlink/experiments/ai-kit/provider/languagemodel.
// This local struct is kept because it is currently unused (runtime code
// normalises via map[string]any) and importing ai-kit would add a
// dependency for no functional benefit.
type LanguageModelV3FinishReason struct {
	Unified string `json:"unified"`
	Raw     string `json:"raw,omitempty"`
}

// LanguageModelV3Usage mirrors @ai-sdk/provider-v6 LanguageModelV3Usage.
// In V6, usage has nested objects with detailed breakdowns.
// The canonical V3 type is languagemodel.Usage in
// brainlink/experiments/ai-kit/provider/languagemodel.
// This local struct is kept because it is currently unused (runtime code
// normalises via map[string]any) and importing ai-kit would add a
// dependency for no functional benefit.
type LanguageModelV3Usage struct {
	InputTokens struct {
		Total      int `json:"total"`
		NoCache    int `json:"noCache,omitempty"`
		CacheRead  int `json:"cacheRead,omitempty"`
		CacheWrite int `json:"cacheWrite,omitempty"`
	} `json:"inputTokens"`
	OutputTokens struct {
		Total     int `json:"total"`
		Text      int `json:"text,omitempty"`
		Reasoning int `json:"reasoning,omitempty"`
	} `json:"outputTokens"`
}

// ModelMessage mirrors @internal/ai-sdk-v5 ModelMessage.
// Stub: AI SDK v5 internal type; ai-kit targets v6. Kept as map alias.
type ModelMessage = map[string]any

// AIV5ResponseMessage mirrors ../../../agent/message-list AIV5ResponseMessage.
// Stub: parallel-stubs architecture — real type in agent package has different shape.
type AIV5ResponseMessage = map[string]any

// ---------------------------------------------------------------------------
// StreamPart — extended LanguageModelV2StreamPart with Mastra-specific fields
// ---------------------------------------------------------------------------

// StreamPart is a richly-typed representation of a LanguageModelV2StreamPart.
// In TS this is a discriminated union; in Go we use a struct with all possible
// fields and a Type discriminator. Fields are populated based on the Type.
//
// LanguageModelV2StreamPart (the wire type) only has Type + Data map.
// StreamPart unpacks the Data map into typed fields for convenient access.
type StreamPart struct {
	Type string

	// Common fields
	ID               string
	Delta            string
	ProviderMetadata map[string]any

	// response-metadata
	ModelID   string
	Timestamp any

	// source
	SourceType string
	URL        string
	Title      string
	Filename   string

	// file
	MediaType string
	Data      any // base64 string for file data

	// tool-call / tool-result / tool-input
	ToolCallID       string
	ToolName         string
	Input            any
	Result           any
	IsError          *bool
	ProviderExecuted *bool

	// finish
	FinishReason string
	Usage        any
	Messages     any

	// error
	Error any

	// raw
	RawValue any

	// stream-start
	Warnings any
}

// StreamPartFromRaw converts a LanguageModelV2StreamPart (wire format with Type + Data map)
// into a richly-typed StreamPart for convenient field access.
func StreamPartFromRaw(raw stream.LanguageModelV2StreamPart) StreamPart {
	sp := StreamPart{Type: raw.Type}
	if raw.Data == nil {
		return sp
	}
	d := raw.Data

	// Common
	if v, ok := d["id"].(string); ok {
		sp.ID = v
	}
	if v, ok := d["delta"].(string); ok {
		sp.Delta = v
	}
	if v, ok := d["providerMetadata"].(map[string]any); ok {
		sp.ProviderMetadata = v
	}

	// response-metadata
	if v, ok := d["modelId"].(string); ok {
		sp.ModelID = v
	}
	sp.Timestamp = d["timestamp"]

	// source
	if v, ok := d["sourceType"].(string); ok {
		sp.SourceType = v
	}
	if v, ok := d["url"].(string); ok {
		sp.URL = v
	}
	if v, ok := d["title"].(string); ok {
		sp.Title = v
	}
	if v, ok := d["filename"].(string); ok {
		sp.Filename = v
	}

	// file
	if v, ok := d["mediaType"].(string); ok {
		sp.MediaType = v
	}
	sp.Data = d["data"]

	// tool fields
	if v, ok := d["toolCallId"].(string); ok {
		sp.ToolCallID = v
	}
	if v, ok := d["toolName"].(string); ok {
		sp.ToolName = v
	}
	sp.Input = d["input"]
	sp.Result = d["result"]
	if v, ok := d["isError"].(bool); ok {
		sp.IsError = &v
	}
	if v, ok := d["providerExecuted"].(bool); ok {
		sp.ProviderExecuted = &v
	}

	// finish
	if v, ok := d["finishReason"].(string); ok {
		sp.FinishReason = v
	}
	sp.Usage = d["usage"]
	sp.Messages = d["messages"]

	// error
	sp.Error = d["error"]

	// raw
	sp.RawValue = d["rawValue"]

	// stream-start
	sp.Warnings = d["warnings"]

	return sp
}

// TransformContext carries context for the transform operation.
type TransformContext struct {
	RunID string
}

// ---------------------------------------------------------------------------
// ConvertFullStreamChunkToMastra
// ---------------------------------------------------------------------------

// ConvertFullStreamChunkToMastra converts an AI SDK v5 LanguageModelV2StreamPart
// to a Mastra ChunkType. Returns nil if the chunk type is not recognized.
//
// This is the v5-specific transform that maps provider stream events
// into Mastra's unified chunk format.
func ConvertFullStreamChunkToMastra(value StreamPart, ctx TransformContext) *stream.ChunkType {
	switch value.Type {
	case "response-metadata":
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type:    "response-metadata",
			Payload: mapFromStreamPart(value),
		}

	case "text-start":
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type: "text-start",
			Payload: map[string]any{
				"id":               value.ID,
				"providerMetadata": value.ProviderMetadata,
			},
		}

	case "text-delta":
		if value.Delta == "" {
			return nil
		}
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type: "text-delta",
			Payload: map[string]any{
				"id":               value.ID,
				"providerMetadata": value.ProviderMetadata,
				"text":             value.Delta,
			},
		}

	case "text-end":
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type:    "text-end",
			Payload: mapFromStreamPart(value),
		}

	case "reasoning-start":
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type: "reasoning-start",
			Payload: map[string]any{
				"id":               value.ID,
				"providerMetadata": value.ProviderMetadata,
			},
		}

	case "reasoning-delta":
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type: "reasoning-delta",
			Payload: map[string]any{
				"id":               value.ID,
				"providerMetadata": value.ProviderMetadata,
				"text":             value.Delta,
			},
		}

	case "reasoning-end":
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type: "reasoning-end",
			Payload: map[string]any{
				"id":               value.ID,
				"providerMetadata": value.ProviderMetadata,
			},
		}

	case "source":
		payload := map[string]any{
			"id":               value.ID,
			"sourceType":       value.SourceType,
			"title":            value.Title,
			"providerMetadata": value.ProviderMetadata,
		}
		if value.SourceType == "document" {
			payload["mimeType"] = value.MediaType
			payload["filename"] = value.Filename
		}
		if value.SourceType == "url" {
			payload["url"] = value.URL
		}
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type:    "source",
			Payload: payload,
		}

	case "file":
		payload := map[string]any{
			"data":     value.Data,
			"mimeType": value.MediaType,
		}
		if dataStr, ok := value.Data.(string); ok {
			payload["base64"] = dataStr
		}
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type:    "file",
			Payload: payload,
		}

	case "tool-call":
		var toolCallInput map[string]any
		if inputStr, ok := value.Input.(string); ok && inputStr != "" {
			if err := json.Unmarshal([]byte(inputStr), &toolCallInput); err != nil {
				fmt.Printf("Error converting tool call input to JSON: %v, input: %s\n", err, inputStr)
			}
		} else if inputMap, ok := value.Input.(map[string]any); ok {
			toolCallInput = inputMap
		}
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type: "tool-call",
			Payload: map[string]any{
				"toolCallId":       value.ToolCallID,
				"toolName":         value.ToolName,
				"args":             toolCallInput,
				"providerExecuted": value.ProviderExecuted,
				"providerMetadata": value.ProviderMetadata,
			},
		}

	case "tool-result":
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type: "tool-result",
			Payload: map[string]any{
				"toolCallId":       value.ToolCallID,
				"toolName":         value.ToolName,
				"result":           value.Result,
				"isError":          value.IsError,
				"providerExecuted": value.ProviderExecuted,
				"providerMetadata": value.ProviderMetadata,
			},
		}

	case "tool-input-start":
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type: "tool-call-input-streaming-start",
			Payload: map[string]any{
				"toolCallId":       value.ID,
				"toolName":         value.ToolName,
				"providerExecuted": value.ProviderExecuted,
				"providerMetadata": value.ProviderMetadata,
			},
		}

	case "tool-input-delta":
		if value.Delta == "" {
			return nil
		}
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type: "tool-call-delta",
			Payload: map[string]any{
				"argsTextDelta":    value.Delta,
				"toolCallId":       value.ID,
				"providerMetadata": value.ProviderMetadata,
			},
		}

	case "tool-input-end":
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type: "tool-call-input-streaming-end",
			Payload: map[string]any{
				"toolCallId":       value.ID,
				"providerMetadata": value.ProviderMetadata,
			},
		}

	case "finish":
		usage := normalizeUsage(value.Usage)
		finishReason := normalizeFinishReason(value.FinishReason)
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type: "finish",
			Payload: map[string]any{
				"stepResult": map[string]any{
					"reason": finishReason,
				},
				"output": map[string]any{
					"usage": usage,
				},
				"metadata": map[string]any{
					"providerMetadata": value.ProviderMetadata,
				},
				"messages": value.Messages,
			},
		}

	case "error":
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type:    "error",
			Payload: mapFromStreamPart(value),
		}

	case "raw":
		return &stream.ChunkType{
			BaseChunkType: stream.BaseChunkType{
				RunID: ctx.RunID,
				From:  stream.ChunkFromAgent,
			},
			Type:    "raw",
			Payload: value.RawValue,
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// ConvertMastraChunkToAISDKv5
// ---------------------------------------------------------------------------

// OutputChunkType is the result of converting a Mastra ChunkType to AI SDK v5 format.
// This is a map[string]any since Go doesn't have discriminated unions.
type OutputChunkType = map[string]any

// ConvertMastraChunkToAISDKv5 converts a Mastra ChunkType to an AI SDK v5
// TextStreamPart/ObjectStreamPart representation.
//
// Returns nil if the chunk type is not recognized.
func ConvertMastraChunkToAISDKv5(chunk stream.ChunkType, mode string) OutputChunkType {
	if mode == "" {
		mode = "stream"
	}

	switch chunk.Type {
	case "start":
		return map[string]any{"type": "start"}

	case "step-start":
		payload := payloadMap(chunk.Payload)
		return map[string]any{
			"type":     "start-step",
			"request":  payload["request"],
			"warnings": orDefault(payload["warnings"], []any{}),
		}

	case "raw":
		return map[string]any{
			"type":     "raw",
			"rawValue": chunk.Payload,
		}

	case "finish":
		payload := payloadMap(chunk.Payload)
		stepResult := mapFromAny(payload["stepResult"])
		output := mapFromAny(payload["output"])
		return map[string]any{
			"type":         "finish",
			"finishReason": stepResult["reason"],
			"totalUsage":   output["usage"],
		}

	case "reasoning-start":
		payload := payloadMap(chunk.Payload)
		return map[string]any{
			"type":             "reasoning-start",
			"id":               payload["id"],
			"providerMetadata": payload["providerMetadata"],
		}

	case "reasoning-delta":
		payload := payloadMap(chunk.Payload)
		return map[string]any{
			"type":             "reasoning-delta",
			"id":               payload["id"],
			"text":             payload["text"],
			"providerMetadata": payload["providerMetadata"],
		}

	case "reasoning-end":
		payload := payloadMap(chunk.Payload)
		return map[string]any{
			"type":             "reasoning-end",
			"id":               payload["id"],
			"providerMetadata": payload["providerMetadata"],
		}

	case "source":
		payload := payloadMap(chunk.Payload)
		sourceType, _ := payload["sourceType"].(string)
		if sourceType == "url" {
			return map[string]any{
				"type":             "source",
				"sourceType":       "url",
				"id":               payload["id"],
				"url":              payload["url"],
				"title":            payload["title"],
				"providerMetadata": payload["providerMetadata"],
			}
		}
		return map[string]any{
			"type":             "source",
			"sourceType":       "document",
			"id":               payload["id"],
			"mediaType":        payload["mimeType"],
			"title":            payload["title"],
			"filename":         payload["filename"],
			"providerMetadata": payload["providerMetadata"],
		}

	case "file":
		payload := payloadMap(chunk.Payload)
		data := payload["data"]
		mediaType, _ := payload["mimeType"].(string)
		if mode == "generate" {
			return map[string]any{
				"type": "file",
				"file": NewDefaultGeneratedFile(DefaultGeneratedFileOptions{
					Data:      data,
					MediaType: mediaType,
				}),
			}
		}
		return map[string]any{
			"type": "file",
			"file": NewDefaultGeneratedFileWithType(DefaultGeneratedFileOptions{
				Data:      data,
				MediaType: mediaType,
			}),
		}

	case "tool-call":
		payload := payloadMap(chunk.Payload)
		return map[string]any{
			"type":             "tool-call",
			"toolCallId":       payload["toolCallId"],
			"providerMetadata": payload["providerMetadata"],
			"providerExecuted": payload["providerExecuted"],
			"toolName":         payload["toolName"],
			"input":            payload["args"],
		}

	case "tool-call-input-streaming-start":
		payload := payloadMap(chunk.Payload)
		return map[string]any{
			"type":             "tool-input-start",
			"id":               payload["toolCallId"],
			"toolName":         payload["toolName"],
			"dynamic":          boolFromAny(payload["dynamic"]),
			"providerMetadata": payload["providerMetadata"],
			"providerExecuted": payload["providerExecuted"],
		}

	case "tool-call-input-streaming-end":
		payload := payloadMap(chunk.Payload)
		return map[string]any{
			"type":             "tool-input-end",
			"id":               payload["toolCallId"],
			"providerMetadata": payload["providerMetadata"],
		}

	case "tool-call-delta":
		payload := payloadMap(chunk.Payload)
		return map[string]any{
			"type":             "tool-input-delta",
			"id":               payload["toolCallId"],
			"delta":            payload["argsTextDelta"],
			"providerMetadata": payload["providerMetadata"],
		}

	case "step-finish":
		payload := payloadMap(chunk.Payload)
		metadata := mapFromAny(payload["metadata"])
		output := mapFromAny(payload["output"])
		stepResult := mapFromAny(payload["stepResult"])
		return map[string]any{
			"type": "finish-step",
			"response": map[string]any{
				"id":        orDefault(payload["id"], ""),
				"timestamp": nil, // time.Now() equivalent in TS
				"modelId":   orDefault(metadata["modelId"], ""),
			},
			"usage":            output["usage"],
			"finishReason":     stepResult["reason"],
			"providerMetadata": metadata["providerMetadata"],
		}

	case "text-delta":
		payload := payloadMap(chunk.Payload)
		return map[string]any{
			"type":             "text-delta",
			"id":               payload["id"],
			"text":             payload["text"],
			"providerMetadata": payload["providerMetadata"],
		}

	case "text-end":
		payload := payloadMap(chunk.Payload)
		return map[string]any{
			"type":             "text-end",
			"id":               payload["id"],
			"providerMetadata": payload["providerMetadata"],
		}

	case "text-start":
		payload := payloadMap(chunk.Payload)
		return map[string]any{
			"type":             "text-start",
			"id":               payload["id"],
			"providerMetadata": payload["providerMetadata"],
		}

	case "tool-result":
		payload := payloadMap(chunk.Payload)
		return map[string]any{
			"type":             "tool-result",
			"input":            payload["args"],
			"toolCallId":       payload["toolCallId"],
			"providerExecuted": payload["providerExecuted"],
			"toolName":         payload["toolName"],
			"output":           payload["result"],
		}

	case "tool-error":
		payload := payloadMap(chunk.Payload)
		return map[string]any{
			"type":             "tool-error",
			"error":            payload["error"],
			"input":            payload["args"],
			"toolCallId":       payload["toolCallId"],
			"providerExecuted": payload["providerExecuted"],
			"toolName":         payload["toolName"],
		}

	case "abort":
		return map[string]any{"type": "abort"}

	case "error":
		payload := payloadMap(chunk.Payload)
		return map[string]any{
			"type":  "error",
			"error": payload["error"],
		}

	case "object":
		return map[string]any{
			"type":   "object",
			"object": chunk.Object,
		}

	default:
		if chunk.Payload != nil {
			result := payloadMap(chunk.Payload)
			result["type"] = chunk.Type
			return result
		}
		return nil
	}
}

// ---------------------------------------------------------------------------
// normalizeUsage
// ---------------------------------------------------------------------------

// isV3Usage checks if usage data is in V3 format (nested objects).
func isV3Usage(usage any) bool {
	if usage == nil {
		return false
	}
	m, ok := usage.(map[string]any)
	if !ok {
		return false
	}
	inputTokens, ok := m["inputTokens"]
	if !ok {
		return false
	}
	inputMap, ok := inputTokens.(map[string]any)
	if !ok {
		return false
	}
	_, hasTotal := inputMap["total"]
	return hasTotal
}

// normalizeUsage normalizes usage from either V2 (flat) or V3 (nested) format
// to Mastra's flat format.
//
// V2 format: { inputTokens: number, outputTokens: number, totalTokens?: number }
// V3 format: { inputTokens: { total, noCache, cacheRead, cacheWrite }, outputTokens: { total, text, reasoning } }
func normalizeUsage(usage any) stream.LanguageModelUsage {
	if usage == nil {
		return stream.LanguageModelUsage{}
	}

	m, ok := usage.(map[string]any)
	if !ok {
		return stream.LanguageModelUsage{}
	}

	if isV3Usage(usage) {
		// V3 format - extract from nested structure
		inputTokensMap := mapFromAny(m["inputTokens"])
		outputTokensMap := mapFromAny(m["outputTokens"])
		inputTokens := intFromAnyV(inputTokensMap["total"])
		outputTokens := intFromAnyV(outputTokensMap["total"])
		return stream.LanguageModelUsage{
			InputTokens:      inputTokens,
			OutputTokens:     outputTokens,
			TotalTokens:      inputTokens + outputTokens,
			ReasoningTokens:  intFromAnyV(outputTokensMap["reasoning"]),
			CachedInputTokens: intFromAnyV(inputTokensMap["cacheRead"]),
		}
	}

	// V2 format - already flat
	inputTokens := intFromAnyV(m["inputTokens"])
	outputTokens := intFromAnyV(m["outputTokens"])
	totalTokens := intFromAnyV(m["totalTokens"])
	if totalTokens == 0 {
		totalTokens = inputTokens + outputTokens
	}
	return stream.LanguageModelUsage{
		InputTokens:      inputTokens,
		OutputTokens:     outputTokens,
		TotalTokens:      totalTokens,
		ReasoningTokens:  intFromAnyV(m["reasoningTokens"]),
		CachedInputTokens: intFromAnyV(m["cachedInputTokens"]),
	}
}

// ---------------------------------------------------------------------------
// normalizeFinishReason
// ---------------------------------------------------------------------------

// normalizeFinishReason normalizes finish reason from either V2/V5 (string)
// or V3/V6 (object) format to a string.
func normalizeFinishReason(finishReason any) string {
	if finishReason == nil {
		return "other"
	}

	// String format (V2/V5 or Mastra-specific)
	if reason, ok := finishReason.(string); ok {
		if reason == "tripwire" || reason == "retry" {
			return reason
		}
		if reason == "unknown" {
			return "other"
		}
		return reason
	}

	// Object format (V3/V6)
	if m, ok := finishReason.(map[string]any); ok {
		if unified, ok := m["unified"].(string); ok {
			return unified
		}
	}

	return "other"
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func payloadMap(payload any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	if m, ok := payload.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func mapFromAny(v any) map[string]any {
	if v == nil {
		return map[string]any{}
	}
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func mapFromStreamPart(value StreamPart) map[string]any {
	result := make(map[string]any)
	result["type"] = value.Type
	if value.ID != "" {
		result["id"] = value.ID
	}
	if value.Delta != "" {
		result["delta"] = value.Delta
	}
	if value.ProviderMetadata != nil {
		result["providerMetadata"] = value.ProviderMetadata
	}
	if value.ModelID != "" {
		result["modelId"] = value.ModelID
	}
	if value.Timestamp != nil {
		result["timestamp"] = value.Timestamp
	}
	if value.SourceType != "" {
		result["sourceType"] = value.SourceType
	}
	if value.URL != "" {
		result["url"] = value.URL
	}
	if value.Title != "" {
		result["title"] = value.Title
	}
	if value.Filename != "" {
		result["filename"] = value.Filename
	}
	if value.MediaType != "" {
		result["mediaType"] = value.MediaType
	}
	if value.Data != nil {
		result["data"] = value.Data
	}
	if value.ToolCallID != "" {
		result["toolCallId"] = value.ToolCallID
	}
	if value.ToolName != "" {
		result["toolName"] = value.ToolName
	}
	if value.Input != nil {
		result["input"] = value.Input
	}
	if value.Result != nil {
		result["result"] = value.Result
	}
	if value.IsError != nil {
		result["isError"] = *value.IsError
	}
	if value.ProviderExecuted != nil {
		result["providerExecuted"] = *value.ProviderExecuted
	}
	if value.FinishReason != "" {
		result["finishReason"] = value.FinishReason
	}
	if value.Usage != nil {
		result["usage"] = value.Usage
	}
	if value.Messages != nil {
		result["messages"] = value.Messages
	}
	if value.Error != nil {
		result["error"] = value.Error
	}
	if value.RawValue != nil {
		result["rawValue"] = value.RawValue
	}
	if value.Warnings != nil {
		result["warnings"] = value.Warnings
	}
	return result
}

func intFromAnyV(v any) int {
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case json.Number:
		n, _ := val.Int64()
		return int(n)
	default:
		return 0
	}
}

func orDefault(v any, def any) any {
	if v == nil {
		return def
	}
	return v
}

func boolFromAny(v any) bool {
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
