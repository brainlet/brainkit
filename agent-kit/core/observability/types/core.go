// Ported from: packages/core/src/observability/types/core.ts
package types

import (
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	requestcontext "github.com/brainlet/brainkit/agent-kit/core/requestcontext"
)

// ============================================================================
// ObservabilityContext
// ============================================================================

// ObservabilityContext is a mixin interface that provides unified observability access.
// All execution contexts (tools, workflow steps, processors) use this
// to gain access to tracing, logging, and metrics.
//
// TracingContext is the source -- it represents your position in the span tree.
// LoggerVNext and Metrics are derived from the current span so that
// log entries and metric data points are automatically correlated to the active trace.
type ObservabilityContext struct {
	// Tracing is the tracing context for span creation and tree navigation.
	Tracing TracingContext
	// LoggerVNext is the logger derived from the current span (trace-correlated).
	// Uses VNext suffix to avoid conflict with IMastraLogger.
	LoggerVNext LoggerContext
	// Metrics is the metrics context derived from the current span (span-tagged).
	Metrics MetricsContext
	// TracingContext is an alias for Tracing. Preferred at forwarding sites where
	// the "Context" suffix clarifies that a structural context object is being passed.
	TracingCtx TracingContext
}

// ============================================================================
// ObservabilityEventBus
// ============================================================================

// ObservabilityEventBus is a generic event bus interface for observability events.
// Implementations handle buffering, batching, and delivery to exporters.
type ObservabilityEventBus interface {
	// Emit emits an event to the bus.
	Emit(event ObservabilityEvent)
	// Subscribe subscribes to events. Returns an unsubscribe function.
	Subscribe(handler func(event ObservabilityEvent)) func()
	// Flush flushes any buffered events.
	Flush() error
	// Shutdown shuts down the bus and releases resources.
	Shutdown() error
}

// ObservabilityEvent is a union of all observability event types.
// In Go we use an interface with a marker method since we can't have
// discriminated unions.
type ObservabilityEvent interface {
	observabilityEventMarker()
}

// Ensure event types implement ObservabilityEvent.
func (TracingEvent) observabilityEventMarker()  {}
func (LogEvent) observabilityEventMarker()      {}
func (MetricEvent) observabilityEventMarker()   {}
func (ScoreEvent) observabilityEventMarker()    {}
func (FeedbackEvent) observabilityEventMarker() {}

// ============================================================================
// ObservabilityInstance
// ============================================================================

// ObservabilityInstance is the primary interface for Observability.
type ObservabilityInstance interface {
	// GetConfig returns the current configuration.
	GetConfig() ObservabilityInstanceConfig
	// GetExporters returns all exporters.
	GetExporters() []ObservabilityExporter
	// GetSpanOutputProcessors returns all span output processors.
	GetSpanOutputProcessors() []SpanOutputProcessor
	// GetLogger returns the logger instance.
	GetLogger() logger.IMastraLogger
	// GetBridge returns the bridge instance if configured.
	GetBridge() ObservabilityBridge
	// StartSpan starts a new span of a specific SpanType.
	StartSpan(options StartSpanOptions) Span
	// RebuildSpan rebuilds a span from exported data for lifecycle operations.
	RebuildSpan(cached ExportedSpan) Span
	// Flush forces flush of any buffered/queued spans from all exporters and the bridge.
	Flush() error
	// Shutdown shuts down tracing and cleans up resources.
	Shutdown() error
	// SetLogger sets the logger with tracing-specific initialization log.
	SetLogger(logger logger.IMastraLogger)
	// GetLoggerContext returns a LoggerContext, optionally correlated to a span.
	GetLoggerContext(span Span) LoggerContext
	// GetMetricsContext returns a MetricsContext, optionally tagged from a span.
	GetMetricsContext(span Span) MetricsContext
}

// ============================================================================
// ObservabilityEntrypoint
// ============================================================================

