// Ported from: packages/core/src/observability/context-factory.test.ts
package observability

import (
	"testing"
	"time"

	"github.com/brainlet/brainkit/agent-kit/core/logger"
	obstypes "github.com/brainlet/brainkit/agent-kit/core/observability/types"
)

// ============================================================================
// Helpers
// ============================================================================

// mockLoggerCtx is a minimal LoggerContext for testing.
type mockLoggerCtx struct{}

func (m *mockLoggerCtx) Debug(_ string, _ ...map[string]any) {}
func (m *mockLoggerCtx) Info(_ string, _ ...map[string]any)  {}
func (m *mockLoggerCtx) Warn(_ string, _ ...map[string]any)  {}
func (m *mockLoggerCtx) Error(_ string, _ ...map[string]any) {}
func (m *mockLoggerCtx) Fatal(_ string, _ ...map[string]any) {}

func newMockLoggerContext() obstypes.LoggerContext {
	return &mockLoggerCtx{}
}

// mockMetricsCtx is a minimal MetricsContext for testing.
type mockMetricsCtx struct{}

func (m *mockMetricsCtx) Counter(_ string) obstypes.Counter     { return &mockCounter{} }
func (m *mockMetricsCtx) Gauge(_ string) obstypes.Gauge         { return &mockGauge{} }
func (m *mockMetricsCtx) Histogram(_ string) obstypes.Histogram { return &mockHistogram{} }

type mockCounter struct{}

func (c *mockCounter) Add(_ float64, _ ...map[string]string) {}

type mockGauge struct{}

func (g *mockGauge) Set(_ float64, _ ...map[string]string) {}

type mockHistogram struct{}

func (h *mockHistogram) Record(_ float64, _ ...map[string]string) {}

func newMockMetricsContext() obstypes.MetricsContext {
	return &mockMetricsCtx{}
}

// mockObsInstance is a mock ObservabilityInstance that returns provided logger/metrics.
type mockObsInstance struct {
	logger             obstypes.LoggerContext
	metrics            obstypes.MetricsContext
	getLoggerCalled    bool
	getLoggerCalledArg obstypes.Span
	getMetricsCalled   bool
	getMetricsCalledArg obstypes.Span
}

func (m *mockObsInstance) GetLoggerContext(span obstypes.Span) obstypes.LoggerContext {
	m.getLoggerCalled = true
	m.getLoggerCalledArg = span
	return m.logger
}

func (m *mockObsInstance) GetMetricsContext(span obstypes.Span) obstypes.MetricsContext {
	m.getMetricsCalled = true
	m.getMetricsCalledArg = span
	return m.metrics
}

// Stub methods to satisfy the ObservabilityInstance interface.
func (m *mockObsInstance) GetConfig() obstypes.ObservabilityInstanceConfig {
	return obstypes.ObservabilityInstanceConfig{}
}
func (m *mockObsInstance) GetExporters() []obstypes.ObservabilityExporter           { return nil }
func (m *mockObsInstance) GetSpanOutputProcessors() []obstypes.SpanOutputProcessor { return nil }
func (m *mockObsInstance) GetLogger() logger.IMastraLogger                         { return nil }
func (m *mockObsInstance) GetBridge() obstypes.ObservabilityBridge                 { return nil }
func (m *mockObsInstance) StartSpan(_ obstypes.StartSpanOptions) obstypes.Span     { return nil }
func (m *mockObsInstance) RebuildSpan(_ obstypes.ExportedSpan) obstypes.Span       { return nil }
func (m *mockObsInstance) Flush() error                                            { return nil }
func (m *mockObsInstance) Shutdown() error                                         { return nil }
func (m *mockObsInstance) SetLogger(_ logger.IMastraLogger)                        {}

// mockSpanForFactory is a test span that returns a mockObsInstance.
type mockSpanForFactory struct {
	spanID   string
	instance *mockObsInstance
}

