// Ported from: packages/huggingface/src/huggingface-config.ts
package huggingface

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// Config holds the configuration for HuggingFace API calls.
type Config struct {
	// Provider is the provider identifier string.
	Provider string

	// URL constructs the API endpoint URL from the model ID and path.
	URL func(opts URLOptions) string

	// Headers returns the headers for API calls.
	Headers func() map[string]string

	// Fetch is an optional custom fetch implementation.
	Fetch providerutils.FetchFunction

	// GenerateID is an optional ID generator function.
	GenerateID providerutils.IdGenerator
}

// URLOptions are the options passed to the URL builder function.
type URLOptions struct {
	ModelID string
	Path    string
}
