// Ported from: packages/provider-utils/src/is-url-supported.test.ts
package providerutils

import (
	"regexp"
	"testing"
)

func TestIsUrlSupported_NoURLsSupported(t *testing.T) {
	result := IsUrlSupported(IsUrlSupportedOptions{
		MediaType:     "text/plain",
		URL:           "https://example.com",
		SupportedUrls: map[string][]*regexp.Regexp{},
	})
	if result {
		t.Error("expected false when no URLs supported")
	}
}

func TestIsUrlSupported_ExactMediaTypeAndURL(t *testing.T) {
	result := IsUrlSupported(IsUrlSupportedOptions{
		MediaType: "text/plain",
		URL:       "https://example.com",
		SupportedUrls: map[string][]*regexp.Regexp{
			"text/plain": {regexp.MustCompile(`https://example\.com`)},
		},
	})
	if !result {
		t.Error("expected true for exact media type and URL match")
	}
}

func TestIsUrlSupported_ExactMediaTypeRegexURL(t *testing.T) {
	result := IsUrlSupported(IsUrlSupportedOptions{
		MediaType: "image/png",
		URL:       "https://images.example.com/cat.png",
		SupportedUrls: map[string][]*regexp.Regexp{
			"image/png": {regexp.MustCompile(`https://images\.example\.com/.+`)},
		},
	})
	if !result {
		t.Error("expected true for exact media type and regex URL match")
	}
}

func TestIsUrlSupported_MediaTypeMismatch(t *testing.T) {
	result := IsUrlSupported(IsUrlSupportedOptions{
		MediaType: "image/png",
		URL:       "https://example.com",
		SupportedUrls: map[string][]*regexp.Regexp{
			"text/plain": {regexp.MustCompile(`https://example\.com`)},
		},
	})
	if result {
		t.Error("expected false for media type mismatch")
	}
}

func TestIsUrlSupported_WildcardMediaType(t *testing.T) {
	result := IsUrlSupported(IsUrlSupportedOptions{
		MediaType: "text/plain",
		URL:       "https://example.com",
		SupportedUrls: map[string][]*regexp.Regexp{
			"*": {regexp.MustCompile(`https://example\.com`)},
		},
	})
	if !result {
		t.Error("expected true for wildcard media type match")
	}
}

func TestIsUrlSupported_WildcardSubtype(t *testing.T) {
	result := IsUrlSupported(IsUrlSupportedOptions{
		MediaType: "image/png",
		URL:       "https://example.com",
		SupportedUrls: map[string][]*regexp.Regexp{
			"image/*": {regexp.MustCompile(`https://example\.com`)},
		},
	})
	if !result {
		t.Error("expected true for wildcard subtype match")
	}
}

func TestIsUrlSupported_FallbackToWildcard(t *testing.T) {
	result := IsUrlSupported(IsUrlSupportedOptions{
		MediaType: "text/plain",
		URL:       "https://any.com",
		SupportedUrls: map[string][]*regexp.Regexp{
			"text/plain": {},
			"*":          {regexp.MustCompile(`https://any\.com`)},
		},
	})
	if !result {
		t.Error("expected true when specific is empty but wildcard matches")
	}
}

func TestIsUrlSupported_EmptyURLArray(t *testing.T) {
	result := IsUrlSupported(IsUrlSupportedOptions{
		MediaType: "text/plain",
		URL:       "https://example.com",
		SupportedUrls: map[string][]*regexp.Regexp{
			"text/plain": {},
		},
	})
	if result {
		t.Error("expected false for empty URL array")
	}
}