func (s *mockSpanForFactory) ID() string                                  { return s.spanID }
func (s *mockSpanForFactory) TraceID() string                             { return "test-trace" }
func (s *mockSpanForFactory) Name() string                                { return "test-span" }
func (s *mockSpanForFactory) Type() obstypes.SpanType                     { return obstypes.SpanTypeGeneric }
func (s *mockSpanForFactory) GetEntityType() *obstypes.EntityType         { return nil }
func (s *mockSpanForFactory) EntityID() string                            { return "" }
func (s *mockSpanForFactory) EntityName() string                          { return "" }
func (s *mockSpanForFactory) Attributes() obstypes.AnySpanAttributes      { return nil }
func (s *mockSpanForFactory) Metadata() map[string]any                    { return nil }
func (s *mockSpanForFactory) Tags() []string                              { return nil }
func (s *mockSpanForFactory) Input() any                                  { return nil }
func (s *mockSpanForFactory) Output() any                                 { return nil }
func (s *mockSpanForFactory) ErrorInfo() *obstypes.SpanErrorInfo          { return nil }
func (s *mockSpanForFactory) IsEvent() bool                               { return false }
func (s *mockSpanForFactory) IsInternal() bool                            { return false }
func (s *mockSpanForFactory) Parent() obstypes.Span                       { return nil }
func (s *mockSpanForFactory) GetTraceState() *obstypes.TraceState         { return nil }
func (s *mockSpanForFactory) End(_ *obstypes.EndSpanOptions)              {}
func (s *mockSpanForFactory) Error(_ obstypes.ErrorSpanOptions)           {}
func (s *mockSpanForFactory) Update(_ obstypes.UpdateSpanOptions)         {}
func (s *mockSpanForFactory) CreateChildSpan(_ obstypes.ChildSpanOptions) obstypes.Span { return nil }
func (s *mockSpanForFactory) CreateEventSpan(_ obstypes.ChildEventOptions) obstypes.Span { return nil }
func (s *mockSpanForFactory) IsRootSpan() bool                            { return true }
func (s *mockSpanForFactory) IsValid() bool                               { return true }
func (s *mockSpanForFactory) GetParentSpanID(_ bool) string               { return "" }
func (s *mockSpanForFactory) FindParent(_ obstypes.SpanType) obstypes.Span { return nil }
func (s *mockSpanForFactory) ExportSpan(_ bool) *obstypes.ExportedSpan    { return nil }
func (s *mockSpanForFactory) ExternalTraceID() string                     { return s.TraceID() }
func (s *mockSpanForFactory) ExecuteInContext(fn func() (any, error)) (any, error) { return fn() }
func (s *mockSpanForFactory) ExecuteInContextSync(fn func() any) any      { return fn() }

func (s *mockSpanForFactory) ObservabilityInstance() obstypes.ObservabilityInstance {
	if s.instance == nil {
		return nil
	}
	return s.instance
}

func (s *mockSpanForFactory) StartTime() time.Time  { return time.Time{} }
func (s *mockSpanForFactory) EndTime() *time.Time   { return nil }

// mockSpanWithInstance creates a mock span that returns the given logger/metrics
// from its ObservabilityInstance (or nil if not provided).
func mockSpanWithInstance(logger obstypes.LoggerContext, metrics obstypes.MetricsContext) (*mockSpanForFactory, *mockObsInstance) {
	inst := &mockObsInstance{
		logger:  logger,
		metrics: metrics,
	}
	span := &mockSpanForFactory{
		spanID:   "test-span",
		instance: inst,
	}
	return span, inst
}

// bareSpan is a span with no observability instance.
type bareSpan struct {
	mockSpanForFactory
}

func newBareSpan() *bareSpan {
	return &bareSpan{
		mockSpanForFactory: mockSpanForFactory{
			spanID:   "bare-span",
			instance: nil,
		},
	}
}

// Compile-time interface checks.
var _ obstypes.Span = (*mockSpanForFactory)(nil)
var _ obstypes.ObservabilityInstance = (*mockObsInstance)(nil)

// ============================================================================
// createObservabilityContext
// ============================================================================

