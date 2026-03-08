// Ported from: packages/core/src/observability/types/index.ts
//
// Package types provides the core observability type definitions for the agent-kit
// observability system.
//
// This package contains types ported from the Mastra TypeScript observability system:
//   - core.go: ObservabilityContext, EventBus, Instance, Entrypoint, Sampling, Config, Exporter, Bridge
//   - tracing.go: SpanType, EntityType, Span interfaces, span attributes, span options
//   - logging.go: LogLevel, LoggerContext, ExportedLog, LogEvent
//   - metrics.go: MetricType, MetricsContext, Counter, Gauge, Histogram, ExportedMetric, MetricEvent
//   - scores.go: ScoreInput, ExportedScore, ScoreEvent
//   - feedback.go: FeedbackInput, ExportedFeedback, FeedbackEvent
package types
