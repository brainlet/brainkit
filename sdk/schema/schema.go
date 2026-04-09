package schema

import (
	"encoding/json"
	"reflect"
)

// StructToJSONSchema converts a Go struct to a JSON Schema object.
// Uses struct tags: `json:"fieldName"` for the property name, `desc:"..."` for description,
// `default:"value"` for default values, `optional:"true"` for optional fields.
// Supports: string, int/float → number, bool, slices → array, nested structs → object.
func StructToJSONSchema(v any) json.RawMessage {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return json.RawMessage(`{"type":"object"}`)
	}

	schema := buildObjectSchema(t)
	data, _ := json.Marshal(schema)
	return data
}

func buildObjectSchema(t reflect.Type) map[string]any {
	properties := map[string]any{}
	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		name := field.Tag.Get("json")
		if name == "" || name == "-" {
			// Fall back to lowercase first char
			name = string([]rune{rune(field.Name[0] + 32)}) + field.Name[1:]
		}
		// Strip ,omitempty
		if idx := len(name); idx > 0 {
			for j, c := range name {
				if c == ',' {
					name = name[:j]
					break
				}
			}
		}

		prop := fieldSchema(field.Type)

		if desc := field.Tag.Get("desc"); desc != "" {
			prop["description"] = desc
		}

		// Parse default tag — coerce to the field's type for JSON Schema
		if def := field.Tag.Get("default"); def != "" {
			prop["default"] = coerceDefault(def, field.Type)
		}

		properties[name] = prop

		// Fields without `optional:"true"` tag are required
		if field.Tag.Get("optional") != "true" {
			required = append(required, name)
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

// coerceDefault converts a string default tag value to the appropriate Go type
// for JSON Schema. "true"/"false" → bool, numeric strings → float64, "[]" → empty slice.
func coerceDefault(s string, t reflect.Type) any {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Bool:
		return s == "true"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		var n float64
		if err := json.Unmarshal([]byte(s), &n); err == nil {
			return n
		}
		return s
	case reflect.Slice, reflect.Array:
		if s == "[]" {
			return []any{}
		}
		var v any
		if err := json.Unmarshal([]byte(s), &v); err == nil {
			return v
		}
		return s
	default:
		return s
	}
}

func fieldSchema(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "number"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "number"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice, reflect.Array:
		items := fieldSchema(t.Elem())
		return map[string]any{"type": "array", "items": items}
	case reflect.Struct:
		return buildObjectSchema(t)
	case reflect.Map:
		return map[string]any{"type": "object"}
	default:
		return map[string]any{"type": "string"}
	}
}
