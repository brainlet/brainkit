// Ported from: packages/ai/src/types/provider-metadata.ts
package aitypes

// ProviderMetadata is additional provider-specific metadata returned from the provider.
//
// This is needed to enable provider-specific functionality that can be
// fully encapsulated in the provider.
//
// Corresponds to SharedV4ProviderMetadata: Record<string, Record<string, JSONValue>>.
type ProviderMetadata = map[string]JSONObject
