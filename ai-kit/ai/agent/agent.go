// Ported from: packages/ai/src/agent/agent.ts
package agent

import (
	"context"

	gt "github.com/brainlet/brainkit/ai-kit/ai/generatetext"
)

// AgentCallParameters contains the parameters for calling an agent.
//
// In the TypeScript source, the prompt and messages fields are mutually exclusive
// (discriminated union). In Go, both can be set but at most one should be non-zero.
type AgentCallParameters struct {
	// Prompt is a text prompt or a list of model messages.
	// You can either use Prompt/PromptMessages or Messages but not both.
	Prompt string

	// PromptMessages is a list of model messages used as the prompt.
	// You can either use Prompt/PromptMessages or Messages but not both.
	PromptMessages []gt.ModelMessage

	// Messages is a list of model messages.
	// You can either use Prompt/PromptMessages or Messages but not both.
	Messages []gt.ModelMessage

	// Ctx is the Go context for cancellation (replaces AbortSignal in TS).
	Ctx context.Context

	// Timeout in milliseconds. Can be specified as a number or as an object with TotalMs.
	Timeout *gt.TimeoutConfiguration

	// Options are call-level options specific to the agent implementation.
	Options interface{}

	// Callbacks

	// ExperimentalOnStart is called when the agent operation begins, before any LLM calls.
	ExperimentalOnStart OnStartCallback

	// ExperimentalOnStepStart is called when a step (LLM call) begins.
	ExperimentalOnStepStart OnStepStartCallback

	// ExperimentalOnToolCallStart is called before each tool execution begins.
	ExperimentalOnToolCallStart OnToolCallStartCallback

	// ExperimentalOnToolCallFinish is called after each tool execution completes.
	ExperimentalOnToolCallFinish OnToolCallFinishCallback

	// OnStepFinish is called when each step (LLM call) is finished.
	OnStepFinish OnStepFinishCallback

	// OnFinish is called when all steps are finished and the response is complete.
	OnFinish OnFinishCallback
}

// AgentStreamParameters contains the parameters for streaming an output from an agent.
type AgentStreamParameters struct {
	AgentCallParameters

	// ExperimentalTransform contains optional stream transformations.
	// They are applied in the order they are provided.
	ExperimentalTransform []gt.StreamTextTransform
}

// Agent receives a prompt (text or messages) and generates or streams an output
// that consists of steps, tool calls, data parts, etc.
//
// You can implement your own Agent by implementing the Agent interface,
// or use the ToolLoopAgent struct.
type Agent interface {
	// Version returns the specification version of the agent interface.
	Version() string

	// ID returns the id of the agent, or empty string if not set.
	ID() string

	// Tools returns the tools that the agent can use.
	Tools() gt.ToolSet

	// Generate generates an output from the agent (non-streaming).
	Generate(params AgentCallParameters) (*gt.GenerateTextResult, error)

	// Stream streams an output from the agent (streaming).
	Stream(params AgentStreamParameters) (*gt.StreamTextResult, error)
}
