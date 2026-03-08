// Ported from: packages/provider/src/shared/v3/shared-v3-provider-options.ts
package shared

import "github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"

// ProviderOptions is additional provider-specific options.
// Options are additional input to the provider.
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
type ProviderOptions = map[string]jsonvalue.JSONObject
