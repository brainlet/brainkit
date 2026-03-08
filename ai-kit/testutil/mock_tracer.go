// Ported from: packages/ai/src/test/mock-tracer.ts
package testutil

import (
	"fmt"

	"github.com/brainlet/brainkit/ai-kit/ai/telemetry"
)

// MockTracerSpanJSON is the JSON-serializable representation of a MockTracerSpan.
type MockTracerSpanJSON struct {
	Name       string                    `json:"name"`
	Attributes telemetry.Attributes      `json:"attributes"`
	Events     []MockTracerSpanEvent     `json:"events"`
	Status     *telemetry.SpanStatus     `json:"status,omitempty"`
}

// MockTracerSpanEvent represents an event recorded on a mock span.
type MockTracerSpanEvent struct {
	Name       string               `json:"name"`
	Attributes telemetry.Attributes `json:"attributes,omitempty"`
	Time       *[2]int              `json:"time,omitempty"`
}

// MockTracerSpan is a Span implementation that records all operations for test assertions.
type MockTracerSpan struct {
	Name        string
	Context     interface{} // opaque context, not used in Go
	Options     *telemetry.SpanStartOption
	SpanAttrs   telemetry.Attributes
	Events      []MockTracerSpanEvent
	StatusVal   *telemetry.SpanStatus
	spanContext telemetry.SpanContext
}

// NewMockTracerSpan creates a new MockTracerSpan.
func NewMockTracerSpan(name string, opts *telemetry.SpanStartOption) *MockTracerSpan {
	attrs := telemetry.Attributes{}
	if opts != nil && opts.Attributes != nil {
		for k, v := range opts.Attributes {
			attrs[k] = v
		}
	}
	return &MockTracerSpan{
		Name:      name,
		Options:   opts,
		SpanAttrs: attrs,
		Events:    []MockTracerSpanEvent{},
		spanContext: telemetry.SpanContext{
			TraceID:    "test-trace-id",
			SpanID:     "test-span-id",
			TraceFlags: 0,
		},
	}
}

func (s *MockTracerSpan) SpanContext() telemetry.SpanContext { return s.spanContext }

func (s *MockTracerSpan) SetAttribute(key string, value telemetry.AttributeValue) telemetry.Span {
	if s.SpanAttrs == nil {
		s.SpanAttrs = telemetry.Attributes{}
	}
	s.SpanAttrs[key] = value
	return s
}

func (s *MockTracerSpan) SetAttributes(attrs telemetry.Attributes) telemetry.Span {
	if s.SpanAttrs == nil {
		s.SpanAttrs = telemetry.Attributes{}
	}
	for k, v := range attrs {
		s.SpanAttrs[k] = v
	}
	return s
}

func (s *MockTracerSpan) AddEvent(name string, attrs ...telemetry.Attributes) telemetry.Span {
	var a telemetry.Attributes
	if len(attrs) > 0 {
		a = attrs[0]
	}
	s.Events = append(s.Events, MockTracerSpanEvent{Name: name, Attributes: a})
	return s
}

func (s *MockTracerSpan) AddLink() telemetry.Span  { return s }
func (s *MockTracerSpan) AddLinks() telemetry.Span { return s }

func (s *MockTracerSpan) SetStatus(status telemetry.SpanStatus) telemetry.Span {
	s.StatusVal = &status
	return s
}

func (s *MockTracerSpan) UpdateName(_ string) telemetry.Span { return s }
func (s *MockTracerSpan) End()                               {}
func (s *MockTracerSpan) IsRecording() bool                  { return false }

func (s *MockTracerSpan) RecordException(err error) {
	errType := "Error"
	errName := "Error"
	errMessage := ""
	errStack := ""

	if err != nil {
		errType = fmt.Sprintf("%T", err)
		errName = errType
		errMessage = err.Error()
		// Go doesn't have stack traces on errors by default
	}

	s.Events = append(s.Events, MockTracerSpanEvent{
		Name: "exception",
		Attributes: telemetry.Attributes{
			"exception.type":    errType,
			"exception.name":    errName,
			"exception.message": errMessage,
			"exception.stack":   errStack,
		},
		Time: &[2]int{0, 0},
	})
}

// MockTracer is a Tracer implementation that records all spans for test assertions.
// This is the testutil version ported from the TS test directory, distinct from
// telemetry.MockTracer which lives in the telemetry package.
type MockTracer struct {
	Spans []*MockTracerSpan
}

// JSONSpans returns a JSON-serializable representation of all recorded spans.
func (t *MockTracer) JSONSpans() []MockTracerSpanJSON {
	result := make([]MockTracerSpanJSON, len(t.Spans))
	for i, span := range t.Spans {
		entry := MockTracerSpanJSON{
			Name:       span.Name,
			Attributes: span.SpanAttrs,
			Events:     span.Events,
		}
		if span.StatusVal != nil {
			entry.Status = span.StatusVal
		}
		result[i] = entry
	}
	return result
}

func (t *MockTracer) StartSpan(name string, opts ...telemetry.SpanStartOption) telemetry.Span {
	var o *telemetry.SpanStartOption
	if len(opts) > 0 {
		o = &opts[0]
	}
	span := NewMockTracerSpan(name, o)
	t.Spans = append(t.Spans, span)
	return span
}

func (t *MockTracer) StartActiveSpan(name string, opts telemetry.SpanStartOption, fn func(telemetry.Span) interface{}) interface{} {
	span := NewMockTracerSpan(name, &opts)
	t.Spans = append(t.Spans, span)
	return fn(span)
}
