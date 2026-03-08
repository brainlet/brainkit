// Ported from: packages/provider-utils/src/convert-to-form-data.test.ts
package providerutils

import "testing"

func TestConvertToFormData_BasicFields(t *testing.T) {
	input := map[string]interface{}{
		"model":  "gpt-4",
		"prompt": "A cat",
	}

	result, err := ConvertToFormData(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Body == nil {
		t.Fatal("expected body to be non-nil")
	}
	if result.ContentType == "" {
		t.Fatal("expected content-type to be set")
	}
}

func TestConvertToFormData_SkipsNilValues(t *testing.T) {
	input := map[string]interface{}{
		"model":  "gpt-4",
		"prompt": nil,
	}

	_, err := ConvertToFormData(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestConvertToFormData_SingleElementArray(t *testing.T) {
	input := map[string]interface{}{
		"items": []interface{}{"single"},
	}

	result, err := ConvertToFormData(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Body == nil {
		t.Fatal("expected body to be non-nil")
	}
}

func TestConvertToFormData_MultiElementArray(t *testing.T) {
	input := map[string]interface{}{
		"items": []interface{}{"a", "b", "c"},
	}

	result, err := ConvertToFormData(input, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Body == nil {
		t.Fatal("expected body to be non-nil")
	}
}

func TestConvertToFormData_NoArrayBrackets(t *testing.T) {
	f := false
	input := map[string]interface{}{
		"items": []interface{}{"a", "b"},
	}

	result, err := ConvertToFormData(input, &ConvertToFormDataOptions{UseArrayBrackets: &f})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Body == nil {
		t.Fatal("expected body to be non-nil")
	}
}
