// Ported from: packages/core/src/agent/trip-wire.ts
package agent

import (
	"github.com/google/uuid"

	"github.com/brainlet/brainkit/agent-kit/core/llm/model"
)

// ---------------------------------------------------------------------------
// TripWireOptions
// ---------------------------------------------------------------------------

// TripWireOptions controls how a tripwire should be handled.
type TripWireOptions struct {
	// Retry indicates the agent should retry with the tripwire reason as feedback.
	// The failed response will be added to message history along with the reason.
	Retry bool `json:"retry,omitempty"`
	// Metadata is strongly typed metadata from the processor.
	// This allows processors to pass structured information about what triggered the tripwire.
	Metadata any `json:"metadata,omitempty"`
}

// ---------------------------------------------------------------------------
// TripWire
// ---------------------------------------------------------------------------

// TripWire is a custom error type for aborting processing with optional retry and metadata.
//
// When returned from a processor, it signals that processing should stop.
//   - Retry: true  -> The agent will retry with the reason as feedback
//   - Metadata     -> Strongly typed data about what triggered the tripwire
type TripWire struct {
	// Reason is the human-readable explanation (also used as the error message).
	Reason string `json:"reason"`
	// Options controls how the tripwire should be handled.
	Options TripWireOptions `json:"options"`
	// ProcessorID is the optional ID of the processor that created this tripwire.
	ProcessorID string `json:"processorId,omitempty"`
}

// NewTripWire creates a new TripWire with the given reason and options.
func NewTripWire(reason string, opts *TripWireOptions, processorID string) *TripWire {
	tw := &TripWire{
		Reason:      reason,
		ProcessorID: processorID,
	}
	if opts != nil {
		tw.Options = *opts
	}
	return tw
}

// Error implements the error interface.
func (tw *TripWire) Error() string {
	return tw.Reason
}

// ---------------------------------------------------------------------------
// TripwireData
// ---------------------------------------------------------------------------

// TripwireData is the data passed to GetModelOutputForTripwire.
type TripwireData struct {
	Reason      string `json:"reason"`
	Retry       bool   `json:"retry,omitempty"`
	Metadata    any    `json:"metadata,omitempty"`
	ProcessorID string `json:"processorId,omitempty"`
}

// ---------------------------------------------------------------------------
// GetModelOutputForTripwireParams
// ---------------------------------------------------------------------------

// GetModelOutputForTripwireParams holds the parameters for GetModelOutputForTripwire.
type GetModelOutputForTripwireParams struct {
	Tripwire    TripwireData
	RunID       string
	Options     InnerAgentExecutionOptions
	Model       MastraLanguageModel
	MessageList *MessageListStub
	ObservabilityContext
}

// ---------------------------------------------------------------------------
// MastraLanguageModel (re-exported from llm/model)
// ---------------------------------------------------------------------------

// MastraLanguageModel is re-exported from llm/model.
type MastraLanguageModel = model.MastraLanguageModel

// ---------------------------------------------------------------------------
// MessageListStub (local stub)
// ---------------------------------------------------------------------------

// MessageListStub is an alias for MessageList (which is now wired to messagelist.MessageList).
type MessageListStub = MessageList

// ---------------------------------------------------------------------------
// ChunkFrom
// ---------------------------------------------------------------------------

// ChunkFrom identifies the origin of a stream chunk.
type ChunkFrom string

const (
	// ChunkFromAgent indicates the chunk originates from an agent.
	ChunkFromAgent ChunkFrom = "agent"
)

// ---------------------------------------------------------------------------
// TripwireChunk
// ---------------------------------------------------------------------------

// TripwireChunk represents a tripwire event in the model output stream.
type TripwireChunk struct {
	Type    string         `json:"type"`
	RunID   string         `json:"runId"`
	From    ChunkFrom      `json:"from"`
	Payload TripwireData   `json:"payload"`
}

// ---------------------------------------------------------------------------
// GetModelOutputForTripwire
// ---------------------------------------------------------------------------

// GetModelOutputForTripwire creates a model output that represents a tripwire event.
// In TypeScript this returns a MastraModelOutput wrapping a ReadableStream that
// emits a single tripwire chunk. In Go we return the structured chunk data and
// metadata that the caller can use to construct the appropriate output.
//
// TODO: Return a proper MastraModelOutput once the stream package is ported.
func GetModelOutputForTripwire(params GetModelOutputForTripwireParams) (*TripwireModelOutput, error) {
	chunk := TripwireChunk{
		Type:  "tripwire",
		RunID: params.RunID,
		From:  ChunkFromAgent,
		Payload: TripwireData{
			Reason:      params.Tripwire.Reason,
			Retry:       params.Tripwire.Retry,
			Metadata:    params.Tripwire.Metadata,
			ProcessorID: params.Tripwire.ProcessorID,
		},
	}

	messageID := uuid.New().String()

	return &TripwireModelOutput{
		Chunk:     chunk,
		MessageID: messageID,
		ModelInfo: ModelInfo{
			ModelID:  params.Model.ModelID(),
			Provider: params.Model.Provider(),
			Version:  params.Model.SpecificationVersion(),
		},
	}, nil
}

// TripwireModelOutput represents the output from a tripwire event.
// TODO: Replace with MastraModelOutput once the stream package is ported.
type TripwireModelOutput struct {
	Chunk     TripwireChunk `json:"chunk"`
	MessageID string        `json:"messageId"`
	ModelInfo ModelInfo     `json:"modelInfo"`
}

// ModelInfo holds basic model identification metadata.
type ModelInfo struct {
	ModelID  string `json:"modelId"`
	Provider string `json:"provider"`
	Version  string `json:"version,omitempty"`
}
