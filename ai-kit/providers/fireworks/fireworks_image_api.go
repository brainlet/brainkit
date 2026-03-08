// Ported from: packages/fireworks/src/fireworks-image-api.ts
package fireworks

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// asyncSubmitResponse is the response from an async image generation submit request.
type asyncSubmitResponse struct {
	RequestID string `json:"request_id"`
}

// asyncPollResponse is the response from polling an async image generation request.
type asyncPollResponse struct {
	ID     string                    `json:"id"`
	Status string                    `json:"status"`
	Result *asyncPollResponseResult  `json:"result"`
}

// asyncPollResponseResult holds the result data from a completed async poll.
type asyncPollResponseResult struct {
	Sample *string `json:"sample,omitempty"`
}

// asyncSubmitResponseSchema is the schema for validating async submit responses.
var asyncSubmitResponseSchema = &providerutils.Schema[asyncSubmitResponse]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[asyncSubmitResponse], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[asyncSubmitResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		var resp asyncSubmitResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return &providerutils.ValidationResult[asyncSubmitResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[asyncSubmitResponse]{
			Success: true,
			Value:   resp,
		}, nil
	},
}

// asyncPollResponseSchema is the schema for validating async poll responses.
var asyncPollResponseSchema = &providerutils.Schema[asyncPollResponse]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[asyncPollResponse], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[asyncPollResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		var resp asyncPollResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return &providerutils.ValidationResult[asyncPollResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[asyncPollResponse]{
			Success: true,
			Value:   resp,
		}, nil
	},
}