// ObservabilityEntrypoint is the entry point interface for the observability system.
// The Mastra type parameter is replaced with any to avoid circular dependencies.
type ObservabilityEntrypoint interface {
	// Shutdown shuts down the observability system.
	Shutdown() error
	// SetMastraContext sets the Mastra context.
	SetMastraContext(mastra any)
	// SetLogger sets the logger.
	SetLogger(logger logger.IMastraLogger)
	// GetSelectedInstance returns the selected observability instance.
	GetSelectedInstance(options ConfigSelectorOptions) ObservabilityInstance
	// RegisterInstance registers an observability instance.
	RegisterInstance(name string, instance ObservabilityInstance, isDefault bool)
	// GetInstance returns a named observability instance.
	GetInstance(name string) ObservabilityInstance
	// GetDefaultInstance returns the default observability instance.
	GetDefaultInstance() ObservabilityInstance
	// ListInstances returns all registered instances.
	ListInstances() map[string]ObservabilityInstance
	// UnregisterInstance removes a named instance.
	UnregisterInstance(name string) bool
	// HasInstance returns true if a named instance exists.
	HasInstance(name string) bool
	// SetConfigSelector sets the config selector function.
	SetConfigSelector(selector ConfigSelector)
	// Clear removes all registered instances.
	Clear()
}

// ============================================================================
// Sampling Strategy
// ============================================================================

// SamplingStrategyType enumerates sampling strategy types.
type SamplingStrategyType string

const (
	SamplingStrategyAlways SamplingStrategyType = "always"
	SamplingStrategyNever  SamplingStrategyType = "never"
	SamplingStrategyRatio  SamplingStrategyType = "ratio"
	SamplingStrategyCustom SamplingStrategyType = "custom"
)

// SamplingStrategy represents the sampling strategy configuration.
// Use one of the constructors: NewAlwaysSampling, NewNeverSampling,
// NewRatioSampling, or NewCustomSampling.
type SamplingStrategy struct {
	// Type is the sampling strategy type.
	Type SamplingStrategyType `json:"type"`
	// Probability is used when Type is SamplingStrategyRatio.
	Probability float64 `json:"probability,omitempty"`
	// Sampler is the custom sampler function used when Type is SamplingStrategyCustom.
	Sampler func(options *CustomSamplerOptions) bool `json:"-"`
}

// NewAlwaysSampling creates an always-on sampling strategy.
func NewAlwaysSampling() SamplingStrategy {
	return SamplingStrategy{Type: SamplingStrategyAlways}
}

// NewNeverSampling creates a never-on sampling strategy.
func NewNeverSampling() SamplingStrategy {
	return SamplingStrategy{Type: SamplingStrategyNever}
}

// NewRatioSampling creates a probability-based sampling strategy.
func NewRatioSampling(probability float64) SamplingStrategy {
	return SamplingStrategy{Type: SamplingStrategyRatio, Probability: probability}
}

// NewCustomSampling creates a custom sampling strategy.
func NewCustomSampling(sampler func(options *CustomSamplerOptions) bool) SamplingStrategy {
	return SamplingStrategy{Type: SamplingStrategyCustom, Sampler: sampler}
}

// CustomSamplerOptions holds options passed when using a custom sampler strategy.
type CustomSamplerOptions struct {
	RequestContext *requestcontext.RequestContext
	Metadata       map[string]any
}

// ============================================================================
// Serialization Options
// ============================================================================

// SerializationOptions controls serialization of span data.
type SerializationOptions struct {
	// MaxStringLength is the maximum length for string values. Default: 1024.
	MaxStringLength *int `json:"maxStringLength,omitempty"`
	// MaxDepth is the maximum depth for nested objects. Default: 6.
	MaxDepth *int `json:"maxDepth,omitempty"`
	// MaxArrayLength is the maximum number of items in arrays. Default: 50.
	MaxArrayLength *int `json:"maxArrayLength,omitempty"`
	// MaxObjectKeys is the maximum number of keys in objects. Default: 50.
	MaxObjectKeys *int `json:"maxObjectKeys,omitempty"`
}

// ============================================================================
// Registry Config
// ============================================================================

// ObservabilityInstanceConfig is the configuration for a single observability instance.
type ObservabilityInstanceConfig struct {
	// Name is the unique identifier for this config in the observability registry.
	Name string `json:"name"`
	// ServiceName is the service name for observability.
	ServiceName string `json:"serviceName"`
	// Sampling controls whether tracing is collected (defaults to ALWAYS).
	Sampling *SamplingStrategy `json:"sampling,omitempty"`
	// Exporters are custom exporters.
	Exporters []ObservabilityExporter `json:"-"`
	// SpanOutputProcessors are custom processors.
	SpanOutputProcessors []SpanOutputProcessor `json:"-"`
	// Bridge is the OpenTelemetry bridge for integration with existing OTEL infrastructure.
	Bridge ObservabilityBridge `json:"-"`
	// IncludeInternalSpans is set to true to see spans internal to mastra operations.
	IncludeInternalSpans bool `json:"includeInternalSpans,omitempty"`
	// RequestContextKeys to automatically extract as metadata for all spans.
	RequestContextKeys []string `json:"requestContextKeys,omitempty"`
	// SerializationOptions controls serialization of span data.
	SerializationOptions *SerializationOptions `json:"serializationOptions,omitempty"`
}

