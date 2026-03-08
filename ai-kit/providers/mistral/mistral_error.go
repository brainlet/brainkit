// Ported from: packages/mistral/src/mistral-error.ts
package mistral

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// MistralErrorData represents the error data returned by the Mistral API.
type MistralErrorData struct {
	Object  string  `json:"object"`
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   *string `json:"param"`
	Code    *string `json:"code"`
}

// mistralErrorDataSchema is the schema for validating MistralErrorData.
var mistralErrorDataSchema = &providerutils.Schema[MistralErrorData]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[MistralErrorData], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[MistralErrorData]{
				Success: false,
				Error:   err,
			}, nil
		}
		var errData MistralErrorData
		if err := json.Unmarshal(data, &errData); err != nil {
			return &providerutils.ValidationResult[MistralErrorData]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[MistralErrorData]{
			Success: true,
			Value:   errData,
		}, nil
	},
}

// mistralFailedResponseHandler handles failed API responses from Mistral.
var mistralFailedResponseHandler providerutils.ResponseHandler[error]

func init() {
	typedHandler := providerutils.CreateJsonErrorResponseHandler(
		mistralErrorDataSchema,
		func(data MistralErrorData) string { return data.Message },
		nil, // no custom retryable logic
	)

	mistralFailedResponseHandler = func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
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
