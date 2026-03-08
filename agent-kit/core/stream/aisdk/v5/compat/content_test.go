// Ported from: packages/core/src/stream/aisdk/v5/compat/content.test.ts
package compat

import (
	"testing"
)

func TestSplitDataUrl(t *testing.T) {
	t.Run("should parse a valid data URL", func(t *testing.T) {
		result := splitDataUrl("data:image/png;base64,iVBORw0KGgo=")
		if result.MediaType != "image/png" {
			t.Errorf("expected media type 'image/png', got %q", result.MediaType)
		}
		if result.Base64Content != "iVBORw0KGgo=" {
			t.Errorf("expected base64 content 'iVBORw0KGgo=', got %q", result.Base64Content)
		}
	})

	t.Run("should parse data URL with different media type", func(t *testing.T) {
		result := splitDataUrl("data:audio/wav;base64,UklGR")
		if result.MediaType != "audio/wav" {
			t.Errorf("expected media type 'audio/wav', got %q", result.MediaType)
		}
		if result.Base64Content != "UklGR" {
			t.Errorf("expected base64 content 'UklGR', got %q", result.Base64Content)
		}
	})

	t.Run("should return empty for invalid data URL without comma", func(t *testing.T) {
		result := splitDataUrl("not-a-data-url")
		if result.MediaType != "" {
			t.Errorf("expected empty media type, got %q", result.MediaType)
		}
		if result.Base64Content != "" {
			t.Errorf("expected empty base64 content, got %q", result.Base64Content)
		}
	})

	t.Run("should return empty for malformed header without colon", func(t *testing.T) {
		result := splitDataUrl("invalid;base64,abc")
		if result.MediaType != "" {
			t.Errorf("expected empty media type, got %q", result.MediaType)
		}
	})
}

func TestConvertToDataContent(t *testing.T) {
	t.Run("should pass through byte slices", func(t *testing.T) {
		input := []byte{0x89, 0x50, 0x4e, 0x47}
		result, err := ConvertToDataContent(input)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		bytes, ok := result.Data.([]byte)
		if !ok {
			t.Fatal("expected Data to be []byte")
		}
		if len(bytes) != 4 {
			t.Errorf("expected 4 bytes, got %d", len(bytes))
		}
		if result.MediaType != "" {
			t.Errorf("expected empty media type for raw bytes, got %q", result.MediaType)
		}
	})

	t.Run("should parse data URL strings", func(t *testing.T) {
		result, err := ConvertToDataContent("data:image/jpeg;base64,/9j/4AAQ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.MediaType != "image/jpeg" {
			t.Errorf("expected media type 'image/jpeg', got %q", result.MediaType)
		}
		data, ok := result.Data.(string)
		if !ok {
			t.Fatal("expected Data to be string")
		}
		if data != "/9j/4AAQ" {
			t.Errorf("expected base64 '/9j/4AAQ', got %q", data)
		}
	})

	t.Run("should return plain strings as-is", func(t *testing.T) {
		result, err := ConvertToDataContent("plain text content")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.MediaType != "" {
			t.Errorf("expected empty media type, got %q", result.MediaType)
		}
	})

	t.Run("should return error for invalid data URL format", func(t *testing.T) {
		_, err := ConvertToDataContent("data:;base64,")
		if err == nil {
			t.Error("expected error for invalid data URL")
		}
	})

	t.Run("should handle other types as-is", func(t *testing.T) {
		result, err := ConvertToDataContent(42)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Data != 42 {
			t.Errorf("expected Data to be 42, got %v", result.Data)
		}
	})
}
