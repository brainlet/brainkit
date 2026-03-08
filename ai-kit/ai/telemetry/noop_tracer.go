// Ported from: packages/ai/src/telemetry/noop-tracer.ts
package telemetry

// Span represents an OpenTelemetry Span interface.
// TODO: import from go.opentelemetry.io/otel/trace
type Span interface {
	// SpanContext returns the SpanContext for this span.
	SpanContext() SpanContext
	// SetAttribute sets a single attribute on the span.
	SetAttribute(key string, value AttributeValue) Span
	// SetAttributes sets multiple attributes on the span.
	SetAttributes(attrs Attributes) Span
	// AddEvent adds an event to the span.
	AddEvent(name string, attrs ...Attributes) Span
	// AddLink adds a link to the span.
	AddLink() Span
	// AddLinks adds links to the span.
	AddLinks() Span
	// SetStatus sets the status of the span.
	SetStatus(status SpanStatus) Span
	// UpdateName updates the name of the span.
	UpdateName(name string) Span
	// End marks the span as ended.
	End()
	// IsRecording returns true if the span is recording events.
	IsRecording() bool
	// RecordException records an exception on the span.
	RecordException(err error)
}

// SpanContext contains information about a span's identity.
type SpanContext struct {
	TraceID    string
	SpanID     string
	TraceFlags int
}

// SpanStatusCode represents the status code of a span.
type SpanStatusCode int

const (
	// SpanStatusCodeUnset is the default status.
	SpanStatusCodeUnset SpanStatusCode = 0
	// SpanStatusCodeOK means the operation completed successfully.
	SpanStatusCodeOK SpanStatusCode = 1
	// SpanStatusCodeError means the operation contains an error.
	SpanStatusCodeError SpanStatusCode = 2
)

// SpanStatus represents the status of a span.
type SpanStatus struct {
	Code    SpanStatusCode
	Message string
}

// noopSpan is a Span implementation that does nothing.
type noopSpan struct{}

func (s *noopSpan) SpanContext() SpanContext {
	return SpanContext{TraceID: "", SpanID: "", TraceFlags: 0}
}
func (s *noopSpan) SetAttribute(key string, value AttributeValue) Span { return s }
func (s *noopSpan) SetAttributes(attrs Attributes) Span                { return s }
func (s *noopSpan) AddEvent(name string, attrs ...Attributes) Span     { return s }
func (s *noopSpan) AddLink() Span                                      { return s }
func (s *noopSpan) AddLinks() Span                                     { return s }
func (s *noopSpan) SetStatus(status SpanStatus) Span                   { return s }
func (s *noopSpan) UpdateName(name string) Span                        { return s }
func (s *noopSpan) End()                                               {}
func (s *noopSpan) IsRecording() bool                                  { return false }
func (s *noopSpan) RecordException(err error)                          {}

// noopTracerImpl is a Tracer implementation that does nothing (null object).
type noopTracerImpl struct{}

func (t *noopTracerImpl) StartSpan(name string, opts ...SpanStartOption) Span {
	return &noopSpan{}
}

func (t *noopTracerImpl) StartActiveSpan(name string, opts SpanStartOption, fn func(Span) interface{}) interface{} {
	return fn(&noopSpan{})
}

// NoopTracer is a Tracer that does nothing (null object pattern).
var NoopTracer Tracer = &noopTracerImpl{}
