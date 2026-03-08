// Ported from: packages/ai/src/model/as-video-model-v4.ts
package model

import (
	"github.com/brainlet/brainkit/ai-kit/provider/videomodel"
)

// AsVideoModelV4 converts a v3 or v4 video model to a v4 video model.
// If the model is already v4, it is returned unchanged.
// If the model is v3, it is wrapped to return "v4" as the specification version.
// Note: There is no v2 video model in the TS SDK.
func AsVideoModelV4(model videomodel.VideoModel) videomodel.VideoModel {
	if model.SpecificationVersion() == "v4" {
		return model
	}

	return &videoModelV4Wrapper{inner: model}
}

// videoModelV4Wrapper wraps a v3 video model to provide a v4 interface.
type videoModelV4Wrapper struct {
	inner videomodel.VideoModel
}

func (w *videoModelV4Wrapper) SpecificationVersion() string { return "v4" }
func (w *videoModelV4Wrapper) Provider() string             { return w.inner.Provider() }
func (w *videoModelV4Wrapper) ModelID() string              { return w.inner.ModelID() }

func (w *videoModelV4Wrapper) MaxVideosPerCall() (*int, error) {
	return w.inner.MaxVideosPerCall()
}

func (w *videoModelV4Wrapper) DoGenerate(options videomodel.CallOptions) (videomodel.GenerateResult, error) {
	return w.inner.DoGenerate(options)
}
