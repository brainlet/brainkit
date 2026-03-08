// Ported from: packages/core/src/agent/message-list/state/types.ts
package state

import (
	"time"
)

// MessageSource represents the origin of a message.
type MessageSource string

const (
	MessageSourceMemory   MessageSource = "memory"
	MessageSourceResponse MessageSource = "response"
	MessageSourceInput    MessageSource = "input"
	MessageSourceSystem   MessageSource = "system"
	MessageSourceContext  MessageSource = "context"
	// Deprecated: use MessageSourceInput instead.
	MessageSourceUser MessageSource = "user"
)

// MemoryInfo holds thread and resource identifiers for memory-linked messages.
type MemoryInfo struct {
	ThreadID   string `json:"threadId"`
	ResourceID string `json:"resourceId,omitempty"`
}

// MastraMessageShared contains fields common to all Mastra messages.
type MastraMessageShared struct {
	ID         string    `json:"id"`
	Role       string    `json:"role"` // "user" | "assistant" | "system"
	CreatedAt  time.Time `json:"createdAt"`
	ThreadID   string    `json:"threadId,omitempty"`
	ResourceID string    `json:"resourceId,omitempty"`
	Type       string    `json:"type,omitempty"`
}

// ProviderMetadata is a flexible map for provider-specific metadata.
// TODO: In TS this maps to AIV5Type.ProviderMetadata — a Record<string, Record<string, unknown>>.
type ProviderMetadata map[string]map[string]any

// ToolInvocation represents a tool call or result.
// TODO: Stub — in TS this comes from @internal/ai-sdk-v4 ToolInvocation type.
type ToolInvocation struct {
	State      string         `json:"state"` // "call" | "partial-call" | "result"
	ToolCallID string         `json:"toolCallId"`
	ToolName   string         `json:"toolName"`
	Args       map[string]any `json:"args,omitempty"`
	Result     any            `json:"result,omitempty"`
	Step       *int           `json:"step,omitempty"`
}

// ExperimentalAttachment represents a file attachment on a message.
// TODO: Stub — in TS this comes from UIMessage['experimental_attachments'][number].
type ExperimentalAttachment struct {
	URL         string `json:"url"`
	ContentType string `json:"contentType,omitempty"`
}

// MastraMessagePart represents a single content part within a message.
// This is a union type covering text, tool-invocation, reasoning, file, source, step-start, and data-* parts.
// TODO: In TS this is a discriminated union of UIMessageV4 parts + AIV5 DataUIPart.
type MastraMessagePart struct {
	Type string `json:"type"` // "text" | "tool-invocation" | "reasoning" | "file" | "source" | "step-start" | "data-*"

	// Text part fields
	Text string `json:"text,omitempty"`

	// Tool invocation part fields
	ToolInvocation *ToolInvocation `json:"toolInvocation,omitempty"`

	// Reasoning part fields
	Reasoning string            `json:"reasoning,omitempty"`
	Details   []ReasoningDetail `json:"details,omitempty"`

	// File part fields
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
	Filename string `json:"filename,omitempty"`

	// Source part fields
	Source *SourceInfo `json:"source,omitempty"`

	// Provider metadata (AIV5 extension)
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`

	// ProviderExecuted indicates if the tool was executed by the provider (e.g., Anthropic web_search).
	ProviderExecuted *bool `json:"providerExecuted,omitempty"`

	// Metadata for arbitrary part-level metadata
	Metadata map[string]any `json:"metadata,omitempty"`

	// DataPayload holds the payload for data-* parts (custom streaming data).
	DataPayload any `json:"dataPayload,omitempty"`
}

// ReasoningDetail is one detail entry inside a reasoning part.
type ReasoningDetail struct {
	Type      string `json:"type"` // "text" | "redacted"
	Text      string `json:"text,omitempty"`
	Signature string `json:"signature,omitempty"`
	Data      string `json:"data,omitempty"`
}

// SourceInfo holds source metadata for a source part.
type SourceInfo struct {
	URL              string           `json:"url"`
	SourceType       string           `json:"sourceType,omitempty"`
	ID               string           `json:"id"`
	Title            string           `json:"title,omitempty"`
	ProviderMetadata ProviderMetadata `json:"providerMetadata,omitempty"`
}

// UIMessageV4Part is a V4-compatible part type (excludes DataUIPart which V4 doesn't support).
// In Go we use the same struct as MastraMessagePart but conceptually exclude data-* types.
type UIMessageV4Part = MastraMessagePart

// MastraMessageContentV2 is the V2 message content format (format 2 === UIMessage in AI SDK v4).
type MastraMessageContentV2 struct {
	Format                  int                      `json:"format"` // always 2
	Parts                   []MastraMessagePart      `json:"parts"`
	ExperimentalAttachments []ExperimentalAttachment  `json:"experimental_attachments,omitempty"`
	Content                 string                   `json:"content,omitempty"`
	ToolInvocations         []ToolInvocation         `json:"toolInvocations,omitempty"`
	Reasoning               string                   `json:"reasoning,omitempty"`
	Annotations             []any                    `json:"annotations,omitempty"`
	Metadata                map[string]any           `json:"metadata,omitempty"`
	ProviderMetadata        ProviderMetadata         `json:"providerMetadata,omitempty"`
}

// MastraDBMessage is the primary message type stored in the database (maps to AI SDK V4 UIMessage).
type MastraDBMessage struct {
	MastraMessageShared
	Content MastraMessageContentV2 `json:"content"`
}

// MastraMessageV1 is the legacy V1 message format.
type MastraMessageV1 struct {
	ID           string    `json:"id"`
	Content      any       `json:"content"` // string or []CoreMessageContent (AIV4 content parts)
	Role         string    `json:"role"`     // "system" | "user" | "assistant" | "tool"
	CreatedAt    time.Time `json:"createdAt"`
	ThreadID     string    `json:"threadId,omitempty"`
	ResourceID   string    `json:"resourceId,omitempty"`
	ToolCallIDs  []string  `json:"toolCallIds,omitempty"`
	ToolCallArgs []map[string]any `json:"toolCallArgs,omitempty"`
	ToolNames    []string  `json:"toolNames,omitempty"`
	Type         string    `json:"type"` // "text" | "tool-call" | "tool-result"
}

// UIMessageWithMetadata extends UIMessageV4 with optional metadata.
// TODO: Stub — in TS this extends the AI SDK V4 UIMessage type.
type UIMessageWithMetadata struct {
	ID                      string                   `json:"id"`
	Role                    string                   `json:"role"`
	Content                 any                      `json:"content"` // string or content array
	CreatedAt               time.Time                `json:"createdAt,omitempty"`
	Parts                   []MastraMessagePart      `json:"parts,omitempty"`
	ExperimentalAttachments []ExperimentalAttachment  `json:"experimental_attachments,omitempty"`
	ToolInvocations         []ToolInvocation         `json:"toolInvocations,omitempty"`
	Reasoning               any                      `json:"reasoning,omitempty"`
	Annotations             []any                    `json:"annotations,omitempty"`
	Metadata                map[string]any           `json:"metadata,omitempty"`
}
