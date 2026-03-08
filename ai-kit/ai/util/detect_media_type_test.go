// Ported from: packages/ai/src/util/detect-media-type.test.ts
package util

import (
	"encoding/base64"
	"testing"
)

func TestDetectMediaType_GIF_Bytes(t *testing.T) {
	gifBytes := []byte{0x47, 0x49, 0x46, 0xFF, 0xFF}
	result := DetectMediaTypeFromBytes(gifBytes, ImageMediaTypeSignatures)
	if result != "image/gif" {
		t.Fatalf("expected image/gif, got %s", result)
	}
}

func TestDetectMediaType_GIF_Base64(t *testing.T) {
	result := DetectMediaTypeFromBase64("R0lGabc123", ImageMediaTypeSignatures)
	if result != "image/gif" {
		t.Fatalf("expected image/gif, got %s", result)
	}
}

func TestDetectMediaType_PNG_Bytes(t *testing.T) {
	pngBytes := []byte{0x89, 0x50, 0x4E, 0x47, 0xFF, 0xFF}
	result := DetectMediaTypeFromBytes(pngBytes, ImageMediaTypeSignatures)
	if result != "image/png" {
		t.Fatalf("expected image/png, got %s", result)
	}
}

func TestDetectMediaType_PNG_Base64(t *testing.T) {
	result := DetectMediaTypeFromBase64("iVBORwabc123", ImageMediaTypeSignatures)
	if result != "image/png" {
		t.Fatalf("expected image/png, got %s", result)
	}
}

func TestDetectMediaType_JPEG_Bytes(t *testing.T) {
	jpegBytes := []byte{0xFF, 0xD8, 0xFF, 0xFF}
	result := DetectMediaTypeFromBytes(jpegBytes, ImageMediaTypeSignatures)
	if result != "image/jpeg" {
		t.Fatalf("expected image/jpeg, got %s", result)
	}
}

func TestDetectMediaType_JPEG_Base64(t *testing.T) {
	result := DetectMediaTypeFromBase64("/9j/abc123", ImageMediaTypeSignatures)
	if result != "image/jpeg" {
		t.Fatalf("expected image/jpeg, got %s", result)
	}
}

func TestDetectMediaType_WebP_Bytes(t *testing.T) {
	webpBytes := []byte{
		0x52, 0x49, 0x46, 0x46,
		0x24, 0x00, 0x00, 0x00,
		0x57, 0x45, 0x42, 0x50,
		0x56, 0x50, 0x38, 0x20,
	}
	result := DetectMediaTypeFromBytes(webpBytes, ImageMediaTypeSignatures)
	if result != "image/webp" {
		t.Fatalf("expected image/webp, got %s", result)
	}
}

func TestDetectMediaType_WebP_Base64(t *testing.T) {
	webpBytes := []byte{
		0x52, 0x49, 0x46, 0x46,
		0x24, 0x00, 0x00, 0x00,
		0x57, 0x45, 0x42, 0x50,
		0x56, 0x50, 0x38, 0x20,
	}
	webpBase64 := base64.StdEncoding.EncodeToString(webpBytes)
	result := DetectMediaTypeFromBase64(webpBase64, ImageMediaTypeSignatures)
	if result != "image/webp" {
		t.Fatalf("expected image/webp, got %s", result)
	}
}

func TestDetectMediaType_NotWebP_WAV_Bytes(t *testing.T) {
	wavBytes := []byte{
		0x52, 0x49, 0x46, 0x46,
		0x24, 0x00, 0x00, 0x00,
		0x57, 0x41, 0x56, 0x45,
		0x66, 0x6D, 0x74, 0x20,
	}
	result := DetectMediaTypeFromBytes(wavBytes, ImageMediaTypeSignatures)
	if result != "" {
		t.Fatalf("expected empty, got %s", result)
	}
}

func TestDetectMediaType_BMP_Bytes(t *testing.T) {
	bmpBytes := []byte{0x42, 0x4D, 0xFF, 0xFF}
	result := DetectMediaTypeFromBytes(bmpBytes, ImageMediaTypeSignatures)
	if result != "image/bmp" {
		t.Fatalf("expected image/bmp, got %s", result)
	}
}

