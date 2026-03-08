// Ported from: packages/core/src/agent/utils.ts
package agent

import (
	"context"
	"fmt"
	"log"

	mastraerror "github.com/brainlet/brainkit/agent-kit/core/error"
)

// ---------------------------------------------------------------------------
// supportedLanguageModelSpecifications
// ---------------------------------------------------------------------------

// SupportedLanguageModelSpecifications lists the model specification versions
// that are supported by the vNext agent loop.
var SupportedLanguageModelSpecifications = []string{"v2", "v3"}

// IsSupportedLanguageModel returns true when the model's specificationVersion
// is one of the supported vNext versions.
func IsSupportedLanguageModel(model LanguageModelLike) bool {
	sv := model.SpecificationVersion()
	for _, v := range SupportedLanguageModelSpecifications {
		if sv == v {
			return true
		}
	}
	return false
}

// LanguageModelLike is the minimal interface needed by IsSupportedLanguageModel.
// Both model.MastraLanguageModel and model.MastraLegacyLanguageModel satisfy this.
// Intentionally kept as local interface (interface segregation): only requires
// SpecificationVersion() while real model.MastraLanguageModel also has Provider()
// and ModelID(). This is the correct Go pattern for minimal dependency.
type LanguageModelLike interface {
	SpecificationVersion() string
}

// ---------------------------------------------------------------------------
// tryGenerateWithJsonFallback
// ---------------------------------------------------------------------------

// AgentLike is the minimal interface used by the JSON-fallback helpers.
// Uses local FullOutput/MastraModelOutputStub stubs because Agent.Generate()
// and Agent.Stream() are not yet implemented. Replace with concrete Agent
// once those methods are ported.
type AgentLike interface {
	Generate(ctx context.Context, prompt MessageListInput, opts AgentExecutionOptions) (*FullOutput, error)
	Stream(ctx context.Context, prompt MessageListInput, opts AgentExecutionOptions) (*MastraModelOutputStub, error)
}

// FullOutput is a stub for ../stream/base/output.FullOutput.
// MISMATCH: real streambase.FullOutput has 20+ fields (Usage, Steps, FinishReason,
// Warnings, ProviderMetadata, Request, Reasoning, ToolCalls, ToolResults, Sources,
// Files, Response, TotalUsage, Error, Tripwire, TraceID, etc.). This stub only has
// Text and Object. Cannot wire until AgentLike.Generate() returns the real type.
type FullOutput struct {
	Text   string `json:"text,omitempty"`
	Object any    `json:"object,omitempty"`
}

// MastraModelOutputStub is a stub for ../stream/base/output.MastraModelOutput.
// MISMATCH: real streambase.MastraModelOutput is a complex struct with streaming
// channels, promises, and many fields. This stub only has Object for testing
// the JSON fallback path. Cannot wire until Agent.Stream() is implemented.
type MastraModelOutputStub struct {
	Object any `json:"object,omitempty"`
}

// TryGenerateWithJsonFallback attempts to generate with structured output.
// If the first attempt fails, it retries with jsonPromptInjection enabled
// to coerce the LLM to respond with JSON text.
func TryGenerateWithJsonFallback(
	ctx context.Context,
	agent AgentLike,
	prompt MessageListInput,
	options AgentExecutionOptions,
) (*FullOutput, error) {
	if options.StructuredOutput == nil || options.StructuredOutput.Schema == nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "STRUCTURED_OUTPUT_OPTIONS_REQUIRED",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Text:     "structuredOutput is required to use TryGenerateWithJsonFallback",
		})
	}

	result, err := agent.Generate(ctx, prompt, options)
	if err != nil {
		log.Printf("Error in TryGenerateWithJsonFallback. Attempting fallback. %v", err)

		// Clone and enable jsonPromptInjection for the fallback.
		fallbackOpts := options
		if fallbackOpts.StructuredOutput != nil {
			so := *fallbackOpts.StructuredOutput
			so.JSONPromptInjection = true
			fallbackOpts.StructuredOutput = &so
		}

		return agent.Generate(ctx, prompt, fallbackOpts)
	}
	return result, nil
}

