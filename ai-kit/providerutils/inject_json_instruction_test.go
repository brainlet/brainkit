// Ported from: packages/provider-utils/src/inject-json-instruction.test.ts
package providerutils

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestInjectJsonInstruction_PromptAndSchema(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
			"age":  map[string]interface{}{"type": "number"},
		},
		"required": []interface{}{"name", "age"},
	}
	prompt := "Generate a person"
	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Prompt: &prompt,
		Schema: schema,
	})

	schemaJSON, _ := json.Marshal(schema)
	expected := "Generate a person\n\nJSON schema:\n" + string(schemaJSON) + "\nYou MUST answer with a JSON object that matches the JSON schema above."
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestInjectJsonInstruction_OnlyPrompt(t *testing.T) {
	prompt := "Generate a person"
	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Prompt: &prompt,
	})
	expected := "Generate a person\n\nYou MUST answer with JSON."
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestInjectJsonInstruction_OnlySchema(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
			"age":  map[string]interface{}{"type": "number"},
		},
		"required": []interface{}{"name", "age"},
	}
	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Schema: schema,
	})

	schemaJSON, _ := json.Marshal(schema)
	expected := "JSON schema:\n" + string(schemaJSON) + "\nYou MUST answer with a JSON object that matches the JSON schema above."
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestInjectJsonInstruction_NoPromptNoSchema(t *testing.T) {
	result := InjectJsonInstruction(InjectJsonInstructionOptions{})
	expected := "You MUST answer with JSON."
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestInjectJsonInstruction_CustomPrefixSuffix(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"name": map[string]interface{}{"type": "string"},
			"age":  map[string]interface{}{"type": "number"},
		},
		"required": []interface{}{"name", "age"},
	}
	prompt := "Generate a person"
	prefix := "Custom prefix:"
	suffix := "Custom suffix"
	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Prompt:       &prompt,
		Schema:       schema,
		SchemaPrefix: &prefix,
		SchemaSuffix: &suffix,
	})

	schemaJSON, _ := json.Marshal(schema)
	expected := "Generate a person\n\nCustom prefix:\n" + string(schemaJSON) + "\nCustom suffix"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestInjectJsonInstruction_EmptyPrompt(t *testing.T) {
	schema := map[string]interface{}{}
	empty := ""
	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Prompt: &empty,
		Schema: schema,
	})
	// Empty prompt should not add newlines
	if strings.HasPrefix(result, "\n") {
		t.Error("empty prompt should not produce leading newlines")
	}
}
