// Ported from: packages/provider/src/embedding-model-middleware/v3/embedding-model-v3-middleware.ts
package middleware

import em "github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"

// EmbeddingModelMiddleware defines middleware that can modify the behavior
// of EmbeddingModel operations.
type EmbeddingModelMiddleware struct {
	// OverrideProvider overrides the provider name if desired.
	OverrideProvider func(model em.EmbeddingModel) string

	// OverrideModelID overrides the model ID if desired.
	OverrideModelID func(model em.EmbeddingModel) string

	// OverrideMaxEmbeddingsPerCall overrides the max embeddings per call if desired.
	OverrideMaxEmbeddingsPerCall func(model em.EmbeddingModel) (*int, error)

	// OverrideSupportsParallelCalls overrides support for parallel calls if desired.
	OverrideSupportsParallelCalls func(model em.EmbeddingModel) (bool, error)

	// TransformParams transforms the parameters before they are passed to the embedding model.
	TransformParams func(opts EmbeddingTransformParamsOptions) (em.CallOptions, error)

	// WrapEmbed wraps the embed operation of the embedding model.
	WrapEmbed func(opts WrapEmbedOptions) (em.Result, error)
}

// SpecificationVersion returns "v3".
func (m EmbeddingModelMiddleware) SpecificationVersion() string { return "v3" }

// EmbeddingTransformParamsOptions are the options for TransformParams.
type EmbeddingTransformParamsOptions struct {
	Params em.CallOptions
	Model  em.EmbeddingModel
}

// WrapEmbedOptions are the options for WrapEmbed.
type WrapEmbedOptions struct {
	// DoEmbed is the original embed function.
	DoEmbed func() (em.Result, error)

	// Params are the parameters for the embed call.
	Params em.CallOptions

	// Model is the embedding model instance.
	Model em.EmbeddingModel
}
