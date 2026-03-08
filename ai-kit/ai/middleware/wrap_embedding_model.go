// Ported from: packages/ai/src/middleware/wrap-embedding-model.ts
package middleware

import (
	em "github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

// WrapEmbeddingModel wraps an EmbeddingModel instance with middleware functionality.
// When multiple middlewares are provided, the first middleware will transform the
// input first, and the last middleware will be wrapped directly around the model.
func WrapEmbeddingModel(opts WrapEmbeddingModelOptions) em.EmbeddingModel {
	model := opts.Model
	middlewares := make([]mw.EmbeddingModelMiddleware, len(opts.Middleware))
	copy(middlewares, opts.Middleware)

	// Reverse so that the first middleware is outermost.
	for i, j := 0, len(middlewares)-1; i < j; i, j = i+1, j-1 {
		middlewares[i], middlewares[j] = middlewares[j], middlewares[i]
	}

	for _, m := range middlewares {
		model = doWrapEmbeddingModel(model, m, opts.ModelID, opts.ProviderID)
	}
	return model
}

// WrapEmbeddingModelOptions holds the options for WrapEmbeddingModel.
type WrapEmbeddingModelOptions struct {
	Model      em.EmbeddingModel
	Middleware []mw.EmbeddingModelMiddleware
	ModelID    *string
	ProviderID *string
}

// wrappedEmbeddingModel implements em.EmbeddingModel.
type wrappedEmbeddingModel struct {
	provider              string
	modelID               string
	maxEmbeddingsPerCall  func() (*int, error)
	supportsParallelCalls func() (bool, error)
	doEmbed               func(options em.CallOptions) (em.Result, error)
}

func (w *wrappedEmbeddingModel) SpecificationVersion() string { return "v3" }
func (w *wrappedEmbeddingModel) Provider() string             { return w.provider }
func (w *wrappedEmbeddingModel) ModelID() string              { return w.modelID }
func (w *wrappedEmbeddingModel) MaxEmbeddingsPerCall() (*int, error) {
	return w.maxEmbeddingsPerCall()
}
func (w *wrappedEmbeddingModel) SupportsParallelCalls() (bool, error) {
	return w.supportsParallelCalls()
}
func (w *wrappedEmbeddingModel) DoEmbed(options em.CallOptions) (em.Result, error) {
	return w.doEmbed(options)
}

func doWrapEmbeddingModel(
	model em.EmbeddingModel,
	middleware mw.EmbeddingModelMiddleware,
	modelID *string,
	providerID *string,
) em.EmbeddingModel {
	transformParams := middleware.TransformParams
	wrapEmbed := middleware.WrapEmbed
	overrideProvider := middleware.OverrideProvider
	overrideModelID := middleware.OverrideModelID
	overrideMaxEmbeddingsPerCall := middleware.OverrideMaxEmbeddingsPerCall
	overrideSupportsParallelCalls := middleware.OverrideSupportsParallelCalls

	doTransform := func(params em.CallOptions) (em.CallOptions, error) {
		if transformParams != nil {
			return transformParams(mw.EmbeddingTransformParamsOptions{
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

	maxEmbeddingsPerCallFn := model.MaxEmbeddingsPerCall
	if overrideMaxEmbeddingsPerCall != nil {
		val, _ := overrideMaxEmbeddingsPerCall(model)
		maxEmbeddingsPerCallFn = func() (*int, error) { return val, nil }
	}

	supportsParallelCallsFn := model.SupportsParallelCalls
	if overrideSupportsParallelCalls != nil {
		val, _ := overrideSupportsParallelCalls(model)
		supportsParallelCallsFn = func() (bool, error) { return val, nil }
	}

	return &wrappedEmbeddingModel{
		provider:              providerStr,
		modelID:               modelIDStr,
		maxEmbeddingsPerCall:  maxEmbeddingsPerCallFn,
		supportsParallelCalls: supportsParallelCallsFn,
		doEmbed: func(params em.CallOptions) (em.Result, error) {
			transformedParams, err := doTransform(params)
			if err != nil {
				return em.Result{}, err
			}
			doEmbedFn := func() (em.Result, error) {
				return model.DoEmbed(transformedParams)
			}
			if wrapEmbed != nil {
				return wrapEmbed(mw.WrapEmbedOptions{
					DoEmbed: doEmbedFn,
					Params:  transformedParams,
					Model:   model,
				})
			}
			return doEmbedFn()
		},
	}
}
