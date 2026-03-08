// Ported from: packages/provider-utils/src/parse-provider-options.ts
package providerutils

import "fmt"

// ParseProviderOptions parses provider-specific options from a provider options map
// using the given schema for validation.
func ParseProviderOptions[T any](
	provider string,
	providerOptions map[string]interface{},
	schema *Schema[T],
) (*T, error) {
	if providerOptions == nil {
		return nil, nil
	}

	value, ok := providerOptions[provider]
	if !ok || value == nil {
		return nil, nil
	}

	result := SafeValidateTypes(value, schema)
	if !result.Success {
		return nil, fmt.Errorf("invalid %s provider options: %w", provider, result.Error)
	}

	return &result.Value, nil
}
