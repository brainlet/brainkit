// Ported from: packages/ai/src/types/language-model-middleware.ts
package aitypes

// LanguageModelMiddleware is middleware for language models.
// Accepts both V3 and V4 middleware types for backward compatibility.
//
// Uses LanguageModelV4Middleware as the base but relaxes specificationVersion
// to accept any string (including "v3") and makes it optional.
//
// In TypeScript this references complex function types from LanguageModelV4Middleware.
// In Go we represent the middleware as a struct with optional function fields.
type LanguageModelMiddleware struct {
	// SpecificationVersion is an optional version string (e.g. "v3", "v4").
	SpecificationVersion string `json:"specificationVersion,omitempty"`

	// OverrideProvider overrides the provider name if desired.
	// TODO: Full typing depends on LanguageModelV4 interface from provider package.
	OverrideProvider func(model any) string `json:"-"`

	// OverrideModelId overrides the model ID if desired.
	OverrideModelId func(model any) string `json:"-"`

	// OverrideSupportedUrls overrides the supported URLs if desired.
	OverrideSupportedUrls func(model any) map[string][]string `json:"-"`

	// TransformParams transforms the parameters before they are passed to the language model.
	// The opType parameter is "generate" or "stream".
	TransformParams func(opType string, params any, model any) (any, error) `json:"-"`

	// WrapGenerate wraps the generate operation of the language model.
	WrapGenerate func(doGenerate func() (any, error), doStream func() (any, error), params any, model any) (any, error) `json:"-"`

	// WrapStream wraps the stream operation of the language model.
	WrapStream func(doGenerate func() (any, error), doStream func() (any, error), params any, model any) (any, error) `json:"-"`
}
