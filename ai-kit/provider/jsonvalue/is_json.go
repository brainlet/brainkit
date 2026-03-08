// Ported from: packages/provider/src/json-value/is-json.ts
package jsonvalue

import "reflect"

// IsJSONValue checks if a value is a valid JSON value.
func IsJSONValue(value any) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string, float64, bool:
		_ = v
		return true
	case int:
		return true
	case int64:
		return true
	case []any:
		for _, item := range v {
			if !IsJSONValue(item) {
				return false
			}
		}
		return true
	case map[string]any:
		for _, val := range v {
			if val != nil && !IsJSONValue(val) {
				return false
			}
		}
		return true
	default:
		rv := reflect.ValueOf(value)
		switch rv.Kind() {
		case reflect.Slice:
			for i := 0; i < rv.Len(); i++ {
				if !IsJSONValue(rv.Index(i).Interface()) {
					return false
				}
			}
			return true
		case reflect.Map:
			if rv.Type().Key().Kind() != reflect.String {
				return false
			}
			for _, key := range rv.MapKeys() {
				val := rv.MapIndex(key).Interface()
				if val != nil && !IsJSONValue(val) {
					return false
				}
			}
			return true
		default:
			return false
		}
	}
}

// IsJSONArray checks if a value is a valid JSON array.
func IsJSONArray(value any) bool {
	arr, ok := value.([]any)
	if !ok {
		return false
	}
	for _, item := range arr {
		if !IsJSONValue(item) {
			return false
		}
	}
	return true
}

// IsJSONObject checks if a value is a valid JSON object.
func IsJSONObject(value any) bool {
	if value == nil {
		return false
	}
	m, ok := value.(map[string]any)
	if !ok {
		return false
	}
	for _, val := range m {
		if val != nil && !IsJSONValue(val) {
			return false
		}
	}
	return true
}