// ObservabilityRegistryConfig is the complete observability registry configuration.
type ObservabilityRegistryConfig struct {
	// Default enables default exporters with sampling: always and sensitive data filtering.
	Default *struct {
		Enabled *bool `json:"enabled,omitempty"`
	} `json:"default,omitempty"`
	// Configs is a map of tracing instance names to their configurations.
	// Values can be ObservabilityInstanceConfig or ObservabilityInstance.
	Configs map[string]any `json:"configs,omitempty"`
	// ConfigSelector is an optional selector function.
	ConfigSelector ConfigSelector `json:"-"`
}

// ============================================================================
// Config Selector
// ============================================================================

// ConfigSelectorOptions holds options passed when using a custom tracing config selector.
type ConfigSelectorOptions struct {
	// RequestContext holds request context.
	RequestContext *requestcontext.RequestContext
}

// ConfigSelector is a function to select which tracing instance to use for a given span.
// Returns the name of the tracing instance, or empty string to use default.
type ConfigSelector func(options ConfigSelectorOptions, availableConfigs map[string]ObservabilityInstance) string

// ============================================================================
// Exporter and Bridge Interfaces
// ============================================================================

// InitExporterOptions holds options for initializing an exporter.
// Mastra is typed as any to avoid circular dependencies.
type InitExporterOptions struct {
	Mastra any
	Config *ObservabilityInstanceConfig
}

// InitBridgeOptions holds options for initializing a bridge.
// Mastra is typed as any to avoid circular dependencies.
type InitBridgeOptions struct {
	Mastra any
	Config *ObservabilityInstanceConfig
}

// AddScoreToTraceArgs holds the arguments for adding a score to a trace.
type AddScoreToTraceArgs struct {
	TraceID    string         `json:"traceId"`
	SpanID     string         `json:"spanId,omitempty"`
	Score      float64        `json:"score"`
	Reason     string         `json:"reason,omitempty"`
	ScorerName string         `json:"scorerName"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// ObservabilityEvents is the shared event interface for exporters and bridges.
type ObservabilityEvents interface {
	// OnTracingEvent handles tracing events.
	OnTracingEvent(event TracingEvent) error
	// OnLogEvent handles log events.
	OnLogEvent(event LogEvent) error
	// OnMetricEvent handles metric events.
	OnMetricEvent(event MetricEvent) error
	// OnScoreEvent handles score events.
	OnScoreEvent(event ScoreEvent) error
	// OnFeedbackEvent handles feedback events.
	OnFeedbackEvent(event FeedbackEvent) error
	// ExportTracingEvent exports tracing events.
	ExportTracingEvent(event TracingEvent) error
}

// ObservabilityExporter is the interface for tracing exporters.
type ObservabilityExporter interface {
	ObservabilityEvents
	// Name returns the exporter name.
	ExporterName() string
	// Init initializes the exporter.
	Init(options InitExporterOptions)
	// SetLogger sets the logger instance on the exporter.
	SetLogger(logger logger.IMastraLogger)
	// AddScoreToTrace adds a score to a trace.
	AddScoreToTrace(args AddScoreToTraceArgs) error
	// Flush forces flush without shutting down.
	Flush() error
	// Shutdown shuts down the exporter.
	Shutdown() error
}

// ObservabilityBridge is the interface for observability bridges.
type ObservabilityBridge interface {
	ObservabilityEvents
	// BridgeName returns the bridge name.
	BridgeName() string
	// Init initializes the bridge.
	Init(options InitBridgeOptions)
	// SetLogger sets the logger instance on the bridge.
	SetLogger(logger logger.IMastraLogger)
	// ExecuteInContext executes an async function within the tracing context of a Mastra span.
	ExecuteInContext(spanID string, fn func() (any, error)) (any, error)
	// ExecuteInContextSync executes a sync function within the tracing context of a Mastra span.
	ExecuteInContextSync(spanID string, fn func() any) any
	// CreateSpan creates a span in the bridge's tracing system.
	CreateSpan(options CreateSpanOptions) *SpanIds
	// Flush forces flush without shutting down.
	Flush() error
	// Shutdown shuts down the bridge.
	Shutdown() error
}
