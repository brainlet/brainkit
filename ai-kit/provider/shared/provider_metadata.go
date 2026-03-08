// Ported from: packages/provider/src/shared/v3/shared-v3-provider-metadata.ts
package shared

import "github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"

// ProviderMetadata is additional provider-specific metadata.
// Metadata are additional outputs from the provider.
// They are passed through to the provider from the AI SDK
// and enable provider-specific functionality
// that can be fully encapsulated in the provider.
//
// The outer record is keyed by the provider name, and the inner
// record is keyed by the provider-specific metadata key.
//
//	{
//	  "anthropic": {
//	    "cacheControl": { "type": "ephemeral" }
//	  }
//	}
type ProviderMetadata = map[string]jsonvalue.JSONObject
