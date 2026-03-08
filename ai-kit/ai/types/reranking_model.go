// Ported from: packages/ai/src/types/reranking-model.ts
package aitypes

// RerankingModel is the reranking model used by the AI SDK.
//
// In TypeScript this is a union: RerankingModelV4 | RerankingModelV3.
// In Go, we represent this as an interface that can hold a model interface implementation.
type RerankingModel = any
