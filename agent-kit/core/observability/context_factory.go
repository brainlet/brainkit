// Ported from: packages/core/src/observability/context-factory.ts
package observability

import (
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
)

// ============================================================================
// Context Derivation
// ============================================================================

// deriveLoggerContext derives a LoggerContext from the current span's ObservabilityInstance.
// Falls back to no-op when there is no span or the instance doesn't support logging.
func deriveLoggerContext(tracing obstypes.TracingContext) obstypes.LoggerContext {
	span := tracing.CurrentSpan
	if span == nil {
		return NoOpLoggerContext
	}
	instance := span.ObservabilityInstance()
	if instance == nil {
		return NoOpLoggerContext
	}
	ctx := instance.GetLoggerContext(span)
	if ctx == nil {
		return NoOpLoggerContext
	}
	return ctx
}

// deriveMetricsContext derives a MetricsContext from the current span's ObservabilityInstance.
// Falls back to no-op when there is no span or the instance doesn't support metrics.
func deriveMetricsContext(tracing obstypes.TracingContext) obstypes.MetricsContext {
	span := tracing.CurrentSpan
	if span == nil {
		return NoOpMetricsContext
	}
	instance := span.ObservabilityInstance()
	if instance == nil {
		return NoOpMetricsContext
	}
	ctx := instance.GetMetricsContext(span)
	if ctx == nil {
		return NoOpMetricsContext
	}
	return ctx
}

// ============================================================================
// Context Factory
// ============================================================================

// CreateObservabilityContext creates an observability context with real or no-op
// implementations for tracing, logging, and metrics.
//
// When a TracingContext with a current span is provided, the logger and metrics
// contexts are derived from the span's ObservabilityInstance so that log entries
// and metric data points are automatically correlated to the active trace.
//
// Pass nil for no-op behavior across all signals.
func CreateObservabilityContext(tracingContext *obstypes.TracingContext) obstypes.ObservabilityContext {
	tracing := NoOpTracingContext
	if tracingContext != nil {
		tracing = *tracingContext
	}

	return obstypes.ObservabilityContext{
		Tracing:     tracing,
		LoggerVNext: deriveLoggerContext(tracing),
		Metrics:     deriveMetricsContext(tracing),
		TracingCtx:  tracing, // alias -- preferred at forwarding sites
	}
}

// ResolveObservabilityContext resolves a partial observability context into a
// complete ObservabilityContext with no-op defaults for any missing fields.
//
// Explicitly provided logger/metrics contexts are preserved (e.g. when set
// upstream). When missing, they are derived from the tracing context's span,
// following the same derivation logic as CreateObservabilityContext.
func ResolveObservabilityContext(partial *obstypes.ObservabilityContext) obstypes.ObservabilityContext {
	if partial == nil {
		return CreateObservabilityContext(nil)
	}

	tracing := partial.Tracing
	if tracing.CurrentSpan == nil {
		tracing = partial.TracingCtx
	}
	if tracing.CurrentSpan == nil {
		tracing = NoOpTracingContext
	}

	loggerVNext := partial.LoggerVNext
	if loggerVNext == nil {
		loggerVNext = deriveLoggerContext(tracing)
	}

	metrics := partial.Metrics
	if metrics == nil {
		metrics = deriveMetricsContext(tracing)
	}

	return obstypes.ObservabilityContext{
		Tracing:     tracing,
		LoggerVNext: loggerVNext,
		Metrics:     metrics,
		TracingCtx:  tracing, // alias -- preferred at forwarding sites
	}
}
