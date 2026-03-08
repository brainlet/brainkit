// Ported from: packages/huggingface/src/huggingface-error.ts
package huggingface

import (
	"net/http"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// ErrorData represents the error data structure from HuggingFace API responses.
type ErrorData struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail is the inner error detail object.
type ErrorDetail struct {
	Message string  `json:"message"`
	Type    *string `json:"type,omitempty"`
	Code    *string `json:"code,omitempty"`
}

// huggingfaceErrorSchema is the schema for parsing HuggingFace error responses.
var huggingfaceErrorSchema = &providerutils.Schema[ErrorData]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[ErrorData], error) {
		m, ok := value.(map[string]interface{})
		if !ok {
			return &providerutils.ValidationResult[ErrorData]{Success: false}, nil
		}

		errObj, ok := m["error"]
		if !ok {
			return &providerutils.ValidationResult[ErrorData]{Success: false}, nil
		}

		errMap, ok := errObj.(map[string]interface{})
		if !ok {
			return &providerutils.ValidationResult[ErrorData]{Success: false}, nil
		}

		msg, _ := errMap["message"].(string)
		result := ErrorData{
			Error: ErrorDetail{
				Message: msg,
			},
		}

		if t, ok := errMap["type"].(string); ok {
			result.Error.Type = &t
		}
		if c, ok := errMap["code"].(string); ok {
			result.Error.Code = &c
		}

		return &providerutils.ValidationResult[ErrorData]{
			Success: true,
			Value:   result,
		}, nil
	},
}

// FailedResponseHandler handles failed responses from the HuggingFace API.
var FailedResponseHandler = providerutils.CreateJsonErrorResponseHandler(
	huggingfaceErrorSchema,
	func(data ErrorData) string {
		return data.Error.Message
	},
	func(resp *http.Response, errorVal *ErrorData) bool {
		return false
	},
)
