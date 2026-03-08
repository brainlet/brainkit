// Ported from: packages/core/src/processor-provider/types.ts
package processorprovider

import (
	"github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// ProcessorPhase
// ---------------------------------------------------------------------------

// ProcessorPhase represents the five processor phases corresponding to the
// five optional methods on Processor.
type ProcessorPhase string

const (
	PhaseProcessInput        ProcessorPhase = "processInput"
	PhaseProcessInputStep    ProcessorPhase = "processInputStep"
	PhaseProcessOutputStream ProcessorPhase = "processOutputStream"
	PhaseProcessOutputResult ProcessorPhase = "processOutputResult"
	PhaseProcessOutputStep   ProcessorPhase = "processOutputStep"
)

// AllProcessorPhases contains all processor phases.
var AllProcessorPhases = []ProcessorPhase{
	PhaseProcessInput,
	PhaseProcessInputStep,
	PhaseProcessOutputStream,
	PhaseProcessOutputResult,
	PhaseProcessOutputStep,
}

// ---------------------------------------------------------------------------
// ProcessorProviderInfo
// ---------------------------------------------------------------------------

// ProcessorProviderInfo holds metadata about a processor provider.
type ProcessorProviderInfo struct {
	// ID is the unique identifier for this provider (e.g., "moderation", "token-limiter").
	ID string `json:"id"`
	// Name is the human-readable name.
	Name string `json:"name"`
	// Description is a short description of the provider.
	Description string `json:"description,omitempty"`
}

// ---------------------------------------------------------------------------
// ProcessorProviderProcessorInfo
// ---------------------------------------------------------------------------

// ProcessorProviderProcessorInfo holds info about a processor available from
// a provider (used for UI listing).
type ProcessorProviderProcessorInfo struct {
	// Slug is the unique slug for this processor within the provider.
	Slug string `json:"slug"`
	// Name is the human-readable name.
	Name string `json:"name"`
	// Description of what this processor does.
	Description string `json:"description,omitempty"`
	// AvailablePhases lists which phases this processor supports.
	AvailablePhases []ProcessorPhase `json:"availablePhases"`
}

// ---------------------------------------------------------------------------
// ProcessorProvider
// ---------------------------------------------------------------------------

// ProcessorProvider is the interface for processor providers that supply
// configurable processors to agents.
//
// Processor providers serve two purposes:
//  1. Discovery -- UI uses Info(), ConfigSchema(), AvailablePhases() to render
//     configuration forms.
//  2. Runtime -- Agent hydration uses CreateProcessor() to instantiate processors
//     from stored config.
type ProcessorProvider interface {
	// Info returns the provider metadata.
	Info() ProcessorProviderInfo

	// ConfigSchema returns the schema describing the configuration this provider
	// accepts. In the TS source this is a Zod ZodSchema used for runtime validation
	// and UI form generation. In Go we return a map describing the schema shape.
	// TODO: Replace with a proper validation/schema type once chosen.
	ConfigSchema() map[string]any

	// AvailablePhases returns which processor phases this provider's processors
	// support. Used by the UI to show which phases can be enabled.
	AvailablePhases() []ProcessorPhase

	// CreateProcessor creates a processor instance from the given configuration.
	// Called during agent hydration to resolve stored processor configs into live
	// instances. The returned Processor may also implement InputProcessor and/or
	// OutputProcessor from the processors package.
	CreateProcessor(config map[string]any) processors.Processor
}
