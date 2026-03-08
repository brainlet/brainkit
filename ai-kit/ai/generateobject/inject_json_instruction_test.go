// Ported from: packages/ai/src/generate-object/inject-json-instruction.test.ts
package generateobject

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestInjectJsonInstruction_BasicCaseWithPromptAndSchema(t *testing.T) {
	basicSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "number"},
		},
		"required": []string{"name", "age"},
	}

	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Prompt: "Generate a person",
		Schema: basicSchema,
	})

	schemaJSON, _ := json.Marshal(basicSchema)
	expected := "Generate a person\n\n" +
		"JSON schema:\n" +
		string(schemaJSON) + "\n" +
		"You MUST answer with a JSON object that matches the JSON schema above."

	if result != expected {
		t.Errorf("unexpected result:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestInjectJsonInstruction_OnlyPromptNoSchema(t *testing.T) {
	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Prompt: "Generate a person",
	})

	expected := "Generate a person\n\nYou MUST answer with JSON."
	if result != expected {
		t.Errorf("unexpected result:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestInjectJsonInstruction_OnlySchemaNoPrompt(t *testing.T) {
	basicSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "number"},
		},
		"required": []string{"name", "age"},
	}

	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Schema: basicSchema,
	})

	schemaJSON, _ := json.Marshal(basicSchema)
	expected := "JSON schema:\n" +
		string(schemaJSON) + "\n" +
		"You MUST answer with a JSON object that matches the JSON schema above."

	if result != expected {
		t.Errorf("unexpected result:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestInjectJsonInstruction_NoPromptNoSchema(t *testing.T) {
	result := InjectJsonInstruction(InjectJsonInstructionOptions{})

	expected := "You MUST answer with JSON."
	if result != expected {
		t.Errorf("unexpected result:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestInjectJsonInstruction_CustomSchemaPrefixAndSuffix(t *testing.T) {
	basicSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "number"},
		},
		"required": []string{"name", "age"},
	}

	customPrefix := "Custom prefix:"
	customSuffix := "Custom suffix"
	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Prompt:       "Generate a person",
		Schema:       basicSchema,
		SchemaPrefix: &customPrefix,
		SchemaSuffix: &customSuffix,
	})

	schemaJSON, _ := json.Marshal(basicSchema)
	expected := "Generate a person\n\n" +
		"Custom prefix:\n" +
		string(schemaJSON) + "\n" +
		"Custom suffix"

	if result != expected {
		t.Errorf("unexpected result:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestInjectJsonInstruction_EmptyStringPrompt(t *testing.T) {
	basicSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "number"},
		},
		"required": []string{"name", "age"},
	}

	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Prompt: "",
		Schema: basicSchema,
	})

	schemaJSON, _ := json.Marshal(basicSchema)
	expected := "JSON schema:\n" +
		string(schemaJSON) + "\n" +
		"You MUST answer with a JSON object that matches the JSON schema above."

	if result != expected {
		t.Errorf("unexpected result:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestInjectJsonInstruction_EmptyObjectSchema(t *testing.T) {
	emptySchema := map[string]any{}

	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Prompt: "Generate something",
		Schema: emptySchema,
	})

	expected := "Generate something\n\n" +
		"JSON schema:\n" +
		"{}\n" +
		"You MUST answer with a JSON object that matches the JSON schema above."

	if result != expected {
		t.Errorf("unexpected result:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestInjectJsonInstruction_ComplexNestedSchema(t *testing.T) {
	complexSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"person": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
					"age":  map[string]any{"type": "number"},
					"address": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"street": map[string]any{"type": "string"},
							"city":   map[string]any{"type": "string"},
						},
					},
				},
			},
		},
	}

	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Prompt: "Generate a complex person",
		Schema: complexSchema,
	})

	schemaJSON, _ := json.Marshal(complexSchema)
	expected := "Generate a complex person\n\n" +
		"JSON schema:\n" +
		string(schemaJSON) + "\n" +
		"You MUST answer with a JSON object that matches the JSON schema above."

	if result != expected {
		t.Errorf("unexpected result:\ngot:  %q\nwant: %q", result, expected)
	}
}

func TestInjectJsonInstruction_SchemaWithSpecialCharacters(t *testing.T) {
	specialSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"special@property": map[string]any{"type": "string"},
		},
	}

	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Schema: specialSchema,
	})

	schemaJSON, _ := json.Marshal(specialSchema)

	// Verify it contains the schema and the suffix.
	if !strings.Contains(result, string(schemaJSON)) {
		t.Errorf("result should contain the schema JSON")
	}
	if !strings.Contains(result, "You MUST answer with a JSON object that matches the JSON schema above.") {
		t.Errorf("result should contain the suffix")
	}
}

func TestInjectJsonInstruction_VeryLongPromptAndSchema(t *testing.T) {
	longPrompt := strings.Repeat("A", 1000)
	longSchema := map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
	props := longSchema["properties"].(map[string]any)
	for i := 0; i < 100; i++ {
		props[strings.Repeat("p", i+1)] = map[string]any{"type": "string"}
	}

	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Prompt: longPrompt,
		Schema: longSchema,
	})

	schemaJSON, _ := json.Marshal(longSchema)
	expected := longPrompt + "\n\n" +
		"JSON schema:\n" +
		string(schemaJSON) + "\n" +
		"You MUST answer with a JSON object that matches the JSON schema above."

	if result != expected {
		t.Errorf("unexpected result length: got %d, want %d", len(result), len(expected))
	}
}

func TestInjectJsonInstruction_NilSchema(t *testing.T) {
	// In Go, nil Schema is equivalent to the TS undefined case.
	result := InjectJsonInstruction(InjectJsonInstructionOptions{
		Schema: nil,
	})

	expected := "You MUST answer with JSON."
	if result != expected {
		t.Errorf("unexpected result:\ngot:  %q\nwant: %q", result, expected)
	}
}
