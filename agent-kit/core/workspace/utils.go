// Ported from: packages/core/src/workspace/utils.ts
package workspace

import (
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ResolveInstructions resolves an instructions override against default instructions.
//
//   - nil -> return default
//   - InstructionsOptionStatic -> return the string as-is
//   - InstructionsOptionFunc -> call with { defaultInstructions, requestContext }
func ResolveInstructions(
	override InstructionsOption,
	getDefault func() string,
	requestContext *requestcontext.RequestContext,
) string {
	if override == nil {
		return getDefault()
	}
	defaultInstructions := getDefault()
	return override.resolveInstructions(defaultInstructions, requestContext)
}
