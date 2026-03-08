// Ported from: packages/core/src/error/utils.ts
package mastraerror

import (
	"encoding/json"
	"fmt"
	"reflect"
)

const defaultMaxDepth = 5

// SerializedError represents a serialized error structure for JSON output.
type SerializedError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	Stack   string `json:"stack,omitempty"`
	Cause   any    `json:"cause,omitempty"`
	// Extra holds any additional custom properties from the original error.
	Extra map[string]any `json:"-"`
}

// MarshalJSON implements json.Marshaler for SerializedError.
// It merges the standard fields with any extra custom properties.
func (s SerializedError) MarshalJSON() ([]byte, error) {
	// Build a combined map for serialization.
	m := make(map[string]any)

	// Add extra properties first (so standard fields take precedence).
	for k, v := range s.Extra {
		m[k] = v
	}

	m["name"] = s.Name
	m["message"] = s.Message
	if s.Stack != "" {
		m["stack"] = s.Stack
	}
	if s.Cause != nil {
		m["cause"] = s.Cause
	}

	return json.Marshal(m)
}

// SerializableError is an error with JSON serialization support.
// In TypeScript this was: Error & { toJSON: () => SerializedError }
// In Go, we make it a struct that implements error and json.Marshaler.
type SerializableError struct {
	message        string
	stack          string
	cause          any // can be *SerializableError, error, or any other value
	name           string
	extra          map[string]any
	serializeStack bool
}

// Error implements the error interface.
func (e *SerializableError) Error() string {
	return e.message
}

// Message returns the error message.
func (e *SerializableError) Message() string {
	return e.message
}

// Name returns the error name.
func (e *SerializableError) Name() string {
	if e.name == "" {
		return "Error"
	}
	return e.name
}

// Stack returns the stack trace string.
func (e *SerializableError) Stack() string {
	return e.stack
}

// CauseValue returns the raw cause value.
func (e *SerializableError) CauseValue() any {
	return e.cause
}

// Unwrap returns the cause as an error for errors.Is/errors.As support.
func (e *SerializableError) Unwrap() error {
	if e.cause == nil {
		return nil
	}
	if err, ok := e.cause.(error); ok {
		return err
	}
	return nil
}

// Extra returns any custom properties attached to this error.
func (e *SerializableError) Extra() map[string]any {
	return e.extra
}

// ToJSON returns the serialized representation of the error.
func (e *SerializableError) ToJSON() SerializedError {
	result := SerializedError{
		Name:    e.Name(),
		Message: e.message,
		Extra:   make(map[string]any),
	}

	// Only include stack in JSON if serializeStack is true.
	if e.serializeStack && e.stack != "" {
		result.Stack = e.stack
	}

	// Serialize cause.
	if e.cause != nil {
		if causeErr, ok := e.cause.(*SerializableError); ok {
			causeJSON := causeErr.ToJSON()
			result.Cause = causeJSON
		} else {
			result.Cause = e.cause
		}
	}

	// Include custom properties.
	for k, v := range e.extra {
		result.Extra[k] = v
	}

	return result
}

// MarshalJSON implements json.Marshaler.
func (e *SerializableError) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.ToJSON())
}

// GetErrorOptions configures how GetErrorFromUnknown behaves.
type GetErrorOptions struct {
	// FallbackMessage is the error message to use if the unknown value cannot be parsed.
	FallbackMessage string
	// MaxDepth is the maximum depth to parse cause chains. Default is 5.
	MaxDepth *int
	// SupportSerialization controls whether to add serialization support. Default is true.
	SupportSerialization *bool
	// SerializeStack controls whether to include the stack in JSON serialization.
	// The stack is always preserved on the error instance for debugging.
	// This option only controls whether it appears in ToJSON() output. Default is true.
	SerializeStack *bool
}

func (o *GetErrorOptions) fallbackMessage() string {
	if o != nil && o.FallbackMessage != "" {
		return o.FallbackMessage
	}
	return "Unknown error"
}

func (o *GetErrorOptions) maxDepth() int {
	if o != nil && o.MaxDepth != nil {
		return *o.MaxDepth
	}
	return defaultMaxDepth
}

func (o *GetErrorOptions) supportSerialization() bool {
	if o != nil && o.SupportSerialization != nil {
		return *o.SupportSerialization
	}
	return true
}

func (o *GetErrorOptions) serializeStack() bool {
	if o != nil && o.SerializeStack != nil {
		return *o.SerializeStack
	}
	return true
}

