// Ported from: packages/ai/src/util/detect-media-type.ts
package util

import (
	"encoding/base64"
)

// MediaTypeSignature describes a media type identified by a byte prefix pattern.
// A nil byte in BytesPrefix means "any byte" (wildcard).
type MediaTypeSignature struct {
	MediaType   string
	BytesPrefix []*byte // nil entries mean wildcard
}

// b is a helper to create a *byte from a byte literal.
func b(v byte) *byte {
	return &v
}

// ImageMediaTypeSignatures holds signatures for common image formats.
var ImageMediaTypeSignatures = []MediaTypeSignature{
	{MediaType: "image/gif", BytesPrefix: []*byte{b(0x47), b(0x49), b(0x46)}},
	{MediaType: "image/png", BytesPrefix: []*byte{b(0x89), b(0x50), b(0x4E), b(0x47)}},
	{MediaType: "image/jpeg", BytesPrefix: []*byte{b(0xFF), b(0xD8)}},
	{MediaType: "image/webp", BytesPrefix: []*byte{
		b(0x52), b(0x49), b(0x46), b(0x46),
		nil, nil, nil, nil,
		b(0x57), b(0x45), b(0x42), b(0x50),
	}},
	{MediaType: "image/bmp", BytesPrefix: []*byte{b(0x42), b(0x4D)}},
	{MediaType: "image/tiff", BytesPrefix: []*byte{b(0x49), b(0x49), b(0x2A), b(0x00)}},
	{MediaType: "image/tiff", BytesPrefix: []*byte{b(0x4D), b(0x4D), b(0x00), b(0x2A)}},
	{MediaType: "image/avif", BytesPrefix: []*byte{
		b(0x00), b(0x00), b(0x00), b(0x20), b(0x66), b(0x74), b(0x79), b(0x70),
		b(0x61), b(0x76), b(0x69), b(0x66),
	}},
	{MediaType: "image/heic", BytesPrefix: []*byte{
		b(0x00), b(0x00), b(0x00), b(0x20), b(0x66), b(0x74), b(0x79), b(0x70),
		b(0x68), b(0x65), b(0x69), b(0x63),
	}},
}

// AudioMediaTypeSignatures holds signatures for common audio formats.
var AudioMediaTypeSignatures = []MediaTypeSignature{
	{MediaType: "audio/mpeg", BytesPrefix: []*byte{b(0xFF), b(0xFB)}},
	{MediaType: "audio/mpeg", BytesPrefix: []*byte{b(0xFF), b(0xFA)}},
	{MediaType: "audio/mpeg", BytesPrefix: []*byte{b(0xFF), b(0xF3)}},
	{MediaType: "audio/mpeg", BytesPrefix: []*byte{b(0xFF), b(0xF2)}},
	{MediaType: "audio/mpeg", BytesPrefix: []*byte{b(0xFF), b(0xE3)}},
	{MediaType: "audio/mpeg", BytesPrefix: []*byte{b(0xFF), b(0xE2)}},
	{MediaType: "audio/wav", BytesPrefix: []*byte{
		b(0x52), b(0x49), b(0x46), b(0x46),
		nil, nil, nil, nil,
		b(0x57), b(0x41), b(0x56), b(0x45),
	}},
	{MediaType: "audio/ogg", BytesPrefix: []*byte{b(0x4F), b(0x67), b(0x67), b(0x53)}},
	{MediaType: "audio/flac", BytesPrefix: []*byte{b(0x66), b(0x4C), b(0x61), b(0x43)}},
	{MediaType: "audio/aac", BytesPrefix: []*byte{b(0x40), b(0x15), b(0x00), b(0x00)}},
	{MediaType: "audio/mp4", BytesPrefix: []*byte{b(0x66), b(0x74), b(0x79), b(0x70)}},
	{MediaType: "audio/webm", BytesPrefix: []*byte{b(0x1A), b(0x45), b(0xDF), b(0xA3)}},
}

// VideoMediaTypeSignatures holds signatures for common video formats.
var VideoMediaTypeSignatures = []MediaTypeSignature{
	{MediaType: "video/mp4", BytesPrefix: []*byte{
		b(0x00), b(0x00), b(0x00), nil,
		b(0x66), b(0x74), b(0x79), b(0x70),
	}},
	{MediaType: "video/webm", BytesPrefix: []*byte{b(0x1A), b(0x45), b(0xDF), b(0xA3)}},
	{MediaType: "video/quicktime", BytesPrefix: []*byte{
		b(0x00), b(0x00), b(0x00), b(0x14),
		b(0x66), b(0x74), b(0x79), b(0x70),
		b(0x71), b(0x74),
	}},
	{MediaType: "video/x-msvideo", BytesPrefix: []*byte{b(0x52), b(0x49), b(0x46), b(0x46)}},
}

// stripID3 strips the ID3v2 header from audio data.
func stripID3(data []byte) []byte {
	if len(data) < 10 {
		return data
	}
	id3Size := (int(data[6]&0x7F) << 21) |
		(int(data[7]&0x7F) << 14) |
		(int(data[8]&0x7F) << 7) |
		int(data[9]&0x7F)

	start := id3Size + 10
	if start > len(data) {
		return data
	}
	return data[start:]
}

// stripID3TagsIfPresentBytes strips ID3v2 tags from byte data if present.
func stripID3TagsIfPresentBytes(data []byte) []byte {
	if len(data) > 10 &&
		data[0] == 0x49 && // 'I'
		data[1] == 0x44 && // 'D'
		data[2] == 0x33 { // '3'
		return stripID3(data)
	}
	return data
}

// stripID3TagsIfPresentBase64 strips ID3v2 tags from base64-encoded data if present.
func stripID3TagsIfPresentBase64(data string) string {
	// "SUQz" is the base64 encoding of "ID3"
	if len(data) >= 4 && data[:4] == "SUQz" {
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return data
		}
		stripped := stripID3(decoded)
		return base64.StdEncoding.EncodeToString(stripped)
	}
	return data
}

// DetectMediaTypeFromBytes detects the media type from raw bytes.
func DetectMediaTypeFromBytes(data []byte, signatures []MediaTypeSignature) string {
	data = stripID3TagsIfPresentBytes(data)
	return matchSignatures(data, signatures)
}

// DetectMediaTypeFromBase64 detects the media type from a base64-encoded string.
func DetectMediaTypeFromBase64(data string, signatures []MediaTypeSignature) string {
	data = stripID3TagsIfPresentBase64(data)

	// Convert the first ~18 bytes (24 base64 chars) for consistent detection logic.
	toConvert := data
	if len(toConvert) > 24 {
		toConvert = toConvert[:24]
	}

	// Pad base64 string to a multiple of 4
	for len(toConvert)%4 != 0 {
		toConvert += "="
	}

	bytes, err := base64.StdEncoding.DecodeString(toConvert)
	if err != nil {
		return ""
	}
	return matchSignatures(bytes, signatures)
}

// matchSignatures checks byte data against a list of signatures.
func matchSignatures(data []byte, signatures []MediaTypeSignature) string {
	for _, sig := range signatures {
		if len(data) < len(sig.BytesPrefix) {
			continue
		}
		match := true
		for j, bp := range sig.BytesPrefix {
			if bp != nil && data[j] != *bp {
				match = false
				break
			}
		}
		if match {
			return sig.MediaType
		}
	}
	return ""
}
