// Ported from: packages/xai/src/xai-error.ts
package xai

import (
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// XaiErrorData represents the structure of an xAI API error response.
type XaiErrorData struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type,omitempty"`
		Param   any    `json:"param,omitempty"`
		Code    any    `json:"code,omitempty"` // can be string or number
	} `json:"error"`
}

// xaiErrorDataSchema is the schema for validating xAI error responses.
var xaiErrorDataSchema = &providerutils.Schema[XaiErrorData]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[XaiErrorData], error) {
		m, ok := value.(map[string]interface{})
		if !ok {
			return &providerutils.ValidationResult[XaiErrorData]{Success: false}, nil
		}
		errObj, ok := m["error"].(map[string]interface{})
		if !ok {
			return &providerutils.ValidationResult[XaiErrorData]{Success: false}, nil
		}
		msg, _ := errObj["message"].(string)
		typ, _ := errObj["type"].(string)
		param := errObj["param"]
		code := errObj["code"]

		var result XaiErrorData
		result.Error.Message = msg
		result.Error.Type = typ
		result.Error.Param = param
		result.Error.Code = code

		return &providerutils.ValidationResult[XaiErrorData]{
			Success: true,
			Value:   result,
		}, nil
	},
}

// xaiTypedFailedResponseHandler is the typed error response handler for xAI API calls.
var xaiTypedFailedResponseHandler = providerutils.CreateJsonErrorResponseHandler(
	xaiErrorDataSchema,
	func(data XaiErrorData) string {
		return data.Error.Message
	},
	nil,
)

// statusCodeErrorResponseHandler wraps CreateStatusCodeErrorResponseHandler to satisfy ResponseHandler[error].
var statusCodeErrorResponseHandler providerutils.ResponseHandler[error] = func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
	handler := providerutils.CreateStatusCodeErrorResponseHandler()
	res, err := handler(opts)
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

// xaiFailedResponseHandler is the standard error response handler for xAI API calls,
// wrapped to satisfy ResponseHandler[error].
var xaiFailedResponseHandler providerutils.ResponseHandler[error] = func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
	res, err := xaiTypedFailedResponseHandler(opts)
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