func TestDetectMediaType_TIFF_LE_Bytes(t *testing.T) {
	tiffBytes := []byte{0x49, 0x49, 0x2A, 0x00, 0xFF}
	result := DetectMediaTypeFromBytes(tiffBytes, ImageMediaTypeSignatures)
	if result != "image/tiff" {
		t.Fatalf("expected image/tiff, got %s", result)
	}
}

func TestDetectMediaType_TIFF_BE_Bytes(t *testing.T) {
	tiffBytes := []byte{0x4D, 0x4D, 0x00, 0x2A, 0xFF}
	result := DetectMediaTypeFromBytes(tiffBytes, ImageMediaTypeSignatures)
	if result != "image/tiff" {
		t.Fatalf("expected image/tiff, got %s", result)
	}
}

func TestDetectMediaType_AVIF_Bytes(t *testing.T) {
	avifBytes := []byte{
		0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70,
		0x61, 0x76, 0x69, 0x66, 0xFF,
	}
	result := DetectMediaTypeFromBytes(avifBytes, ImageMediaTypeSignatures)
	if result != "image/avif" {
		t.Fatalf("expected image/avif, got %s", result)
	}
}

func TestDetectMediaType_HEIC_Bytes(t *testing.T) {
	heicBytes := []byte{
		0x00, 0x00, 0x00, 0x20, 0x66, 0x74, 0x79, 0x70,
		0x68, 0x65, 0x69, 0x63, 0xFF,
	}
	result := DetectMediaTypeFromBytes(heicBytes, ImageMediaTypeSignatures)
	if result != "image/heic" {
		t.Fatalf("expected image/heic, got %s", result)
	}
}

func TestDetectMediaType_MP3_Bytes(t *testing.T) {
	mp3Bytes := []byte{0xFF, 0xFB}
	result := DetectMediaTypeFromBytes(mp3Bytes, AudioMediaTypeSignatures)
	if result != "audio/mpeg" {
		t.Fatalf("expected audio/mpeg, got %s", result)
	}
}

func TestDetectMediaType_MP3_WithID3_Bytes(t *testing.T) {
	mp3WithID3Bytes := []byte{
		0x49, 0x44, 0x33, // 'ID3'
		0x03, 0x00, // version
		0x00,                                                       // flags
		0x00, 0x00, 0x00, 0x0A,                                     // size (10 bytes)
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // 10 bytes of ID3 data
		0xFF, 0xFB, 0x00, 0x00, // MP3 frame header
	}
	result := DetectMediaTypeFromBytes(mp3WithID3Bytes, AudioMediaTypeSignatures)
	if result != "audio/mpeg" {
		t.Fatalf("expected audio/mpeg, got %s", result)
	}
}

func TestDetectMediaType_MP3_WithID3_Base64(t *testing.T) {
	mp3WithID3Bytes := []byte{
		0x49, 0x44, 0x33,
		0x03, 0x00,
		0x00,
		0x00, 0x00, 0x00, 0x0A,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xFF, 0xFB, 0x00, 0x00,
	}
	mp3Base64 := base64.StdEncoding.EncodeToString(mp3WithID3Bytes)
	result := DetectMediaTypeFromBase64(mp3Base64, AudioMediaTypeSignatures)
	if result != "audio/mpeg" {
		t.Fatalf("expected audio/mpeg, got %s", result)
	}
}

func TestDetectMediaType_WAV_Bytes(t *testing.T) {
	wavBytes := []byte{
		0x52, 0x49, 0x46, 0x46,
		0x24, 0x00, 0x00, 0x00,
		0x57, 0x41, 0x56, 0x45,
		0x66, 0x6D, 0x74, 0x20,
	}
	result := DetectMediaTypeFromBytes(wavBytes, AudioMediaTypeSignatures)
	if result != "audio/wav" {
		t.Fatalf("expected audio/wav, got %s", result)
	}
}

func TestDetectMediaType_WAV_Base64(t *testing.T) {
	wavBytes := []byte{
		0x52, 0x49, 0x46, 0x46,
		0x24, 0x00, 0x00, 0x00,
		0x57, 0x41, 0x56, 0x45,
		0x66, 0x6D, 0x74, 0x20,
	}
	wavBase64 := base64.StdEncoding.EncodeToString(wavBytes)
	result := DetectMediaTypeFromBase64(wavBase64, AudioMediaTypeSignatures)
	if result != "audio/wav" {
		t.Fatalf("expected audio/wav, got %s", result)
	}
}

