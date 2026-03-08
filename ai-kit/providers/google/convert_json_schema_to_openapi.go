// Ported from: packages/google/src/convert-json-schema-to-openapi-schema.ts
package google

// ConvertJSONSchemaToOpenAPISchema converts a JSON Schema 7 definition to
// OpenAPI Schema 3.0 format as expected by the Google Generative AI API.
func ConvertJSONSchemaToOpenAPISchema(jsonSchema map[string]any, isRoot bool) map[string]any {
	if jsonSchema == nil {
		return nil
	}

	if isEmptyObjectSchema(jsonSchema) {
		if isRoot {
			return nil
		}
		if desc, ok := jsonSchema["description"].(string); ok && desc != "" {
			return map[string]any{"type": "object", "description": desc}
		}
		return map[string]any{"type": "object"}
	}

	result := make(map[string]any)

	if desc, ok := jsonSchema["description"].(string); ok && desc != "" {
		result["description"] = desc
	}
	if req, ok := jsonSchema["required"]; ok {
		result["required"] = req
	}
	if format, ok := jsonSchema["format"].(string); ok && format != "" {
		result["format"] = format
	}

	// Handle const
	if constValue, ok := jsonSchema["const"]; ok {
		result["enum"] = []any{constValue}
	}

	// Handle type
	if typ, ok := jsonSchema["type"]; ok {
		switch t := typ.(type) {
		case []any:
			hasNull := false
			var nonNullTypes []any
			for _, item := range t {
				if s, ok := item.(string); ok && s == "null" {
					hasNull = true
				} else {
					nonNullTypes = append(nonNullTypes, item)
				}
			}
			if len(nonNullTypes) == 0 {
				result["type"] = "null"
			} else {
				anyOfItems := make([]any, len(nonNullTypes))
				for i, nt := range nonNullTypes {
					anyOfItems[i] = map[string]any{"type": nt}
				}
				result["anyOf"] = anyOfItems
				if hasNull {
					result["nullable"] = true
				}
			}
		default:
			result["type"] = typ
		}
	}

	// Handle enum
	if enumValues, ok := jsonSchema["enum"]; ok {
		result["enum"] = enumValues
	}

	// Handle properties
	if props, ok := jsonSchema["properties"].(map[string]any); ok {
		convertedProps := make(map[string]any)
		for key, value := range props {
			if subSchema, ok := value.(map[string]any); ok {
				converted := ConvertJSONSchemaToOpenAPISchema(subSchema, false)
				if converted != nil {
					convertedProps[key] = converted
				}
			} else if boolVal, ok := value.(bool); ok {
				convertedProps[key] = convertBoolSchema(boolVal)
			}
		}
		result["properties"] = convertedProps
	}

	// Handle items
	if items, ok := jsonSchema["items"]; ok {
		switch it := items.(type) {
		case []any:
			convertedItems := make([]any, 0, len(it))
			for _, item := range it {
				if subSchema, ok := item.(map[string]any); ok {
					converted := ConvertJSONSchemaToOpenAPISchema(subSchema, false)
					if converted != nil {
						convertedItems = append(convertedItems, converted)
					}
				}
			}
			result["items"] = convertedItems
		case map[string]any:
			converted := ConvertJSONSchemaToOpenAPISchema(it, false)
			if converted != nil {
				result["items"] = converted
			}
		}
	}

	// Handle allOf
	if allOf, ok := jsonSchema["allOf"].([]any); ok {
		convertedAllOf := convertSchemaArray(allOf)
		if len(convertedAllOf) > 0 {
			result["allOf"] = convertedAllOf
		}
	}

	// Handle anyOf
	if anyOf, ok := jsonSchema["anyOf"].([]any); ok {
		// Check if anyOf includes a null type
		hasNullSchema := false
		for _, schema := range anyOf {
			if m, ok := schema.(map[string]any); ok {
				if t, ok := m["type"].(string); ok && t == "null" {
					hasNullSchema = true
					break
				}
			}
		}

		if hasNullSchema {
			var nonNullSchemas []any
			for _, schema := range anyOf {
				if m, ok := schema.(map[string]any); ok {
					if t, ok := m["type"].(string); ok && t == "null" {
						continue
					}
				}
				nonNullSchemas = append(nonNullSchemas, schema)
			}

			if len(nonNullSchemas) == 1 {
				if subSchema, ok := nonNullSchemas[0].(map[string]any); ok {
					converted := ConvertJSONSchemaToOpenAPISchema(subSchema, false)
					if converted != nil {
						result["nullable"] = true
						for k, v := range converted {
							result[k] = v
						}
					}
				}
			} else {
				convertedAnyOf := convertSchemaArray(nonNullSchemas)
				if len(convertedAnyOf) > 0 {
					result["anyOf"] = convertedAnyOf
					result["nullable"] = true
				}
			}
		} else {
			convertedAnyOf := convertSchemaArray(anyOf)
			if len(convertedAnyOf) > 0 {
				result["anyOf"] = convertedAnyOf
			}
		}
	}

	// Handle oneOf
	if oneOf, ok := jsonSchema["oneOf"].([]any); ok {
		convertedOneOf := convertSchemaArray(oneOf)
		if len(convertedOneOf) > 0 {
			result["oneOf"] = convertedOneOf
		}
	}

	// Handle minLength
	if minLength, ok := jsonSchema["minLength"]; ok {
		result["minLength"] = minLength
	}

	return result
}

func convertSchemaArray(schemas []any) []any {
	var result []any
	for _, item := range schemas {
		if subSchema, ok := item.(map[string]any); ok {
			converted := ConvertJSONSchemaToOpenAPISchema(subSchema, false)
			if converted != nil {
				result = append(result, converted)
			}
		} else if boolVal, ok := item.(bool); ok {
			result = append(result, convertBoolSchema(boolVal))
		}
	}
	return result
}

func convertBoolSchema(b bool) map[string]any {
	return map[string]any{"type": "boolean", "properties": map[string]any{}}
}

func isEmptyObjectSchema(schema map[string]any) bool {
	if schema == nil {
		return false
	}
	typ, ok := schema["type"].(string)
	if !ok || typ != "object" {
		return false
	}
	props, hasProps := schema["properties"]
	if hasProps {
		if m, ok := props.(map[string]any); ok && len(m) > 0 {
			return false
		}
	}
	if ap, ok := schema["additionalProperties"]; ok {
		if b, ok := ap.(bool); ok && b {
			return false
		}
	}
	return true
}
