// Ported from: packages/openai/src/openai-config.ts
package openai

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// OpenAIConfig holds the configuration for the OpenAI provider.
type OpenAIConfig struct {
	// Provider is the provider identifier (e.g. "openai").
	Provider string

	// URL constructs the API URL from model ID and path.
	URL func(options struct {
		ModelID string
		Path    string
	}) string

	// Headers returns the HTTP headers to include with requests.
	Headers func() map[string]string

	// Fetch is an optional custom HTTP fetch function.
	Fetch providerutils.FetchFunction

	// GenerateID is an optional custom ID generator.
	GenerateID func() string

	// FileIDPrefixes are file ID prefixes used to identify file IDs in Responses API.
	// When nil, all file data is treated as base64 content.
	FileIDPrefixes []string
}
