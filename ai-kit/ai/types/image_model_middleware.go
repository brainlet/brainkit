// Ported from: packages/ai/src/types/image-model-middleware.ts
package aitypes

// ImageModelMiddleware is middleware for image models.
// Accepts both V3 and V4 middleware types for backward compatibility.
//
// Uses ImageModelV4Middleware as the base but relaxes specificationVersion
// to accept any string (including "v3") and makes it optional.
type ImageModelMiddleware struct {
	// SpecificationVersion is an optional version string (e.g. "v3", "v4").
	SpecificationVersion string `json:"specificationVersion,omitempty"`

	// OverrideProvider overrides the provider name if desired.
	// TODO: Full typing depends on ImageModelV4 interface from provider package.
	OverrideProvider func(model any) string `json:"-"`

	// OverrideModelId overrides the model ID if desired.
	OverrideModelId func(model any) string `json:"-"`

	// OverrideMaxImagesPerCall overrides the limit of how many images
	// can be generated in a single API call if desired.
	OverrideMaxImagesPerCall func(model any) *int `json:"-"`

	// TransformParams transforms the parameters before they are passed to the image model.
	TransformParams func(params any, model any) (any, error) `json:"-"`

	// WrapGenerate wraps the generate operation of the image model.
	WrapGenerate func(doGenerate func() (any, error), params any, model any) (any, error) `json:"-"`
}
