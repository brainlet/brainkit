// Ported from: packages/provider/src/language-model/v3/language-model-v3-stream-part.ts
package languagemodel

import "github.com/brainlet/brainkit/ai-kit/provider/shared"

// StreamPart is a sealed interface representing parts of a language model stream.
// Implementations cover text blocks, reasoning blocks, tool operations, files,
// sources, metadata, and control signals.
//
// Types that also implement StreamPart: File, SourceURL, SourceDocument,
// ToolApprovalRequest, ToolCall, ToolResult.
type StreamPart interface {
	isStreamPart()
}

// --- Text blocks ---

// StreamPartTextStart signals the start of a text block.
type StreamPartTextStart struct {
	ID               string
	ProviderMetadata shared.ProviderMetadata
}

func (StreamPartTextStart) isStreamPart() {}

// StreamPartTextDelta contains a text delta within a text block.
type StreamPartTextDelta struct {
	ID               string
	Delta            string
	ProviderMetadata shared.ProviderMetadata
}

func (StreamPartTextDelta) isStreamPart() {}

// StreamPartTextEnd signals the end of a text block.
type StreamPartTextEnd struct {
	ID               string
	ProviderMetadata shared.ProviderMetadata
}

func (StreamPartTextEnd) isStreamPart() {}

// --- Reasoning blocks ---

// StreamPartReasoningStart signals the start of a reasoning block.
type StreamPartReasoningStart struct {
	ID               string
	ProviderMetadata shared.ProviderMetadata
}

func (StreamPartReasoningStart) isStreamPart() {}

// StreamPartReasoningDelta contains a reasoning delta within a reasoning block.
type StreamPartReasoningDelta struct {
	ID               string
	Delta            string
	ProviderMetadata shared.ProviderMetadata
}

func (StreamPartReasoningDelta) isStreamPart() {}

// StreamPartReasoningEnd signals the end of a reasoning block.
type StreamPartReasoningEnd struct {
	ID               string
	ProviderMetadata shared.ProviderMetadata
}

func (StreamPartReasoningEnd) isStreamPart() {}

// --- Tool input blocks ---

// StreamPartToolInputStart signals the start of tool input.
type StreamPartToolInputStart struct {
	ID               string
	ToolName         string
	ProviderMetadata shared.ProviderMetadata
	ProviderExecuted *bool
	Dynamic          *bool
	Title            *string
}

func (StreamPartToolInputStart) isStreamPart() {}

// StreamPartToolInputDelta contains a tool input delta.
type StreamPartToolInputDelta struct {
	ID               string
	Delta            string
	ProviderMetadata shared.ProviderMetadata
}

func (StreamPartToolInputDelta) isStreamPart() {}

// StreamPartToolInputEnd signals the end of tool input.
type StreamPartToolInputEnd struct {
	ID               string
	ProviderMetadata shared.ProviderMetadata
}

func (StreamPartToolInputEnd) isStreamPart() {}

// --- Stream lifecycle ---

// StreamPartStreamStart is the stream start event with warnings for the call.
type StreamPartStreamStart struct {
	Warnings []shared.Warning
}

func (StreamPartStreamStart) isStreamPart() {}

// StreamPartResponseMetadata contains metadata for the response.
// Sent as a separate stream part so it can be sent once available.
type StreamPartResponseMetadata struct {
	ResponseMetadata
}

func (StreamPartResponseMetadata) isStreamPart() {}

// StreamPartFinish contains metadata available after the stream is finished.
type StreamPartFinish struct {
	Usage            Usage
	FinishReason     FinishReason
	ProviderMetadata shared.ProviderMetadata
}

func (StreamPartFinish) isStreamPart() {}

// StreamPartRaw contains raw chunks if enabled.
type StreamPartRaw struct {
	RawValue any
}

func (StreamPartRaw) isStreamPart() {}

// StreamPartError contains error parts that are streamed,
// allowing for multiple errors.
type StreamPartError struct {
	Error any
}

func (StreamPartError) isStreamPart() {}
