// Ported from: packages/core/src/a2a/error.ts
package a2a

import "fmt"

// A2AError is a custom error type for A2A server operations, incorporating
// JSON-RPC error codes. It corresponds to MastraA2AError in the TS source.
type A2AError struct {
	// Code is the JSON-RPC error code (a KnownErrorCode or arbitrary int).
	Code int
	// Msg is a human-readable description of the error.
	Msg string
	// Data is optional additional data about the error.
	Data any
	// TaskID is an optional task ID providing context.
	TaskID string
}

// Error implements the error interface.
func (e *A2AError) Error() string {
	if e.TaskID != "" {
		return fmt.Sprintf("A2AError(%d): %s [task=%s]", e.Code, e.Msg, e.TaskID)
	}
	return fmt.Sprintf("A2AError(%d): %s", e.Code, e.Msg)
}

// NewA2AError creates a new A2AError with the given code, message, and optional
// data and taskId.
func NewA2AError(code int, message string, data any, taskID string) *A2AError {
	return &A2AError{
		Code:   code,
		Msg:    message,
		Data:   data,
		TaskID: taskID,
	}
}

// ToJSONRPCError formats the error into a standard JSON-RPC error object structure.
func (e *A2AError) ToJSONRPCError() JSONRPCError {
	rpcErr := JSONRPCError{
		Code:    e.Code,
		Message: e.Msg,
	}
	if e.Data != nil {
		rpcErr.Data = e.Data
	}
	return rpcErr
}

// ---------------------------------------------------------------------------
// Static factory functions for common errors
// ---------------------------------------------------------------------------

// ParseError creates an A2AError for JSON Parse Error (-32700).
func ParseError(message string, data any) *A2AError {
	return NewA2AError(int(ErrorCodeParseError), message, data, "")
}

// InvalidRequest creates an A2AError for Invalid Request (-32600).
func InvalidRequest(message string, data any) *A2AError {
	return NewA2AError(int(ErrorCodeInvalidRequest), message, data, "")
}

// MethodNotFound creates an A2AError for Method Not Found (-32601).
func MethodNotFound(method string) *A2AError {
	return NewA2AError(int(ErrorCodeMethodNotFound), fmt.Sprintf("Method not found: %s", method), nil, "")
}

// InvalidParams creates an A2AError for Invalid Params (-32602).
func InvalidParams(message string, data any) *A2AError {
	return NewA2AError(int(ErrorCodeInvalidParams), message, data, "")
}

// InternalError creates an A2AError for Internal Error (-32603).
func InternalError(message string, data any) *A2AError {
	return NewA2AError(int(ErrorCodeInternalError), message, data, "")
}

// TaskNotFound creates an A2AError for Task Not Found (-32001).
func TaskNotFound(taskID string) *A2AError {
	return NewA2AError(int(ErrorCodeTaskNotFound), fmt.Sprintf("Task not found: %s", taskID), nil, taskID)
}

// TaskNotCancelable creates an A2AError for Task Not Cancelable (-32002).
func TaskNotCancelable(taskID string) *A2AError {
	return NewA2AError(int(ErrorCodeTaskNotCancelable), fmt.Sprintf("Task not cancelable: %s", taskID), nil, taskID)
}

// PushNotificationNotSupported creates an A2AError for Push Notification
// Not Supported (-32003).
func PushNotificationNotSupported() *A2AError {
	return NewA2AError(int(ErrorCodePushNotificationNotSupported), "Push Notification is not supported", nil, "")
}

// UnsupportedOperation creates an A2AError for Unsupported Operation (-32004).
func UnsupportedOperation(operation string) *A2AError {
	return NewA2AError(int(ErrorCodeUnsupportedOperation), fmt.Sprintf("Unsupported operation: %s", operation), nil, "")
}
