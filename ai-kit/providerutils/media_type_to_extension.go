// Ported from: packages/provider-utils/src/media-type-to-extension.ts
package providerutils

import "strings"

// mediaTypeSubtypeMap maps specific media subtypes to their file extensions.
var mediaTypeSubtypeMap = map[string]string{
	"mpeg":  "mp3",
	"x-wav": "wav",
	"opus":  "ogg",
	"mp4":   "m4a",
	"x-m4a": "m4a",
}

// MediaTypeToExtension maps a media type to its corresponding file extension.
func MediaTypeToExtension(mediaType string) string {
	lower := strings.ToLower(mediaType)
	parts := strings.SplitN(lower, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	subtype := parts[1]
	if ext, ok := mediaTypeSubtypeMap[subtype]; ok {
		return ext
	}
	return subtype
}
