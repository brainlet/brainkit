// Ported from: packages/ai/src/middleware/wrap-image-model.ts
package middleware

import (
	im "github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

// WrapImageModel wraps an ImageModel instance with middleware functionality.
// When multiple middlewares are provided, the first middleware will transform the
// input first, and the last middleware will be wrapped directly around the model.
func WrapImageModel(opts WrapImageModelOptions) im.ImageModel {
	model := opts.Model
	middlewares := make([]mw.ImageModelMiddleware, len(opts.Middleware))
	copy(middlewares, opts.Middleware)

	// Reverse so that the first middleware is outermost.
	for i, j := 0, len(middlewares)-1; i < j; i, j = i+1, j-1 {
		middlewares[i], middlewares[j] = middlewares[j], middlewares[i]
	}

	for _, m := range middlewares {
		model = doWrapImageModel(model, m, opts.ModelID, opts.ProviderID)
	}
	return model
}

// WrapImageModelOptions holds the options for WrapImageModel.
type WrapImageModelOptions struct {
	Model      im.ImageModel
	Middleware []mw.ImageModelMiddleware
	ModelID    *string
	ProviderID *string
}

// wrappedImageModel implements im.ImageModel.
type wrappedImageModel struct {
	provider         string
	modelID          string
	maxImagesPerCall func() (*int, error)
	doGenerate       func(options im.CallOptions) (im.GenerateResult, error)
}

func (w *wrappedImageModel) SpecificationVersion() string { return "v3" }
func (w *wrappedImageModel) Provider() string             { return w.provider }
func (w *wrappedImageModel) ModelID() string              { return w.modelID }
func (w *wrappedImageModel) MaxImagesPerCall() (*int, error) {
	return w.maxImagesPerCall()
}
func (w *wrappedImageModel) DoGenerate(options im.CallOptions) (im.GenerateResult, error) {
	return w.doGenerate(options)
}

func doWrapImageModel(
	model im.ImageModel,
	middleware mw.ImageModelMiddleware,
	modelID *string,
	providerID *string,
) im.ImageModel {
	transformParams := middleware.TransformParams
	wrapGenerate := middleware.WrapGenerate
	overrideProvider := middleware.OverrideProvider
	overrideModelID := middleware.OverrideModelID
	overrideMaxImagesPerCall := middleware.OverrideMaxImagesPerCall

	doTransform := func(params im.CallOptions) (im.CallOptions, error) {
		if transformParams != nil {
			return transformParams(mw.ImageTransformParamsOptions{
				Params: params,
				Model:  model,
			})
		}
		return params, nil
	}

	providerStr := model.Provider()
	if overrideProvider != nil {
		providerStr = overrideProvider(model)
	}
	if providerID != nil {
		providerStr = *providerID
	}

	modelIDStr := model.ModelID()
	if overrideModelID != nil {
		modelIDStr = overrideModelID(model)
	}
	if modelID != nil {
		modelIDStr = *modelID
	}

	maxImagesPerCallFn := model.MaxImagesPerCall
	if overrideMaxImagesPerCall != nil {
		val, _ := overrideMaxImagesPerCall(model)
		maxImagesPerCallFn = func() (*int, error) { return val, nil }
	}

	return &wrappedImageModel{
		provider:         providerStr,
		modelID:          modelIDStr,
		maxImagesPerCall: maxImagesPerCallFn,
		doGenerate: func(params im.CallOptions) (im.GenerateResult, error) {
			transformedParams, err := doTransform(params)
			if err != nil {
				return im.GenerateResult{}, err
			}
			doGenerateFn := func() (im.GenerateResult, error) {
				return model.DoGenerate(transformedParams)
			}
			if wrapGenerate != nil {
				return wrapGenerate(mw.WrapImageGenerateOptions{
					DoGenerate: doGenerateFn,
					Params:     transformedParams,
					Model:      model,
				})
			}
			return doGenerateFn()
		},
	}
}
