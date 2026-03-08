// Ported from: packages/ai/src/agent/create-agent-ui-stream.ts
package agent

import (
	"context"
	"fmt"

	gt "github.com/brainlet/brainkit/ai-kit/ai/generatetext"
)

// CreateAgentUIStreamOptions contains options for CreateAgentUIStream.
type CreateAgentUIStreamOptions struct {
	// Agent is the agent to run.
	Agent Agent

	// UIMessages is the input UI messages.
	UIMessages []interface{}

	// Ctx is the Go context for cancellation (replaces AbortSignal in TS).
	Ctx context.Context

	// Timeout in milliseconds. Optional.
	Timeout *gt.TimeoutConfiguration

	// Options are call-level options specific to the agent implementation.
	Options interface{}

	// ExperimentalTransform contains optional stream transformations.
	ExperimentalTransform []gt.StreamTextTransform

	// OnStepFinish is called when each step is finished. Optional.
	OnStepFinish OnStepFinishCallback

	// OriginalMessages are the original messages for backwards compatibility.
	// TODO: remove in v7 (per TS source comment).
	OriginalMessages []UIMessage
}

// CreateAgentUIStream runs the agent and streams the output as a UI message stream.
//
// It validates the UI messages, converts them to model messages, runs the agent's
// stream method, and returns the result as a UI message stream.
//
// TODO: This function requires validateUIMessages and convertToModelMessages
// from the ui package, which are not yet ported. The current implementation
// provides a structural port that delegates to the agent's Stream method.
func CreateAgentUIStream(opts CreateAgentUIStreamOptions) (*gt.StreamTextResult, error) {
	if opts.Agent == nil {
		return nil, fmt.Errorf("agent is required")
	}

	ctx := opts.Ctx
	if ctx == nil {
		ctx = context.Background()
	}

	// TODO: validateUIMessages(opts.UIMessages, opts.Agent.Tools())
	// TODO: convertToModelMessages(validatedMessages, opts.Agent.Tools())
	//
	// For now, we pass through to the agent's Stream method.
	// The full implementation would:
	// 1. Validate UI messages
	// 2. Convert to model messages
	// 3. Stream with the agent
	// 4. Convert result to UI message stream via toUIMessageStream

	params := AgentStreamParameters{
		AgentCallParameters: AgentCallParameters{
			Ctx:          ctx,
			Timeout:      opts.Timeout,
			Options:      opts.Options,
			OnStepFinish: opts.OnStepFinish,
		},
		ExperimentalTransform: opts.ExperimentalTransform,
	}

	// Convert UIMessages to model messages would happen here
	// For structural fidelity, we pass the messages through as-is
	// since the conversion functions are not yet ported.

	result, err := opts.Agent.Stream(params)
	if err != nil {
		return nil, fmt.Errorf("agent stream: %w", err)
	}

	// TODO: result.ToUIMessageStream(uiMessageStreamOptions)
	// The TS source calls result.toUIMessageStream({...uiMessageStreamOptions, originalMessages})
	// which converts the StreamTextResult into an AsyncIterableStream of UIMessageChunks.

	return result, nil
}
