// Ported from: packages/openai-compatible/src/openai-compatible-error.ts
package openaicompatible

import (
	"encoding/json"
	"net/http"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// ErrorDataError contains the error detail fields within the OpenAI-compatible error response.
type ErrorDataError struct {
	Message string      `json:"message"`
	Type    *string     `json:"type,omitempty"`
	Param   interface{} `json:"param,omitempty"`
	Code    interface{} `json:"code,omitempty"` // Can be string or number
}

// ErrorData represents the OpenAI-compatible error response structure.
type ErrorData struct {
	Error ErrorDataError `json:"error"`
}

// ProviderErrorStructure defines the structure for parsing and handling
// provider-specific error responses.
type ProviderErrorStructure[T any] struct {
	// ErrorSchema is the schema used to validate and parse the error response body.
	ErrorSchema *providerutils.Schema[T]

	// ErrorToMessage extracts a human-readable error message from the parsed error data.
	ErrorToMessage func(T) string

	// IsRetryable optionally determines whether the error is retryable based on
	// the HTTP response and parsed error data.
	IsRetryable func(resp *http.Response, err *T) bool
}

// DefaultErrorStructure is the default error structure for OpenAI-compatible providers.
var DefaultErrorStructure = ProviderErrorStructure[ErrorData]{
	ErrorSchema: &providerutils.Schema[ErrorData]{
		Validate: func(value interface{}) (*providerutils.ValidationResult[ErrorData], error) {
			data, err := json.Marshal(value)
			if err != nil {
				return &providerutils.ValidationResult[ErrorData]{
					Success: false,
					Error:   err,
				}, nil
			}
			var errData ErrorData
			if err := json.Unmarshal(data, &errData); err != nil {
				return &providerutils.ValidationResult[ErrorData]{
					Success: false,
					Error:   err,
				}, nil
			}
			return &providerutils.ValidationResult[ErrorData]{
				Success: true,
				Value:   errData,
			}, nil
		},
	},
	ErrorToMessage: func(data ErrorData) string {
		return data.Error.Message
	},
}
