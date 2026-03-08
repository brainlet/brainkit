// Ported from: packages/provider-utils/src/normalize-headers.ts
package providerutils

import "strings"

// NormalizeHeaders normalizes a header map by lower-casing all keys.
// Entries with empty values are removed; nil maps return an empty map.
func NormalizeHeaders(headers map[string]string) map[string]string {
	if headers == nil {
		return map[string]string{}
	}
	normalized := make(map[string]string, len(headers))
	for k, v := range headers {
		if v != "" {
			normalized[strings.ToLower(k)] = v
		}
	}
	return normalized
}
