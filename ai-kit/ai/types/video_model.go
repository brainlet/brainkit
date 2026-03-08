// Ported from: packages/ai/src/types/video-model.ts
package aitypes

// VideoModel is a video model that can be a string (model ID) or a video model object.
//
// In TypeScript this is a union: string | Experimental_VideoModelV4 | Experimental_VideoModelV3.
// In Go, we represent this as an interface that can hold either a string model ID
// or a model interface implementation.
type VideoModel = any

// VideoModelProviderMetadata is provider-specific metadata for video model calls.
//
// Corresponds to SharedV4ProviderMetadata from @ai-sdk/provider.
type VideoModelProviderMetadata = ProviderMetadata
