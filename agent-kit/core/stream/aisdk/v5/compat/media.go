// Ported from: packages/core/src/stream/aisdk/v5/compat/media.ts
package compat

import (
	"encoding/base64"
	"strings"
)

// ---------------------------------------------------------------------------
// MediaTypeSignature
// ---------------------------------------------------------------------------

// MediaTypeSignature defines a signature for detecting media types from
// binary data or base64-encoded strings.
type MediaTypeSignature struct {
	MediaType    string
	BytesPrefix  []byte
	Base64Prefix string
}

// ---------------------------------------------------------------------------
// Image media type signatures
// ---------------------------------------------------------------------------

// ImageMediaTypeSignatures contains signatures for detecting image media types.
var ImageMediaTypeSignatures = []MediaTypeSignature{
	{
		MediaType:    "image/gif",
		BytesPrefix:  []byte{0x47, 0x49, 0x46},
		Base64Prefix: "R0lG",
	},
	{
		MediaType:    "image/png",
		BytesPrefix:  []byte{0x89, 0x50, 0x4e, 0x47},
		Base64Prefix: "iVBORw",
	},
	{
		MediaType:    "image/jpeg",
		BytesPrefix:  []byte{0xff, 0xd8},
		Base64Prefix: "/9j/",
	},
	{
		MediaType:    "image/webp",
		BytesPrefix:  []byte{0x52, 0x49, 0x46, 0x46},
		Base64Prefix: "UklGRg",
	},
	{
		MediaType:    "image/bmp",
		BytesPrefix:  []byte{0x42, 0x4d},
		Base64Prefix: "Qk",
	},
	{
		MediaType:    "image/tiff",
		BytesPrefix:  []byte{0x49, 0x49, 0x2a, 0x00},
		Base64Prefix: "SUkqAA",
	},
	{
		MediaType:    "image/tiff",
		BytesPrefix:  []byte{0x4d, 0x4d, 0x00, 0x2a},
		Base64Prefix: "TU0AKg",
	},
	{
		MediaType:    "image/avif",
		BytesPrefix:  []byte{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70, 0x61, 0x76, 0x69, 0x66},
		Base64Prefix: "AAAAIGZ0eXBhdmlm",
	},
	{
		MediaType:    "image/heic",
		BytesPrefix:  []byte{0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70, 0x68, 0x65, 0x69, 0x63},
		Base64Prefix: "AAAAIGZ0eXBoZWlj",
	},
}

// ---------------------------------------------------------------------------
// Audio media type signatures
// ---------------------------------------------------------------------------

// AudioMediaTypeSignatures contains signatures for detecting audio media types.
var AudioMediaTypeSignatures = []MediaTypeSignature{
	{
		MediaType:    "audio/mpeg",
		BytesPrefix:  []byte{0xff, 0xfb},
		Base64Prefix: "//s=",
	},
	{
		MediaType:    "audio/mpeg",
		BytesPrefix:  []byte{0xff, 0xfa},
		Base64Prefix: "//o=",
	},
	{
		MediaType:    "audio/mpeg",
		BytesPrefix:  []byte{0xff, 0xf3},
		Base64Prefix: "//M=",
	},
	{
		MediaType:    "audio/mpeg",
		BytesPrefix:  []byte{0xff, 0xf2},
		Base64Prefix: "//I=",
	},
	{
		MediaType:    "audio/mpeg",
		BytesPrefix:  []byte{0xff, 0xe3},
		Base64Prefix: "/+M=",
	},
	{
		MediaType:    "audio/mpeg",
		BytesPrefix:  []byte{0xff, 0xe2},
		Base64Prefix: "/+I=",
	},
	{
		MediaType:    "audio/wav",
		BytesPrefix:  []byte{0x52, 0x49, 0x46, 0x46},
		Base64Prefix: "UklGR",
	},
	{
		MediaType:    "audio/ogg",
		BytesPrefix:  []byte{0x4f, 0x67, 0x67, 0x53},
		Base64Prefix: "T2dnUw",
	},
	{
		MediaType:    "audio/flac",
		BytesPrefix:  []byte{0x66, 0x4c, 0x61, 0x43},
		Base64Prefix: "ZkxhQw",
	},
	{
		MediaType:    "audio/aac",
		BytesPrefix:  []byte{0x40, 0x15, 0x00, 0x00},
		Base64Prefix: "QBUA",
	},
	{
		MediaType:    "audio/mp4",
		BytesPrefix:  []byte{0x66, 0x74, 0x79, 0x70},
		Base64Prefix: "ZnR5cA",
	},
	{
		MediaType:    "audio/webm",
		BytesPrefix:  []byte{0x1a, 0x45, 0xdf, 0xa3},
		Base64Prefix: "GkXf",
	},
}

// ---------------------------------------------------------------------------
// stripID3
// ---------------------------------------------------------------------------

// stripID3 removes an ID3v2 tag header from MP3 data.
// It calculates the tag size from the synchsafe integer at bytes 6-9
// and returns the data starting after the tag.
func stripID3(data []byte) []byte {
	if len(data) < 10 {
		return data
	}
	id3Size := (int(data[6]&0x7f) << 21) |
		(int(data[7]&0x7f) << 14) |
		(int(data[8]&0x7f) << 7) |
		int(data[9]&0x7f)

	start := id3Size + 10
	if start > len(data) {
		return data
	}
	return data[start:]
}

// ---------------------------------------------------------------------------
// stripID3TagsIfPresent
// ---------------------------------------------------------------------------

// stripID3BytesIfPresent checks for and strips ID3v2 tags from byte data.
func stripID3BytesIfPresent(data []byte) []byte {
	if len(data) > 10 &&
		data[0] == 0x49 && // 'I'
		data[1] == 0x44 && // 'D'
		data[2] == 0x33 { // '3'
		return stripID3(data)
	}
	return data
}

// stripID3StringIfPresent checks for and strips ID3v2 tags from base64-encoded string data.
// "SUQz" is the base64 encoding of "ID3".
func stripID3StringIfPresent(data string) string {
	if strings.HasPrefix(data, "SUQz") {
		// Decode, strip, re-encode
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			return data
		}
		stripped := stripID3(decoded)
		return base64.StdEncoding.EncodeToString(stripped)
	}
	return data
}

// ---------------------------------------------------------------------------
// DetectMediaType
// ---------------------------------------------------------------------------

// DetectMediaTypeParams configures media type detection.
type DetectMediaTypeParams struct {
	// Data is either a base64 string or raw bytes.
	Data any // string or []byte
	// Signatures is the set of signatures to check against.
	Signatures []MediaTypeSignature
}

// DetectMediaType detects the media type of data by checking magic bytes
// or base64 prefixes against known signatures.
//
// It handles ID3v2 tag stripping for MP3 files before checking signatures.
//
// Returns the detected media type string, or empty string if no match.
func DetectMediaType(params DetectMediaTypeParams) string {
	switch data := params.Data.(type) {
	case string:
		// Strip ID3 tags if present (base64-encoded)
		processedData := stripID3StringIfPresent(data)
		for _, sig := range params.Signatures {
			if strings.HasPrefix(processedData, sig.Base64Prefix) {
				return sig.MediaType
			}
		}

	case []byte:
		// Strip ID3 tags if present (raw bytes)
		processedData := stripID3BytesIfPresent(data)
		for _, sig := range params.Signatures {
			if len(processedData) >= len(sig.BytesPrefix) {
				match := true
				for i, b := range sig.BytesPrefix {
					if processedData[i] != b {
						match = false
						break
					}
				}
				if match {
					return sig.MediaType
				}
			}
		}
	}

	return ""
}
