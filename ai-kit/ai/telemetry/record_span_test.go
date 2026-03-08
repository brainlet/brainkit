// Ported from: packages/ai/src/telemetry/record-span.test.ts
package telemetry

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRecordSpanTyped(t *testing.T) {
	t.Run("should execute the function and return its result", func(t *testing.T) {
		tracer := &MockTracer{}

		result, err := RecordSpanTyped(RecordSpanOptions{
			Name:       "test-span",
			Tracer:     tracer,
			Attributes: Attributes{"key": "value"},
			Fn: func(span Span) (interface{}, error) {
				return "test-result", nil
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, "test-result", result)
		assert.Equal(t, 1, len(tracer.Spans))
		assert.Equal(t, "test-span", tracer.Spans[0].SpanName)
		assert.Equal(t, Attributes{"key": "value"}, tracer.Spans[0].SpanAttributes)
	})

	t.Run("should end span when endWhenDone is true (default)", func(t *testing.T) {
		tracer := &MockTracer{}

		_, err := RecordSpanTyped(RecordSpanOptions{
			Name:       "test-span",
			Tracer:     tracer,
			Attributes: Attributes{},
			Fn: func(span Span) (interface{}, error) {
				return "result", nil
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, len(tracer.Spans))
		assert.True(t, tracer.Spans[0].Ended)
	})

	t.Run("should not end span when endWhenDone is false", func(t *testing.T) {
		tracer := &MockTracer{}
		endWhenDone := false

		_, err := RecordSpanTyped(RecordSpanOptions{
			Name:        "test-span",
			Tracer:      tracer,
			Attributes:  Attributes{},
			EndWhenDone: &endWhenDone,
			Fn: func(span Span) (interface{}, error) {
				return "result", nil
			},
		})

		assert.NoError(t, err)
		assert.Equal(t, 1, len(tracer.Spans))
		assert.False(t, tracer.Spans[0].Ended)
	})

	t.Run("should record error and end span on exception", func(t *testing.T) {
		tracer := &MockTracer{}
		testErr := errors.New("Test error")

		_, err := RecordSpanTyped(RecordSpanOptions{
			Name:       "test-span",
			Tracer:     tracer,
			Attributes: Attributes{},
			Fn: func(span Span) (interface{}, error) {
				return nil, testErr
			},
		})

		assert.Error(t, err)
		assert.Equal(t, "Test error", err.Error())
		assert.Equal(t, 1, len(tracer.Spans))
		assert.NotNil(t, tracer.Spans[0].Status)
		assert.Equal(t, SpanStatusCodeError, tracer.Spans[0].Status.Code)
		assert.Equal(t, "Test error", tracer.Spans[0].Status.Message)
		assert.Equal(t, 1, len(tracer.Spans[0].Events))
		assert.Equal(t, "exception", tracer.Spans[0].Events[0].Name)
	})
}

func TestRecordErrorOnSpan(t *testing.T) {
	t.Run("should record exception for error instances", func(t *testing.T) {
		span := &MockSpan{}
		testErr := errors.New("Test error")

		RecordErrorOnSpan(span, testErr)

		assert.Equal(t, 1, len(span.Events))
		assert.Equal(t, "exception", span.Events[0].Name)
		assert.NotNil(t, span.Status)
		assert.Equal(t, SpanStatusCodeError, span.Status.Code)
		assert.Equal(t, "Test error", span.Status.Message)
	})

	t.Run("should set error status for nil error", func(t *testing.T) {
		span := &MockSpan{}

		RecordErrorOnSpan(span, nil)

		assert.NotNil(t, span.Status)
		assert.Equal(t, SpanStatusCodeError, span.Status.Code)
	})
}
