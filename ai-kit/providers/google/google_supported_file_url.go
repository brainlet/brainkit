// Ported from: packages/google/src/google-supported-file-url.ts
package google

import (
	"net/url"
	"regexp"
	"strings"
)

// IsSupportedFileURL checks if the given URL is natively supported by the Google
// Generative AI provider (e.g. Google Generative Language files API or YouTube URLs).
func IsSupportedFileURL(u *url.URL) bool {
	urlString := u.String()

	// Google Generative Language files API
	if strings.HasPrefix(urlString, "https://generativelanguage.googleapis.com/v1beta/files/") {
		return true
	}

	// YouTube URLs (public or unlisted videos)
	youtubeRegexes := []*regexp.Regexp{
		regexp.MustCompile(`^https://(?:www\.)?youtube\.com/watch\?v=[\w-]+(?:&[\w=&.-]*)?$`),
		regexp.MustCompile(`^https://youtu\.be/[\w-]+(?:\?[\w=&.-]*)?$`),
	}

	for _, re := range youtubeRegexes {
		if re.MatchString(urlString) {
			return true
		}
	}

	return false
}
