// Ported from: packages/ai/src/model/as-transcription-model-v4.ts
package model

import (
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
)

// AsTranscriptionModelV4 converts a v2, v3, or v4 transcription model to a v4 transcription model.
// If the model is already v4, it is returned unchanged.
// If the model is v2, it is first converted to v3 via AsTranscriptionModelV3, then wrapped as v4.
// If the model is v3, it is wrapped to return "v4" as the specification version.
func AsTranscriptionModelV4(model transcriptionmodel.TranscriptionModel) transcriptionmodel.TranscriptionModel {
	if model.SpecificationVersion() == "v4" {
		return model
	}

	// First convert v2 to v3, then proxy v3 as v4
	v3Model := model
	if model.SpecificationVersion() == "v2" {
		v3Model = AsTranscriptionModelV3(model)
	}

	return &transcriptionModelV4Wrapper{inner: v3Model}
}

// transcriptionModelV4Wrapper wraps a v3 transcription model to provide a v4 interface.
type transcriptionModelV4Wrapper struct {
	inner transcriptionmodel.TranscriptionModel
}

func (w *transcriptionModelV4Wrapper) SpecificationVersion() string { return "v4" }
func (w *transcriptionModelV4Wrapper) Provider() string             { return w.inner.Provider() }
func (w *transcriptionModelV4Wrapper) ModelID() string              { return w.inner.ModelID() }

func (w *transcriptionModelV4Wrapper) DoGenerate(options transcriptionmodel.CallOptions) (transcriptionmodel.GenerateResult, error) {
	return w.inner.DoGenerate(options)
}
