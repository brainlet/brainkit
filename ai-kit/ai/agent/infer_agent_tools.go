// Ported from: packages/ai/src/agent/infer-agent-tools.ts
package agent

import (
	gt "github.com/brainlet/brainkit/ai-kit/ai/generatetext"
)

// InferAgentTools returns the ToolSet from an Agent.
//
// In TypeScript, InferAgentTools is a conditional type that extracts
// the TOOLS type parameter from an Agent<any, TOOLS, any>.
// In Go, generics are not needed since ToolSet is a concrete type (map[string]Tool).
// This function provides the equivalent runtime utility.
func InferAgentTools(a Agent) gt.ToolSet {
	if a == nil {
		return nil
	}
	return a.Tools()
}
