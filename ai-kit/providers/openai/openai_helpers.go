// Ported from: packages/openai/src (shared helper utilities)
package openai

import "github.com/brainlet/brainkit/ai-kit/provider/shared"

// providerOptionsToMap converts shared.ProviderOptions to map[string]interface{}.
func providerOptionsToMap(opts shared.ProviderOptions) map[string]interface{} {
	if opts == nil {
		return nil
	}
	result := make(map[string]interface{}, len(opts))
	for k, v := range opts {
		result[k] = v
	}
	return result
}

// convertHeadersPtrMap converts map[string]*string to map[string]string,
// dropping nil values.
func convertHeadersPtrMap(headers map[string]*string) map[string]string {
	if headers == nil {
		return nil
	}
	result := make(map[string]string, len(headers))
	for k, v := range headers {
		if v != nil {
			result[k] = *v
		}
	}
	return result
}
