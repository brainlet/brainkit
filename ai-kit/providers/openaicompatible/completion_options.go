// Ported from: packages/openai-compatible/src/completion/openai-compatible-completion-options.ts
package openaicompatible

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// CompletionModelID is the identifier for an OpenAI-compatible completion model.
type CompletionModelID = string

// CompletionOptions contains provider-specific options for OpenAI-compatible
// completion models.
type CompletionOptions struct {
	// Echo controls whether to echo back the prompt in addition to the completion.
	Echo *bool `json:"echo,omitempty"`

	// LogitBias modifies the likelihood of specified tokens appearing in the
	// completion. Maps token IDs (as strings) to bias values from -100 to 100.
	LogitBias map[string]float64 `json:"logitBias,omitempty"`

	// Suffix is appended after the completion of inserted text.
	Suffix *string `json:"suffix,omitempty"`

	// User is a unique identifier representing the end-user, which can help
	// providers monitor and detect abuse.
	User *string `json:"user,omitempty"`
}

// CompletionOptionsSchema is the providerutils.Schema used to validate and parse
// CompletionOptions from provider options maps.
var CompletionOptionsSchema = &providerutils.Schema[CompletionOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[CompletionOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[CompletionOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts CompletionOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[CompletionOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[CompletionOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}