// GetErrorFromUnknown safely converts an unknown value to a SerializableError.
// It normalizes any value into a proper error with JSON serialization support.
func GetErrorFromUnknown(unknown any, opts ...*GetErrorOptions) *SerializableError {
	var options *GetErrorOptions
	if len(opts) > 0 {
		options = opts[0]
	}

	fallbackMsg := options.fallbackMessage()
	maxDepth := options.maxDepth()
	serializeStack := options.serializeStack()

	// If it's already a *SerializableError, update serialization settings and return.
	if sErr, ok := unknown.(*SerializableError); ok {
		sErr.serializeStack = serializeStack
		// Recursively process cause chain with depth protection.
		if sErr.cause != nil {
			if maxDepth > 0 {
				childOpts := &GetErrorOptions{
					FallbackMessage:      fallbackMsg,
					MaxDepth:             intPtr(maxDepth - 1),
					SupportSerialization: boolPtr(true),
					SerializeStack:       &serializeStack,
				}
				sErr.cause = GetErrorFromUnknown(sErr.cause, childOpts)
			} else {
				// At max depth: stop the cause chain.
				sErr.cause = nil
			}
		}
		return sErr
	}

	// If it's a standard Go error (but not a *SerializableError).
	if err, ok := unknown.(error); ok {
		result := &SerializableError{
			message:        err.Error(),
			name:           "Error",
			serializeStack: serializeStack,
			extra:          make(map[string]any),
		}

		// Try to extract cause via Unwrap.
		if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
			if cause := unwrapper.Unwrap(); cause != nil {
				if maxDepth > 0 {
					childOpts := &GetErrorOptions{
						FallbackMessage:      fallbackMsg,
						MaxDepth:             intPtr(maxDepth - 1),
						SupportSerialization: boolPtr(true),
						SerializeStack:       &serializeStack,
					}
					result.cause = GetErrorFromUnknown(cause, childOpts)
				}
			}
		}

		// Copy extra fields if the error is a MastraBaseError.
		if mErr, ok := err.(*MastraBaseError); ok {
			result.name = "MastraError"
			_ = mErr // extra fields already captured via message
		}

		return result
	}

	// If it's a map (object-like).
	if m, ok := unknown.(map[string]any); ok {
		msg := ""
		if msgVal, exists := m["message"]; exists {
			if s, ok := msgVal.(string); ok {
				msg = s
			}
		}
		if msg == "" {
			msg = safeParseErrorObject(unknown)
		}

		result := &SerializableError{
			message:        msg,
			name:           "Error",
			serializeStack: serializeStack,
			extra:          make(map[string]any),
		}

		// Preserve cause from the map.
		if causeVal, exists := m["cause"]; exists && causeVal != nil {
			if causeErr, ok := causeVal.(error); ok {
				if maxDepth > 0 {
					childOpts := &GetErrorOptions{
						FallbackMessage:      fallbackMsg,
						MaxDepth:             intPtr(maxDepth - 1),
						SupportSerialization: boolPtr(true),
						SerializeStack:       &serializeStack,
					}
					result.cause = GetErrorFromUnknown(causeErr, childOpts)
				} else {
					result.cause = causeVal
				}
			} else if maxDepth > 0 {
				childOpts := &GetErrorOptions{
					FallbackMessage:      fallbackMsg,
					MaxDepth:             intPtr(maxDepth - 1),
					SupportSerialization: boolPtr(true),
					SerializeStack:       &serializeStack,
				}
				result.cause = GetErrorFromUnknown(causeVal, childOpts)
			}
		}

		// Copy extra properties from the map (excluding "message", "cause", "stack").
		if stackVal, exists := m["stack"]; exists {
			if s, ok := stackVal.(string); ok {
				result.stack = s
			}
		}
		for k, v := range m {
			if k == "message" || k == "cause" || k == "stack" {
				continue
			}
			result.extra[k] = v
		}

		return result
	}

	// If it's a string.
	if s, ok := unknown.(string); ok {
		return &SerializableError{
			message:        s,
			name:           "Error",
			serializeStack: serializeStack,
			extra:          make(map[string]any),
		}
	}

	// If it's a slice or array (in TS, arrays are objects and get safeParseErrorObject'd).
	if unknown != nil {
		rv := reflect.ValueOf(unknown)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			msg := safeParseErrorObject(unknown)
			return &SerializableError{
				message:        msg,
				name:           "Error",
				serializeStack: serializeStack,
				extra:          make(map[string]any),
			}
		}
	}

	// For all other types (nil, numbers, booleans, etc.), use the fallback message.
	return &SerializableError{
		message:        fallbackMsg,
		name:           "Error",
		serializeStack: serializeStack,
		extra:          make(map[string]any),
	}
}

// safeParseErrorObject safely converts an object to a string representation.
// Uses json.Marshal first, but falls back to fmt.Sprintf if:
// - json.Marshal fails (e.g., circular references)
// - json.Marshal returns "{}" (e.g., Error objects with no enumerable properties)
func safeParseErrorObject(obj any) string {
	if obj == nil {
		return fmt.Sprintf("%v", obj)
	}

	b, err := json.Marshal(obj)
	if err != nil {
		return fmt.Sprintf("%v", obj)
	}

	s := string(b)
	if s == "{}" {
		return fmt.Sprintf("%v", s)
	}

	return s
}

// Serialize converts any error value into a SerializedError.
// This is the Go equivalent of the TypeScript `serialize(error: unknown): SerializedError`.
func Serialize(err any) SerializedError {
	return GetErrorFromUnknown(err).ToJSON()
}

// intPtr returns a pointer to an int value.
func intPtr(i int) *int {
	return &i
}
