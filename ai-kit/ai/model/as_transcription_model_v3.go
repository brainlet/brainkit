// Ported from: packages/ai/src/model/as-transcription-model-v3.ts
package model

import (
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/ai/util"
)

// AsTranscriptionModelV3 converts a v2 or v3 transcription model to a v3 transcription model.
// If the model is already v3, it is returned unchanged.
// If the model is v2, it wraps the model to adapt the interface.
func AsTranscriptionModelV3(model transcriptionmodel.TranscriptionModel) transcriptionmodel.TranscriptionModel {
	if model.SpecificationVersion() == "v3" {
		return model
	}

	util.LogV2CompatibilityWarning(model.Provider(), model.ModelID())

	return &transcriptionModelV3Wrapper{inner: model}
}

// transcriptionModelV3Wrapper wraps a v2 transcription model to provide a v3 interface.
type transcriptionModelV3Wrapper struct {
	inner transcriptionmodel.TranscriptionModel
}

func (w *transcriptionModelV3Wrapper) SpecificationVersion() string { return "v3" }
func (w *transcriptionModelV3Wrapper) Provider() string             { return w.inner.Provider() }
func (w *transcriptionModelV3Wrapper) ModelID() string              { return w.inner.ModelID() }

func (w *transcriptionModelV3Wrapper) DoGenerate(options transcriptionmodel.CallOptions) (transcriptionmodel.GenerateResult, error) {
	return w.inner.DoGenerate(options)
}
