// Ported from: packages/core/src/stream/aisdk/v5/compat/media.test.ts
package compat

import (
	"encoding/base64"
	"testing"
)

func TestStripID3(t *testing.T) {
	t.Run("should return data unchanged if less than 10 bytes", func(t *testing.T) {
		data := []byte{0x01, 0x02, 0x03}
		result := stripID3(data)
		if len(result) != 3 {
			t.Errorf("expected 3 bytes, got %d", len(result))
		}
	})

	t.Run("should strip ID3 header based on synchsafe size", func(t *testing.T) {
		// Build a fake ID3v2 header: "ID3" + version + flags + size (4 bytes synchsafe)
		// Size = 0 means just the 10-byte header
		data := make([]byte, 20)
		data[0] = 0x49 // I
		data[1] = 0x44 // D
		data[2] = 0x33 // 3
		data[3] = 0x04 // version
		data[4] = 0x00 // flags
		// Synchsafe int for size=0: bytes 6-9 all zero
		data[6] = 0x00
		data[7] = 0x00
		data[8] = 0x00
		data[9] = 0x00
		// The actual audio data starts at byte 10
		data[10] = 0xFF
		data[11] = 0xFB

		result := stripID3(data)
		if len(result) != 10 {
			t.Errorf("expected 10 bytes after stripping, got %d", len(result))
		}
		if result[0] != 0xFF || result[1] != 0xFB {
			t.Errorf("expected audio data, got %v", result[:2])
		}
	})

	t.Run("should return original data if computed start exceeds length", func(t *testing.T) {
		data := make([]byte, 10)
		data[6] = 0x7F // Large synchsafe int
		data[7] = 0x7F
		data[8] = 0x7F
		data[9] = 0x7F
		result := stripID3(data)
		if len(result) != 10 {
			t.Errorf("expected original data, got %d bytes", len(result))
		}
	})
}

func TestStripID3BytesIfPresent(t *testing.T) {
	t.Run("should strip ID3 tags when present", func(t *testing.T) {
		data := make([]byte, 15)
		data[0] = 0x49 // I
		data[1] = 0x44 // D
		data[2] = 0x33 // 3
		data[3] = 0x04
		data[4] = 0x00
		data[6] = 0x00
		data[7] = 0x00
		data[8] = 0x00
		data[9] = 0x00
		data[10] = 0xFF
		result := stripID3BytesIfPresent(data)
		if result[0] != 0xFF {
			t.Errorf("expected first byte 0xFF after stripping, got %02x", result[0])
		}
	})

	t.Run("should not modify data without ID3 header", func(t *testing.T) {
		data := []byte{0xFF, 0xFB, 0x90, 0x00}
		result := stripID3BytesIfPresent(data)
		if len(result) != 4 {
			t.Errorf("expected 4 bytes, got %d", len(result))
		}
	})
}

func TestStripID3StringIfPresent(t *testing.T) {
	t.Run("should strip ID3 tags from base64 encoded data", func(t *testing.T) {
		// Create ID3v2 header with size 0 followed by audio data
		raw := make([]byte, 14)
		raw[0] = 0x49 // I
		raw[1] = 0x44 // D
		raw[2] = 0x33 // 3
		raw[3] = 0x04
		raw[4] = 0x00
		raw[6] = 0x00
		raw[7] = 0x00
		raw[8] = 0x00
		raw[9] = 0x00
		raw[10] = 0xFF
		raw[11] = 0xFB
		raw[12] = 0x90
		raw[13] = 0x00

		encoded := base64.StdEncoding.EncodeToString(raw)
		result := stripID3StringIfPresent(encoded)

		decoded, err := base64.StdEncoding.DecodeString(result)
		if err != nil {
			t.Fatalf("failed to decode result: %v", err)
		}
		if decoded[0] != 0xFF || decoded[1] != 0xFB {
			t.Errorf("expected audio data after stripping, got %v", decoded[:2])
		}
	})

	t.Run("should not modify strings without ID3 prefix", func(t *testing.T) {
		input := "UklGRg==" // RIFF prefix
		result := stripID3StringIfPresent(input)
		if result != input {
			t.Errorf("expected unchanged string, got %q", result)
		}
	})
}

