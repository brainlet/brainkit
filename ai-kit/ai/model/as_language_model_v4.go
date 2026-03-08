// Ported from: packages/ai/src/model/as-language-model-v4.ts
package model

import (
	"regexp"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
)

// AsLanguageModelV4 converts a v2, v3, or v4 language model to a v4 language model.
// If the model is already v4, it is returned unchanged.
// If the model is v2, it is first converted to v3 via AsLanguageModelV3, then wrapped as v4.
// If the model is v3, it is wrapped to return "v4" as the specification version.
func AsLanguageModelV4(model languagemodel.LanguageModel) languagemodel.LanguageModel {
	if model.SpecificationVersion() == "v4" {
		return model
	}

	// First convert v2 to v3, then proxy v3 as v4
	v3Model := model
	if model.SpecificationVersion() == "v2" {
		v3Model = AsLanguageModelV3(model)
	}

	return &languageModelV4Wrapper{inner: v3Model}
}

// languageModelV4Wrapper wraps a v3 language model to provide a v4 interface.
// In the TS SDK, this uses a Proxy that only overrides specificationVersion.
type languageModelV4Wrapper struct {
	inner languagemodel.LanguageModel
}

func (w *languageModelV4Wrapper) SpecificationVersion() string { return "v4" }
func (w *languageModelV4Wrapper) Provider() string             { return w.inner.Provider() }
func (w *languageModelV4Wrapper) ModelID() string              { return w.inner.ModelID() }

func (w *languageModelV4Wrapper) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return w.inner.SupportedUrls()
}

func (w *languageModelV4Wrapper) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	return w.inner.DoGenerate(options)
}

func (w *languageModelV4Wrapper) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
	return w.inner.DoStream(options)
}