func TestCreateObservabilityContext(t *testing.T) {
	t.Run("returns no-op contexts when called without arguments", func(t *testing.T) {
		ctx := CreateObservabilityContext(nil)

		if ctx.Tracing.CurrentSpan != nil {
			t.Errorf("expected Tracing.CurrentSpan to be nil, got %v", ctx.Tracing.CurrentSpan)
		}
		if ctx.LoggerVNext != NoOpLoggerContext {
			t.Errorf("expected LoggerVNext to be NoOpLoggerContext")
		}
		if ctx.Metrics != NoOpMetricsContext {
			t.Errorf("expected Metrics to be NoOpMetricsContext")
		}
	})

	t.Run("returns tracingContext alias pointing to tracing", func(t *testing.T) {
		ctx := CreateObservabilityContext(nil)

		// TracingCtx should equal Tracing (same value)
		if ctx.TracingCtx.CurrentSpan != ctx.Tracing.CurrentSpan {
			t.Errorf("expected TracingCtx to equal Tracing")
		}
	})

	t.Run("uses provided tracing context when passed", func(t *testing.T) {
		span, _ := mockSpanWithInstance(nil, nil)
		tracing := obstypes.TracingContext{CurrentSpan: span}

		ctx := CreateObservabilityContext(&tracing)

		if ctx.Tracing.CurrentSpan != span {
			t.Errorf("expected Tracing.CurrentSpan to be the mock span")
		}
		if ctx.TracingCtx.CurrentSpan != span {
			t.Errorf("expected TracingCtx.CurrentSpan to be the mock span")
		}
	})

	t.Run("derives logger context from span observability instance", func(t *testing.T) {
		logger := newMockLoggerContext()
		span, inst := mockSpanWithInstance(logger, nil)
		tracing := obstypes.TracingContext{CurrentSpan: span}

		ctx := CreateObservabilityContext(&tracing)

		if ctx.LoggerVNext != logger {
			t.Errorf("expected LoggerVNext to be the mock logger")
		}
		if !inst.getLoggerCalled {
			t.Errorf("expected GetLoggerContext to have been called")
		}
		if inst.getLoggerCalledArg != span {
			t.Errorf("expected GetLoggerContext to have been called with the span")
		}
	})

	t.Run("derives metrics context from span observability instance", func(t *testing.T) {
		metrics := newMockMetricsContext()
		span, inst := mockSpanWithInstance(nil, metrics)
		tracing := obstypes.TracingContext{CurrentSpan: span}

		ctx := CreateObservabilityContext(&tracing)

		if ctx.Metrics != metrics {
			t.Errorf("expected Metrics to be the mock metrics")
		}
		if !inst.getMetricsCalled {
			t.Errorf("expected GetMetricsContext to have been called")
		}
		if inst.getMetricsCalledArg != span {
			t.Errorf("expected GetMetricsContext to have been called with the span")
		}
	})

	t.Run("derives both logger and metrics from span observability instance", func(t *testing.T) {
		logger := newMockLoggerContext()
		metrics := newMockMetricsContext()
		span, _ := mockSpanWithInstance(logger, metrics)
		tracing := obstypes.TracingContext{CurrentSpan: span}

		ctx := CreateObservabilityContext(&tracing)

		if ctx.LoggerVNext != logger {
			t.Errorf("expected LoggerVNext to be the mock logger")
		}
		if ctx.Metrics != metrics {
			t.Errorf("expected Metrics to be the mock metrics")
		}
	})

	t.Run("falls back to no-op when instance does not implement getLoggerContext", func(t *testing.T) {
		// Instance returns nil for logger, so deriveLoggerContext falls back to no-op.
		metrics := newMockMetricsContext()
		span, _ := mockSpanWithInstance(nil, metrics)
		tracing := obstypes.TracingContext{CurrentSpan: span}

		ctx := CreateObservabilityContext(&tracing)

		if ctx.LoggerVNext != NoOpLoggerContext {
			t.Errorf("expected LoggerVNext to fall back to NoOpLoggerContext")
		}
	})

	t.Run("falls back to no-op when instance does not implement getMetricsContext", func(t *testing.T) {
		// Instance returns nil for metrics, so deriveMetricsContext falls back to no-op.
		logger := newMockLoggerContext()
		span, _ := mockSpanWithInstance(logger, nil)
		tracing := obstypes.TracingContext{CurrentSpan: span}

		ctx := CreateObservabilityContext(&tracing)

		if ctx.Metrics != NoOpMetricsContext {
			t.Errorf("expected Metrics to fall back to NoOpMetricsContext")
		}
	})

	t.Run("falls back to no-op when span has no observability instance", func(t *testing.T) {
		span := newBareSpan()
		tracing := obstypes.TracingContext{CurrentSpan: span}

		ctx := CreateObservabilityContext(&tracing)

		if ctx.LoggerVNext != NoOpLoggerContext {
			t.Errorf("expected LoggerVNext to fall back to NoOpLoggerContext")
		}
		if ctx.Metrics != NoOpMetricsContext {
			t.Errorf("expected Metrics to fall back to NoOpMetricsContext")
		}
	})
}

// ============================================================================
// resolveObservabilityContext
// ============================================================================

