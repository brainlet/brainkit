// Ported from: packages/core/src/processors/processors/workspace-instructions.ts
package concreteprocessors

import (
	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// Stub types for unported dependencies
// ---------------------------------------------------------------------------

// AnyWorkspace is a stub for ../../workspace/workspace.AnyWorkspace.
// TODO: import from workspace package once ported.
type AnyWorkspace interface {
	// GetInstructions returns workspace environment instructions.
	// The opts map may contain a "requestContext" key.
	GetInstructions(opts map[string]any) string
}

// ---------------------------------------------------------------------------
// WorkspaceInstructionsProcessorOptions
// ---------------------------------------------------------------------------

// WorkspaceInstructionsProcessorOptions configures the WorkspaceInstructionsProcessor.
type WorkspaceInstructionsProcessorOptions struct {
	// Workspace instance to derive instructions from.
	Workspace AnyWorkspace
}

// ---------------------------------------------------------------------------
// WorkspaceInstructionsProcessor
// ---------------------------------------------------------------------------

// WorkspaceInstructionsProcessor injects workspace environment instructions
// (filesystem paths, sandbox info, mount states) into the system message so
// agents understand which paths are accessible in shell commands vs. file tools.
//
// Auto-wired by Agent when a workspace is configured.
type WorkspaceInstructionsProcessor struct {
	processors.BaseProcessor
	workspace AnyWorkspace
}

// NewWorkspaceInstructionsProcessor creates a new WorkspaceInstructionsProcessor.
func NewWorkspaceInstructionsProcessor(opts WorkspaceInstructionsProcessorOptions) *WorkspaceInstructionsProcessor {
	return &WorkspaceInstructionsProcessor{
		BaseProcessor: processors.NewBaseProcessor("workspace-instructions-processor", "Workspace Instructions Processor"),
		workspace:     opts.Workspace,
	}
}

// ProcessInputStep injects workspace instructions as a system message.
func (wip *WorkspaceInstructionsProcessor) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	opts := map[string]any{}
	if args.RequestContext != nil {
		opts["requestContext"] = args.RequestContext
	}

	instructions := wip.workspace.GetInstructions(opts)
	if instructions == "" {
		return nil, nil, nil
	}

	result := &processors.ProcessInputStepResult{
		SystemMessages: []processors.CoreMessageV4{
			{
				Role:    "system",
				Content: instructions,
			},
		},
	}

	return result, nil, nil
}

// ProcessInput is not implemented for this processor.
func (wip *WorkspaceInstructionsProcessor) ProcessInput(args processors.ProcessInputArgs) ([]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error) {
	return nil, nil, nil, nil
}

// ProcessOutputStream is not implemented for this processor.
func (wip *WorkspaceInstructionsProcessor) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	return &args.Part, nil
}

// ProcessOutputResult is not implemented for this processor.
func (wip *WorkspaceInstructionsProcessor) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ProcessOutputStep is not implemented for this processor.
func (wip *WorkspaceInstructionsProcessor) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}
