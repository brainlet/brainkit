// Ported from: packages/openai/src/openai-error.ts
package openai

import (
	"encoding/json"
	"net/http"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// OpenAIErrorDataError contains the error detail fields within the OpenAI error response.
type OpenAIErrorDataError struct {
	Message string      `json:"message"`
	Type    *string     `json:"type,omitempty"`
	Param   interface{} `json:"param,omitempty"`
	Code    interface{} `json:"code,omitempty"` // Can be string or number
}

// OpenAIErrorData represents the OpenAI error response structure.
type OpenAIErrorData struct {
	Error OpenAIErrorDataError `json:"error"`
}

// openaiErrorDataSchema is the schema for parsing OpenAI error responses.
var openaiErrorDataSchema = &providerutils.Schema[OpenAIErrorData]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[OpenAIErrorData], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[OpenAIErrorData]{
				Success: false,
				Error:   err,
			}, nil
		}
		var errData OpenAIErrorData
		if err := json.Unmarshal(data, &errData); err != nil {
			return &providerutils.ValidationResult[OpenAIErrorData]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[OpenAIErrorData]{
			Success: true,
			Value:   errData,
		}, nil
	},
}

// openaiFailedResponseHandler is the failed response handler for OpenAI API errors.
var openaiFailedResponseHandler providerutils.ResponseHandler[error]

func init() {
	typedHandler := providerutils.CreateJsonErrorResponseHandler(
		openaiErrorDataSchema,
		func(data OpenAIErrorData) string {
			return data.Error.Message
		},
		func(resp *http.Response, err *OpenAIErrorData) bool {
			return false
		},
	)
	openaiFailedResponseHandler = func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
		res, err := typedHandler(opts)
		if err != nil {
			return nil, err
		}
		if res == nil {
			return nil, nil
		}
		return &providerutils.ResponseHandlerResult[error]{
			Value:           res.Value,
			RawValue:        res.RawValue,
			ResponseHeaders: res.ResponseHeaders,
		}, nil
	}
}
