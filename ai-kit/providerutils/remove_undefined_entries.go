// Ported from: packages/provider-utils/src/remove-undefined-entries.ts
package providerutils

// RemoveNilEntries removes entries from a map where the value is nil.
// In Go, this applies to map[string]*T where nil pointers are removed.
// For map[string]interface{}, nil values are removed.
func RemoveNilEntries(record map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range record {
		if v != nil {
			result[k] = v
		}
	}
	return result
}

// RemoveEmptyStringEntries removes entries from a string map where the value is empty.
func RemoveEmptyStringEntries(record map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range record {
		if v != "" {
			result[k] = v
		}
	}
	return result
}
