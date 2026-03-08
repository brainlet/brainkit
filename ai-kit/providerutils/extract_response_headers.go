// Ported from: packages/provider-utils/src/extract-response-headers.ts
package providerutils

import "net/http"

// ExtractResponseHeaders extracts the headers from an HTTP response and
// returns them as a key-value map. Multi-value headers are joined with ", ".
func ExtractResponseHeaders(resp *http.Response) map[string]string {
	if resp == nil || resp.Header == nil {
		return map[string]string{}
	}
	result := make(map[string]string, len(resp.Header))
	for k, v := range resp.Header {
		if len(v) == 1 {
			result[k] = v[0]
		} else {
			joined := ""
			for i, s := range v {
				if i > 0 {
					joined += ", "
				}
				joined += s
			}
			result[k] = joined
		}
	}
	return result
}
