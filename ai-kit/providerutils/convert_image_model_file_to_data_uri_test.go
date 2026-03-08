// Ported from: packages/provider-utils/src/convert-image-model-file-to-data-uri.test.ts
package providerutils

import "testing"

func TestConvertImageModelFileToDataUri_URLFile(t *testing.T) {
	result := ConvertImageModelFileToDataUri(ImageModelFile{
		Type: "url",
		URL:  "https://example.com/image.png",
	})
	if result != "https://example.com/image.png" {
		t.Errorf("expected URL passthrough, got %q", result)
	}
}

func TestConvertImageModelFileToDataUri_URLWithParams(t *testing.T) {
	result := ConvertImageModelFileToDataUri(ImageModelFile{
		Type: "url",
		URL:  "https://example.com/image.png?width=100&height=200",
	})
	if result != "https://example.com/image.png?width=100&height=200" {
		t.Errorf("expected URL passthrough, got %q", result)
	}
}

func TestConvertImageModelFileToDataUri_Base64String(t *testing.T) {
	result := ConvertImageModelFileToDataUri(ImageModelFile{
		Type:      "file",
		MediaType: "image/png",
		Data:      "iVBORw0KGgoAAAANSUhEUg==",
	})
	expected := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUg=="
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConvertImageModelFileToDataUri_DifferentMediaType(t *testing.T) {
	result := ConvertImageModelFileToDataUri(ImageModelFile{
		Type:      "file",
		MediaType: "image/jpeg",
		Data:      "base64data",
	})
	expected := "data:image/jpeg;base64,base64data"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConvertImageModelFileToDataUri_Bytes(t *testing.T) {
	// "Hello" in bytes
	data := []byte{72, 101, 108, 108, 111}
	result := ConvertImageModelFileToDataUri(ImageModelFile{
		Type:      "file",
		MediaType: "image/png",
		Data:      data,
	})
	expected := "data:image/png;base64,SGVsbG8="
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConvertImageModelFileToDataUri_EmptyBytes(t *testing.T) {
	result := ConvertImageModelFileToDataUri(ImageModelFile{
		Type:      "file",
		MediaType: "image/png",
		Data:      []byte{},
	})
	expected := "data:image/png;base64,"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestConvertImageModelFileToDataUri_Webp(t *testing.T) {
	data := []byte{72, 101, 108, 108, 111}
	result := ConvertImageModelFileToDataUri(ImageModelFile{
		Type:      "file",
		MediaType: "image/webp",
		Data:      data,
	})
	expected := "data:image/webp;base64,SGVsbG8="
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
