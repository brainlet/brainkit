// Ported from: packages/core/src/llm/model/model.ts (adapter layer for ai-kit integration)
//
// This file provides adapter types and conversion helpers to bridge the gap between
// agent-kit's model types (LanguageModelV1, CoreMessage, ToolSet, etc.) and ai-kit's
// generatetext/generateobject function signatures.
package model

import (
	"context"
	"fmt"

	genobj "github.com/brainlet/brainkit/ai-kit/ai/generateobject"
	gentext "github.com/brainlet/brainkit/ai-kit/ai/generatetext"
)

// ---------------------------------------------------------------------------
// Adapter: agent-kit LanguageModelV1 → ai-kit generatetext.LanguageModel
// ---------------------------------------------------------------------------

// modelAdapterForGenText adapts agent-kit's LanguageModelV1 to ai-kit's
// generatetext.LanguageModel interface (Provider() + ModelID()).
type modelAdapterForGenText struct {
	model LanguageModelV1
}

// Provider implements generatetext.LanguageModel.
func (a *modelAdapterForGenText) Provider() string { return a.model.Provider() }

// ModelID implements generatetext.LanguageModel.
func (a *modelAdapterForGenText) ModelID() string { return a.model.ModelID() }

// ---------------------------------------------------------------------------
// Adapter: agent-kit LanguageModelV1 → ai-kit generateobject.LanguageModel
// ---------------------------------------------------------------------------

// modelAdapterForGenObj adapts agent-kit's LanguageModelV1 to ai-kit's
// generateobject.LanguageModel interface (Provider() + ModelID() + DoGenerate()).
type modelAdapterForGenObj struct {
	model LanguageModelV1
}

// Provider implements generateobject.LanguageModel.
func (a *modelAdapterForGenObj) Provider() string { return a.model.Provider() }

// ModelID implements generateobject.LanguageModel.
func (a *modelAdapterForGenObj) ModelID() string { return a.model.ModelID() }

// DoGenerate implements generateobject.LanguageModel.
// It attempts to delegate to the underlying model if it supports a compatible
// DoGenerate method via type assertion. This follows the TS pattern where
// LanguageModelV3 has DoGenerate(LanguageModelV3CallOptions).
func (a *modelAdapterForGenObj) DoGenerate(ctx context.Context, opts genobj.DoGenerateObjectOptions) (*genobj.DoGenerateObjectResult, error) {
	// Try LanguageModelV3 (Mastra's V3 interface has DoGenerate).
	type doGeneratorV3 interface {
		DoGenerate(options LanguageModelV3CallOptions) (LanguageModelV3StreamResult, error)
	}
	if dg, ok := a.model.(doGeneratorV3); ok {
		callOpts := LanguageModelV3CallOptions{
			ProviderOptions: flattenProviderOptions(opts.ProviderOptions),
		}
		result, err := dg.DoGenerate(callOpts)
		if err != nil {
			return nil, err
		}
		// Extract text from the stream result if possible.
		text := ""
		if result.Stream != nil {
			if s, ok := result.Stream.(string); ok {
				text = s
			}
		}
		return &genobj.DoGenerateObjectResult{
			Text: text,
		}, nil
	}

	// Try LanguageModelV2 as fallback.
	type doGeneratorV2 interface {
		DoGenerate(options LanguageModelV2CallOptions) (LanguageModelV2StreamResult, error)
	}
	if dg, ok := a.model.(doGeneratorV2); ok {
		callOpts := LanguageModelV2CallOptions{
			ProviderOptions: flattenProviderOptions(opts.ProviderOptions),
		}
		result, err := dg.DoGenerate(callOpts)
		if err != nil {
			return nil, err
		}
		text := ""
		if result.Stream != nil {
			if s, ok := result.Stream.(string); ok {
				text = s
			}
		}
		return &genobj.DoGenerateObjectResult{
			Text: text,
		}, nil
	}

	return nil, fmt.Errorf("underlying model does not support DoGenerate; model type: %T", a.model)
}

// ---------------------------------------------------------------------------
// Adapter: agent-kit LanguageModelV1 → ai-kit generateobject.StreamLanguageModel
// ---------------------------------------------------------------------------

