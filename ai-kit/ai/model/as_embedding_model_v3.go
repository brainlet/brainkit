// Ported from: packages/ai/src/model/as-embedding-model-v3.ts
package model

import (
	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/ai/util"
)

// AsEmbeddingModelV3 converts a v2 or v3 embedding model to a v3 embedding model.
// If the model is already v3, it is returned unchanged.
// If the model is v2, it wraps the model to adapt the interface.
func AsEmbeddingModelV3(model embeddingmodel.EmbeddingModel) embeddingmodel.EmbeddingModel {
	if model.SpecificationVersion() == "v3" {
		return model
	}

	util.LogV2CompatibilityWarning(model.Provider(), model.ModelID())

	return &embeddingModelV3Wrapper{inner: model}
}

// embeddingModelV3Wrapper wraps a v2 embedding model to provide a v3 interface.
type embeddingModelV3Wrapper struct {
	inner embeddingmodel.EmbeddingModel
}

func (w *embeddingModelV3Wrapper) SpecificationVersion() string { return "v3" }
func (w *embeddingModelV3Wrapper) Provider() string             { return w.inner.Provider() }
func (w *embeddingModelV3Wrapper) ModelID() string              { return w.inner.ModelID() }

func (w *embeddingModelV3Wrapper) MaxEmbeddingsPerCall() (*int, error) {
	return w.inner.MaxEmbeddingsPerCall()
}

func (w *embeddingModelV3Wrapper) SupportsParallelCalls() (bool, error) {
	return w.inner.SupportsParallelCalls()
}

func (w *embeddingModelV3Wrapper) DoEmbed(options embeddingmodel.CallOptions) (embeddingmodel.Result, error) {
	return w.inner.DoEmbed(options)
}
