// Ported from: packages/ai/src/model/as-speech-model-v4.ts
package model

import (
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
)

// AsSpeechModelV4 converts a v2, v3, or v4 speech model to a v4 speech model.
// If the model is already v4, it is returned unchanged.
// If the model is v2, it is first converted to v3 via AsSpeechModelV3, then wrapped as v4.
// If the model is v3, it is wrapped to return "v4" as the specification version.
func AsSpeechModelV4(model speechmodel.SpeechModel) speechmodel.SpeechModel {
	if model.SpecificationVersion() == "v4" {
		return model
	}

	// First convert v2 to v3, then proxy v3 as v4
	v3Model := model
	if model.SpecificationVersion() == "v2" {
		v3Model = AsSpeechModelV3(model)
	}

	return &speechModelV4Wrapper{inner: v3Model}
}

// speechModelV4Wrapper wraps a v3 speech model to provide a v4 interface.
type speechModelV4Wrapper struct {
	inner speechmodel.SpeechModel
}

func (w *speechModelV4Wrapper) SpecificationVersion() string { return "v4" }
func (w *speechModelV4Wrapper) Provider() string             { return w.inner.Provider() }
func (w *speechModelV4Wrapper) ModelID() string              { return w.inner.ModelID() }

func (w *speechModelV4Wrapper) DoGenerate(options speechmodel.CallOptions) (speechmodel.GenerateResult, error) {
	return w.inner.DoGenerate(options)
}
