// Ported from: packages/ai/src/telemetry/record-span.ts
package telemetry

import "fmt"

// RecordSpanOptions configures the RecordSpan call.
type RecordSpanOptions struct {
	// Name is the name of the span.
	Name string
	// Tracer is the tracer to use.
	Tracer Tracer
	// Attributes are the span attributes.
	Attributes Attributes
	// Fn is the function to execute within the span.
	Fn func(span Span) (interface{}, error)
	// EndWhenDone controls whether the span is ended when the function completes.
	// Defaults to true.
	EndWhenDone *bool
}

// RecordSpan starts a span, executes fn within it, and handles errors.
func RecordSpan(opts RecordSpanOptions) (interface{}, error) {
	endWhenDone := true
	if opts.EndWhenDone != nil {
		endWhenDone = *opts.EndWhenDone
	}

	spanOpts := SpanStartOption{Attributes: opts.Attributes}
	result := opts.Tracer.StartActiveSpan(opts.Name, spanOpts, func(span Span) interface{} {
		type resultOrError struct {
			value interface{}
			err   error
		}

		res, err := opts.Fn(span)
		if err != nil {
			RecordErrorOnSpan(span, err)
			span.End()
			return resultOrError{value: nil, err: err}
		}

		if endWhenDone {
			span.End()
		}

		return resultOrError{value: res, err: nil}
	})

	roe := result.(struct {
		value interface{}
		err   error
	})
	return roe.value, roe.err
}

// RecordErrorOnSpan records an error on a span. Sets the span status to error.
// If the error has a message, it will be included in the status.
func RecordErrorOnSpan(span Span, err error) {
	if err != nil {
		span.RecordException(err)
		span.SetStatus(SpanStatus{
			Code:    SpanStatusCodeError,
			Message: err.Error(),
		})
	} else {
		span.SetStatus(SpanStatus{Code: SpanStatusCodeError})
	}
}

// recordSpanResultOrError is used internally by RecordSpan.
type recordSpanResultOrError struct {
	value interface{}
	err   error
}

// RecordSpanTyped is a typed version of RecordSpan that avoids interface{} casting issues.
// This is the preferred way to use RecordSpan in Go.
func RecordSpanTyped(opts RecordSpanOptions) (interface{}, error) {
	endWhenDone := true
	if opts.EndWhenDone != nil {
		endWhenDone = *opts.EndWhenDone
	}

	spanOpts := SpanStartOption{Attributes: opts.Attributes}

	var fnResult interface{}
	var fnErr error

	opts.Tracer.StartActiveSpan(opts.Name, spanOpts, func(span Span) interface{} {
		result, err := opts.Fn(span)
		if err != nil {
			RecordErrorOnSpan(span, err)
			span.End()
			fnResult = nil
			fnErr = err
			return nil
		}

		if endWhenDone {
			span.End()
		}

		fnResult = result
		fnErr = nil
		return result
	})

	return fnResult, fnErr
}

// MockSpanEvent represents an event recorded on a mock span, for testing.
type MockSpanEvent struct {
	Name       string
	Attributes Attributes
}

// MockSpan is a test implementation of Span that records all operations.
type MockSpan struct {
	SpanName       string
	SpanAttributes Attributes
	Events         []MockSpanEvent
	Status         *SpanStatus
	Ended          bool
}

func (s *MockSpan) SpanContext() SpanContext {
	return SpanContext{TraceID: "test-trace-id", SpanID: "test-span-id", TraceFlags: 0}
}

func (s *MockSpan) SetAttribute(key string, value AttributeValue) Span {
	if s.SpanAttributes == nil {
		s.SpanAttributes = Attributes{}
	}
	s.SpanAttributes[key] = value
	return s
}

func (s *MockSpan) SetAttributes(attrs Attributes) Span {
	if s.SpanAttributes == nil {
		s.SpanAttributes = Attributes{}
	}
	for k, v := range attrs {
		s.SpanAttributes[k] = v
	}
	return s
}

func (s *MockSpan) AddEvent(name string, attrs ...Attributes) Span {
	var a Attributes
	if len(attrs) > 0 {
		a = attrs[0]
	}
	s.Events = append(s.Events, MockSpanEvent{Name: name, Attributes: a})
	return s
}

func (s *MockSpan) AddLink() Span  { return s }
func (s *MockSpan) AddLinks() Span { return s }
func (s *MockSpan) SetStatus(status SpanStatus) Span {
	s.Status = &status
	return s
}
func (s *MockSpan) UpdateName(name string) Span { return s }
func (s *MockSpan) End()                        { s.Ended = true }
func (s *MockSpan) IsRecording() bool           { return false }
func (s *MockSpan) RecordException(err error) {
	errName := "Error"
	errMessage := ""
	errStack := ""
	if err != nil {
		errMessage = err.Error()
		errName = fmt.Sprintf("%T", err)
	}
	s.Events = append(s.Events, MockSpanEvent{
		Name: "exception",
		Attributes: Attributes{
			"exception.type":    errName,
			"exception.name":    errName,
			"exception.message": errMessage,
			"exception.stack":   errStack,
		},
	})
}

// MockTracer is a test implementation of Tracer that records all spans.
type MockTracer struct {
	Spans []*MockSpan
}

func (t *MockTracer) StartSpan(name string, opts ...SpanStartOption) Span {
	var attrs Attributes
	if len(opts) > 0 {
		attrs = opts[0].Attributes
	}
	span := &MockSpan{
		SpanName:       name,
		SpanAttributes: attrs,
	}
	t.Spans = append(t.Spans, span)
	return span
}

func (t *MockTracer) StartActiveSpan(name string, opts SpanStartOption, fn func(Span) interface{}) interface{} {
	span := &MockSpan{
		SpanName:       name,
		SpanAttributes: opts.Attributes,
	}
	t.Spans = append(t.Spans, span)
	return fn(span)
}
