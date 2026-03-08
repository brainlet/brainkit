// Ported from: packages/ai/src/util/prepare-headers.ts
package util

import "net/http"

// PrepareHeaders merges default headers into the provided headers map.
// Existing keys in headers are NOT overwritten by defaultHeaders.
func PrepareHeaders(headers http.Header, defaultHeaders map[string]string) http.Header {
	if headers == nil {
		headers = make(http.Header)
	}

	for key, value := range defaultHeaders {
		if headers.Get(key) == "" {
			headers.Set(key, value)
		}
	}

	return headers
}
