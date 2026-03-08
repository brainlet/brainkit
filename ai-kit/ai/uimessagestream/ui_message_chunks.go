// Ported from: packages/ai/src/ui-message-stream/ui-message-chunks.ts
package uimessagestream

import (
	"encoding/json"
	"strings"

	aitypes "github.com/brainlet/brainkit/ai-kit/ai/types"
)

// UIMessageChunk represents any chunk that can appear in a UI message stream.
// In TypeScript this is a discriminated union on the "type" field.
// In Go we use a struct with a Type discriminator and optional fields.
type UIMessageChunk struct {
	Type string `json:"type"`

	// Common fields
	ID    string `json:"id,omitempty"`
	Delta string `json:"delta,omitempty"`

	// Error
	ErrorText string `json:"errorText,omitempty"`

	// Tool fields
	ToolCallID       string `json:"toolCallId,omitempty"`
	ToolName         string `json:"toolName,omitempty"`
	InputTextDelta   string `json:"inputTextDelta,omitempty"`
	Input            any    `json:"input,omitempty"`
	Output           any    `json:"output,omitempty"`
	ProviderExecuted *bool  `json:"providerExecuted,omitempty"`
	Dynamic          *bool  `json:"dynamic,omitempty"`
	Preliminary      *bool  `json:"preliminary,omitempty"`
	Title            string `json:"title,omitempty"`

	// Approval
	ApprovalID string `json:"approvalId,omitempty"`

	// Source fields
	SourceID  string `json:"sourceId,omitempty"`
	URL       string `json:"url,omitempty"`
	MediaType string `json:"mediaType,omitempty"`
	Filename  string `json:"filename,omitempty"`

	// Data chunk fields
	Data      any  `json:"data,omitempty"`
	Transient *bool `json:"transient,omitempty"`

	// Start/Finish fields
	MessageID       string `json:"messageId,omitempty"`
	MessageMetadata any    `json:"messageMetadata,omitempty"`
	FinishReason    string `json:"finishReason,omitempty"`

	// Abort
	Reason string `json:"reason,omitempty"`

	// Provider metadata (shared across many chunk types)
	ProviderMetadata aitypes.ProviderMetadata `json:"providerMetadata,omitempty"`
}

// IsDataUIMessageChunk checks whether the chunk is a data chunk (type starts with "data-").
func IsDataUIMessageChunk(chunk UIMessageChunk) bool {
	return strings.HasPrefix(chunk.Type, "data-")
}

// MarshalUIMessageChunk serializes a UIMessageChunk to JSON.
func MarshalUIMessageChunk(chunk UIMessageChunk) ([]byte, error) {
	return json.Marshal(chunk)
}

// UnmarshalUIMessageChunk deserializes JSON into a UIMessageChunk.
func UnmarshalUIMessageChunk(data []byte) (UIMessageChunk, error) {
	var chunk UIMessageChunk
	err := json.Unmarshal(data, &chunk)
	return chunk, err
}
