// Ported from: packages/ai/src/types/embedding-model.ts
package aitypes

// EmbeddingModel is the embedding model used by the AI SDK.
//
// In TypeScript this is a union: string | EmbeddingModelV4 | EmbeddingModelV3 | EmbeddingModelV2<string>.
// In Go, we represent this as an interface that can hold either a string model ID
// or a model interface implementation.
type EmbeddingModel = any

// Embedding is a vector, i.e. an array of numbers.
// It is e.g. used to represent a text as a vector of word embeddings.
//
// Corresponds to EmbeddingModelV4Embedding from @ai-sdk/provider.
type Embedding = []float64
