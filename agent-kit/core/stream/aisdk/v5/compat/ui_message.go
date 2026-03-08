// Ported from: packages/core/src/stream/aisdk/v5/compat/ui-message.ts
package compat

import (
	"fmt"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// UIMessage mirrors the TS UIMessage from @internal/ai-sdk-v5.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type UIMessage struct {
	ID       string `json:"id"`
	Role     string `json:"role"`
	Content  string `json:"content"`
	Metadata any    `json:"metadata,omitempty"`
}

// UIMessageChunk is a chunk emitted to the UI message stream.
// It uses a Type discriminator and optional fields depending on the type.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 internal types remain local stubs.
type UIMessageChunk struct {
	Type string `json:"type"`

	// text-start / text-delta / text-end
	ID    string `json:"id,omitempty"`
	Delta string `json:"delta,omitempty"`

	// reasoning-start / reasoning-delta / reasoning-end (same fields as text)

	// file
	MediaType string `json:"mediaType,omitempty"`
	URL       string `json:"url,omitempty"`

	// source-url
	SourceID string `json:"sourceId,omitempty"`
	Title    string `json:"title,omitempty"`
	// URL is reused for source-url

	// source-document
	Filename string `json:"filename,omitempty"`
	// MediaType and Title are reused

	// tool-input-start / tool-input-available
	ToolCallID     string `json:"toolCallId,omitempty"`
	ToolName       string `json:"toolName,omitempty"`
	InputTextDelta string `json:"inputTextDelta,omitempty"`
	Input          any    `json:"input,omitempty"`

	// tool-output-available
	Output any `json:"output,omitempty"`

	// tool-output-error
	ErrorText string `json:"errorText,omitempty"`

	// error (same ErrorText)

	// Common optional metadata fields
	ProviderMetadata map[string]any `json:"providerMetadata,omitempty"`
	ProviderExecuted *bool          `json:"providerExecuted,omitempty"`
	Dynamic          *bool          `json:"dynamic,omitempty"`

	// start/finish
	MessageMetadata any    `json:"messageMetadata,omitempty"`
	MessageID       string `json:"messageId,omitempty"`
}

// TextStreamPart mirrors the TS TextStreamPart from @internal/ai-sdk-v5.
// It represents a single part of a text stream.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V4/V5 types remain local stubs.
type TextStreamPart struct {
	Type string `json:"type"`

	// Shared
	ID               string         `json:"id,omitempty"`
	ProviderMetadata map[string]any `json:"providerMetadata,omitempty"`

	// text-delta
	Text string `json:"text,omitempty"`

	// file
	File *TextStreamPartFile `json:"file,omitempty"`

	// source
	SourceType string `json:"sourceType,omitempty"`
	URL        string `json:"url,omitempty"`
	Title      string `json:"title,omitempty"`
	MediaType  string `json:"mediaType,omitempty"`
	Filename   string `json:"filename,omitempty"`

	// tool-input-start / tool-call / tool-result / tool-error
	ToolCallID       string `json:"toolCallId,omitempty"`
	ToolName         string `json:"toolName,omitempty"`
	Delta            string `json:"delta,omitempty"`
	Input            any    `json:"input,omitempty"`
	Output           any    `json:"output,omitempty"`
	Result           any    `json:"result,omitempty"`
	ProviderExecuted *bool  `json:"providerExecuted,omitempty"`
	Dynamic          *bool  `json:"dynamic,omitempty"`

	// error
	Error any `json:"error,omitempty"`
}

// TextStreamPartFile holds file data within a TextStreamPart.
type TextStreamPartFile struct {
	MediaType string `json:"mediaType"`
	Base64    string `json:"base64"`
}

// ---------------------------------------------------------------------------
// IdGeneratorFn
// ---------------------------------------------------------------------------

// IdGeneratorFn is a function that generates unique IDs.
type IdGeneratorFn func() string

// ---------------------------------------------------------------------------
// GetResponseUIMessageId
// ---------------------------------------------------------------------------

// GetResponseUIMessageIdParams configures GetResponseUIMessageId.
type GetResponseUIMessageIdParams struct {
	OriginalMessages  []UIMessage
	ResponseMessageID any // string | IdGeneratorFn | nil
}

// GetResponseUIMessageId determines the response message ID based on
// the original messages and the provided responseMessageId.
//
// When there are no original messages (i.e. no persistence),
// the assistant message ID generation is handled on the client side.
func GetResponseUIMessageId(params GetResponseUIMessageIdParams) string {
	// No original messages means no persistence — client handles IDs
	if params.OriginalMessages == nil {
		return ""
	}

	// If the last message is from the assistant, reuse its ID
	if len(params.OriginalMessages) > 0 {
		lastMsg := params.OriginalMessages[len(params.OriginalMessages)-1]
		if lastMsg.Role == "assistant" {
			return lastMsg.ID
		}
	}

	// Use the provided responseMessageId
	switch v := params.ResponseMessageID.(type) {
	case string:
		return v
	case IdGeneratorFn:
		return v()
	case func() string:
		return v()
	default:
		return ""
	}
}

// ---------------------------------------------------------------------------
// ConvertFullStreamChunkToUIMessageStream
// ---------------------------------------------------------------------------

// ConvertFullStreamChunkToUIMessageStreamParams configures the conversion.
type ConvertFullStreamChunkToUIMessageStreamParams struct {
	Part              TextStreamPart
	MessageMetadata   any
	SendReasoning     bool
	SendSources       bool
	OnError           func(err any) string
	SendStart         bool
	SendFinish        bool
	ResponseMessageID string
}

