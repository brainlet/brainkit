// Ported from: packages/ai/src/model/as-reranking-model-v4.ts
package model

import (
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
)

// AsRerankingModelV4 converts a v3 or v4 reranking model to a v4 reranking model.
// If the model is already v4, it is returned unchanged.
// If the model is v3, it is wrapped to return "v4" as the specification version.
func AsRerankingModelV4(model rerankingmodel.RerankingModel) rerankingmodel.RerankingModel {
	if model.SpecificationVersion() == "v4" {
		return model
	}

	return &rerankingModelV4Wrapper{inner: model}
}

// rerankingModelV4Wrapper wraps a v3 reranking model to provide a v4 interface.
type rerankingModelV4Wrapper struct {
	inner rerankingmodel.RerankingModel
}

func (w *rerankingModelV4Wrapper) SpecificationVersion() string { return "v4" }
func (w *rerankingModelV4Wrapper) Provider() string             { return w.inner.Provider() }
func (w *rerankingModelV4Wrapper) ModelID() string              { return w.inner.ModelID() }

func (w *rerankingModelV4Wrapper) DoRerank(options rerankingmodel.CallOptions) (rerankingmodel.RerankResult, error) {
	return w.inner.DoRerank(options)
}
