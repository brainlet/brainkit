// Ported from: packages/ai/src/generate-speech/generated-audio-file.ts
package generatespeech

import (
	"fmt"
	"strings"
)

// GeneratedAudioFile represents a generated audio file.
type GeneratedAudioFile struct {
	// Data is the raw binary data of the audio file.
	Data []byte
	// MediaType is the MIME type of the audio file (e.g., "audio/mp3").
	MediaType string
	// Format is the audio format (e.g., "mp3", "wav").
	Format string
}

// NewGeneratedAudioFile creates a new GeneratedAudioFile, determining the format
// from the media type if not explicitly provided.
func NewGeneratedAudioFile(data []byte, mediaType string) (*GeneratedAudioFile, error) {
	format := "mp3"

	if mediaType != "" {
		parts := strings.Split(mediaType, "/")
		if len(parts) == 2 {
			// Handle special cases for audio formats
			if mediaType != "audio/mpeg" {
				format = parts[1]
			}
		}
	}

	if format == "" {
		return nil, fmt.Errorf("audio format must be provided or determinable from media type")
	}

	return &GeneratedAudioFile{
		Data:      data,
		MediaType: mediaType,
		Format:    format,
	}, nil
}
