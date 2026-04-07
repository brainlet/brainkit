package tools

import (
	"encoding/json"
	"testing"
)

func TestStructToJSONSchema(t *testing.T) {
	type Input struct {
		Name   string  `json:"name" desc:"The user's name"`
		Age    int     `json:"age" desc:"Age in years"`
		Score  float64 `json:"score"`
		Active bool    `json:"active"`
	}

	schema := StructToJSONSchema(Input{})
	var parsed map[string]any
	if err := json.Unmarshal(schema, &parsed); err != nil {
		t.Fatal(err)
	}

	if parsed["type"] != "object" {
		t.Errorf("expected type=object, got %v", parsed["type"])
	}

	props := parsed["properties"].(map[string]any)
	if len(props) != 4 {
		t.Errorf("expected 4 properties, got %d", len(props))
	}

	nameProp := props["name"].(map[string]any)
	if nameProp["type"] != "string" {
		t.Errorf("name type: %v", nameProp["type"])
	}
	if nameProp["description"] != "The user's name" {
		t.Errorf("name desc: %v", nameProp["description"])
	}

	ageProp := props["age"].(map[string]any)
	if ageProp["type"] != "number" {
		t.Errorf("age type: %v", ageProp["type"])
	}

	activeProp := props["active"].(map[string]any)
	if activeProp["type"] != "boolean" {
		t.Errorf("active type: %v", activeProp["type"])
	}

	required := parsed["required"].([]any)
	if len(required) != 4 {
		t.Errorf("expected 4 required, got %v", required)
	}

	t.Logf("Schema: %s", schema)
}

func TestStructToJSONSchema_Nested(t *testing.T) {
	type Address struct {
		City    string `json:"city"`
		Country string `json:"country"`
	}
	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	schema := StructToJSONSchema(Person{})
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	props := parsed["properties"].(map[string]any)
	addrProp := props["address"].(map[string]any)
	if addrProp["type"] != "object" {
		t.Errorf("address type: %v", addrProp["type"])
	}
	addrProps := addrProp["properties"].(map[string]any)
	if len(addrProps) != 2 {
		t.Errorf("expected 2 address properties, got %d", len(addrProps))
	}
	t.Logf("Schema: %s", schema)
}

func TestStructToJSONSchema_Array(t *testing.T) {
	type Input struct {
		Tags []string `json:"tags" desc:"List of tags"`
	}

	schema := StructToJSONSchema(Input{})
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	props := parsed["properties"].(map[string]any)
	tagsProp := props["tags"].(map[string]any)
	if tagsProp["type"] != "array" {
		t.Errorf("tags type: %v", tagsProp["type"])
	}
	items := tagsProp["items"].(map[string]any)
	if items["type"] != "string" {
		t.Errorf("tags items type: %v", items["type"])
	}
	t.Logf("Schema: %s", schema)
}

func TestStructToJSONSchema_Optional(t *testing.T) {
	type Input struct {
		Name  string `json:"name"`
		Email string `json:"email" optional:"true"`
	}

	schema := StructToJSONSchema(Input{})
	var parsed map[string]any
	json.Unmarshal(schema, &parsed)

	required := parsed["required"].([]any)
	if len(required) != 1 || required[0] != "name" {
		t.Errorf("expected only 'name' required, got %v", required)
	}
	t.Logf("Schema: %s", schema)
}
