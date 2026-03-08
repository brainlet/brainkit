// Ported from: packages/groq/src/groq-config.ts
package groq

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// GroqConfig holds the configuration for Groq API models.
type GroqConfig struct {
	// Provider is the provider identifier string.
	Provider string

	// URL constructs the API URL from modelId and path.
	URL func(modelId string, path string) string

	// Headers returns the HTTP headers to include with requests.
	Headers func() map[string]string

	// Fetch is an optional custom HTTP fetch function.
	Fetch providerutils.FetchFunction

	// GenerateID is an optional custom ID generator function.
	GenerateID func() string
}
