// Ported from: packages/core/src/workspace/types.ts
package workspace

import (
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// WorkspaceStatus represents the current status of a workspace.
type WorkspaceStatus string

const (
	WorkspaceStatusPending      WorkspaceStatus = "pending"
	WorkspaceStatusInitializing WorkspaceStatus = "initializing"
	WorkspaceStatusReady        WorkspaceStatus = "ready"
	WorkspaceStatusPaused       WorkspaceStatus = "paused"
	WorkspaceStatusError        WorkspaceStatus = "error"
	WorkspaceStatusDestroying   WorkspaceStatus = "destroying"
	WorkspaceStatusDestroyed    WorkspaceStatus = "destroyed"
)

// InstructionsOption represents the instructions configuration for workspace providers.
//
// It can be either:
//   - A static string that fully replaces the default instructions.
//   - A function that receives the default instructions and optional
//     request context, allowing the caller to extend or customise per-request.
//
// Use InstructionsOptionStatic for a plain string override, or
// InstructionsOptionFunc for the function form.
type InstructionsOption interface {
	// resolveInstructions resolves the instructions against the given defaults and context.
	resolveInstructions(defaultInstructions string, requestContext *requestcontext.RequestContext) string
}

// InstructionsOptionStatic is a static string that replaces the default instructions.
type InstructionsOptionStatic string

func (s InstructionsOptionStatic) resolveInstructions(_ string, _ *requestcontext.RequestContext) string {
	return string(s)
}

// InstructionsOptionFuncArgs contains the arguments passed to an InstructionsOptionFunc.
type InstructionsOptionFuncArgs struct {
	DefaultInstructions string
	RequestContext      *requestcontext.RequestContext
}

// InstructionsOptionFunc is a function that receives the default instructions
// and optional request context, returning customised instructions.
type InstructionsOptionFunc func(opts InstructionsOptionFuncArgs) string

func (f InstructionsOptionFunc) resolveInstructions(defaultInstructions string, rc *requestcontext.RequestContext) string {
	return f(InstructionsOptionFuncArgs{
		DefaultInstructions: defaultInstructions,
		RequestContext:      rc,
	})
}
