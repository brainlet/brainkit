// Ported from: packages/provider/src/language-model-middleware/v3/language-model-v3-middleware.ts
package middleware

import (
	"regexp"

	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// LanguageModelMiddleware defines middleware that can modify the behavior
// of LanguageModel operations.
type LanguageModelMiddleware struct {
	// OverrideProvider overrides the provider name if desired.
	OverrideProvider func(model lm.LanguageModel) string

	// OverrideModelID overrides the model ID if desired.
	OverrideModelID func(model lm.LanguageModel) string

	// OverrideSupportedUrls overrides the supported URLs if desired.
	OverrideSupportedUrls func(model lm.LanguageModel) (map[string][]*regexp.Regexp, error)

	// TransformParams transforms the parameters before they are passed to the language model.
	TransformParams func(opts TransformParamsOptions) (lm.CallOptions, error)

	// WrapGenerate wraps the generate operation of the language model.
	WrapGenerate func(opts WrapGenerateOptions) (lm.GenerateResult, error)

	// WrapStream wraps the stream operation of the language model.
	WrapStream func(opts WrapStreamOptions) (lm.StreamResult, error)
}

// SpecificationVersion returns "v3".
func (m LanguageModelMiddleware) SpecificationVersion() string { return "v3" }

// TransformParamsOptions are the options for TransformParams.
type TransformParamsOptions struct {
	// Type is the type of operation ("generate" or "stream").
	Type string

	// Params are the original parameters for the language model call.
	Params lm.CallOptions

	// Model is the language model instance.
	Model lm.LanguageModel
}

// WrapGenerateOptions are the options for WrapGenerate.
type WrapGenerateOptions struct {
	// DoGenerate is the original generate function.
	DoGenerate func() (lm.GenerateResult, error)

	// DoStream is the original stream function.
	DoStream func() (lm.StreamResult, error)

	// Params are the parameters for the generate call.
	// If TransformParams middleware is used, this will be the transformed parameters.
	Params lm.CallOptions

	// Model is the language model instance.
	Model lm.LanguageModel
}

// WrapStreamOptions are the options for WrapStream.
type WrapStreamOptions struct {
	// DoGenerate is the original generate function.
	DoGenerate func() (lm.GenerateResult, error)

	// DoStream is the original stream function.
	DoStream func() (lm.StreamResult, error)

	// Params are the parameters for the stream call.
	// If TransformParams middleware is used, this will be the transformed parameters.
	Params lm.CallOptions

	// Model is the language model instance.
	Model lm.LanguageModel
}
