// Ported from: packages/core/src/processor-provider/phase-filtered-processor.ts
package processorprovider

import (
	"github.com/brainlet/brainkit/agent-kit/core/processors"
)

// PhaseFilteredProcessor wraps an existing Processor and only exposes the
// selected phases. Unselected phase methods are left as no-ops so the
// runner/step-creator skips them.
//
// In the TS source, unselected phases are literally undefined on the object,
// and the runner checks for their existence. In Go, we implement all interface
// methods but provide HasPhase() so the runner can check which phases are
// active before calling.
type PhaseFilteredProcessor struct {
	id             string
	name           string
	description    string
	processorIndex int

	inner         processors.Processor
	enabledPhases map[ProcessorPhase]bool

	// Cached interface assertions for the inner processor.
	innerInput  processors.InputProcessorMethods
	innerOutput processors.OutputProcessorMethods
}

// NewPhaseFilteredProcessor creates a new PhaseFilteredProcessor wrapping
// the inner processor with only the specified phases enabled.
//
// A phase is only truly enabled if it appears in enabledPhases AND the inner
// processor actually implements the corresponding interface method.
func NewPhaseFilteredProcessor(inner processors.Processor, enabledPhases []ProcessorPhase) *PhaseFilteredProcessor {
	phaseMap := make(map[ProcessorPhase]bool, len(enabledPhases))
	for _, p := range enabledPhases {
		phaseMap[p] = true
	}

	pfp := &PhaseFilteredProcessor{
		id:            inner.ID(),
		name:          inner.Name(),
		description:   inner.Description(),
		inner:         inner,
		enabledPhases: phaseMap,
	}

	// Cache interface assertions for the inner processor.
	// Only set if the inner actually implements the interface.
	if ip, ok := inner.(processors.InputProcessorMethods); ok {
		pfp.innerInput = ip
	}
	if op, ok := inner.(processors.OutputProcessorMethods); ok {
		pfp.innerOutput = op
	}

	// Narrow enabledPhases to only phases the inner actually supports.
	// This mirrors the TS behavior where even if a phase is requested,
	// it remains undefined if the inner doesn't implement it.
	if pfp.innerInput == nil {
		delete(pfp.enabledPhases, PhaseProcessInput)
		delete(pfp.enabledPhases, PhaseProcessInputStep)
	}
	if pfp.innerOutput == nil {
		delete(pfp.enabledPhases, PhaseProcessOutputStream)
		delete(pfp.enabledPhases, PhaseProcessOutputResult)
		delete(pfp.enabledPhases, PhaseProcessOutputStep)
	}

	return pfp
}

// HasPhase reports whether the given phase is enabled on this processor.
// The runner should call this before invoking a phase method.
// This mirrors the TS pattern where the runner checks if the method is defined.
func (p *PhaseFilteredProcessor) HasPhase(phase ProcessorPhase) bool {
	return p.enabledPhases[phase]
}

// ---------------------------------------------------------------------------
// Processor interface implementation
// ---------------------------------------------------------------------------

// ID returns the processor's unique identifier.
func (p *PhaseFilteredProcessor) ID() string { return p.id }

// Name returns the processor's optional human-readable name.
func (p *PhaseFilteredProcessor) Name() string { return p.name }

// Description returns the processor's optional description.
func (p *PhaseFilteredProcessor) Description() string { return p.description }

// ProcessorIndex returns the index of this processor in the workflow.
func (p *PhaseFilteredProcessor) ProcessorIndex() int { return p.processorIndex }

// SetProcessorIndex sets the index of this processor in the workflow.
func (p *PhaseFilteredProcessor) SetProcessorIndex(index int) { p.processorIndex = index }

// ---------------------------------------------------------------------------
// InputProcessorMethods implementation (delegates to inner if phase enabled)
// ---------------------------------------------------------------------------

// ProcessInput delegates to the inner processor's ProcessInput if the
// processInput phase is enabled and the inner implements it.
// Returns (nil, nil, nil, nil) when disabled, indicating no transformation.
func (p *PhaseFilteredProcessor) ProcessInput(args processors.ProcessInputArgs) ([]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error) {
	if !p.enabledPhases[PhaseProcessInput] || p.innerInput == nil {
		return nil, nil, nil, nil
	}
	return p.innerInput.ProcessInput(args)
}

// ProcessInputStep delegates to the inner processor's ProcessInputStep if the
// processInputStep phase is enabled and the inner implements it.
// Returns (nil, nil, nil) when disabled, indicating no transformation.
func (p *PhaseFilteredProcessor) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	if !p.enabledPhases[PhaseProcessInputStep] || p.innerInput == nil {
		return nil, nil, nil
	}
	return p.innerInput.ProcessInputStep(args)
}

// ---------------------------------------------------------------------------
// OutputProcessorMethods implementation (delegates to inner if phase enabled)
// ---------------------------------------------------------------------------

// ProcessOutputStream delegates to the inner processor's ProcessOutputStream
// if the processOutputStream phase is enabled and the inner implements it.
// Returns (nil, nil) when disabled, indicating the part should be passed through.
func (p *PhaseFilteredProcessor) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	if !p.enabledPhases[PhaseProcessOutputStream] || p.innerOutput == nil {
		return nil, nil
	}
	return p.innerOutput.ProcessOutputStream(args)
}

// ProcessOutputResult delegates to the inner processor's ProcessOutputResult
// if the processOutputResult phase is enabled and the inner implements it.
// Returns (nil, nil, nil) when disabled, indicating no transformation.
func (p *PhaseFilteredProcessor) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	if !p.enabledPhases[PhaseProcessOutputResult] || p.innerOutput == nil {
		return nil, nil, nil
	}
	return p.innerOutput.ProcessOutputResult(args)
}

// ProcessOutputStep delegates to the inner processor's ProcessOutputStep
// if the processOutputStep phase is enabled and the inner implements it.
// Returns (nil, nil, nil) when disabled, indicating no transformation.
func (p *PhaseFilteredProcessor) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	if !p.enabledPhases[PhaseProcessOutputStep] || p.innerOutput == nil {
		return nil, nil, nil
	}
	return p.innerOutput.ProcessOutputStep(args)
}

// ---------------------------------------------------------------------------
// MastraRegistrable implementation
// ---------------------------------------------------------------------------

// RegisterMastra delegates to the inner processor if it implements
// MastraRegistrable.
func (p *PhaseFilteredProcessor) RegisterMastra(mastra processors.Mastra) {
	if registrable, ok := p.inner.(processors.MastraRegistrable); ok {
		registrable.RegisterMastra(mastra)
	}
}
