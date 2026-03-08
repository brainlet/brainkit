// Ported from: packages/ai/src/generate-text/output.test.ts
package generatetext

import (
	"strings"
	"testing"
)

var testOutputContext = OutputContext{
	Response: LanguageModelResponseMetadata{
		ID:      "123",
		ModelID: "456",
	},
	Usage: LanguageModelUsage{},
	FinishReason: "length",
}

// --- TextOutput tests ---

func TestTextOutput_ResponseFormat(t *testing.T) {
	text1 := TextOutput()
	rf, err := text1.ResponseFormat()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rf.Type != "text" {
		t.Errorf("expected type 'text', got %q", rf.Type)
	}
}

func TestTextOutput_ParseCompleteOutput(t *testing.T) {
	text1 := TextOutput()
	result, err := text1.ParseCompleteOutput("some output", testOutputContext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "some output" {
		t.Errorf("expected 'some output', got %v", result)
	}
}

func TestTextOutput_ParseCompleteOutput_EmptyString(t *testing.T) {
	text1 := TextOutput()
	result, err := text1.ParseCompleteOutput("", testOutputContext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %v", result)
	}
}

func TestTextOutput_ParsePartialOutput(t *testing.T) {
	text1 := TextOutput()
	result := text1.ParsePartialOutput("partial text")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Partial != "partial text" {
		t.Errorf("expected 'partial text', got %v", result.Partial)
	}
}

func TestTextOutput_ParsePartialOutput_EmptyString(t *testing.T) {
	text1 := TextOutput()
	result := text1.ParsePartialOutput("")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Partial != "" {
		t.Errorf("expected empty string, got %v", result.Partial)
	}
}

// --- ObjectOutput tests ---

func TestObjectOutput_ResponseFormat(t *testing.T) {
	obj := ObjectOutput(ObjectOutputOptions{
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{"type": "string"},
			},
		},
	})
	rf, err := obj.ResponseFormat()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rf.Type != "json" {
		t.Errorf("expected type 'json', got %q", rf.Type)
	}
	if rf.Schema == nil {
		t.Error("expected schema to be set")
	}
}

func TestObjectOutput_ResponseFormat_WithNameDescription(t *testing.T) {
	obj := ObjectOutput(ObjectOutputOptions{
		Schema: map[string]interface{}{
			"type": "object",
		},
		Name:        "test-name",
		Description: "test description",
	})
	rf, err := obj.ResponseFormat()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rf.Name != "test-name" {
		t.Errorf("expected name 'test-name', got %q", rf.Name)
	}
	if rf.Description != "test description" {
		t.Errorf("expected description 'test description', got %q", rf.Description)
	}
}

func TestObjectOutput_ParseCompleteOutput(t *testing.T) {
	obj := ObjectOutput(ObjectOutputOptions{
		Schema: map[string]interface{}{
			"type": "object",
		},
	})
	result, err := obj.ParseCompleteOutput(`{ "content": "test" }`, testOutputContext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["content"] != "test" {
		t.Errorf("expected content 'test', got %v", m["content"])
	}
}

func TestObjectOutput_ParseCompleteOutput_InvalidJSON(t *testing.T) {
	obj := ObjectOutput(ObjectOutputOptions{
		Schema: map[string]interface{}{
			"type": "object",
		},
	})
	_, err := obj.ParseCompleteOutput("{ broken json", testOutputContext)
	if err == nil {
		t.Fatal("expected error for broken JSON")
	}
	if !strings.Contains(err.Error(), "could not parse") {
		t.Errorf("expected 'could not parse' in error, got %q", err.Error())
	}
}

func TestObjectOutput_ParsePartialOutput_ValidJSON(t *testing.T) {
	obj := ObjectOutput(ObjectOutputOptions{
		Schema: map[string]interface{}{
			"type": "object",
		},
	})
	result := obj.ParsePartialOutput(`{ "content": "test" }`)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	m, ok := result.Partial.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result.Partial)
	}
	if m["content"] != "test" {
		t.Errorf("expected content 'test', got %v", m["content"])
	}
}

func TestObjectOutput_ParsePartialOutput_EmptyString(t *testing.T) {
	obj := ObjectOutput(ObjectOutputOptions{
		Schema: map[string]interface{}{
			"type": "object",
		},
	})
	result := obj.ParsePartialOutput("")
	if result != nil {
		t.Errorf("expected nil for empty string, got %v", result)
	}
}

// --- ArrayOutput tests ---

func TestArrayOutput_ResponseFormat(t *testing.T) {
	arr := ArrayOutput(ArrayOutputOptions{
		Element: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{"type": "string"},
			},
		},
	})
	rf, err := arr.ResponseFormat()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rf.Type != "json" {
		t.Errorf("expected type 'json', got %q", rf.Type)
	}
	schema := rf.Schema
	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties in schema")
	}
	elements, ok := props["elements"].(map[string]interface{})
	if !ok {
		t.Fatal("expected elements in properties")
	}
	if elements["type"] != "array" {
		t.Errorf("expected elements type 'array', got %v", elements["type"])
	}
}

