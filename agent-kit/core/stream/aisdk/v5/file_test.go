// Ported from: packages/core/src/stream/aisdk/v5/file.test.ts
package v5

import (
	"encoding/base64"
	"testing"
)

func TestDefaultGeneratedFile(t *testing.T) {
	t.Run("should create from base64 string", func(t *testing.T) {
		data := base64.StdEncoding.EncodeToString([]byte("hello world"))
		f := NewDefaultGeneratedFile(DefaultGeneratedFileOptions{
			Data:      data,
			MediaType: "text/plain",
		})
		if f.MediaType() != "text/plain" {
			t.Errorf("expected media type 'text/plain', got %q", f.MediaType())
		}
		if f.Base64() != data {
			t.Errorf("expected base64 data to match")
		}
	})

	t.Run("should create from byte slice", func(t *testing.T) {
		data := []byte("hello world")
		f := NewDefaultGeneratedFile(DefaultGeneratedFileOptions{
			Data:      data,
			MediaType: "text/plain",
		})
		if f.MediaType() != "text/plain" {
			t.Errorf("expected media type 'text/plain', got %q", f.MediaType())
		}
		bytes := f.Bytes()
		if string(bytes) != "hello world" {
			t.Errorf("expected 'hello world', got %q", string(bytes))
		}
	})

	t.Run("should lazily compute Base64 from bytes", func(t *testing.T) {
		data := []byte("test data")
		f := NewDefaultGeneratedFile(DefaultGeneratedFileOptions{
			Data:      data,
			MediaType: "application/octet-stream",
		})

		b64 := f.Base64()
		expected := base64.StdEncoding.EncodeToString(data)
		if b64 != expected {
			t.Errorf("expected %q, got %q", expected, b64)
		}

		// Second call should return cached value
		b64Again := f.Base64()
		if b64Again != b64 {
			t.Error("expected cached base64 to match")
		}
	})

	t.Run("should lazily compute Bytes from base64", func(t *testing.T) {
		original := []byte("test data")
		data := base64.StdEncoding.EncodeToString(original)
		f := NewDefaultGeneratedFile(DefaultGeneratedFileOptions{
			Data:      data,
			MediaType: "application/octet-stream",
		})

		bytes := f.Bytes()
		if string(bytes) != "test data" {
			t.Errorf("expected 'test data', got %q", string(bytes))
		}

		// Second call should return cached value
		bytesAgain := f.Bytes()
		if string(bytesAgain) != string(bytes) {
			t.Error("expected cached bytes to match")
		}
	})

	t.Run("should handle empty data", func(t *testing.T) {
		f := NewDefaultGeneratedFile(DefaultGeneratedFileOptions{
			Data:      "",
			MediaType: "text/plain",
		})
		if f.Base64() != "" {
			t.Errorf("expected empty base64, got %q", f.Base64())
		}
	})

	t.Run("should implement GeneratedFile interface", func(t *testing.T) {
		f := NewDefaultGeneratedFile(DefaultGeneratedFileOptions{
			Data:      "dGVzdA==",
			MediaType: "text/plain",
		})
		var gf GeneratedFile = f
		if gf.MediaType() != "text/plain" {
			t.Errorf("expected 'text/plain', got %q", gf.MediaType())
		}
	})
}

func TestDefaultGeneratedFileWithType(t *testing.T) {
	t.Run("should have Type field set to file", func(t *testing.T) {
		f := NewDefaultGeneratedFileWithType(DefaultGeneratedFileOptions{
			Data:      "dGVzdA==",
			MediaType: "text/plain",
		})
		if f.Type != "file" {
			t.Errorf("expected type 'file', got %q", f.Type)
		}
	})

	t.Run("should inherit DefaultGeneratedFile methods", func(t *testing.T) {
		data := base64.StdEncoding.EncodeToString([]byte("hello"))
		f := NewDefaultGeneratedFileWithType(DefaultGeneratedFileOptions{
			Data:      data,
			MediaType: "text/plain",
		})
		if f.MediaType() != "text/plain" {
			t.Errorf("expected media type 'text/plain', got %q", f.MediaType())
		}
		if f.Base64() != data {
			t.Errorf("expected base64 to match")
		}
		bytes := f.Bytes()
		if string(bytes) != "hello" {
			t.Errorf("expected 'hello', got %q", string(bytes))
		}
	})
}