// modelAdapterForStreamObj adapts agent-kit's LanguageModelV1 to ai-kit's
// generateobject.StreamLanguageModel interface (Provider() + ModelID() + DoStream()).
type modelAdapterForStreamObj struct {
	model LanguageModelV1
}

// Provider implements generateobject.StreamLanguageModel.
func (a *modelAdapterForStreamObj) Provider() string { return a.model.Provider() }

// ModelID implements generateobject.StreamLanguageModel.
func (a *modelAdapterForStreamObj) ModelID() string { return a.model.ModelID() }

// DoStream implements generateobject.StreamLanguageModel.
// It attempts to delegate to the underlying model if it supports a compatible
// DoStream method via type assertion.
func (a *modelAdapterForStreamObj) DoStream(ctx context.Context, opts genobj.DoStreamObjectOptions) (<-chan genobj.StreamChunk, error) {
	// Try LanguageModelV3 (Mastra's V3 interface has DoStream).
	type doStreamerV3 interface {
		DoStream(options LanguageModelV3CallOptions) (LanguageModelV3StreamResult, error)
	}
	if ds, ok := a.model.(doStreamerV3); ok {
		callOpts := LanguageModelV3CallOptions{
			ProviderOptions: flattenProviderOptions(opts.ProviderOptions),
		}
		result, err := ds.DoStream(callOpts)
		if err != nil {
			return nil, err
		}
		// Wrap the stream result into a channel of StreamChunks.
		ch := make(chan genobj.StreamChunk, 1)
		go func() {
			defer close(ch)
			// If the stream is a string, emit it as a single text-delta then finish.
			if s, ok := result.Stream.(string); ok {
				ch <- genobj.StreamChunk{
					Type:      "text-delta",
					TextDelta: s,
				}
			}
			ch <- genobj.StreamChunk{
				Type:         "finish",
				FinishReason: "stop",
			}
		}()
		return ch, nil
	}

	// Try LanguageModelV2 as fallback.
	type doStreamerV2 interface {
		DoStream(options LanguageModelV2CallOptions) (LanguageModelV2StreamResult, error)
	}
	if ds, ok := a.model.(doStreamerV2); ok {
		callOpts := LanguageModelV2CallOptions{
			ProviderOptions: flattenProviderOptions(opts.ProviderOptions),
		}
		result, err := ds.DoStream(callOpts)
		if err != nil {
			return nil, err
		}
		ch := make(chan genobj.StreamChunk, 1)
		go func() {
			defer close(ch)
			if s, ok := result.Stream.(string); ok {
				ch <- genobj.StreamChunk{
					Type:      "text-delta",
					TextDelta: s,
				}
			}
			ch <- genobj.StreamChunk{
				Type:         "finish",
				FinishReason: "stop",
			}
		}()
		return ch, nil
	}

	return nil, fmt.Errorf("underlying model does not support DoStream; model type: %T", a.model)
}

// ---------------------------------------------------------------------------
// Message conversion helpers
// ---------------------------------------------------------------------------

// convertToGenTextMessages converts agent-kit CoreMessages to ai-kit ModelMessages.
// Both share the same structural shape (Role string, Content any).
func convertToGenTextMessages(messages []CoreMessage) []gentext.ModelMessage {
	result := make([]gentext.ModelMessage, len(messages))
	for i, m := range messages {
		result[i] = gentext.ModelMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	return result
}

// convertMessagesToPrompt converts a slice of CoreMessages into a single prompt string
// for use with generateobject/streamobject which take a prompt string.
func convertMessagesToPrompt(messages []CoreMessage) string {
	var prompt string
	for _, m := range messages {
		if s, ok := m.Content.(string); ok {
			if prompt != "" {
				prompt += "\n"
			}
			prompt += s
		}
	}
	return prompt
}

// flattenProviderOptions converts map[string]map[string]any to map[string]any
// by merging the nested maps. This bridges the type difference between ai-kit's
// generateobject (map[string]map[string]any) and agent-kit's call options (map[string]any).
func flattenProviderOptions(opts map[string]map[string]any) map[string]any {
	if opts == nil {
		return nil
	}
	result := make(map[string]any, len(opts))
	for k, v := range opts {
		result[k] = v
	}
	return result
}
