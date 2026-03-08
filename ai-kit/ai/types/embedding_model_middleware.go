// Ported from: packages/ai/src/types/embedding-model-middleware.ts
package aitypes

// EmbeddingModelMiddleware is middleware for embedding models.
// Accepts both V3 and V4 middleware types for backward compatibility.
//
// Uses EmbeddingModelV4Middleware as the base but relaxes specificationVersion
// to accept any string (including "v3") and makes it optional.
type EmbeddingModelMiddleware struct {
	// SpecificationVersion is an optional version string (e.g. "v3", "v4").
	SpecificationVersion string `json:"specificationVersion,omitempty"`

	// OverrideProvider overrides the provider name if desired.
	// TODO: Full typing depends on EmbeddingModelV4 interface from provider package.
	OverrideProvider func(model any) string `json:"-"`

	// OverrideModelId overrides the model ID if desired.
	OverrideModelId func(model any) string `json:"-"`

	// OverrideMaxEmbeddingsPerCall overrides the limit of how many embeddings
	// can be generated in a single API call if desired.
	OverrideMaxEmbeddingsPerCall func(model any) *int `json:"-"`

	// OverrideSupportsParallelCalls overrides support for handling multiple
	// embedding calls in parallel, if desired.
	OverrideSupportsParallelCalls func(model any) bool `json:"-"`

	// TransformParams transforms the parameters before they are passed to the embed model.
	TransformParams func(params any, model any) (any, error) `json:"-"`

	// WrapEmbed wraps the embed operation of the embedding model.
	WrapEmbed func(doEmbed func() (any, error), params any, model any) (any, error) `json:"-"`
}
