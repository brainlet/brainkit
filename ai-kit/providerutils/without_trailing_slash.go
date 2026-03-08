// Ported from: packages/provider-utils/src/without-trailing-slash.ts
package providerutils

import "strings"

// WithoutTrailingSlash removes a trailing slash from a URL string.
// Returns the input unchanged if there is no trailing slash.
func WithoutTrailingSlash(url string) string {
	return strings.TrimRight(url, "/")
}
