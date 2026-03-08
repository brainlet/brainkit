// Ported from: packages/core/src/loop/workflows/run-state.ts
package workflows

import (
	"sync"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// MastraLanguageModel is a stub for ../../llm/model/shared_types.MastraLanguageModel.
// Stub: real interface has methods ModelID(), Provider(), SpecificationVersion() (no Get prefix).
// This stub uses GetModelID(), GetProvider(), GetSpecificationVersion(). Method name mismatch.
type MastraLanguageModel interface {
	GetModelID() string
	GetSpecificationVersion() string
	GetProvider() string
}

// StreamInternal is a stub for ../types.StreamInternal.
// Stub: can't import parent loop package (loop imports loop/workflows — would create cycle).
// This version has simplified fields compared to real loop.StreamInternal. Cycle risk.
type StreamInternal struct {
	GenerateID  func() string
	CurrentDate func() string
	ThreadID    string
	ResourceID  string
}

// ---------------------------------------------------------------------------
// ModelMetadata
// ---------------------------------------------------------------------------

// ModelMetadata holds model identification metadata.
type ModelMetadata struct {
	ModelID       string `json:"modelId"`
	ModelVersion  string `json:"modelVersion"`
	ModelProvider string `json:"modelProvider"`
}

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

// State holds the mutable internal state of an AgenticRunState.
type State struct {
	StepResult          map[string]any `json:"stepResult,omitempty"`
	ResponseMetadata    map[string]any `json:"responseMetadata,omitempty"`
	ModelMetadata       ModelMetadata  `json:"modelMetadata"`
	HasToolCallStreaming bool          `json:"hasToolCallStreaming"`
	HasErrored          bool          `json:"hasErrored"`
	ReasoningDeltas     []string       `json:"reasoningDeltas"`
	TextDeltas          []string       `json:"textDeltas"`
	IsReasoning         bool          `json:"isReasoning"`
	IsStreaming          bool          `json:"isStreaming"`
	ProviderOptions     map[string]any `json:"providerOptions,omitempty"`
}

// ---------------------------------------------------------------------------
// AgenticRunState
// ---------------------------------------------------------------------------

// AgenticRunState wraps mutable run state and provides thread-safe access.
// It mirrors the TS class with private #state.
type AgenticRunState struct {
	mu    sync.RWMutex
	state State
}

// NewAgenticRunState creates a new AgenticRunState initialised from the
// provided StreamInternal and MastraLanguageModel, matching the TS constructor.
func NewAgenticRunState(internal *StreamInternal, model MastraLanguageModel) *AgenticRunState {
	var id, timestamp string
	if internal != nil {
		if internal.GenerateID != nil {
			id = internal.GenerateID()
		}
		if internal.CurrentDate != nil {
			timestamp = internal.CurrentDate()
		}
	}

	modelID := model.GetModelID()
	modelVersion := model.GetSpecificationVersion()
	modelProvider := model.GetProvider()

	return &AgenticRunState{
		state: State{
			ResponseMetadata: map[string]any{
				"id":            id,
				"timestamp":     timestamp,
				"modelId":       modelID,
				"modelVersion":  modelVersion,
				"modelProvider": modelProvider,
				"headers":       nil,
			},
			ModelMetadata: ModelMetadata{
				ModelID:       modelID,
				ModelVersion:  modelVersion,
				ModelProvider: modelProvider,
			},
			IsReasoning:         false,
			IsStreaming:          false,
			ProviderOptions:     nil,
			HasToolCallStreaming: false,
			HasErrored:          false,
			ReasoningDeltas:     []string{},
			TextDeltas:          []string{},
			StepResult:          nil,
		},
	}
}

// SetState merges the given partial state into the current state.
// Fields in the provided State override the corresponding fields in the
// current state. Only non-zero-value fields are considered overrides.
//
// For fine-grained control over which fields to update, callers should use
// the SetXxx helper methods below.
func (rs *AgenticRunState) SetState(partial State) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if partial.StepResult != nil {
		rs.state.StepResult = partial.StepResult
	}
	if partial.ResponseMetadata != nil {
		rs.state.ResponseMetadata = partial.ResponseMetadata
	}
	if partial.ModelMetadata != (ModelMetadata{}) {
		rs.state.ModelMetadata = partial.ModelMetadata
	}
	if partial.ReasoningDeltas != nil {
		rs.state.ReasoningDeltas = partial.ReasoningDeltas
	}
	if partial.TextDeltas != nil {
		rs.state.TextDeltas = partial.TextDeltas
	}
	if partial.ProviderOptions != nil {
		rs.state.ProviderOptions = partial.ProviderOptions
	}
	// Booleans are always applied since there is no way to distinguish
	// "not provided" from "false" in a plain struct. Callers should use
	// the full State struct or the individual setters.
	rs.state.HasToolCallStreaming = partial.HasToolCallStreaming
	rs.state.HasErrored = partial.HasErrored
	rs.state.IsReasoning = partial.IsReasoning
	rs.state.IsStreaming = partial.IsStreaming
}

// SetFields applies a mutator function under the write lock, giving the
// caller direct access to modify any field. This is the Go equivalent of
// the TS `setState(partial)` pattern where only specific fields are
// overridden.
func (rs *AgenticRunState) SetFields(fn func(s *State)) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	fn(&rs.state)
}

// GetState returns a snapshot copy of the current state.
func (rs *AgenticRunState) GetState() State {
	rs.mu.RLock()
	defer rs.mu.RUnlock()
	// Return a copy so callers cannot mutate internal state.
	cp := rs.state
	return cp
}
