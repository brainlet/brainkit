// Ported from: packages/core/src/llm/model/aisdk/v6/model.ts (conversion helpers)
//
// This file contains conversion functions between ai-kit types
// (brainlink/experiments/ai-kit/provider/languagemodel) and agent-kit's
// internal aisdk types used by Mastra's streaming architecture.
package v6

import (
	"github.com/brainlet/brainkit/agent-kit/core/llm/model/aisdk"
	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// toAIKitCallOptions converts Mastra-level call options to ai-kit CallOptions.
func toAIKitCallOptions(opts LanguageModelV3CallOptions) lm.CallOptions {
	var callOpts lm.CallOptions
	if opts.ProviderOptions != nil {
		// ProviderOptions in Mastra is map[string]any; ai-kit expects
		// map[string]map[string]any. We convert by type-asserting inner values.
		provOpts := make(map[string]map[string]any)
		for k, v := range opts.ProviderOptions {
			if m, ok := v.(map[string]any); ok {
				provOpts[k] = m
			}
		}
		callOpts.ProviderOptions = provOpts
	}
	return callOpts
}

// aiKitGenerateResultToAisdk converts an ai-kit GenerateResult to the
// internal aisdk.GenerateResult used by CreateStreamFromGenerateResult.
func aiKitGenerateResultToAisdk(result lm.GenerateResult) *aisdk.GenerateResult {
	content := make([]map[string]any, len(result.Content))
	for i, c := range result.Content {
		content[i] = contentToMap(c)
	}

	warnings := make([]any, len(result.Warnings))
	for i, w := range result.Warnings {
		warnings[i] = w
	}

	var response *aisdk.GenerateResultResponse
	if result.Response != nil {
		response = &aisdk.GenerateResultResponse{
			ID:      derefString(result.Response.ID),
			ModelID: derefString(result.Response.ModelID),
		}
		if result.Response.Timestamp != nil {
			response.Timestamp = *result.Response.Timestamp
		}
	}

	return &aisdk.GenerateResult{
		Warnings:         warnings,
		Response:         response,
		Content:          content,
		FinishReason:     result.FinishReason,
		Usage:            result.Usage,
		ProviderMetadata: result.ProviderMetadata,
	}
}

// aiKitGenerateResponseToStreamResponse converts an ai-kit
// GenerateResultResponse to the local StreamResultResponse type.
func aiKitGenerateResponseToStreamResponse(resp *lm.GenerateResultResponse) *StreamResultResponse {
	if resp == nil {
		return nil
	}
	sr := &StreamResultResponse{
		ID:      derefString(resp.ID),
		ModelID: derefString(resp.ModelID),
	}
	if resp.Timestamp != nil {
		sr.Timestamp = *resp.Timestamp
	}
	return sr
}

// convertStreamPartsToEvents converts an ai-kit StreamPart channel to an
// aisdk.StreamEvent channel for Mastra's streaming architecture.
func convertStreamPartsToEvents(input <-chan lm.StreamPart) <-chan aisdk.StreamEvent {
	output := make(chan aisdk.StreamEvent)
	go func() {
		defer close(output)
		for part := range input {
			event := convertStreamPart(part)
			if event.Type != "" {
				output <- event
			}
		}
	}()
	return output
}

// convertStreamPart converts a single ai-kit StreamPart to an aisdk.StreamEvent.
func convertStreamPart(part lm.StreamPart) aisdk.StreamEvent {
	switch p := part.(type) {
	// Stream lifecycle
	case lm.StreamPartStreamStart:
		warnings := make([]any, len(p.Warnings))
		for i, w := range p.Warnings {
			warnings[i] = w
		}
		return aisdk.StreamEvent{Type: "stream-start", Warnings: warnings}

	case lm.StreamPartResponseMetadata:
		return aisdk.StreamEvent{
			Type:      "response-metadata",
			ID:        derefString(p.ID),
			ModelID:   derefString(p.ModelID),
			Timestamp: p.Timestamp,
		}

	case lm.StreamPartFinish:
		return aisdk.StreamEvent{
			Type:             "finish",
			FinishReason:     p.FinishReason,
			Usage:            p.Usage,
			ProviderMetadata: p.ProviderMetadata,
		}

	// Text blocks
	case lm.StreamPartTextStart:
		return aisdk.StreamEvent{Type: "text-start", ID: p.ID, ProviderMetadata: p.ProviderMetadata}
	case lm.StreamPartTextDelta:
		return aisdk.StreamEvent{Type: "text-delta", ID: p.ID, Delta: p.Delta, ProviderMetadata: p.ProviderMetadata}
	case lm.StreamPartTextEnd:
		return aisdk.StreamEvent{Type: "text-end", ID: p.ID, ProviderMetadata: p.ProviderMetadata}

	// Reasoning blocks
	case lm.StreamPartReasoningStart:
		return aisdk.StreamEvent{Type: "reasoning-start", ID: p.ID, ProviderMetadata: p.ProviderMetadata}
	case lm.StreamPartReasoningDelta:
		return aisdk.StreamEvent{Type: "reasoning-delta", ID: p.ID, Delta: p.Delta, ProviderMetadata: p.ProviderMetadata}
	case lm.StreamPartReasoningEnd:
		return aisdk.StreamEvent{Type: "reasoning-end", ID: p.ID, ProviderMetadata: p.ProviderMetadata}

	// Tool input blocks
	case lm.StreamPartToolInputStart:
		return aisdk.StreamEvent{Type: "tool-input-start", ID: p.ID, ToolName: p.ToolName}
	case lm.StreamPartToolInputDelta:
		return aisdk.StreamEvent{Type: "tool-input-delta", ID: p.ID, Delta: p.Delta}
	case lm.StreamPartToolInputEnd:
		return aisdk.StreamEvent{Type: "tool-input-end", ID: p.ID}

	// Content types that also implement StreamPart
	case lm.File:
		return aisdk.StreamEvent{Type: "file", MediaType: p.MediaType, Data: p.Data}
	case lm.ToolCall:
		return aisdk.StreamEvent{Type: "tool-call", ID: p.ToolCallID, ToolName: p.ToolName, Delta: p.Input}
	case lm.ToolResult:
		return aisdk.StreamEvent{Type: "tool-result", ID: p.ToolCallID, Delta: p.Result}
	case lm.SourceURL:
		return aisdk.StreamEvent{
			Type:             "source",
			SourceType:       "url",
			ID:               p.ID,
			URL:              p.URL,
			Title:            derefString(p.Title),
			ProviderMetadata: p.ProviderMetadata,
		}
	case lm.SourceDocument:
		return aisdk.StreamEvent{
			Type:       "source",
			SourceType: "document",
			ID:         p.ID,
			Title:      p.Title,
			MediaType:  p.MediaType,
			Filename:   derefString(p.Filename),
		}

	// Raw and error
	case lm.StreamPartRaw:
		return aisdk.StreamEvent{Type: "raw", Data: p.RawValue}
	case lm.StreamPartError:
		return aisdk.StreamEvent{Type: "error", Data: p.Error}

	default:
		return aisdk.StreamEvent{}
	}
}

// contentToMap converts an ai-kit Content interface to a map[string]any
// for use with aisdk.CreateStreamFromGenerateResult.
func contentToMap(c lm.Content) map[string]any {
	switch v := c.(type) {
	case lm.Text:
		return map[string]any{
			"type":             "text",
			"text":             v.Text,
			"providerMetadata": v.ProviderMetadata,
		}
	case lm.ToolCall:
		return map[string]any{
			"type":       "tool-call",
			"toolCallId": v.ToolCallID,
			"toolName":   v.ToolName,
			"input":      v.Input,
		}
	case lm.ToolResult:
		return map[string]any{
			"type":       "tool-result",
			"toolCallId": v.ToolCallID,
			"toolName":   v.ToolName,
			"result":     v.Result,
		}
	case lm.Reasoning:
		return map[string]any{
			"type":             "reasoning",
			"text":             v.Text,
			"providerMetadata": v.ProviderMetadata,
		}
	case lm.File:
		return map[string]any{
			"type":      "file",
			"mediaType": v.MediaType,
			"data":      v.Data,
		}
	case lm.SourceURL:
		return map[string]any{
			"type":             "source",
			"sourceType":       "url",
			"id":               v.ID,
			"url":              v.URL,
			"title":            derefString(v.Title),
			"providerMetadata": v.ProviderMetadata,
		}
	case lm.SourceDocument:
		return map[string]any{
			"type":       "source",
			"sourceType": "document",
			"id":         v.ID,
			"title":      v.Title,
			"mediaType":  v.MediaType,
		}
	default:
		return map[string]any{"type": "unknown"}
	}
}

// derefString safely dereferences a *string, returning "" if nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
