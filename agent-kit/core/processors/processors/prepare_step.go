// Ported from: packages/core/src/processors/processors/prepare-step.ts
package concreteprocessors

import (
	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// Stub types for unported dependencies
// ---------------------------------------------------------------------------

// PrepareStepFunction is a stub for ../../loop/types.PrepareStepFunction.
// TODO: import from loop package once ported.
type PrepareStepFunction func(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, error)

// ---------------------------------------------------------------------------
// PrepareStepProcessor
// ---------------------------------------------------------------------------

// PrepareStepProcessor wraps a PrepareStepFunction as a processor.
type PrepareStepProcessor struct {
	processors.BaseProcessor
	prepareStep PrepareStepFunction
}

// PrepareStepProcessorOptions holds options for PrepareStepProcessor.
type PrepareStepProcessorOptions struct {
	PrepareStep PrepareStepFunction
}

// NewPrepareStepProcessor creates a new PrepareStepProcessor.
func NewPrepareStepProcessor(opts PrepareStepProcessorOptions) *PrepareStepProcessor {
	return &PrepareStepProcessor{
		BaseProcessor: processors.NewBaseProcessor("prepare-step", "Prepare Step Processor"),
		prepareStep:   opts.PrepareStep,
	}
}

// ProcessInputStep delegates to the wrapped PrepareStepFunction.
func (p *PrepareStepProcessor) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	if p.prepareStep == nil {
		return nil, nil, nil
	}

	result, err := p.prepareStep(args)
	if err != nil {
		return nil, nil, err
	}
	return result, nil, nil
}

// ProcessInput is not implemented for this processor.
func (p *PrepareStepProcessor) ProcessInput(args processors.ProcessInputArgs) ([]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error) {
	return nil, nil, nil, nil
}

// ProcessOutputStream is not implemented for this processor.
func (p *PrepareStepProcessor) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	return &args.Part, nil
}

// ProcessOutputResult is not implemented for this processor.
func (p *PrepareStepProcessor) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ProcessOutputStep is not implemented for this processor.
func (p *PrepareStepProcessor) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}
