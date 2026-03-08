// Ported from: packages/provider-utils/src/types/provider-options.ts
package providerutils

// ProviderOptions represents additional provider-specific options.
// They are passed through to the provider from the AI SDK and enable
// provider-specific functionality that can be fully encapsulated in the provider.
type ProviderOptions = map[string]map[string]interface{}