// TryStreamWithJsonFallback attempts to stream with structured output.
// If the first attempt fails or returns a nil object, it retries with
// jsonPromptInjection enabled.
func TryStreamWithJsonFallback(
	ctx context.Context,
	agent AgentLike,
	prompt MessageListInput,
	options AgentExecutionOptions,
) (*MastraModelOutputStub, error) {
	if options.StructuredOutput == nil || options.StructuredOutput.Schema == nil {
		return nil, mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "STRUCTURED_OUTPUT_OPTIONS_REQUIRED",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Text:     "structuredOutput is required to use TryStreamWithJsonFallback",
		})
	}

	result, err := agent.Stream(ctx, prompt, options)
	if err != nil {
		log.Printf("Error in TryStreamWithJsonFallback. Attempting fallback. %v", err)

		fallbackOpts := options
		if fallbackOpts.StructuredOutput != nil {
			so := *fallbackOpts.StructuredOutput
			so.JSONPromptInjection = true
			fallbackOpts.StructuredOutput = &so
		}

		return agent.Stream(ctx, prompt, fallbackOpts)
	}

	if result.Object == nil {
		fallbackErr := mastraerror.NewMastraError(mastraerror.ErrorDefinition{
			ID:       "STRUCTURED_OUTPUT_OBJECT_UNDEFINED",
			Domain:   mastraerror.ErrorDomainAgent,
			Category: mastraerror.ErrorCategoryUser,
			Text:     "structuredOutput object is undefined",
		})
		log.Printf("Error in TryStreamWithJsonFallback. Attempting fallback. %v", fallbackErr)

		fallbackOpts := options
		if fallbackOpts.StructuredOutput != nil {
			so := *fallbackOpts.StructuredOutput
			so.JSONPromptInjection = true
			fallbackOpts.StructuredOutput = &so
		}

		return agent.Stream(ctx, prompt, fallbackOpts)
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// resolveThreadIdFromArgs
// ---------------------------------------------------------------------------

// ThreadIDResult represents a resolved thread ID with optional partial thread data.
type ThreadIDResult struct {
	StorageThreadType
}

// ResolveThreadIdFromArgs resolves a thread ID from various argument formats.
// It supports:
//   - memory.thread as a string ID
//   - memory.thread as an object with an "id" field
//   - a top-level threadId string
//
// Returns nil if no thread ID can be resolved.
func ResolveThreadIdFromArgs(args ResolveThreadArgs) *ThreadIDResult {
	if args.Memory != nil && args.Memory.Thread != nil {
		switch t := args.Memory.Thread.(type) {
		case string:
			if t != "" {
				return &ThreadIDResult{
					StorageThreadType: StorageThreadType{ID: t},
				}
			}
		case map[string]any:
			if id, ok := t["id"].(string); ok && id != "" {
				result := &ThreadIDResult{
					StorageThreadType: StorageThreadType{ID: id},
				}
				// Copy optional fields from the map.
				if title, ok := t["title"].(string); ok {
					result.Title = title
				}
				if meta, ok := t["metadata"].(map[string]any); ok {
					result.Metadata = meta
				}
				if resID, ok := t["resourceId"].(string); ok {
					result.ResourceID = resID
				}
				return result
			}
		case StorageThreadType:
			if t.ID != "" {
				return &ThreadIDResult{StorageThreadType: t}
			}
		case *StorageThreadType:
			if t != nil && t.ID != "" {
				return &ThreadIDResult{StorageThreadType: *t}
			}
		default:
			// Try to extract ID via interface.
			type hasID interface{ GetID() string }
			if obj, ok := args.Memory.Thread.(hasID); ok {
				if id := obj.GetID(); id != "" {
					return &ThreadIDResult{
						StorageThreadType: StorageThreadType{ID: id},
					}
				}
			}
		}
	}

	if args.ThreadID != "" {
		return &ThreadIDResult{
			StorageThreadType: StorageThreadType{ID: args.ThreadID},
		}
	}

	return nil
}

// ResolveThreadArgs holds the arguments for ResolveThreadIdFromArgs.
type ResolveThreadArgs struct {
	Memory   *AgentMemoryOption `json:"memory,omitempty"`
	ThreadID string             `json:"threadId,omitempty"`
}

// ---------------------------------------------------------------------------
// resolveMaybePromise (Go equivalent)
// ---------------------------------------------------------------------------

// ResolveDynamic resolves a value that may be produced by a dynamic function.
// In TypeScript this was resolveMaybePromise handling sync/async duality.
// In Go, if the value is a function, we call it; otherwise return it directly.
// The caller is responsible for providing the correct concrete type via generics.
func ResolveDynamic[T any](value any, resolve func(T) T) T {
	switch v := value.(type) {
	case T:
		return resolve(v)
	case func() T:
		return resolve(v())
	default:
		panic(fmt.Sprintf("agent.ResolveDynamic: unexpected type %T", value))
	}
}
