// Ported from: packages/ai/src/types/image-model.ts
package aitypes

// ImageModel is the image model used by the AI SDK.
//
// In TypeScript this is a union: string | ImageModelV4 | ImageModelV3 | ImageModelV2.
// In Go, we represent this as an interface that can hold either a string model ID
// or a model interface implementation.
type ImageModel = any

// ImageModelProviderMetadata is metadata from the model provider for this call.
//
// In TypeScript this is: ImageModelV4ProviderMetadata | ImageModelV2ProviderMetadata.
// Both are Record<string, { images: JSONArray } & JSONValue>.
type ImageModelProviderMetadata = map[string]JSONValue
