// Ported from: packages/ai/src/model/as-embedding-model-v4.ts
package model

import (
	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
)

// AsEmbeddingModelV4 converts a v2, v3, or v4 embedding model to a v4 embedding model.
// If the model is already v4, it is returned unchanged.
// If the model is v2, it is first converted to v3 via AsEmbeddingModelV3, then wrapped as v4.
// If the model is v3, it is wrapped to return "v4" as the specification version.
func AsEmbeddingModelV4(model embeddingmodel.EmbeddingModel) embeddingmodel.EmbeddingModel {
	if model.SpecificationVersion() == "v4" {
		return model
	}

	// First convert v2 to v3, then proxy v3 as v4
	v3Model := model
	if model.SpecificationVersion() == "v2" {
		v3Model = AsEmbeddingModelV3(model)
	}

	return &embeddingModelV4Wrapper{inner: v3Model}
}

// embeddingModelV4Wrapper wraps a v3 embedding model to provide a v4 interface.
type embeddingModelV4Wrapper struct {
	inner embeddingmodel.EmbeddingModel
}

func (w *embeddingModelV4Wrapper) SpecificationVersion() string { return "v4" }
func (w *embeddingModelV4Wrapper) Provider() string             { return w.inner.Provider() }
func (w *embeddingModelV4Wrapper) ModelID() string              { return w.inner.ModelID() }

func (w *embeddingModelV4Wrapper) MaxEmbeddingsPerCall() (*int, error) {
	return w.inner.MaxEmbeddingsPerCall()
}

func (w *embeddingModelV4Wrapper) SupportsParallelCalls() (bool, error) {
	return w.inner.SupportsParallelCalls()
}

func (w *embeddingModelV4Wrapper) DoEmbed(options embeddingmodel.CallOptions) (embeddingmodel.Result, error) {
	return w.inner.DoEmbed(options)
}
