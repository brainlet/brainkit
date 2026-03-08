// Ported from: packages/core/src/observability/types/metrics.ts
package types

import "time"

// ============================================================================
// Metric Type
// ============================================================================

// MetricType identifies the type of metric.
type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

// ============================================================================
// MetricsContext (API Interface)
// ============================================================================

// MetricsContext is the API interface for emitting metrics.
// Provides counter, gauge, and histogram metric types.
type MetricsContext interface {
	Counter(name string) Counter
	Gauge(name string) Gauge
	Histogram(name string) Histogram
}

// Counter is a monotonically increasing metric instrument.
type Counter interface {
	Add(value float64, additionalLabels ...map[string]string)
}

// Gauge is a metric instrument that can go up and down.
type Gauge interface {
	Set(value float64, additionalLabels ...map[string]string)
}

// Histogram is a metric instrument that records a distribution of values.
type Histogram interface {
	Record(value float64, additionalLabels ...map[string]string)
}

// ============================================================================
// ExportedMetric (Event Bus Transport)
// ============================================================================

// ExportedMetric is metric data transported via the event bus.
// Must be JSON-serializable.
type ExportedMetric struct {
	// Timestamp is when the metric was recorded.
	Timestamp time.Time `json:"timestamp"`
	// Name is the metric name (e.g., mastra_agent_duration_ms).
	Name string `json:"name"`
	// MetricType is the type of metric.
	MetricType MetricType `json:"metricType"`
	// Value is the metric value (single observation).
	Value float64 `json:"value"`
	// Labels for dimensional filtering.
	Labels map[string]string `json:"labels"`
	// Metadata is user-defined metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ============================================================================
// MetricEvent (Event Bus Event)
// ============================================================================

// MetricEvent is a metric event emitted to the ObservabilityBus.
type MetricEvent struct {
	Type   string         `json:"type"` // always "metric"
	Metric ExportedMetric `json:"metric"`
}

// NewMetricEvent creates a new MetricEvent with the type set to "metric".
func NewMetricEvent(metric ExportedMetric) MetricEvent {
	return MetricEvent{
		Type:   "metric",
		Metric: metric,
	}
}

// ============================================================================
// Cardinality Protection
// ============================================================================

// DefaultBlockedLabels contains labels to block from metrics to prevent cardinality explosion.
var DefaultBlockedLabels = []string{
	"trace_id",
	"span_id",
	"run_id",
	"request_id",
	"user_id",
	"resource_id",
	"session_id",
	"thread_id",
}

// CardinalityConfig holds cardinality protection configuration.
type CardinalityConfig struct {
	// BlockedLabels to block from metrics. Replaces the default list entirely when set.
	// nil (default) uses DefaultBlockedLabels.
	// Empty slice disables label blocking.
	BlockedLabels []string `json:"blockedLabels,omitempty"`
	// BlockUUIDs controls whether to block UUID-like values in labels. Default: true.
	BlockUUIDs *bool `json:"blockUUIDs,omitempty"`
}

// MetricsConfig holds metrics-specific configuration.
type MetricsConfig struct {
	// Enabled indicates whether metrics are enabled.
	Enabled *bool `json:"enabled,omitempty"`
	// Cardinality protection settings.
	Cardinality *CardinalityConfig `json:"cardinality,omitempty"`
}
