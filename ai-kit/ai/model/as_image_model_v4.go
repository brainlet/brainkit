// Ported from: packages/ai/src/model/as-image-model-v4.ts
package model

import (
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
)

// AsImageModelV4 converts a v2, v3, or v4 image model to a v4 image model.
// If the model is already v4, it is returned unchanged.
// If the model is v2, it is first converted to v3 via AsImageModelV3, then wrapped as v4.
// If the model is v3, it is wrapped to return "v4" as the specification version.
func AsImageModelV4(model imagemodel.ImageModel) imagemodel.ImageModel {
	if model.SpecificationVersion() == "v4" {
		return model
	}

	// First convert v2 to v3, then proxy v3 as v4
	v3Model := model
	if model.SpecificationVersion() == "v2" {
		v3Model = AsImageModelV3(model)
	}

	return &imageModelV4Wrapper{inner: v3Model}
}

// imageModelV4Wrapper wraps a v3 image model to provide a v4 interface.
type imageModelV4Wrapper struct {
	inner imagemodel.ImageModel
}

func (w *imageModelV4Wrapper) SpecificationVersion() string { return "v4" }
func (w *imageModelV4Wrapper) Provider() string             { return w.inner.Provider() }
func (w *imageModelV4Wrapper) ModelID() string              { return w.inner.ModelID() }

func (w *imageModelV4Wrapper) MaxImagesPerCall() (*int, error) {
	return w.inner.MaxImagesPerCall()
}

func (w *imageModelV4Wrapper) DoGenerate(options imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
	return w.inner.DoGenerate(options)
}
