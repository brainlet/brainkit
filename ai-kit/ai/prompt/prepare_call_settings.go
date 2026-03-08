// Ported from: packages/ai/src/prompt/prepare-call-settings.ts
package prompt

import (
	"math"

	aierror "github.com/brainlet/brainkit/ai-kit/ai/error"
)

// PrepareCallSettingsInput is a subset of CallSettings that can be validated.
type PrepareCallSettingsInput struct {
	MaxOutputTokens  *int
	Temperature      *float64
	TopP             *float64
	TopK             *int
	PresencePenalty  *float64
	FrequencyPenalty *float64
	Seed             *int
	StopSequences    []string
}

// PrepareCallSettingsOutput is the validated result.
type PrepareCallSettingsOutput struct {
	MaxOutputTokens  *int
	Temperature      *float64
	TopP             *float64
	TopK             *int
	PresencePenalty  *float64
	FrequencyPenalty *float64
	Seed             *int
	StopSequences    []string
}

// PrepareCallSettings validates call settings and returns a new object with limited values.
func PrepareCallSettings(input PrepareCallSettingsInput) (*PrepareCallSettingsOutput, error) {
	if input.MaxOutputTokens != nil {
		v := *input.MaxOutputTokens
		if v != int(math.Floor(float64(v))) {
			return nil, aierror.NewInvalidArgumentError(
				"maxOutputTokens", v, "maxOutputTokens must be an integer",
			)
		}
		if v < 1 {
			return nil, aierror.NewInvalidArgumentError(
				"maxOutputTokens", v, "maxOutputTokens must be >= 1",
			)
		}
	}

	// In Go, the types are enforced at compile time for temperature, topP, topK,
	// presencePenalty, frequencyPenalty. The TypeScript version checks typeof === 'number'
	// because JS allows passing strings at runtime. We keep the same validation pattern
	// for API parity, even though Go's type system prevents most of these.

	if input.Seed != nil {
		v := *input.Seed
		if v != int(math.Floor(float64(v))) {
			return nil, aierror.NewInvalidArgumentError(
				"seed", v, "seed must be an integer",
			)
		}
	}

	return &PrepareCallSettingsOutput{
		MaxOutputTokens:  input.MaxOutputTokens,
		Temperature:      input.Temperature,
		TopP:             input.TopP,
		TopK:             input.TopK,
		PresencePenalty:  input.PresencePenalty,
		FrequencyPenalty: input.FrequencyPenalty,
		StopSequences:    input.StopSequences,
		Seed:             input.Seed,
	}, nil
}
