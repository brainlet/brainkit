// Ported from: packages/core/src/agent/message-list/prompt/attachments-to-parts.test.ts
package prompt

import (
	"strings"
	"testing"
)

func TestAttachmentsToParts(t *testing.T) {
	t.Run("should handle regular HTTP URLs", func(t *testing.T) {
		attachments := []Attachment{
			{URL: "https://example.com/image.png", ContentType: "image/png"},
		}

		parts, err := AttachmentsToParts(attachments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(parts))
		}
		if parts[0].Type != "image" {
			t.Errorf("expected type image, got %s", parts[0].Type)
		}
		if parts[0].Data != "https://example.com/image.png" {
			t.Errorf("expected data https://example.com/image.png, got %s", parts[0].Data)
		}
		if parts[0].MimeType != "image/png" {
			t.Errorf("expected mimeType image/png, got %s", parts[0].MimeType)
		}
	})

	t.Run("should handle data URIs", func(t *testing.T) {
		base64Data := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
		dataUri := "data:image/png;base64," + base64Data

		attachments := []Attachment{
			{URL: dataUri, ContentType: "image/png"},
		}

		parts, err := AttachmentsToParts(attachments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(parts))
		}
		if parts[0].Type != "image" {
			t.Errorf("expected type image, got %s", parts[0].Type)
		}
		if parts[0].MimeType != "image/png" {
			t.Errorf("expected mimeType image/png, got %s", parts[0].MimeType)
		}
		if !strings.Contains(parts[0].Data, "data:image/png;base64,") {
			t.Errorf("expected data to contain data:image/png;base64, got %s", parts[0].Data)
		}
	})

	t.Run("should handle raw base64 strings by converting them to data URIs", func(t *testing.T) {
		base64Data := "iVBORw0KGgoAAAANSUhEUgAAAAgAAAAIAQMAAAD+wSzIAAAABlBMVEX///+/v7+jQ3Y5AAAADklEQVQI12P4AIX8EAgALgAD/aNpbtEAAAAASUVORK5CYII"

		attachments := []Attachment{
			{URL: base64Data, ContentType: "image/png"},
		}

		parts, err := AttachmentsToParts(attachments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(parts))
		}
		if parts[0].Type != "image" {
			t.Errorf("expected type image, got %s", parts[0].Type)
		}
		if parts[0].MimeType != "image/png" {
			t.Errorf("expected mimeType image/png, got %s", parts[0].MimeType)
		}
		if !strings.Contains(parts[0].Data, "data:image/png;base64,") {
			t.Errorf("expected data to contain data:image/png;base64,")
		}
		if !strings.Contains(parts[0].Data, base64Data) {
			t.Errorf("expected data to contain original base64 data")
		}
	})

	t.Run("should handle raw base64 strings for non-image files", func(t *testing.T) {
		base64Data := "SGVsbG8gV29ybGQh" // "Hello World!" in base64

		attachments := []Attachment{
			{URL: base64Data, ContentType: "text/plain"},
		}

		parts, err := AttachmentsToParts(attachments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(parts))
		}
		if parts[0].Type != "file" {
			t.Errorf("expected type file, got %s", parts[0].Type)
		}
		if parts[0].MimeType != "text/plain" {
			t.Errorf("expected mimeType text/plain, got %s", parts[0].MimeType)
		}
		if !strings.Contains(parts[0].Data, "data:text/plain;base64,") {
			t.Errorf("expected data to contain data:text/plain;base64,")
		}
		if !strings.Contains(parts[0].Data, base64Data) {
			t.Errorf("expected data to contain original base64 data")
		}
	})

	t.Run("should handle multiple attachments with mixed formats", func(t *testing.T) {
		base64Data := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg=="
		dataUri := "data:image/jpeg;base64," + base64Data
		httpUrl := "https://example.com/image.png"

		attachments := []Attachment{
			{URL: httpUrl, ContentType: "image/png"},
			{URL: dataUri, ContentType: "image/jpeg"},
			{URL: base64Data, ContentType: "image/gif"},
		}

		parts, err := AttachmentsToParts(attachments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(parts) != 3 {
			t.Fatalf("expected 3 parts, got %d", len(parts))
		}

		// First: HTTP URL
		if parts[0].Type != "image" {
			t.Errorf("expected first part type image, got %s", parts[0].Type)
		}
		if parts[0].Data != httpUrl {
			t.Errorf("expected first part data %s, got %s", httpUrl, parts[0].Data)
		}
		if parts[0].MimeType != "image/png" {
			t.Errorf("expected first part mimeType image/png, got %s", parts[0].MimeType)
		}

		// Second: Data URI
		if parts[1].Type != "image" {
			t.Errorf("expected second part type image, got %s", parts[1].Type)
		}
		if parts[1].MimeType != "image/jpeg" {
			t.Errorf("expected second part mimeType image/jpeg, got %s", parts[1].MimeType)
		}

		// Third: Raw base64 (should be converted to data URI)
		if parts[2].Type != "image" {
			t.Errorf("expected third part type image, got %s", parts[2].Type)
		}
		if parts[2].MimeType != "image/gif" {
			t.Errorf("expected third part mimeType image/gif, got %s", parts[2].MimeType)
		}
		if !strings.Contains(parts[2].Data, "data:image/gif;base64,") {
			t.Errorf("expected third part data to contain data:image/gif;base64,")
		}
	})

	t.Run("should handle HTTPS URLs with query parameters", func(t *testing.T) {
		attachments := []Attachment{
			{URL: "https://example.com/image.png?size=large&format=webp", ContentType: "image/png"},
		}

		parts, err := AttachmentsToParts(attachments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(parts))
		}
		if parts[0].Type != "image" {
			t.Errorf("expected type image, got %s", parts[0].Type)
		}
		if parts[0].Data != "https://example.com/image.png?size=large&format=webp" {
			t.Errorf("expected URL with query params, got %s", parts[0].Data)
		}
	})

	t.Run("should handle HTTP URLs (not just HTTPS)", func(t *testing.T) {
		attachments := []Attachment{
			{URL: "http://example.com/photo.jpg", ContentType: "image/jpeg"},
		}

		parts, err := AttachmentsToParts(attachments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(parts))
		}
		if parts[0].Type != "image" {
			t.Errorf("expected type image, got %s", parts[0].Type)
		}
		if parts[0].Data != "http://example.com/photo.jpg" {
			t.Errorf("expected URL http://example.com/photo.jpg, got %s", parts[0].Data)
		}
	})

	t.Run("should handle non-image file URLs", func(t *testing.T) {
		attachments := []Attachment{
			{URL: "https://example.com/document.pdf", ContentType: "application/pdf"},
		}

		parts, err := AttachmentsToParts(attachments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(parts))
		}
		if parts[0].Type != "file" {
			t.Errorf("expected type file, got %s", parts[0].Type)
		}
		if parts[0].Data != "https://example.com/document.pdf" {
			t.Errorf("expected data https://example.com/document.pdf, got %s", parts[0].Data)
		}
		if parts[0].MimeType != "application/pdf" {
			t.Errorf("expected mimeType application/pdf, got %s", parts[0].MimeType)
		}
	})

	t.Run("should handle URLs with various image formats", func(t *testing.T) {
		imageFormats := []Attachment{
			{URL: "https://example.com/image.webp", ContentType: "image/webp"},
			{URL: "https://example.com/image.gif", ContentType: "image/gif"},
			{URL: "https://example.com/image.svg", ContentType: "image/svg+xml"},
			{URL: "https://example.com/image.bmp", ContentType: "image/bmp"},
		}

		parts, err := AttachmentsToParts(imageFormats)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(parts) != 4 {
			t.Fatalf("expected 4 parts, got %d", len(parts))
		}

		for i, format := range imageFormats {
			if parts[i].Type != "image" {
				t.Errorf("part %d: expected type image, got %s", i, parts[i].Type)
			}
			if parts[i].Data != format.URL {
				t.Errorf("part %d: expected data %s, got %s", i, format.URL, parts[i].Data)
			}
			if parts[i].MimeType != format.ContentType {
				t.Errorf("part %d: expected mimeType %s, got %s", i, format.ContentType, parts[i].MimeType)
			}
		}
	})

	t.Run("should handle URLs with special characters", func(t *testing.T) {
		attachments := []Attachment{
			{URL: "https://example.com/images/my%20image%20(1).png", ContentType: "image/png"},
		}

		parts, err := AttachmentsToParts(attachments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(parts) != 1 {
			t.Fatalf("expected 1 part, got %d", len(parts))
		}
		// The URL should be preserved (Go's url.Parse may normalize it)
		if !strings.Contains(parts[0].Data, "example.com") {
			t.Errorf("expected URL to contain example.com, got %s", parts[0].Data)
		}
	})

	t.Run("should not convert URLs to data URIs", func(t *testing.T) {
		attachments := []Attachment{
			{URL: "https://example.com/image.png", ContentType: "image/png"},
		}

		parts, err := AttachmentsToParts(attachments)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(parts[0].Data, "data:") {
			t.Errorf("expected URL to not contain data:, got %s", parts[0].Data)
		}
		if !strings.HasPrefix(parts[0].Data, "https://") {
			t.Errorf("expected URL to start with https://, got %s", parts[0].Data)
		}
	})
}