func TestDetectMediaType_NotWAV_WebP_Bytes(t *testing.T) {
	webpBytes := []byte{
		0x52, 0x49, 0x46, 0x46,
		0x24, 0x00, 0x00, 0x00,
		0x57, 0x45, 0x42, 0x50,
		0x56, 0x50, 0x38, 0x20,
	}
	result := DetectMediaTypeFromBytes(webpBytes, AudioMediaTypeSignatures)
	if result != "" {
		t.Fatalf("expected empty, got %s", result)
	}
}

func TestDetectMediaType_OGG_Bytes(t *testing.T) {
	oggBytes := []byte{0x4F, 0x67, 0x67, 0x53}
	result := DetectMediaTypeFromBytes(oggBytes, AudioMediaTypeSignatures)
	if result != "audio/ogg" {
		t.Fatalf("expected audio/ogg, got %s", result)
	}
}

func TestDetectMediaType_FLAC_Bytes(t *testing.T) {
	flacBytes := []byte{0x66, 0x4C, 0x61, 0x43}
	result := DetectMediaTypeFromBytes(flacBytes, AudioMediaTypeSignatures)
	if result != "audio/flac" {
		t.Fatalf("expected audio/flac, got %s", result)
	}
}

func TestDetectMediaType_AAC_Bytes(t *testing.T) {
	aacBytes := []byte{0x40, 0x15, 0x00, 0x00}
	result := DetectMediaTypeFromBytes(aacBytes, AudioMediaTypeSignatures)
	if result != "audio/aac" {
		t.Fatalf("expected audio/aac, got %s", result)
	}
}

func TestDetectMediaType_MP4_Bytes(t *testing.T) {
	mp4Bytes := []byte{0x66, 0x74, 0x79, 0x70}
	result := DetectMediaTypeFromBytes(mp4Bytes, AudioMediaTypeSignatures)
	if result != "audio/mp4" {
		t.Fatalf("expected audio/mp4, got %s", result)
	}
}

func TestDetectMediaType_WEBM_Bytes(t *testing.T) {
	webmBytes := []byte{0x1A, 0x45, 0xDF, 0xA3}
	result := DetectMediaTypeFromBytes(webmBytes, AudioMediaTypeSignatures)
	if result != "audio/webm" {
		t.Fatalf("expected audio/webm, got %s", result)
	}
}

func TestDetectMediaType_Unknown_Image(t *testing.T) {
	unknownBytes := []byte{0x00, 0x01, 0x02, 0x03}
	result := DetectMediaTypeFromBytes(unknownBytes, ImageMediaTypeSignatures)
	if result != "" {
		t.Fatalf("expected empty, got %s", result)
	}
}

func TestDetectMediaType_Unknown_Audio(t *testing.T) {
	unknownBytes := []byte{0x00, 0x01, 0x02, 0x03}
	result := DetectMediaTypeFromBytes(unknownBytes, AudioMediaTypeSignatures)
	if result != "" {
		t.Fatalf("expected empty, got %s", result)
	}
}

func TestDetectMediaType_Empty_Image(t *testing.T) {
	emptyBytes := []byte{}
	result := DetectMediaTypeFromBytes(emptyBytes, ImageMediaTypeSignatures)
	if result != "" {
		t.Fatalf("expected empty, got %s", result)
	}
}

func TestDetectMediaType_Empty_Audio(t *testing.T) {
	emptyBytes := []byte{}
	result := DetectMediaTypeFromBytes(emptyBytes, AudioMediaTypeSignatures)
	if result != "" {
		t.Fatalf("expected empty, got %s", result)
	}
}

func TestDetectMediaType_Short_Image(t *testing.T) {
	shortBytes := []byte{0x89, 0x50} // Incomplete PNG
	result := DetectMediaTypeFromBytes(shortBytes, ImageMediaTypeSignatures)
	if result != "" {
		t.Fatalf("expected empty, got %s", result)
	}
}

func TestDetectMediaType_Short_Audio(t *testing.T) {
	shortBytes := []byte{0x4F, 0x67} // Incomplete OGG
	result := DetectMediaTypeFromBytes(shortBytes, AudioMediaTypeSignatures)
	if result != "" {
		t.Fatalf("expected empty, got %s", result)
	}
}
