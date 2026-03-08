// Ported from: packages/provider-utils/src/is-url-supported.ts
package providerutils

import (
	"regexp"
	"strings"
)

// IsUrlSupportedOptions are the parameters for IsUrlSupported.
type IsUrlSupportedOptions struct {
	// MediaType is the IANA media type of the URL. Case-insensitive.
	MediaType string
	// URL is the URL to check.
	URL string
	// SupportedUrls is a map where keys are case-sensitive media types (or "*"/"*/*")
	// and values are arrays of regexp patterns for URLs.
	SupportedUrls map[string][]*regexp.Regexp
}

// IsUrlSupported checks if the given URL is supported natively by the model.
func IsUrlSupported(opts IsUrlSupportedOptions) bool {
	url := strings.ToLower(opts.URL)
	mediaType := strings.ToLower(opts.MediaType)

	type entry struct {
		mediaTypePrefix string
		regexes         []*regexp.Regexp
	}

	var entries []entry
	for key, regexes := range opts.SupportedUrls {
		mt := strings.ToLower(key)
		if mt == "*" || mt == "*/*" {
			entries = append(entries, entry{mediaTypePrefix: "", regexes: regexes})
		} else {
			prefix := strings.Replace(mt, "*", "", 1)
			entries = append(entries, entry{mediaTypePrefix: prefix, regexes: regexes})
		}
	}

	for _, e := range entries {
		if !strings.HasPrefix(mediaType, e.mediaTypePrefix) {
			continue
		}
		for _, pattern := range e.regexes {
			if pattern.MatchString(url) {
				return true
			}
		}
	}

	return false
}