// ConvertFullStreamChunkToUIMessageStream converts a TextStreamPart to a
// UIMessageChunk for the UI message stream.
//
// This mirrors the TS convertFullStreamChunkToUIMessageStream function with
// exhaustive switch coverage over all part types.
//
// Returns nil if the part should not be included in the UI stream.
func ConvertFullStreamChunkToUIMessageStream(params ConvertFullStreamChunkToUIMessageStreamParams) *UIMessageChunk {
	part := params.Part

	switch part.Type {
	case "text-start":
		return &UIMessageChunk{
			Type:             "text-start",
			ID:               part.ID,
			ProviderMetadata: part.ProviderMetadata,
		}

	case "text-delta":
		return &UIMessageChunk{
			Type:             "text-delta",
			ID:               part.ID,
			Delta:            part.Text,
			ProviderMetadata: part.ProviderMetadata,
		}

	case "text-end":
		return &UIMessageChunk{
			Type:             "text-end",
			ID:               part.ID,
			ProviderMetadata: part.ProviderMetadata,
		}

	case "reasoning-start":
		return &UIMessageChunk{
			Type:             "reasoning-start",
			ID:               part.ID,
			ProviderMetadata: part.ProviderMetadata,
		}

	case "reasoning-delta":
		if params.SendReasoning {
			return &UIMessageChunk{
				Type:             "reasoning-delta",
				ID:               part.ID,
				Delta:            part.Text,
				ProviderMetadata: part.ProviderMetadata,
			}
		}
		return nil

	case "reasoning-end":
		return &UIMessageChunk{
			Type:             "reasoning-end",
			ID:               part.ID,
			ProviderMetadata: part.ProviderMetadata,
		}

	case "file":
		if part.File != nil {
			return &UIMessageChunk{
				Type:      "file",
				MediaType: part.File.MediaType,
				URL:       fmt.Sprintf("data:%s;base64,%s", part.File.MediaType, part.File.Base64),
			}
		}
		return nil

	case "source":
		if params.SendSources && part.SourceType == "url" {
			return &UIMessageChunk{
				Type:             "source-url",
				SourceID:         part.ID,
				URL:              part.URL,
				Title:            part.Title,
				ProviderMetadata: part.ProviderMetadata,
			}
		}
		if params.SendSources && part.SourceType == "document" {
			return &UIMessageChunk{
				Type:             "source-document",
				SourceID:         part.ID,
				MediaType:        part.MediaType,
				Title:            part.Title,
				Filename:         part.Filename,
				ProviderMetadata: part.ProviderMetadata,
			}
		}
		return nil

	case "tool-input-start":
		return &UIMessageChunk{
			Type:             "tool-input-start",
			ToolCallID:       part.ID,
			ToolName:         part.ToolName,
			ProviderExecuted: part.ProviderExecuted,
			Dynamic:          part.Dynamic,
		}

	case "tool-input-delta":
		return &UIMessageChunk{
			Type:           "tool-input-delta",
			ToolCallID:     part.ID,
			InputTextDelta: part.Delta,
		}

	case "tool-call":
		return &UIMessageChunk{
			Type:             "tool-input-available",
			ToolCallID:       part.ToolCallID,
			ToolName:         part.ToolName,
			Input:            part.Input,
			ProviderExecuted: part.ProviderExecuted,
			ProviderMetadata: part.ProviderMetadata,
			Dynamic:          part.Dynamic,
		}

	case "tool-result":
		return &UIMessageChunk{
			Type:             "tool-output-available",
			ToolCallID:       part.ToolCallID,
			Output:           part.Output,
			ProviderExecuted: part.ProviderExecuted,
			Dynamic:          part.Dynamic,
		}

	case "tool-output":
		// tool-output is a custom mastra chunk type used in ToolStream
		if outputMap, ok := part.Output.(map[string]any); ok {
			chunk := &UIMessageChunk{}
			if t, ok := outputMap["type"].(string); ok {
				chunk.Type = t
			}
			// Spread all fields from output
			if id, ok := outputMap["id"].(string); ok {
				chunk.ID = id
			}
			if delta, ok := outputMap["delta"].(string); ok {
				chunk.Delta = delta
			}
			return chunk
		}
		return nil

	case "tool-error":
		errorText := ""
		if params.OnError != nil {
			errorText = params.OnError(part.Error)
		}
		return &UIMessageChunk{
			Type:             "tool-output-error",
			ToolCallID:       part.ToolCallID,
			ErrorText:        errorText,
			ProviderExecuted: part.ProviderExecuted,
			Dynamic:          part.Dynamic,
		}

	case "error":
		errorText := ""
		if params.OnError != nil {
			errorText = params.OnError(part.Error)
		}
		return &UIMessageChunk{
			Type:      "error",
			ErrorText: errorText,
		}

	case "start-step":
		return &UIMessageChunk{Type: "start-step"}

	case "finish-step":
		return &UIMessageChunk{Type: "finish-step"}

	case "start":
		if params.SendStart {
			chunk := &UIMessageChunk{Type: "start"}
			if params.MessageMetadata != nil {
				chunk.MessageMetadata = params.MessageMetadata
			}
			if params.ResponseMessageID != "" {
				chunk.MessageID = params.ResponseMessageID
			}
			return chunk
		}
		return nil

	case "finish":
		if params.SendFinish {
			chunk := &UIMessageChunk{Type: "finish"}
			if params.MessageMetadata != nil {
				chunk.MessageMetadata = params.MessageMetadata
			}
			return chunk
		}
		return nil

	case "abort":
		return &UIMessageChunk{Type: "abort"}

	case "tool-input-end":
		// Not included in UI message streams
		return nil

	case "raw":
		// Raw chunks are not included in UI message streams
		// as they contain provider-specific data for developer use
		return nil

	default:
		// Unknown chunk type
		return nil
	}
}
