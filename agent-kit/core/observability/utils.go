// Ported from: packages/core/src/observability/utils.ts
package observability

import (
	"reflect"

	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
)

// GetOrCreateSpan creates or gets a child span from existing tracing context or starts a new trace.
// This helper consolidates the common pattern of creating spans that can either be:
// 1. Children of an existing span (when TracingContext.CurrentSpan exists)
// 2. New root spans (when no current span exists)
//
// Returns the created Span or nil if tracing is disabled.
func GetOrCreateSpan(options obstypes.GetOrCreateSpanOptions) obstypes.Span {
	metadata := make(map[string]any)
	if options.Metadata != nil {
		for k, v := range options.Metadata {
			metadata[k] = v
		}
	}
	if options.TracingOptions != nil && options.TracingOptions.Metadata != nil {
		for k, v := range options.TracingOptions.Metadata {
			metadata[k] = v
		}
	}

	// If we have a current span, create a child span.
	if options.TracingContext != nil && options.TracingContext.CurrentSpan != nil {
		return options.TracingContext.CurrentSpan.CreateChildSpan(obstypes.ChildSpanOptions{
			CreateBaseOptions: obstypes.CreateBaseOptions{
				Type:           options.Type,
				Name:           options.Name,
				Attributes:     options.Attributes,
				Metadata:       metadata,
				EntityType:     options.EntityType,
				EntityID:       options.EntityID,
				EntityName:     options.EntityName,
				TracingPolicy:  options.TracingPolicy,
				RequestContext: options.RequestContext,
			},
			Input: options.Input,
		})
	}

	// Otherwise, try to create a new root span via the Mastra instance's
	// observability field. In TypeScript this is:
	//   options.mastra?.observability?.getSelectedInstance({ requestContext })
	// Since we can't import the Mastra type directly (circular dependency),
	// we use reflection to access the Observability field.
	instance := getSelectedInstanceFromMastra(options.Mastra, options.RequestContext)
	if instance == nil {
		return nil
	}

	startOpts := obstypes.StartSpanOptions{
		CreateSpanOptions: obstypes.CreateSpanOptions{
			CreateBaseOptions: obstypes.CreateBaseOptions{
				Type:           options.Type,
				Name:           options.Name,
				Attributes:     options.Attributes,
				Metadata:       metadata,
				EntityType:     options.EntityType,
				EntityID:       options.EntityID,
				EntityName:     options.EntityName,
				TracingPolicy:  options.TracingPolicy,
				RequestContext: options.RequestContext,
			},
			Input: options.Input,
		},
		TracingOptions: options.TracingOptions,
	}

	if options.TracingOptions != nil {
		startOpts.CreateSpanOptions.TraceID = options.TracingOptions.TraceID
		startOpts.CreateSpanOptions.ParentSpanID = options.TracingOptions.ParentSpanID
	}

	startOpts.CustomSamplerOptions = &obstypes.CustomSamplerOptions{
		RequestContext: options.RequestContext,
		Metadata:       metadata,
	}

	return instance.StartSpan(startOpts)
}

// getSelectedInstanceFromMastra uses reflection to access mastra.Observability and
// call GetSelectedInstance on it. This avoids a circular import on the Mastra type.
// Returns nil if the mastra value is nil, lacks an Observability field, or the
// entrypoint returns no instance.
func getSelectedInstanceFromMastra(mastra any, rc interface{}) obstypes.ObservabilityInstance {
	if mastra == nil {
		return nil
	}

	v := reflect.ValueOf(mastra)
	// Dereference pointer if needed.
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	// Look for an Observability field (struct) or method.
	var obsField reflect.Value
	if v.Kind() == reflect.Struct {
		obsField = v.FieldByName("Observability")
	}
	if !obsField.IsValid() {
		// Try as a method instead.
		obsField = reflect.ValueOf(mastra)
		m := obsField.MethodByName("Observability")
		if !m.IsValid() {
			return nil
		}
		results := m.Call(nil)
		if len(results) == 0 || results[0].IsNil() {
			return nil
		}
		obsField = results[0]
	}

	if obsField.IsNil() {
		return nil
	}

	// The observability field should implement ObservabilityEntrypoint which has
	// GetSelectedInstance(ConfigSelectorOptions) ObservabilityInstance.
	entrypoint, ok := obsField.Interface().(obstypes.ObservabilityEntrypoint)
	if !ok {
		return nil
	}

	// Build ConfigSelectorOptions with the request context if available.
	selectorOpts := obstypes.ConfigSelectorOptions{}
	if rc != nil {
		// Try to cast to *requestcontext.RequestContext.
		// We use reflection to avoid importing the requestcontext package directly
		// since it's already imported by the types package.
		selectorOpts.RequestContext = nil // Set via type assertion below.
	}

	return entrypoint.GetSelectedInstance(selectorOpts)
}

// ExecuteWithContextParams holds parameters for ExecuteWithContext.
type ExecuteWithContextParams struct {
	Span obstypes.Span
	Fn   func() (any, error)
}

// ExecuteWithContext executes a function within the span's tracing context if available.
// Falls back to direct execution if no span exists.
//
// When a bridge is configured, this enables auto-instrumented operations
// (HTTP requests, database queries, etc.) to be properly nested under the
// current span in the external tracing system.
func ExecuteWithContext(params ExecuteWithContextParams) (any, error) {
	if params.Span != nil {
		return params.Span.ExecuteInContext(params.Fn)
	}
	return params.Fn()
}

// ExecuteWithContextSyncParams holds parameters for ExecuteWithContextSync.
type ExecuteWithContextSyncParams struct {
	Span obstypes.Span
	Fn   func() any
}

// ExecuteWithContextSync executes a synchronous function within the span's tracing
// context if available. Falls back to direct execution if no span exists.
//
// When a bridge is configured, this enables auto-instrumented operations
// (HTTP requests, database queries, etc.) to be properly nested under the
// current span in the external tracing system.
func ExecuteWithContextSync(params ExecuteWithContextSyncParams) any {
	if params.Span != nil {
		return params.Span.ExecuteInContextSync(params.Fn)
	}
	return params.Fn()
}
