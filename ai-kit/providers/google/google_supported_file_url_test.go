// Ported from: packages/google/src/google-supported-file-url.test.ts
package google

import (
	"net/url"
	"testing"
)

func mustParseURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}
	return u
}

func TestIsSupportedFileURL(t *testing.T) {
	t.Run("should return true for valid Google generative language file URLs", func(t *testing.T) {
		validURLs := []string{
			"https://generativelanguage.googleapis.com/v1beta/files/00000000-00000000-00000000-00000000",
			"https://generativelanguage.googleapis.com/v1beta/files/test123",
		}
		for _, rawURL := range validURLs {
			u := mustParseURL(rawURL)
			if !IsSupportedFileURL(u) {
				t.Errorf("expected %q to be supported", rawURL)
			}
		}
	})

	t.Run("should return true for valid YouTube URLs", func(t *testing.T) {
		validYouTubeURLs := []string{
			"https://www.youtube.com/watch?v=dQw4w9WgXcQ",
			"https://youtube.com/watch?v=dQw4w9WgXcQ",
			"https://youtu.be/dQw4w9WgXcQ",
			"https://www.youtube.com/watch?v=dQw4w9WgXcQ&feature=youtu.be",
			"https://youtu.be/dQw4w9WgXcQ?t=42",
		}
		for _, rawURL := range validYouTubeURLs {
			u := mustParseURL(rawURL)
			if !IsSupportedFileURL(u) {
				t.Errorf("expected %q to be supported", rawURL)
			}
		}
	})

	t.Run("should return false for invalid YouTube URLs", func(t *testing.T) {
		invalidYouTubeURLs := []string{
			"https://youtube.com/channel/UCdQw4w9WgXcQ",
			"https://youtube.com/playlist?list=PLdQw4w9WgXcQ",
			"https://m.youtube.com/watch?v=dQw4w9WgXcQ",
			"http://youtube.com/watch?v=dQw4w9WgXcQ",
			"https://vimeo.com/123456789",
		}
		for _, rawURL := range invalidYouTubeURLs {
			u := mustParseURL(rawURL)
			if IsSupportedFileURL(u) {
				t.Errorf("expected %q to NOT be supported", rawURL)
			}
		}
	})

	t.Run("should return false for non-Google generative language file URLs", func(t *testing.T) {
		testCases := []string{
			"https://example.com",
			"https://example.com/foo/bar",
			"https://generativelanguage.googleapis.com",
			"https://generativelanguage.googleapis.com/v1/other",
			"http://generativelanguage.googleapis.com/v1beta/files/test",
			"https://api.googleapis.com/v1beta/files/test",
		}
		for _, rawURL := range testCases {
			u := mustParseURL(rawURL)
			if IsSupportedFileURL(u) {
				t.Errorf("expected %q to NOT be supported", rawURL)
			}
		}
	})
}
