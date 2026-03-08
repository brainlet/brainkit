// Ported from: packages/core/src/observability/no-op.ts
package observability

import (
	"github.com/brainlet/brainkit/agent-kit/core/logger"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
)

// ============================================================================
// No-Op Metric Instruments
// ============================================================================

// noOpCounter is a counter that silently discards all operations.
type noOpCounter struct{}

func (c noOpCounter) Add(_ float64, _ ...map[string]string) {}

// noOpGauge is a gauge that silently discards all operations.
type noOpGauge struct{}

func (g noOpGauge) Set(_ float64, _ ...map[string]string) {}

// noOpHistogram is a histogram that silently discards all operations.
type noOpHistogram struct{}

func (h noOpHistogram) Record(_ float64, _ ...map[string]string) {}

// ============================================================================
// No-Op TracingContext
// ============================================================================

// NoOpTracingContext is a no-op tracing context used when observability is not configured.
var NoOpTracingContext = obstypes.TracingContext{
	CurrentSpan: nil,
}

// ============================================================================
// No-Op LoggerContext
// ============================================================================

// noOpLoggerCtx silently discards all log calls.
type noOpLoggerCtx struct{}

func (l noOpLoggerCtx) Debug(_ string, _ ...map[string]any) {}
func (l noOpLoggerCtx) Info(_ string, _ ...map[string]any)  {}
func (l noOpLoggerCtx) Warn(_ string, _ ...map[string]any)  {}
func (l noOpLoggerCtx) Error(_ string, _ ...map[string]any) {}
func (l noOpLoggerCtx) Fatal(_ string, _ ...map[string]any) {}

// NoOpLoggerContext is a no-op LoggerContext that silently discards all log calls.
var NoOpLoggerContext obstypes.LoggerContext = noOpLoggerCtx{}

// ============================================================================
// No-Op MetricsContext
// ============================================================================

// noOpMetricsCtx silently discards all metric operations.
type noOpMetricsCtx struct{}

func (m noOpMetricsCtx) Counter(_ string) obstypes.Counter     { return noOpCounter{} }
func (m noOpMetricsCtx) Gauge(_ string) obstypes.Gauge         { return noOpGauge{} }
func (m noOpMetricsCtx) Histogram(_ string) obstypes.Histogram { return noOpHistogram{} }

// NoOpMetricsContext is a no-op MetricsContext that silently discards all operations.
var NoOpMetricsContext obstypes.MetricsContext = noOpMetricsCtx{}

// ============================================================================
// NoOpObservability
// ============================================================================

// NoOpObservability is a no-op implementation of ObservabilityEntrypoint.
type NoOpObservability struct{}

// Shutdown is a no-op.
func (n *NoOpObservability) Shutdown() error { return nil }

// SetMastraContext is a no-op.
func (n *NoOpObservability) SetMastraContext(_ any) {}

// SetLogger is a no-op.
func (n *NoOpObservability) SetLogger(_ logger.IMastraLogger) {}

// GetSelectedInstance always returns nil.
func (n *NoOpObservability) GetSelectedInstance(_ obstypes.ConfigSelectorOptions) obstypes.ObservabilityInstance {
	return nil
}

// RegisterInstance is a no-op.
func (n *NoOpObservability) RegisterInstance(_ string, _ obstypes.ObservabilityInstance, _ bool) {}

// GetInstance always returns nil.
func (n *NoOpObservability) GetInstance(_ string) obstypes.ObservabilityInstance {
	return nil
}

// GetDefaultInstance always returns nil.
func (n *NoOpObservability) GetDefaultInstance() obstypes.ObservabilityInstance {
	return nil
}

// ListInstances returns an empty map.
func (n *NoOpObservability) ListInstances() map[string]obstypes.ObservabilityInstance {
	return make(map[string]obstypes.ObservabilityInstance)
}

// UnregisterInstance always returns false.
func (n *NoOpObservability) UnregisterInstance(_ string) bool {
	return false
}

// HasInstance always returns false.
func (n *NoOpObservability) HasInstance(_ string) bool {
	return false
}

// SetConfigSelector is a no-op.
func (n *NoOpObservability) SetConfigSelector(_ obstypes.ConfigSelector) {}

// Clear is a no-op.
func (n *NoOpObservability) Clear() {}

// Compile-time assertion that NoOpObservability implements ObservabilityEntrypoint.
var _ obstypes.ObservabilityEntrypoint = (*NoOpObservability)(nil)
