// Ported from: packages/ai/src/embed/embed.ts
package embed

import (
	"context"
)

// EmbeddingModel is the interface for embedding models.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type EmbeddingModel interface {
	// Provider returns the provider name.
	Provider() string
	// ModelID returns the model identifier.
	ModelID() string
	// DoEmbed performs the embedding operation.
	DoEmbed(ctx context.Context, opts DoEmbedOptions) (*DoEmbedResult, error)
}

// DoEmbedOptions are the options passed to EmbeddingModel.DoEmbed.
type DoEmbedOptions struct {
	Values          []string
	Headers         map[string]string
	ProviderOptions map[string]map[string]any
}

// DoEmbedResult is the result from EmbeddingModel.DoEmbed.
type DoEmbedResult struct {
	Embeddings       []Embedding
	Usage            *EmbeddingModelUsage
	Warnings         []Warning
	ProviderMetadata ProviderMetadata
	Response         *EmbedResponseData
}

// EmbedOptions are the options for the Embed function.
type EmbedOptions struct {
	// Model is the embedding model to use.
	Model EmbeddingModel

	// Value is the value that should be embedded.
	Value string

	// MaxRetries is the maximum number of retries per embedding model call.
	// Set to 0 to disable retries. Default: 2.
	MaxRetries *int

	// Headers are additional headers to include in the request.
	// Only applicable for HTTP-based providers.
	Headers map[string]string

	// ProviderOptions are additional provider-specific options.
	ProviderOptions map[string]map[string]any
}

// Embed embeds a value using an embedding model.
// The type of the value is defined by the embedding model.
func Embed(ctx context.Context, opts EmbedOptions) (*EmbedResult, error) {
	model := opts.Model

	result, err := model.DoEmbed(ctx, DoEmbedOptions{
		Values:          []string{opts.Value},
		Headers:         opts.Headers,
		ProviderOptions: opts.ProviderOptions,
	})
	if err != nil {
		return nil, err
	}

	embedding := result.Embeddings[0]

	usage := EmbeddingModelUsage{}
	if result.Usage != nil {
		usage = *result.Usage
	}

	return &EmbedResult{
		Value:            opts.Value,
		Embedding:        embedding,
		Usage:            usage,
		Warnings:         result.Warnings,
		ProviderMetadata: result.ProviderMetadata,
		Response:         result.Response,
	}, nil
}
