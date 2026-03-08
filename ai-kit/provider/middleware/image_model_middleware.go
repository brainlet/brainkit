// Ported from: packages/provider/src/image-model-middleware/v3/image-model-v3-middleware.ts
package middleware

import im "github.com/brainlet/brainkit/ai-kit/provider/imagemodel"

// ImageModelMiddleware defines middleware that can modify the behavior
// of ImageModel operations.
type ImageModelMiddleware struct {
	// OverrideProvider overrides the provider name if desired.
	OverrideProvider func(model im.ImageModel) string

	// OverrideModelID overrides the model ID if desired.
	OverrideModelID func(model im.ImageModel) string

	// OverrideMaxImagesPerCall overrides the max images per call if desired.
	OverrideMaxImagesPerCall func(model im.ImageModel) (*int, error)

	// TransformParams transforms the parameters before they are passed to the image model.
	TransformParams func(opts ImageTransformParamsOptions) (im.CallOptions, error)

	// WrapGenerate wraps the generate operation of the image model.
	WrapGenerate func(opts WrapImageGenerateOptions) (im.GenerateResult, error)
}

// SpecificationVersion returns "v3".
func (m ImageModelMiddleware) SpecificationVersion() string { return "v3" }

// ImageTransformParamsOptions are the options for TransformParams.
type ImageTransformParamsOptions struct {
	Params im.CallOptions
	Model  im.ImageModel
}

// WrapImageGenerateOptions are the options for WrapGenerate.
type WrapImageGenerateOptions struct {
	// DoGenerate is the original generate function.
	DoGenerate func() (im.GenerateResult, error)

	// Params are the parameters for the generate call.
	Params im.CallOptions

	// Model is the image model instance.
	Model im.ImageModel
}