func TestArrayOutput_ParseCompleteOutput(t *testing.T) {
	arr := ArrayOutput(ArrayOutputOptions{
		Element: map[string]interface{}{
			"type": "object",
		},
	})
	result, err := arr.ParseCompleteOutput(`{ "elements": [{ "content": "test" }] }`, testOutputContext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	slice, ok := result.([]interface{})
	if !ok {
		t.Fatalf("expected slice, got %T", result)
	}
	if len(slice) != 1 {
		t.Fatalf("expected 1 element, got %d", len(slice))
	}
}

func TestArrayOutput_ParseCompleteOutput_InvalidJSON(t *testing.T) {
	arr := ArrayOutput(ArrayOutputOptions{
		Element: map[string]interface{}{"type": "object"},
	})
	_, err := arr.ParseCompleteOutput("{ broken json", testOutputContext)
	if err == nil {
		t.Fatal("expected error for broken JSON")
	}
}

func TestArrayOutput_ParsePartialOutput_ValidJSON(t *testing.T) {
	arr := ArrayOutput(ArrayOutputOptions{
		Element: map[string]interface{}{"type": "object"},
	})
	result := arr.ParsePartialOutput(`{ "elements": [{ "content": "a" }, { "content": "b" }] }`)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	slice, ok := result.Partial.([]interface{})
	if !ok {
		t.Fatalf("expected slice, got %T", result.Partial)
	}
	if len(slice) != 2 {
		t.Errorf("expected 2 elements, got %d", len(slice))
	}
}

func TestArrayOutput_ParsePartialOutput_MissingElements(t *testing.T) {
	arr := ArrayOutput(ArrayOutputOptions{
		Element: map[string]interface{}{"type": "object"},
	})
	result := arr.ParsePartialOutput(`{ "foo": [1,2,3] }`)
	if result != nil {
		t.Errorf("expected nil when elements missing, got %v", result)
	}
}

func TestArrayOutput_ParsePartialOutput_NotArray(t *testing.T) {
	arr := ArrayOutput(ArrayOutputOptions{
		Element: map[string]interface{}{"type": "object"},
	})
	result := arr.ParsePartialOutput(`{ "elements": "not-an-array" }`)
	if result != nil {
		t.Errorf("expected nil when elements is not array, got %v", result)
	}
}

func TestArrayOutput_ParsePartialOutput_EmptyArray(t *testing.T) {
	arr := ArrayOutput(ArrayOutputOptions{
		Element: map[string]interface{}{"type": "object"},
	})
	result := arr.ParsePartialOutput(`{ "elements": [] }`)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	slice, ok := result.Partial.([]interface{})
	if !ok {
		t.Fatalf("expected slice, got %T", result.Partial)
	}
	if len(slice) != 0 {
		t.Errorf("expected 0 elements, got %d", len(slice))
	}
}

// --- ChoiceOutput tests ---

func TestChoiceOutput_ResponseFormat(t *testing.T) {
	ch := ChoiceOutput(ChoiceOutputOptions{
		Options: []string{"aaa", "aab", "ccc"},
	})
	rf, err := ch.ResponseFormat()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rf.Type != "json" {
		t.Errorf("expected type 'json', got %q", rf.Type)
	}
}

func TestChoiceOutput_ResponseFormat_WithNameDescription(t *testing.T) {
	ch := ChoiceOutput(ChoiceOutputOptions{
		Options:     []string{"aaa", "aab", "ccc"},
		Name:        "test-choice-name",
		Description: "test choice description",
	})
	rf, err := ch.ResponseFormat()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rf.Name != "test-choice-name" {
		t.Errorf("expected name 'test-choice-name', got %q", rf.Name)
	}
}

func TestChoiceOutput_ParseCompleteOutput_Valid(t *testing.T) {
	ch := ChoiceOutput(ChoiceOutputOptions{
		Options: []string{"aaa", "aab", "ccc"},
	})
	result, err := ch.ParseCompleteOutput(`{ "result": "aaa" }`, testOutputContext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "aaa" {
		t.Errorf("expected 'aaa', got %v", result)
	}
}

func TestChoiceOutput_ParseCompleteOutput_InvalidJSON(t *testing.T) {
	ch := ChoiceOutput(ChoiceOutputOptions{
		Options: []string{"aaa", "aab", "ccc"},
	})
	_, err := ch.ParseCompleteOutput("{ broken json", testOutputContext)
	if err == nil {
		t.Fatal("expected error for broken JSON")
	}
}

func TestChoiceOutput_ParseCompleteOutput_MissingResult(t *testing.T) {
	ch := ChoiceOutput(ChoiceOutputOptions{
		Options: []string{"aaa", "aab", "ccc"},
	})
	_, err := ch.ParseCompleteOutput(`{}`, testOutputContext)
	if err == nil {
		t.Fatal("expected error for missing result")
	}
}

func TestChoiceOutput_ParseCompleteOutput_InvalidChoice(t *testing.T) {
	ch := ChoiceOutput(ChoiceOutputOptions{
		Options: []string{"aaa", "aab", "ccc"},
	})
	_, err := ch.ParseCompleteOutput(`{ "result": "d" }`, testOutputContext)
	if err == nil {
		t.Fatal("expected error for invalid choice")
	}
}

func TestChoiceOutput_ParseCompleteOutput_NotString(t *testing.T) {
	ch := ChoiceOutput(ChoiceOutputOptions{
		Options: []string{"aaa", "aab", "ccc"},
	})
	_, err := ch.ParseCompleteOutput(`{ "result": 5 }`, testOutputContext)
	if err == nil {
		t.Fatal("expected error for non-string result")
	}
}

func TestChoiceOutput_ParsePartialOutput_Valid(t *testing.T) {
	ch := ChoiceOutput(ChoiceOutputOptions{
		Options: []string{"aaa", "aab", "ccc"},
	})
	result := ch.ParsePartialOutput(`{ "result": "aaa" }`)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Partial != "aaa" {
		t.Errorf("expected 'aaa', got %v", result.Partial)
	}
}

func TestChoiceOutput_ParsePartialOutput_InvalidJSON(t *testing.T) {
	ch := ChoiceOutput(ChoiceOutputOptions{
		Options: []string{"aaa", "aab", "ccc"},
	})
	result := ch.ParsePartialOutput("{ broken json")
	if result != nil {
		t.Errorf("expected nil for broken JSON, got %v", result)
	}
}

func TestChoiceOutput_ParsePartialOutput_MissingResult(t *testing.T) {
	ch := ChoiceOutput(ChoiceOutputOptions{
		Options: []string{"aaa", "aab", "ccc"},
	})
	result := ch.ParsePartialOutput(`{}`)
	if result != nil {
		t.Errorf("expected nil for missing result, got %v", result)
	}
}

func TestChoiceOutput_ParsePartialOutput_NotString(t *testing.T) {
	ch := ChoiceOutput(ChoiceOutputOptions{
		Options: []string{"aaa", "aab", "ccc"},
	})
	result := ch.ParsePartialOutput(`{ "result": 5 }`)
	if result != nil {
		t.Errorf("expected nil for non-string result, got %v", result)
	}
}

// --- JSONOutput tests ---

func TestJSONOutput_ResponseFormat(t *testing.T) {
	j := JSONOutput(JSONOutputOptions{})
	rf, err := j.ResponseFormat()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rf.Type != "json" {
		t.Errorf("expected type 'json', got %q", rf.Type)
	}
}

func TestJSONOutput_ResponseFormat_WithNameDescription(t *testing.T) {
	j := JSONOutput(JSONOutputOptions{
		Name:        "test-json-name",
		Description: "test json description",
	})
	rf, err := j.ResponseFormat()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rf.Name != "test-json-name" {
		t.Errorf("expected name 'test-json-name', got %q", rf.Name)
	}
	if rf.Description != "test json description" {
		t.Errorf("expected description 'test json description', got %q", rf.Description)
	}
}

func TestJSONOutput_ParseCompleteOutput_Valid(t *testing.T) {
	j := JSONOutput(JSONOutputOptions{})
	result, err := j.ParseCompleteOutput(`{"a": 1, "b": [2,3]}`, testOutputContext)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["a"] != float64(1) {
		t.Errorf("expected a=1, got %v", m["a"])
	}
}

func TestJSONOutput_ParseCompleteOutput_Invalid(t *testing.T) {
	j := JSONOutput(JSONOutputOptions{})
	_, err := j.ParseCompleteOutput(`{ a: 1 }`, testOutputContext)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestJSONOutput_ParseCompleteOutput_JustText(t *testing.T) {
	j := JSONOutput(JSONOutputOptions{})
	_, err := j.ParseCompleteOutput(`foo`, testOutputContext)
	if err == nil {
		t.Fatal("expected error for non-JSON text")
	}
}

func TestJSONOutput_ParsePartialOutput_Valid(t *testing.T) {
	j := JSONOutput(JSONOutputOptions{})
	result := j.ParsePartialOutput(`{ "foo": 1, "bar": [2, 3] }`)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	m, ok := result.Partial.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result.Partial)
	}
	if m["foo"] != float64(1) {
		t.Errorf("expected foo=1, got %v", m["foo"])
	}
}

func TestJSONOutput_ParsePartialOutput_Invalid(t *testing.T) {
	j := JSONOutput(JSONOutputOptions{})
	result := j.ParsePartialOutput(`invalid!`)
	if result != nil {
		t.Errorf("expected nil for invalid JSON, got %v", result)
	}
}

func TestJSONOutput_ParsePartialOutput_EmptyString(t *testing.T) {
	j := JSONOutput(JSONOutputOptions{})
	result := j.ParsePartialOutput(``)
	if result != nil {
		t.Errorf("expected nil for empty string, got %v", result)
	}
}
