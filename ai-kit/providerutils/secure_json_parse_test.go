// Ported from: packages/provider-utils/src/secure-json-parse.test.ts
// Licensed under BSD-3-Clause (this file only)
package providerutils

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSecureJsonParse_ParsesObject(t *testing.T) {
	result, err := SecureJsonParse(`{"a": 5, "b": 6}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var expected interface{}
	json.Unmarshal([]byte(`{"a": 5, "b": 6}`), &expected)
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestSecureJsonParse_ParsesNull(t *testing.T) {
	result, err := SecureJsonParse(`null`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestSecureJsonParse_ParsesZero(t *testing.T) {
	result, err := SecureJsonParse(`0`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != float64(0) {
		t.Errorf("expected 0, got %v", result)
	}
}

func TestSecureJsonParse_ParsesString(t *testing.T) {
	result, err := SecureJsonParse(`"X"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "X" {
		t.Errorf("expected 'X', got %v", result)
	}
}

func TestSecureJsonParse_AllowsConstructorWithNonObject(t *testing.T) {
	result, err := SecureJsonParse(`{ "constructor": "string value" }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["constructor"] != "string value" {
		t.Errorf("expected 'string value', got %v", m["constructor"])
	}
}

func TestSecureJsonParse_AllowsConstructorWithNull(t *testing.T) {
	result, err := SecureJsonParse(`{ "constructor": null }`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["constructor"] != nil {
		t.Errorf("expected nil, got %v", m["constructor"])
	}
}

func TestSecureJsonParse_ErrorsOnProtoProperty(t *testing.T) {
	text := `{ "a": 5, "b": 6, "__proto__": { "x": 7 }, "c": { "d": 0, "e": "text", "__proto__": { "y": 8 }, "f": { "g": 2 } } }`
	_, err := SecureJsonParse(text)
	if err == nil {
		t.Fatal("expected error for __proto__ property")
	}
}

func TestSecureJsonParse_ErrorsOnConstructorWithPrototype(t *testing.T) {
	text := `{ "a": 5, "b": 6, "constructor": { "x": 7 }, "c": { "d": 0, "e": "text", "__proto__": { "y": 8 }, "f": { "g": 2 } } }`
	_, err := SecureJsonParse(text)
	if err == nil {
		t.Fatal("expected error for constructor.prototype property")
	}
}

func TestIsParsableJson_ValidJSON(t *testing.T) {
	if !IsParsableJson(`{"foo": "bar"}`) {
		t.Error("expected true for valid JSON object")
	}
	if !IsParsableJson(`[1, 2, 3]`) {
		t.Error("expected true for valid JSON array")
	}
	if !IsParsableJson(`"hello"`) {
		t.Error("expected true for valid JSON string")
	}
}

func TestIsParsableJson_InvalidJSON(t *testing.T) {
	if IsParsableJson(`invalid`) {
		t.Error("expected false for 'invalid'")
	}
	if IsParsableJson(`{foo: "bar"}`) {
		t.Error("expected false for malformed JSON")
	}
}
