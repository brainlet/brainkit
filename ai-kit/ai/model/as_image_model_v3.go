// Ported from: packages/ai/src/model/as-image-model-v3.ts
package model

import (
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/ai/util"
)

// AsImageModelV3 converts a v2 or v3 image model to a v3 image model.
// If the model is already v3, it is returned unchanged.
// If the model is v2, it wraps the model to adapt the interface.
func AsImageModelV3(model imagemodel.ImageModel) imagemodel.ImageModel {
	if model.SpecificationVersion() == "v3" {
		return model
	}

	util.LogV2CompatibilityWarning(model.Provider(), model.ModelID())

	return &imageModelV3Wrapper{inner: model}
}

// imageModelV3Wrapper wraps a v2 image model to provide a v3 interface.
type imageModelV3Wrapper struct {
	inner imagemodel.ImageModel
}

func (w *imageModelV3Wrapper) SpecificationVersion() string { return "v3" }
func (w *imageModelV3Wrapper) Provider() string             { return w.inner.Provider() }
func (w *imageModelV3Wrapper) ModelID() string              { return w.inner.ModelID() }

func (w *imageModelV3Wrapper) MaxImagesPerCall() (*int, error) {
	return w.inner.MaxImagesPerCall()
}

func (w *imageModelV3Wrapper) DoGenerate(options imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
	return w.inner.DoGenerate(options)
}
