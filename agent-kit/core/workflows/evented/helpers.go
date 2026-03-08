// Ported from: packages/core/src/workflows/evented/helpers.ts
package evented

// ---------------------------------------------------------------------------
// TripwireChunk
// ---------------------------------------------------------------------------

// TripwireChunkPayload holds the payload data for a tripwire chunk.
type TripwireChunkPayload struct {
	Reason      string `json:"reason"`
	Retry       *bool  `json:"retry,omitempty"`
	Metadata    any    `json:"metadata,omitempty"`
	ProcessorID string `json:"processorId,omitempty"`
}

// TripwireChunk represents a tripwire chunk in the stream.
// These chunks are emitted when a processor triggers a tripwire.
// TS equivalent: export interface TripwireChunk
type TripwireChunk struct {
	Type    string              `json:"type"`
	Payload TripwireChunkPayload `json:"payload"`
}

// IsTripwireChunk checks if a value is a TripwireChunk.
// TS equivalent: export function isTripwireChunk(chunk: unknown): chunk is TripwireChunk
func IsTripwireChunk(chunk any) bool {
	if chunk == nil {
		return false
	}
	m, ok := chunk.(map[string]any)
	if !ok {
		// Also check for typed TripwireChunk
		if tc, ok2 := chunk.(TripwireChunk); ok2 {
			return tc.Type == "tripwire"
		}
		if tc, ok2 := chunk.(*TripwireChunk); ok2 {
			return tc != nil && tc.Type == "tripwire"
		}
		return false
	}
	t, ok := m["type"]
	if !ok {
		return false
	}
	ts, ok := t.(string)
	if !ok {
		return false
	}
	if ts != "tripwire" {
		return false
	}
	_, hasPayload := m["payload"]
	return hasPayload
}

// TripWire is a stub for the TripWire error type.
// TODO: Replace with actual TripWire from agent/trip-wire package when ported.
type TripWire struct {
	Message     string
	Options     *TripWireOptions
	ProcessorID string
}

// TripWireOptions holds options for a TripWire.
type TripWireOptions struct {
	Retry    *bool
	Metadata any
}

// Error implements the error interface.
func (tw *TripWire) Error() string {
	return tw.Message
}

// CreateTripWireFromChunk creates a TripWire error from a tripwire chunk.
// TS equivalent: export function createTripWireFromChunk(chunk: TripwireChunk): TripWire
func CreateTripWireFromChunk(chunk TripwireChunk) *TripWire {
	reason := chunk.Payload.Reason
	if reason == "" {
		reason = "Agent tripwire triggered"
	}
	return &TripWire{
		Message: reason,
		Options: &TripWireOptions{
			Retry:    chunk.Payload.Retry,
			Metadata: chunk.Payload.Metadata,
		},
		ProcessorID: chunk.Payload.ProcessorID,
	}
}

// GetTextDeltaFromChunk extracts text delta from a stream chunk, handling V1 vs V2 differences.
//
// V1 (AI SDK v4): Uses chunk.textDelta for raw text
// V2 (AI SDK v5): Uses chunk.payload.text for normalized text
//
// TS equivalent: export function getTextDeltaFromChunk(chunk, isV2Model): string | undefined
func GetTextDeltaFromChunk(chunk map[string]any, isV2Model bool) (string, bool) {
	chunkType, _ := chunk["type"].(string)
	if chunkType != "text-delta" {
		return "", false
	}
	if isV2Model {
		payload, ok := chunk["payload"].(map[string]any)
		if !ok {
			return "", false
		}
		text, ok := payload["text"].(string)
		return text, ok
	}
	text, ok := chunk["textDelta"].(string)
	return text, ok
}

// ---------------------------------------------------------------------------
// ResolveCurrentState
// ---------------------------------------------------------------------------

// ResolveStateParams holds parameters for resolving the current workflow state.
// TS equivalent: export interface ResolveStateParams
type ResolveStateParams struct {
	// StepResult is the state from a step result (highest priority).
	StepResult any
	// StepResults is the state from all step results.
	StepResults map[string]any
	// State is the state passed directly.
	State map[string]any
}

// ResolveCurrentState resolves the current workflow state from multiple potential sources.
// Priority order: stepResult.__state > stepResults.__state > state > empty object
// TS equivalent: export function resolveCurrentState(params: ResolveStateParams): Record<string, unknown>
func ResolveCurrentState(params ResolveStateParams) map[string]any {
	// Try stepResult.__state first
	if params.StepResult != nil {
		if m, ok := params.StepResult.(map[string]any); ok {
			if state, ok := m["__state"]; ok {
				if stateMap, ok := state.(map[string]any); ok {
					return stateMap
				}
			}
		}
	}
	// Try stepResults.__state
	if params.StepResults != nil {
		if state, ok := params.StepResults["__state"]; ok {
			if stateMap, ok := state.(map[string]any); ok {
				return stateMap
			}
		}
	}
	// Try state directly
	if params.State != nil {
		return params.State
	}
	return map[string]any{}
}
