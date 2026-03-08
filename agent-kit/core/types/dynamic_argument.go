// Ported from: packages/core/src/types/dynamic-argument.ts
package types

import (
	"errors"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
	"github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// MastraRef represents the top-level Mastra orchestrator.
// Defined here (not imported from core package) to break circular dependency:
// core imports types (for IdGeneratorContext, etc.), so types cannot import core.
// core.Mastra struct satisfies this interface.
// Dynamic argument resolver functions receive this to access framework services
// like logging and storage.
type MastraRef interface {
	// GetLogger returns the configured logger instance.
	GetLogger() logger.IMastraLogger
}

// DynamicArgumentContext provides context for resolving dynamic arguments.
// This corresponds to the parameter object { requestContext, mastra } in the
// TypeScript DynamicArgument<T> function variant.
type DynamicArgumentContext struct {
	RequestContext *requestcontext.RequestContext
	Mastra         MastraRef
}

// DynamicArgumentFunc is a function that resolves a dynamic argument.
// It corresponds to the function variant of the TypeScript union type:
//
//	DynamicArgument<T> = T | (({ requestContext, mastra }) => Promise<T> | T)
//
// In Go, the async (Promise) aspect is handled by returning (T, error).
type DynamicArgumentFunc[T any] func(ctx DynamicArgumentContext) (T, error)

// DynamicArgument holds either a static value or a resolver function.
// This is the Go representation of the TypeScript union type:
//
//	DynamicArgument<T> = T | (({ requestContext, mastra }) => Promise<T> | T)
//
// Use NewStaticArgument or NewDynamicArgument to construct, and Resolve to obtain the value.
type DynamicArgument[T any] struct {
	static   T
	resolver DynamicArgumentFunc[T]
	isDynamic bool
}

// NewStaticArgument creates a DynamicArgument that holds a fixed value.
func NewStaticArgument[T any](value T) DynamicArgument[T] {
	return DynamicArgument[T]{
		static:    value,
		isDynamic: false,
	}
}

// NewDynamicArgument creates a DynamicArgument backed by a resolver function.
func NewDynamicArgument[T any](fn DynamicArgumentFunc[T]) DynamicArgument[T] {
	return DynamicArgument[T]{
		resolver:  fn,
		isDynamic: true,
	}
}

// Resolve returns the argument's value. For static arguments it returns the
// value directly; for dynamic arguments it calls the resolver function.
func (da DynamicArgument[T]) Resolve(ctx DynamicArgumentContext) (T, error) {
	if da.isDynamic {
		if da.resolver == nil {
			var zero T
			return zero, errors.New("dynamic argument has nil resolver")
		}
		return da.resolver(ctx)
	}
	return da.static, nil
}

// IsDynamic reports whether the argument is backed by a resolver function.
func (da DynamicArgument[T]) IsDynamic() bool {
	return da.isDynamic
}

// ValidateNonEmpty checks that a string is not empty at runtime.
// This is the Go equivalent of the TypeScript compile-time type:
//
//	NonEmpty<T extends string> = T extends '' ? never : T
//
// Since Go cannot enforce non-empty strings at compile time, this provides
// runtime validation instead.
func ValidateNonEmpty(s string) error {
	if s == "" {
		return errors.New("string must not be empty")
	}
	return nil
}

// IdType represents the type of ID being generated.
type IdType string

const (
	// IdTypeThread identifies a conversation thread ID.
	IdTypeThread IdType = "thread"
	// IdTypeMessage identifies a message within a thread.
	IdTypeMessage IdType = "message"
	// IdTypeRun identifies an agent or workflow execution run.
	IdTypeRun IdType = "run"
	// IdTypeStep identifies a workflow step.
	IdTypeStep IdType = "step"
	// IdTypeGeneric identifies a generic ID with no specific type.
	IdTypeGeneric IdType = "generic"
)

// IdGeneratorSource represents the Mastra primitive requesting the ID.
type IdGeneratorSource string

const (
	// IdGeneratorSourceAgent indicates the ID is requested by an agent.
	IdGeneratorSourceAgent IdGeneratorSource = "agent"
	// IdGeneratorSourceWorkflow indicates the ID is requested by a workflow.
	IdGeneratorSourceWorkflow IdGeneratorSource = "workflow"
	// IdGeneratorSourceMemory indicates the ID is requested by memory.
	IdGeneratorSourceMemory IdGeneratorSource = "memory"
)

// IdGeneratorContext contains context information passed to the ID generator function.
// This allows users to generate context-aware IDs based on the context
// in which the ID is being generated.
type IdGeneratorContext struct {
	// IdType is the type of ID being generated.
	IdType IdType `json:"idType"`

	// Source is the Mastra primitive requesting the ID.
	Source *IdGeneratorSource `json:"source,omitempty"`

	// EntityId is the ID of the entity (agent, workflow, etc.) requesting the ID.
	EntityId *string `json:"entityId,omitempty"`

	// ThreadId is the thread ID, if applicable (e.g., for message IDs).
	ThreadId *string `json:"threadId,omitempty"`

	// ResourceId is the resource ID, if applicable (e.g., user ID for threads).
	ResourceId *string `json:"resourceId,omitempty"`

	// Role is the message role, if generating a message ID.
	Role *string `json:"role,omitempty"`

	// StepType is the step type, if generating a workflow step ID.
	StepType *string `json:"stepType,omitempty"`
}

// MastraIdGenerator is a custom ID generator function for creating unique identifiers.
// Receives optional context about what type of ID is being generated
// and where it's being requested from.
//
// The returned string must be non-empty (corresponding to the TypeScript
// NonEmpty<string> return type).
//
// Example usage:
//
//	generator := func(ctx *types.IdGeneratorContext) string {
//	    if ctx != nil && ctx.IdType == types.IdTypeMessage && ctx.ThreadId != nil {
//	        return fmt.Sprintf("msg-%s-%d", *ctx.ThreadId, time.Now().UnixMilli())
//	    }
//	    if ctx != nil && ctx.IdType == types.IdTypeRun && ctx.EntityId != nil {
//	        return fmt.Sprintf("run-%s-%d", *ctx.EntityId, time.Now().UnixMilli())
//	    }
//	    return uuid.NewString()
//	}
type MastraIdGenerator func(ctx *IdGeneratorContext) string
