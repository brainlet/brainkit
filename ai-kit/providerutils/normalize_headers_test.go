// Ported from: packages/provider-utils/src/normalize-headers.test.ts
package providerutils

import "testing"

func TestNormalizeHeaders_NilInput(t *testing.T) {
	result := NormalizeHeaders(nil)
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestNormalizeHeaders_LowercaseKeys(t *testing.T) {
	input := map[string]string{
		"CONTENT-TYPE":    "application/json",
		"X-CUSTOM-HEADER": "test-value",
	}
	result := NormalizeHeaders(input)
	if result["content-type"] != "application/json" {
		t.Errorf("expected 'application/json', got %q", result["content-type"])
	}
	if result["x-custom-header"] != "test-value" {
		t.Errorf("expected 'test-value', got %q", result["x-custom-header"])
	}
}

func TestNormalizeHeaders_FilterEmptyValues(t *testing.T) {
	input := map[string]string{
		"Authorization": "Bearer token",
		"X-Feature":     "",
		"Content-Type":  "application/json",
	}
	result := NormalizeHeaders(input)
	if _, ok := result["x-feature"]; ok {
		t.Error("expected empty values to be filtered out")
	}
	if result["authorization"] != "Bearer token" {
		t.Errorf("expected 'Bearer token', got %q", result["authorization"])
	}
	if result["content-type"] != "application/json" {
		t.Errorf("expected 'application/json', got %q", result["content-type"])
	}
}
