// Ported from: packages/provider-utils/src/media-type-to-extension.test.ts
package providerutils

import "testing"

func TestMediaTypeToExtension(t *testing.T) {
	tests := []struct {
		mediaType string
		expected  string
	}{
		{"audio/mpeg", "mp3"},
		{"audio/mp3", "mp3"},
		{"audio/wav", "wav"},
		{"audio/x-wav", "wav"},
		{"audio/webm", "webm"},
		{"audio/ogg", "ogg"},
		{"audio/opus", "ogg"},
		{"audio/mp4", "m4a"},
		{"audio/x-m4a", "m4a"},
		{"audio/flac", "flac"},
		{"audio/aac", "aac"},
		{"AUDIO/MPEG", "mp3"},
		{"AUDIO/MP3", "mp3"},
		{"nope", ""},
	}
	for _, tt := range tests {
		t.Run(tt.mediaType, func(t *testing.T) {
			result := MediaTypeToExtension(tt.mediaType)
			if result != tt.expected {
				t.Errorf("MediaTypeToExtension(%q) = %q, want %q", tt.mediaType, result, tt.expected)
			}
		})
	}
}
