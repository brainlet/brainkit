package sdk

import (
	"encoding/json"
	"fmt"
	"time"
)

// ErrCallerClosed is returned when Call runs on a closed Caller.
var ErrCallerClosed = fmt.Errorf("caller: closed")

// NoDeadlineError is returned when a typed Call receives no deadline.
type NoDeadlineError struct{}

func (e *NoDeadlineError) Error() string {
	return "sdk.Call: ctx has no deadline (use context.WithTimeout or WithCallTimeout)"
}
func (e *NoDeadlineError) Code() string            { return "VALIDATION_ERROR" }
func (e *NoDeadlineError) Details() map[string]any { return nil }

// CallTimeoutError is returned when the caller's deadline elapses before a
// terminal reply arrives.
type CallTimeoutError struct {
	Topic   string
	Elapsed time.Duration
}

func (e *CallTimeoutError) Error() string {
	return fmt.Sprintf("call timeout on %s after %s", e.Topic, e.Elapsed)
}
func (e *CallTimeoutError) Code() string { return "CALL_TIMEOUT" }
func (e *CallTimeoutError) Details() map[string]any {
	return map[string]any{"topic": e.Topic, "elapsed": e.Elapsed.String()}
}

// CallCancelledError is returned when the caller's context is cancelled.
type CallCancelledError struct {
	Topic string
	Cause error
}

func (e *CallCancelledError) Error() string {
	return fmt.Sprintf("call cancelled on %s: %v", e.Topic, e.Cause)
}
func (e *CallCancelledError) Unwrap() error { return e.Cause }
func (e *CallCancelledError) Code() string  { return "CALL_CANCELLED" }
func (e *CallCancelledError) Details() map[string]any {
	return map[string]any{"topic": e.Topic}
}

// CallDecodeError is returned when a reply payload cannot be decoded.
type CallDecodeError struct {
	Topic   string
	Payload json.RawMessage
	Cause   error
}

func (e *CallDecodeError) Error() string {
	return fmt.Sprintf("call decode on %s: %v", e.Topic, e.Cause)
}
func (e *CallDecodeError) Unwrap() error { return e.Cause }
func (e *CallDecodeError) Code() string  { return "CALL_DECODE_ERROR" }
func (e *CallDecodeError) Details() map[string]any {
	return map[string]any{"topic": e.Topic}
}

// BufferOverflowError is returned when a streaming call overflows with the
// BufferError policy.
type BufferOverflowError struct {
	CorrelationID string
	Topic         string
}

func (e *BufferOverflowError) Error() string {
	return fmt.Sprintf("call stream buffer overflow on %s", e.Topic)
}
func (e *BufferOverflowError) Code() string { return "CALL_BUFFER_OVERFLOW" }
func (e *BufferOverflowError) Details() map[string]any {
	return map[string]any{"topic": e.Topic, "correlationId": e.CorrelationID}
}

// HandlerFailedError is returned when a remote handler's retries are
// exhausted before a terminal reply arrives.
type HandlerFailedError struct {
	Topic   string
	Retries int
	Cause   error
}

func (e *HandlerFailedError) Error() string {
	return fmt.Sprintf("remote handler failed on %s after %d retries: %v", e.Topic, e.Retries, e.Cause)
}
func (e *HandlerFailedError) Unwrap() error { return e.Cause }
func (e *HandlerFailedError) Code() string  { return "HANDLER_FAILED" }
func (e *HandlerFailedError) Details() map[string]any {
	return map[string]any{"topic": e.Topic, "retries": e.Retries}
}