func TestResolveObservabilityContext(t *testing.T) {
	t.Run("returns no-op contexts when called with empty partial", func(t *testing.T) {
		ctx := ResolveObservabilityContext(&obstypes.ObservabilityContext{})

		if ctx.Tracing.CurrentSpan != nil {
			t.Errorf("expected Tracing.CurrentSpan to be nil, got %v", ctx.Tracing.CurrentSpan)
		}
		if ctx.LoggerVNext != NoOpLoggerContext {
			t.Errorf("expected LoggerVNext to be NoOpLoggerContext")
		}
		if ctx.Metrics != NoOpMetricsContext {
			t.Errorf("expected Metrics to be NoOpMetricsContext")
		}
	})

	t.Run("uses provided logger context when passed", func(t *testing.T) {
		logger := newMockLoggerContext()
		ctx := ResolveObservabilityContext(&obstypes.ObservabilityContext{
			LoggerVNext: logger,
		})

		if ctx.LoggerVNext != logger {
			t.Errorf("expected LoggerVNext to be the provided logger")
		}
	})

	t.Run("uses provided metrics context when passed", func(t *testing.T) {
		metrics := newMockMetricsContext()
		ctx := ResolveObservabilityContext(&obstypes.ObservabilityContext{
			Metrics: metrics,
		})

		if ctx.Metrics != metrics {
			t.Errorf("expected Metrics to be the provided metrics")
		}
	})

	t.Run("uses all provided contexts when passed", func(t *testing.T) {
		span, _ := mockSpanWithInstance(nil, nil)
		tracing := obstypes.TracingContext{CurrentSpan: span}
		logger := newMockLoggerContext()
		metrics := newMockMetricsContext()

		ctx := ResolveObservabilityContext(&obstypes.ObservabilityContext{
			Tracing:     tracing,
			LoggerVNext: logger,
			Metrics:     metrics,
		})

		if ctx.Tracing.CurrentSpan != span {
			t.Errorf("expected Tracing.CurrentSpan to be the provided span")
		}
		if ctx.LoggerVNext != logger {
			t.Errorf("expected LoggerVNext to be the provided logger")
		}
		if ctx.Metrics != metrics {
			t.Errorf("expected Metrics to be the provided metrics")
		}
	})

	t.Run("prefers tracing over tracingContext alias", func(t *testing.T) {
		span1, _ := mockSpanWithInstance(nil, nil)
		span2, _ := mockSpanWithInstance(nil, nil)
		span2.spanID = "span-2"

		ctx := ResolveObservabilityContext(&obstypes.ObservabilityContext{
			Tracing:    obstypes.TracingContext{CurrentSpan: span1},
			TracingCtx: obstypes.TracingContext{CurrentSpan: span2},
		})

		if ctx.Tracing.CurrentSpan != span1 {
			t.Errorf("expected Tracing.CurrentSpan to be span1 (Tracing takes precedence)")
		}
	})

	t.Run("falls back to tracingContext alias when tracing is missing", func(t *testing.T) {
		span, _ := mockSpanWithInstance(nil, nil)

		ctx := ResolveObservabilityContext(&obstypes.ObservabilityContext{
			TracingCtx: obstypes.TracingContext{CurrentSpan: span},
		})

		if ctx.Tracing.CurrentSpan != span {
			t.Errorf("expected Tracing.CurrentSpan to fall back to TracingCtx span")
		}
	})

	t.Run("derives logger from span when not explicitly provided", func(t *testing.T) {
		logger := newMockLoggerContext()
		span, inst := mockSpanWithInstance(logger, nil)
		tracing := obstypes.TracingContext{CurrentSpan: span}

		ctx := ResolveObservabilityContext(&obstypes.ObservabilityContext{
			Tracing: tracing,
		})

		if ctx.LoggerVNext != logger {
			t.Errorf("expected LoggerVNext to be derived from span")
		}
		if !inst.getLoggerCalled {
			t.Errorf("expected GetLoggerContext to have been called")
		}
		if inst.getLoggerCalledArg != span {
			t.Errorf("expected GetLoggerContext to have been called with the span")
		}
	})

	t.Run("derives metrics from span when not explicitly provided", func(t *testing.T) {
		metrics := newMockMetricsContext()
		span, inst := mockSpanWithInstance(nil, metrics)
		tracing := obstypes.TracingContext{CurrentSpan: span}

		ctx := ResolveObservabilityContext(&obstypes.ObservabilityContext{
			Tracing: tracing,
		})

		if ctx.Metrics != metrics {
			t.Errorf("expected Metrics to be derived from span")
		}
		if !inst.getMetricsCalled {
			t.Errorf("expected GetMetricsContext to have been called")
		}
		if inst.getMetricsCalledArg != span {
			t.Errorf("expected GetMetricsContext to have been called with the span")
		}
	})

	t.Run("prefers explicit logger over derived logger", func(t *testing.T) {
		explicitLogger := newMockLoggerContext()
		derivedLogger := newMockLoggerContext()
		span, _ := mockSpanWithInstance(derivedLogger, nil)
		tracing := obstypes.TracingContext{CurrentSpan: span}

		ctx := ResolveObservabilityContext(&obstypes.ObservabilityContext{
			Tracing:     tracing,
			LoggerVNext: explicitLogger,
		})

		if ctx.LoggerVNext != explicitLogger {
			t.Errorf("expected LoggerVNext to be the explicit logger, not derived")
		}
	})

	t.Run("prefers explicit metrics over derived metrics", func(t *testing.T) {
		explicitMetrics := newMockMetricsContext()
		derivedMetrics := newMockMetricsContext()
		span, _ := mockSpanWithInstance(nil, derivedMetrics)
		tracing := obstypes.TracingContext{CurrentSpan: span}

		ctx := ResolveObservabilityContext(&obstypes.ObservabilityContext{
			Tracing: tracing,
			Metrics: explicitMetrics,
		})

		if ctx.Metrics != explicitMetrics {
			t.Errorf("expected Metrics to be the explicit metrics, not derived")
		}
	})
}

