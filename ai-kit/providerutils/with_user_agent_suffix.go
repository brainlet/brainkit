// Ported from: packages/provider-utils/src/with-user-agent-suffix.ts
package providerutils

import "strings"

// WithUserAgentSuffix appends suffix parts to the "user-agent" header.
// If a "user-agent" header already exists, the suffix parts are appended to it.
// If no "user-agent" header exists, a new one is created with the suffix parts.
// Automatically removes empty entries from the headers.
func WithUserAgentSuffix(headers map[string]string, suffixParts ...string) map[string]string {
	normalized := NormalizeHeaders(headers)

	currentUA := normalized["user-agent"]
	parts := []string{}
	if currentUA != "" {
		parts = append(parts, currentUA)
	}
	for _, p := range suffixParts {
		if p != "" {
			parts = append(parts, p)
		}
	}
	normalized["user-agent"] = strings.Join(parts, " ")

	return normalized
}
