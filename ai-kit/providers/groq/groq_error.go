// Ported from: packages/groq/src/groq-error.ts
package groq

import (
	"encoding/json"
	"net/http"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// GroqErrorData represents the Groq API error response structure.
type GroqErrorData struct {
	Error GroqErrorDetail `json:"error"`
}

// GroqErrorDetail contains the error detail fields.
type GroqErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// groqErrorDataSchema is the schema for parsing Groq error responses.
var groqErrorDataSchema = &providerutils.Schema[GroqErrorData]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[GroqErrorData], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[GroqErrorData]{
				Success: false,
				Error:   err,
			}, nil
		}
		var errData GroqErrorData
		if err := json.Unmarshal(data, &errData); err != nil {
			return &providerutils.ValidationResult[GroqErrorData]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[GroqErrorData]{
			Success: true,
			Value:   errData,
		}, nil
	},
}

// groqFailedResponseHandler is the response handler for failed Groq API responses.
var groqFailedResponseHandler providerutils.ResponseHandler[error]

func init() {
	typedHandler := providerutils.CreateJsonErrorResponseHandler(
		groqErrorDataSchema,
		func(data GroqErrorData) string {
			return data.Error.Message
		},
		func(resp *http.Response, errorVal *GroqErrorData) bool {
			return false
		},
	)
	groqFailedResponseHandler = func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
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
