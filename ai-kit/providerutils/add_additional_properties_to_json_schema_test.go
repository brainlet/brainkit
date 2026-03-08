// Ported from: packages/provider-utils/src/add-additional-properties-to-json-schema.test.ts
package providerutils

import (
	"reflect"
	"testing"
)

func TestAddAdditionalProperties_RecursiveObjects(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"user": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
				},
			},
			"age": map[string]interface{}{"type": "number"},
		},
	}

	result := AddAdditionalPropertiesToJsonSchema(schema)

	if result["additionalProperties"] != false {
		t.Error("expected additionalProperties: false on root")
	}
	user := result["properties"].(map[string]interface{})["user"].(map[string]interface{})
	if user["additionalProperties"] != false {
		t.Error("expected additionalProperties: false on nested object")
	}
}

func TestAddAdditionalProperties_ArrayItems(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"ingredients": map[string]interface{}{
				"type": "array",
				"items": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"name":   map[string]interface{}{"type": "string"},
						"amount": map[string]interface{}{"type": "string"},
					},
					"required": []interface{}{"name", "amount"},
				},
			},
		},
		"required": []interface{}{"ingredients"},
	}

	result := AddAdditionalPropertiesToJsonSchema(schema)

	if result["additionalProperties"] != false {
		t.Error("expected additionalProperties: false on root")
	}
	ingredients := result["properties"].(map[string]interface{})["ingredients"].(map[string]interface{})
	items := ingredients["items"].(map[string]interface{})
	if items["additionalProperties"] != false {
		t.Error("expected additionalProperties: false on items")
	}
}

func TestAddAdditionalProperties_UnionTypeWithObject(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"response": map[string]interface{}{
				"type": []interface{}{"object", "null"},
				"properties": map[string]interface{}{
					"name": map[string]interface{}{"type": "string"},
				},
			},
		},
	}

	result := AddAdditionalPropertiesToJsonSchema(schema)

	response := result["properties"].(map[string]interface{})["response"].(map[string]interface{})
	if response["additionalProperties"] != false {
		t.Error("expected additionalProperties: false on union type including object")
	}
}

func TestAddAdditionalProperties_AnyOf(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"response": map[string]interface{}{
				"anyOf": []interface{}{
					map[string]interface{}{"type": "object", "properties": map[string]interface{}{"name": map[string]interface{}{"type": "string"}}},
					map[string]interface{}{"type": "object", "properties": map[string]interface{}{"amount": map[string]interface{}{"type": "string"}}},
				},
			},
		},
	}

	result := AddAdditionalPropertiesToJsonSchema(schema)

	response := result["properties"].(map[string]interface{})["response"].(map[string]interface{})
	anyOf := response["anyOf"].([]interface{})
	for i, item := range anyOf {
		m := item.(map[string]interface{})
		if m["additionalProperties"] != false {
			t.Errorf("expected additionalProperties: false on anyOf[%d]", i)
		}
	}
}

func TestAddAdditionalProperties_Definitions(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"node": map[string]interface{}{"$ref": "#/definitions/Node"},
		},
		"definitions": map[string]interface{}{
			"Node": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"value": map[string]interface{}{"type": "string"},
					"next":  map[string]interface{}{"$ref": "#/definitions/Node"},
				},
			},
		},
	}

	result := AddAdditionalPropertiesToJsonSchema(schema)

	node := result["definitions"].(map[string]interface{})["Node"].(map[string]interface{})
	if node["additionalProperties"] != false {
		t.Error("expected additionalProperties: false on definition")
	}
}

func TestAddAdditionalProperties_NonObjectSchemaUnchanged(t *testing.T) {
	schema := map[string]interface{}{"type": "string"}
	result := AddAdditionalPropertiesToJsonSchema(schema)
	if !reflect.DeepEqual(result, map[string]interface{}{"type": "string"}) {
		t.Errorf("expected non-object schema to be unchanged, got %v", result)
	}
}

func TestAddAdditionalProperties_OverwritesExisting(t *testing.T) {
	schema := map[string]interface{}{
		"type":                 "object",
		"additionalProperties": true,
		"properties": map[string]interface{}{
			"meta": map[string]interface{}{
				"type":                 "object",
				"additionalProperties": true,
				"properties": map[string]interface{}{
					"id": map[string]interface{}{"type": "string"},
				},
			},
		},
	}

	result := AddAdditionalPropertiesToJsonSchema(schema)

	if result["additionalProperties"] != false {
		t.Error("expected additionalProperties to be overwritten to false on root")
	}
	meta := result["properties"].(map[string]interface{})["meta"].(map[string]interface{})
	if meta["additionalProperties"] != false {
		t.Error("expected additionalProperties to be overwritten to false on nested")
	}
}
