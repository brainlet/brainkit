// Ported from: packages/core/src/observability/index.ts
//
// Package observability provides the core observability utilities and types for
// the agent-kit framework.
//
// To use observability, create an ObservabilityInstance and pass it to the Mastra
// constructor. This package provides no-op defaults for all observability
// primitives (tracing, logging, metrics) that are used when observability is
// not configured.
//
// Sub-packages:
//   - types: All observability type definitions (SpanType, EntityType, interfaces, etc.)
//
// Exported from this package:
//   - NoOpTracingContext, NoOpLoggerContext, NoOpMetricsContext: No-op defaults
//   - NoOpObservability: No-op ObservabilityEntrypoint implementation
//   - CreateObservabilityContext, ResolveObservabilityContext: Context factory functions
//   - GetOrCreateSpan: Span creation helper
//   - ExecuteWithContext, ExecuteWithContextSync: Context execution helpers
package observability
