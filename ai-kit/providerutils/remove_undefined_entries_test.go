// Ported from: packages/provider-utils/src/remove-undefined-entries.test.ts
package providerutils

import "testing"

func TestRemoveNilEntries_Basic(t *testing.T) {
	input := map[string]interface{}{
		"a": 1,
		"b": nil,
		"c": "test",
		"d": nil,
	}
	result := RemoveNilEntries(input)
	if len(result) != 2 {
		t.Errorf("expected 2 entries, got %d", len(result))
	}
	if result["a"] != 1 {
		t.Errorf("expected a=1, got %v", result["a"])
	}
	if result["c"] != "test" {
		t.Errorf("expected c='test', got %v", result["c"])
	}
}

func TestRemoveNilEntries_Empty(t *testing.T) {
	input := map[string]interface{}{}
	result := RemoveNilEntries(input)
	if len(result) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result))
	}
}

func TestRemoveNilEntries_AllNil(t *testing.T) {
	input := map[string]interface{}{
		"a": nil,
		"b": nil,
	}
	result := RemoveNilEntries(input)
	if len(result) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result))
	}
}

func TestRemoveNilEntries_PreserveFalsy(t *testing.T) {
	input := map[string]interface{}{
		"a": false,
		"b": 0,
		"c": "",
		"d": nil,
	}
	result := RemoveNilEntries(input)
	if len(result) != 3 {
		t.Errorf("expected 3 entries, got %d", len(result))
	}
	if result["a"] != false {
		t.Errorf("expected a=false, got %v", result["a"])
	}
	if result["b"] != 0 {
		t.Errorf("expected b=0, got %v", result["b"])
	}
	if result["c"] != "" {
		t.Errorf("expected c='', got %v", result["c"])
	}
}
