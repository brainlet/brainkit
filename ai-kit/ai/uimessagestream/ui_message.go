// Ported from: packages/ai/src/ui/ui-messages.ts (subset needed by this package)
//
// This file contains the UIMessage and UIMessagePart types needed by the
// uimessagestream package. The full ui-messages module is in the ui package;
// these are local stubs to avoid circular imports.
package uimessagestream

import (
	"encoding/json"

	aitypes "github.com/brainlet/brainkit/ai-kit/ai/types"
)

// UIMessage represents an AI SDK UI message used in the client and between
// frontend and API routes.
type UIMessage struct {
	// ID is a unique identifier for the message.
	ID string `json:"id"`

	// Role is the role of the message: "system", "user", or "assistant".
	Role string `json:"role"`

	// Metadata is optional metadata for the message.
	Metadata any `json:"metadata,omitempty"`

	// Parts contains the parts of the message for UI rendering.
	Parts []UIMessagePart `json:"parts"`
}

// DeepCopy returns a deep copy of the UIMessage by round-tripping through JSON.
func (m UIMessage) DeepCopy() UIMessage {
	data, _ := json.Marshal(m)
	var copy UIMessage
	_ = json.Unmarshal(data, &copy)
	return copy
}

// UIMessagePart represents a single part of a UIMessage.
// In TypeScript this is a discriminated union. In Go we use a struct with a
// Type field as discriminator.
type UIMessagePart struct {
	Type string `json:"type"`

	// Text part fields
	Text             string                   `json:"text,omitempty"`
	State            string                   `json:"state,omitempty"`
	ProviderMetadata aitypes.ProviderMetadata `json:"providerMetadata,omitempty"`

	// Source fields
	SourceID string `json:"sourceId,omitempty"`
	URL      string `json:"url,omitempty"`
	Title    string `json:"title,omitempty"`

	// File / source-document fields
	MediaType string `json:"mediaType,omitempty"`
	Filename  string `json:"filename,omitempty"`

	// Tool fields
	ToolCallID           string                   `json:"toolCallId,omitempty"`
	ToolName             string                   `json:"toolName,omitempty"`
	Input                any                      `json:"input,omitempty"`
	Output               any                      `json:"output,omitempty"`
	ErrorText            string                   `json:"errorText,omitempty"`
	RawInput             any                      `json:"rawInput,omitempty"`
	ProviderExecuted     *bool                    `json:"providerExecuted,omitempty"`
	Preliminary          *bool                    `json:"preliminary,omitempty"`
	CallProviderMetadata aitypes.ProviderMetadata `json:"callProviderMetadata,omitempty"`
	Dynamic              *bool                    `json:"dynamic,omitempty"`
	Approval             *ToolApproval            `json:"approval,omitempty"`

	// Data part fields
	ID   string `json:"id,omitempty"`
	Data any    `json:"data,omitempty"`
}

// ToolApproval holds approval information for tool invocations.
type ToolApproval struct {
	ID       string `json:"id"`
	Approved *bool  `json:"approved,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

// IdGenerator is a function that generates unique IDs.
// Corresponds to IdGenerator from @ai-sdk/provider-utils.
type IdGenerator func() string
