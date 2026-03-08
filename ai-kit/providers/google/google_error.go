// Ported from: packages/google/src/google-error.ts
package google

import (
	"net/http"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// GoogleErrorData represents the error structure returned by the Google
// Generative AI API.
type GoogleErrorData struct {
	Error GoogleErrorDetail `json:"error"`
}

// GoogleErrorDetail contains the details of a Google API error.
type GoogleErrorDetail struct {
	Code    *int   `json:"code"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// googleErrorSchema is the schema for parsing Google error responses.
var googleErrorSchema = &providerutils.Schema[GoogleErrorData]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[GoogleErrorData], error) {
		m, ok := value.(map[string]interface{})
		if !ok {
			return &providerutils.ValidationResult[GoogleErrorData]{Success: false}, nil
		}
		errObj, ok := m["error"].(map[string]interface{})
		if !ok {
			return &providerutils.ValidationResult[GoogleErrorData]{Success: false}, nil
		}
		result := GoogleErrorData{
			Error: GoogleErrorDetail{
				Message: stringFromMap(errObj, "message"),
				Status:  stringFromMap(errObj, "status"),
			},
		}
		if code, ok := errObj["code"].(float64); ok {
			c := int(code)
			result.Error.Code = &c
		}
		return &providerutils.ValidationResult[GoogleErrorData]{
			Success: true,
			Value:   result,
		}, nil
	},
}

// googleTypedFailedResponseHandler is the typed error response handler.
var googleTypedFailedResponseHandler = providerutils.CreateJsonErrorResponseHandler(
	googleErrorSchema,
	func(data GoogleErrorData) string {
		return data.Error.Message
	},
	func(resp *http.Response, errorVal *GoogleErrorData) bool {
		return false
	},
)

// GoogleFailedResponseHandler is the response handler for failed Google API responses,
// wrapped to satisfy ResponseHandler[error].
var GoogleFailedResponseHandler providerutils.ResponseHandler[error] = func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
	res, err := googleTypedFailedResponseHandler(opts)
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

func stringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
