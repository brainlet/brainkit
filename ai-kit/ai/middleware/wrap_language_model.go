// Ported from: packages/ai/src/middleware/wrap-language-model.ts
package middleware

import (
	"regexp"

	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
)

// WrapLanguageModel wraps a LanguageModel instance with middleware functionality.
// When multiple middlewares are provided, the first middleware will transform the
// input first, and the last middleware will be wrapped directly around the model.
func WrapLanguageModel(opts WrapLanguageModelOptions) lm.LanguageModel {
	model := opts.Model
	middlewares := make([]mw.LanguageModelMiddleware, len(opts.Middleware))
	copy(middlewares, opts.Middleware)

	// Reverse so that the first middleware is outermost.
	for i, j := 0, len(middlewares)-1; i < j; i, j = i+1, j-1 {
		middlewares[i], middlewares[j] = middlewares[j], middlewares[i]
	}

	for _, m := range middlewares {
		model = doWrapLanguageModel(model, m, opts.ModelID, opts.ProviderID)
	}
	return model
}

// WrapLanguageModelOptions holds the options for WrapLanguageModel.
type WrapLanguageModelOptions struct {
	Model      lm.LanguageModel
	Middleware []mw.LanguageModelMiddleware
	ModelID    *string
	ProviderID *string
}

// wrappedLanguageModel implements lm.LanguageModel.
type wrappedLanguageModel struct {
	provider      string
	modelID       string
	supportedUrls func() (map[string][]*regexp.Regexp, error)
	doGenerate    func(options lm.CallOptions) (lm.GenerateResult, error)
	doStream      func(options lm.CallOptions) (lm.StreamResult, error)
}

func (w *wrappedLanguageModel) SpecificationVersion() string { return "v3" }
func (w *wrappedLanguageModel) Provider() string             { return w.provider }
func (w *wrappedLanguageModel) ModelID() string              { return w.modelID }
func (w *wrappedLanguageModel) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return w.supportedUrls()
}
func (w *wrappedLanguageModel) DoGenerate(options lm.CallOptions) (lm.GenerateResult, error) {
	return w.doGenerate(options)
}
func (w *wrappedLanguageModel) DoStream(options lm.CallOptions) (lm.StreamResult, error) {
	return w.doStream(options)
}

func doWrapLanguageModel(
	model lm.LanguageModel,
	middleware mw.LanguageModelMiddleware,
	modelID *string,
	providerID *string,
) lm.LanguageModel {
	transformParams := middleware.TransformParams
	wrapGenerate := middleware.WrapGenerate
	wrapStream := middleware.WrapStream
	overrideProvider := middleware.OverrideProvider
	overrideModelID := middleware.OverrideModelID
	overrideSupportedUrls := middleware.OverrideSupportedUrls

	doTransform := func(params lm.CallOptions, callType string) (lm.CallOptions, error) {
		if transformParams != nil {
			return transformParams(mw.TransformParamsOptions{
				Type:   callType,
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

	supportedUrlsFn := model.SupportedUrls
	if overrideSupportedUrls != nil {
		captured, capturedErr := overrideSupportedUrls(model)
		supportedUrlsFn = func() (map[string][]*regexp.Regexp, error) {
			return captured, capturedErr
		}
	}

	return &wrappedLanguageModel{
		provider:      providerStr,
		modelID:       modelIDStr,
		supportedUrls: supportedUrlsFn,
		doGenerate: func(params lm.CallOptions) (lm.GenerateResult, error) {
			transformedParams, err := doTransform(params, "generate")
			if err != nil {
				return lm.GenerateResult{}, err
			}
			doGenerateFn := func() (lm.GenerateResult, error) {
				return model.DoGenerate(transformedParams)
			}
			doStreamFn := func() (lm.StreamResult, error) {
				return model.DoStream(transformedParams)
			}
			if wrapGenerate != nil {
				return wrapGenerate(mw.WrapGenerateOptions{
					DoGenerate: doGenerateFn,
					DoStream:   doStreamFn,
					Params:     transformedParams,
					Model:      model,
				})
			}
			return doGenerateFn()
		},
		doStream: func(params lm.CallOptions) (lm.StreamResult, error) {
			transformedParams, err := doTransform(params, "stream")
			if err != nil {
				return lm.StreamResult{}, err
			}
			doGenerateFn := func() (lm.GenerateResult, error) {
				return model.DoGenerate(transformedParams)
			}
			doStreamFn := func() (lm.StreamResult, error) {
				return model.DoStream(transformedParams)
			}
			if wrapStream != nil {
				return wrapStream(mw.WrapStreamOptions{
					DoGenerate: doGenerateFn,
					DoStream:   doStreamFn,
					Params:     transformedParams,
					Model:      model,
				})
			}
			return doStreamFn()
		},
	}
}
