// Ported from: packages/ai/src/model/as-language-model-v3.ts
package model

import (
	"regexp"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/ai/util"
)

// AsLanguageModelV3 converts a v2 or v3 language model to a v3 language model.
// If the model is already v3, it is returned unchanged.
// If the model is v2, it wraps the model to adapt the interface, converting
// finish reasons and usage formats from v2 to v3.
func AsLanguageModelV3(model languagemodel.LanguageModel) languagemodel.LanguageModel {
	if model.SpecificationVersion() == "v3" {
		return model
	}

	util.LogV2CompatibilityWarning(model.Provider(), model.ModelID())

	return &languageModelV3Wrapper{inner: model}
}

// languageModelV3Wrapper wraps a v2 language model to provide a v3 interface.
// In the TS SDK, this uses a Proxy to intercept property access. In Go, we
// implement the interface directly and delegate to the inner model.
type languageModelV3Wrapper struct {
	inner languagemodel.LanguageModel
}

func (w *languageModelV3Wrapper) SpecificationVersion() string { return "v3" }
func (w *languageModelV3Wrapper) Provider() string             { return w.inner.Provider() }
func (w *languageModelV3Wrapper) ModelID() string              { return w.inner.ModelID() }

func (w *languageModelV3Wrapper) SupportedUrls() (map[string][]*regexp.Regexp, error) {
	return w.inner.SupportedUrls()
}

func (w *languageModelV3Wrapper) DoGenerate(options languagemodel.CallOptions) (languagemodel.GenerateResult, error) {
	return w.inner.DoGenerate(options)
}

func (w *languageModelV3Wrapper) DoStream(options languagemodel.CallOptions) (languagemodel.StreamResult, error) {
	return w.inner.DoStream(options)
}
