// Ported from: packages/openai/src/image/openai-image-options.ts
package openai

import "strings"

// OpenAIImageModelID is the identifier for an OpenAI image model.
type OpenAIImageModelID = string

// ModelMaxImagesPerCall maps known OpenAI image model IDs to their
// maximum number of images per call.
// https://platform.openai.com/docs/guides/images
var ModelMaxImagesPerCall = map[string]int{
	"dall-e-3":             1,
	"dall-e-2":             10,
	"gpt-image-1":         10,
	"gpt-image-1-mini":    10,
	"gpt-image-1.5":       10,
	"chatgpt-image-latest": 10,
}

// defaultResponseFormatPrefixes are model ID prefixes that have a default
// response format (and therefore do not need response_format: "b64_json").
var defaultResponseFormatPrefixes = []string{
	"chatgpt-image-",
	"gpt-image-1-mini",
	"gpt-image-1.5",
	"gpt-image-1",
}

// HasDefaultResponseFormat returns true if the given model ID has a default
// response format and does not need an explicit response_format parameter.
func HasDefaultResponseFormat(modelID string) bool {
	for _, prefix := range defaultResponseFormatPrefixes {
		if strings.HasPrefix(modelID, prefix) {
			return true
		}
	}
	return false
}
