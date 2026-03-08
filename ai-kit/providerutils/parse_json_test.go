// Ported from: packages/provider-utils/src/parse-json.test.ts
// Note: Zod-specific tests are skipped since Go has no Zod equivalent.
package providerutils

import "testing"

func TestParseJSON_BasicNoSchema(t *testing.T) {
	result, err := ParseJSON[interface{}](`{"foo": "bar"}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["foo"] != "bar" {
		t.Errorf("expected foo='bar', got %v", m["foo"])
	}
}

func TestParseJSON_InvalidJSON(t *testing.T) {
	_, err := ParseJSON[interface{}]("invalid json", nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !IsJSONParseError(err) {
		t.Errorf("expected JSONParseError, got %T", err)
	}
}

func TestSafeParseJSON_BasicNoSchema(t *testing.T) {
	result := SafeParseJSON[interface{}](`{"foo": "bar"}`, nil)
	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.Error)
	}
	m, ok := result.RawValue.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map in RawValue, got %T", result.RawValue)
	}
	if m["foo"] != "bar" {
		t.Errorf("expected foo='bar', got %v", m["foo"])
	}
}

func TestSafeParseJSON_InvalidJSON(t *testing.T) {
	result := SafeParseJSON[interface{}]("invalid json", nil)
	if result.Success {
		t.Fatal("expected failure for invalid JSON")
	}
	if result.Error == nil {
		t.Fatal("expected error to be set")
	}
	if !IsJSONParseError(result.Error) {
		t.Errorf("expected JSONParseError, got %T", result.Error)
	}
}

func TestIsParsableJson_Valid(t *testing.T) {
	tests := []string{
		`{"foo": "bar"}`,
		`[1, 2, 3]`,
		`"hello"`,
	}
	for _, input := range tests {
		if !IsParsableJson(input) {
			t.Errorf("expected true for %q", input)
		}
	}
}

func TestIsParsableJson_Invalid(t *testing.T) {
	tests := []string{
		"invalid",
		`{foo: "bar"}`,
		`{"foo": }`,
	}
	for _, input := range tests {
		if IsParsableJson(input) {
			t.Errorf("expected false for %q", input)
		}
	}
}
