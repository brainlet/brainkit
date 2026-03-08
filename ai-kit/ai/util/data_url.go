// Ported from: packages/ai/src/util/data-url.ts
package util

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// GetTextFromDataURL converts a data URL of type text/* to a text string.
func GetTextFromDataURL(dataURL string) (string, error) {
	parts := strings.SplitN(dataURL, ",", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid data URL format")
	}

	header := parts[0]
	base64Content := parts[1]

	headerParts := strings.SplitN(header, ";", 2)
	if len(headerParts) == 0 {
		return "", fmt.Errorf("invalid data URL format")
	}

	schemeParts := strings.SplitN(headerParts[0], ":", 2)
	if len(schemeParts) < 2 || schemeParts[1] == "" {
		return "", fmt.Errorf("invalid data URL format")
	}

	decoded, err := base64.StdEncoding.DecodeString(base64Content)
	if err != nil {
		return "", fmt.Errorf("error decoding data URL")
	}

	return string(decoded), nil
}
