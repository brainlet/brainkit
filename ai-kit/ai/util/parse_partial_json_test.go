// Ported from: packages/ai/src/util/parse-partial-json.test.ts
package util

import (
	"testing"
)

func TestParsePartialJSON_NilInput(t *testing.T) {
	result := ParsePartialJSON(nil)
	if result.State != ParseStateUndefinedInput {
		t.Fatalf("expected undefined-input, got %s", result.State)
	}
	if result.Value != nil {
		t.Fatalf("expected nil value, got %v", result.Value)
	}
}

func TestParsePartialJSON_ValidJSON(t *testing.T) {
	text := `{"key": "value"}`
	result := ParsePartialJSON(&text)
	if result.State != ParseStateSuccessfulParse {
		t.Fatalf("expected successful-parse, got %s", result.State)
	}
	m, ok := result.Value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	if m["key"] != "value" {
		t.Fatalf("expected key=value, got %v", m["key"])
	}
}

func TestParsePartialJSON_PartialJSON(t *testing.T) {
	text := `{"key": "value"`
	result := ParsePartialJSON(&text)
	if result.State != ParseStateRepairedParse {
		t.Fatalf("expected repaired-parse, got %s", result.State)
	}
	m, ok := result.Value.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result.Value)
	}
	if m["key"] != "value" {
		t.Fatalf("expected key=value, got %v", m["key"])
	}
}

func TestParsePartialJSON_InvalidJSON(t *testing.T) {
	text := "}}}not json at all{{{"
	result := ParsePartialJSON(&text)
	if result.State != ParseStateFailedParse {
		t.Fatalf("expected failed-parse, got %s", result.State)
	}
}