// ============================================================================
// noOpLoggerContext
// ============================================================================

func TestNoOpLoggerContext(t *testing.T) {
	t.Run("has all required methods", func(t *testing.T) {
		// Verify NoOpLoggerContext satisfies LoggerContext interface.
		var lc obstypes.LoggerContext = NoOpLoggerContext
		if lc == nil {
			t.Fatal("NoOpLoggerContext should not be nil")
		}
	})

	t.Run("debug does not throw", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Debug should not panic: %v", r)
			}
		}()
		NoOpLoggerContext.Debug("test message")
		NoOpLoggerContext.Debug("test message", map[string]any{"key": "value"})
	})

	t.Run("info does not throw", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Info should not panic: %v", r)
			}
		}()
		NoOpLoggerContext.Info("test message")
		NoOpLoggerContext.Info("test message", map[string]any{"key": "value"})
	})

	t.Run("warn does not throw", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Warn should not panic: %v", r)
			}
		}()
		NoOpLoggerContext.Warn("test message")
		NoOpLoggerContext.Warn("test message", map[string]any{"key": "value"})
	})

	t.Run("error does not throw", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Error should not panic: %v", r)
			}
		}()
		NoOpLoggerContext.Error("test message")
		NoOpLoggerContext.Error("test message", map[string]any{"key": "value"})
	})

	t.Run("fatal does not throw", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Fatal should not panic: %v", r)
			}
		}()
		NoOpLoggerContext.Fatal("test message")
		NoOpLoggerContext.Fatal("test message", map[string]any{"key": "value"})
	})
}

// ============================================================================
// noOpMetricsContext
// ============================================================================

func TestNoOpMetricsContext(t *testing.T) {
	t.Run("has all required methods", func(t *testing.T) {
		// Verify NoOpMetricsContext satisfies MetricsContext interface.
		var mc obstypes.MetricsContext = NoOpMetricsContext
		if mc == nil {
			t.Fatal("NoOpMetricsContext should not be nil")
		}
	})

	t.Run("counter", func(t *testing.T) {
		t.Run("returns an object with add method", func(t *testing.T) {
			counter := NoOpMetricsContext.Counter("test_counter")
			if counter == nil {
				t.Fatal("Counter should not return nil")
			}
		})

		t.Run("add does not throw", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Add should not panic: %v", r)
				}
			}()
			counter := NoOpMetricsContext.Counter("test_counter")
			counter.Add(1)
			counter.Add(5, map[string]string{"label": "value"})
		})
	})

	t.Run("gauge", func(t *testing.T) {
		t.Run("returns an object with set method", func(t *testing.T) {
			gauge := NoOpMetricsContext.Gauge("test_gauge")
			if gauge == nil {
				t.Fatal("Gauge should not return nil")
			}
		})

		t.Run("set does not throw", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Set should not panic: %v", r)
				}
			}()
			gauge := NoOpMetricsContext.Gauge("test_gauge")
			gauge.Set(42)
			gauge.Set(100, map[string]string{"label": "value"})
		})
	})

	t.Run("histogram", func(t *testing.T) {
		t.Run("returns an object with record method", func(t *testing.T) {
			histogram := NoOpMetricsContext.Histogram("test_histogram")
			if histogram == nil {
				t.Fatal("Histogram should not return nil")
			}
		})

		t.Run("record does not throw", func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Record should not panic: %v", r)
				}
			}()
			histogram := NoOpMetricsContext.Histogram("test_histogram")
			histogram.Record(0.5)
			histogram.Record(123.45, map[string]string{"label": "value"})
		})
	})
}
