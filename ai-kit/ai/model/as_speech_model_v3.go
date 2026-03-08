// Ported from: packages/ai/src/model/as-speech-model-v3.ts
package model

import (
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/ai/util"
)

// AsSpeechModelV3 converts a v2 or v3 speech model to a v3 speech model.
// If the model is already v3, it is returned unchanged.
// If the model is v2, it wraps the model to adapt the interface.
func AsSpeechModelV3(model speechmodel.SpeechModel) speechmodel.SpeechModel {
	if model.SpecificationVersion() == "v3" {
		return model
	}

	util.LogV2CompatibilityWarning(model.Provider(), model.ModelID())

	return &speechModelV3Wrapper{inner: model}
}

// speechModelV3Wrapper wraps a v2 speech model to provide a v3 interface.
type speechModelV3Wrapper struct {
	inner speechmodel.SpeechModel
}

func (w *speechModelV3Wrapper) SpecificationVersion() string { return "v3" }
func (w *speechModelV3Wrapper) Provider() string             { return w.inner.Provider() }
func (w *speechModelV3Wrapper) ModelID() string              { return w.inner.ModelID() }

func (w *speechModelV3Wrapper) DoGenerate(options speechmodel.CallOptions) (speechmodel.GenerateResult, error) {
	return w.inner.DoGenerate(options)
}
