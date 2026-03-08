// Ported from: packages/provider-utils/src/add-additional-properties-to-json-schema.ts
package providerutils

// AddAdditionalPropertiesToJsonSchema recursively adds additionalProperties: false
// to the JSON schema. This is necessary because some providers (e.g. OpenAI) do not
// support additionalProperties: true.
func AddAdditionalPropertiesToJsonSchema(jsonSchema map[string]interface{}) map[string]interface{} {
	schemaType := jsonSchema["type"]

	isObject := false
	if s, ok := schemaType.(string); ok && s == "object" {
		isObject = true
	}
	if arr, ok := schemaType.([]interface{}); ok {
		for _, item := range arr {
			if s, ok := item.(string); ok && s == "object" {
				isObject = true
				break
			}
		}
	}

	if isObject {
		jsonSchema["additionalProperties"] = false
		if properties, ok := jsonSchema["properties"].(map[string]interface{}); ok {
			for key, val := range properties {
				properties[key] = visit(val)
			}
		}
	}

	if items, ok := jsonSchema["items"]; ok && items != nil {
		if arr, ok := items.([]interface{}); ok {
			for i, item := range arr {
				arr[i] = visit(item)
			}
			jsonSchema["items"] = arr
		} else {
			jsonSchema["items"] = visit(items)
		}
	}

	if anyOf, ok := jsonSchema["anyOf"].([]interface{}); ok {
		for i, item := range anyOf {
			anyOf[i] = visit(item)
		}
	}

	if allOf, ok := jsonSchema["allOf"].([]interface{}); ok {
		for i, item := range allOf {
			allOf[i] = visit(item)
		}
	}

	if oneOf, ok := jsonSchema["oneOf"].([]interface{}); ok {
		for i, item := range oneOf {
			oneOf[i] = visit(item)
		}
	}

	if definitions, ok := jsonSchema["definitions"].(map[string]interface{}); ok {
		for key, val := range definitions {
			definitions[key] = visit(val)
		}
	}

	return jsonSchema
}

func visit(def interface{}) interface{} {
	if _, ok := def.(bool); ok {
		return def
	}
	if m, ok := def.(map[string]interface{}); ok {
		return AddAdditionalPropertiesToJsonSchema(m)
	}
	return def
}
