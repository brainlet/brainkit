// Ported from: packages/ai/src/agent/infer-agent-ui-message.ts
package agent

// InferAgentUIMessage is a type alias used in the TypeScript source to infer
// the UI message type of an agent.
//
// In TypeScript:
//
//	type InferAgentUIMessage<AGENT, MESSAGE_METADATA = unknown> = UIMessage<
//	  MESSAGE_METADATA,
//	  never,
//	  InferUITools<InferAgentTools<AGENT>>
//	>;
//
// In Go, this is a runtime concept rather than a compile-time type inference.
// The UIMessage type is defined in the ui package. This file exists for
// structural parity with the TypeScript source.

// UIMessage is a stub for the UI message type.
// TODO: import from brainlink/experiments/ai-kit/ui once ported.
type UIMessage struct {
	Role  string
	ID    string
	Parts []interface{}
}

// InferAgentUIMessage returns the UIMessage type associated with an agent.
// In Go, this is a pass-through since we don't have generic type inference.
// The caller constructs UIMessage values directly.
func InferAgentUIMessage(_ Agent) UIMessage {
	return UIMessage{}
}