func TestDetectMediaType(t *testing.T) {
	t.Run("should detect PNG from bytes", func(t *testing.T) {
		data := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}
		result := DetectMediaType(DetectMediaTypeParams{
			Data:       data,
			Signatures: ImageMediaTypeSignatures,
		})
		if result != "image/png" {
			t.Errorf("expected 'image/png', got %q", result)
		}
	})

	t.Run("should detect JPEG from bytes", func(t *testing.T) {
		data := []byte{0xff, 0xd8, 0xff, 0xe0}
		result := DetectMediaType(DetectMediaTypeParams{
			Data:       data,
			Signatures: ImageMediaTypeSignatures,
		})
		if result != "image/jpeg" {
			t.Errorf("expected 'image/jpeg', got %q", result)
		}
	})

	t.Run("should detect GIF from bytes", func(t *testing.T) {
		data := []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}
		result := DetectMediaType(DetectMediaTypeParams{
			Data:       data,
			Signatures: ImageMediaTypeSignatures,
		})
		if result != "image/gif" {
			t.Errorf("expected 'image/gif', got %q", result)
		}
	})

	t.Run("should detect PNG from base64 string", func(t *testing.T) {
		result := DetectMediaType(DetectMediaTypeParams{
			Data:       "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAA",
			Signatures: ImageMediaTypeSignatures,
		})
		if result != "image/png" {
			t.Errorf("expected 'image/png', got %q", result)
		}
	})

	t.Run("should detect JPEG from base64 string", func(t *testing.T) {
		result := DetectMediaType(DetectMediaTypeParams{
			Data:       "/9j/4AAQSkZJRgABAQAAAQABAAD",
			Signatures: ImageMediaTypeSignatures,
		})
		if result != "image/jpeg" {
			t.Errorf("expected 'image/jpeg', got %q", result)
		}
	})

	t.Run("should detect audio/wav from bytes", func(t *testing.T) {
		data := []byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00}
		result := DetectMediaType(DetectMediaTypeParams{
			Data:       data,
			Signatures: AudioMediaTypeSignatures,
		})
		if result != "audio/wav" {
			t.Errorf("expected 'audio/wav', got %q", result)
		}
	})

	t.Run("should detect audio/ogg from base64", func(t *testing.T) {
		result := DetectMediaType(DetectMediaTypeParams{
			Data:       "T2dnUwACAAAAAAAAAABrGS4=",
			Signatures: AudioMediaTypeSignatures,
		})
		if result != "audio/ogg" {
			t.Errorf("expected 'audio/ogg', got %q", result)
		}
	})

	t.Run("should return empty string for unknown format", func(t *testing.T) {
		data := []byte{0x00, 0x01, 0x02, 0x03}
		result := DetectMediaType(DetectMediaTypeParams{
			Data:       data,
			Signatures: ImageMediaTypeSignatures,
		})
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("should return empty string for empty bytes", func(t *testing.T) {
		result := DetectMediaType(DetectMediaTypeParams{
			Data:       []byte{},
			Signatures: ImageMediaTypeSignatures,
		})
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("should return empty string for unknown base64", func(t *testing.T) {
		result := DetectMediaType(DetectMediaTypeParams{
			Data:       "AAAA",
			Signatures: ImageMediaTypeSignatures,
		})
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("should handle ID3 tagged MP3 bytes", func(t *testing.T) {
		// ID3v2 header with size 0 followed by MP3 sync bytes
		data := make([]byte, 14)
		data[0] = 0x49 // I
		data[1] = 0x44 // D
		data[2] = 0x33 // 3
		data[3] = 0x04
		data[4] = 0x00
		data[6] = 0x00
		data[7] = 0x00
		data[8] = 0x00
		data[9] = 0x00
		data[10] = 0xff
		data[11] = 0xfb
		data[12] = 0x90
		data[13] = 0x00

		result := DetectMediaType(DetectMediaTypeParams{
			Data:       data,
			Signatures: AudioMediaTypeSignatures,
		})
		if result != "audio/mpeg" {
			t.Errorf("expected 'audio/mpeg', got %q", result)
		}
	})
}
