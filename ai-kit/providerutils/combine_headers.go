// Ported from: packages/provider-utils/src/combine-headers.ts
package providerutils

// CombineHeaders merges multiple header maps into one.
// Later maps override earlier ones. Nil maps are skipped.
func CombineHeaders(headers ...map[string]string) map[string]string {
	combined := make(map[string]string)
	for _, h := range headers {
		if h == nil {
			continue
		}
		for k, v := range h {
			combined[k] = v
		}
	}
	return combined
}
