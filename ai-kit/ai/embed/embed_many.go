// Ported from: packages/ai/src/embed/embed-many.ts
package embed

import (
	"context"
	"sync"
)

// EmbedManyOptions are the options for the EmbedMany function.
type EmbedManyOptions struct {
	// Model is the embedding model to use.
	Model EmbeddingModel

	// Values are the values that should be embedded.
	Values []string

	// MaxRetries is the maximum number of retries per embedding model call.
	// Set to 0 to disable retries. Default: 2.
	MaxRetries *int

	// Headers are additional headers to include in the request.
	// Only applicable for HTTP-based providers.
	Headers map[string]string

	// ProviderOptions are additional provider-specific options.
	ProviderOptions map[string]map[string]any

	// MaxParallelCalls is the maximum number of concurrent requests.
	// Default: 0 (no limit).
	MaxParallelCalls int

	// MaxEmbeddingsPerCall overrides the model's maxEmbeddingsPerCall.
	// If zero, defers to the model's limit.
	MaxEmbeddingsPerCall int
}

// EmbedMany embeds several values using an embedding model.
// It automatically splits large requests into smaller chunks if the model
// has a limit on how many embeddings can be generated in a single call.
func EmbedMany(ctx context.Context, opts EmbedManyOptions) (*EmbedManyResult, error) {
	model := opts.Model
	maxPerCall := opts.MaxEmbeddingsPerCall

	// If no per-call limit, embed everything in a single call.
	if maxPerCall <= 0 {
		result, err := model.DoEmbed(ctx, DoEmbedOptions{
			Values:          opts.Values,
			Headers:         opts.Headers,
			ProviderOptions: opts.ProviderOptions,
		})
		if err != nil {
			return nil, err
		}

		usage := EmbeddingModelUsage{}
		if result.Usage != nil {
			usage = *result.Usage
		}

		return &EmbedManyResult{
			Values:           opts.Values,
			Embeddings:       result.Embeddings,
			Usage:            usage,
			Warnings:         result.Warnings,
			ProviderMetadata: result.ProviderMetadata,
			Responses:        []*EmbedResponseData{result.Response},
		}, nil
	}

	// Split values into chunks.
	valueChunks := splitArray(opts.Values, maxPerCall)

	// Determine parallelism.
	maxParallel := opts.MaxParallelCalls
	if maxParallel <= 0 {
		maxParallel = len(valueChunks) // no limit
	}

	// Process chunks with controlled parallelism.
	var allEmbeddings []Embedding
	var allWarnings []Warning
	var allResponses []*EmbedResponseData
	var totalTokens int
	var providerMeta ProviderMetadata

	// Process in batches of maxParallel.
	parallelBatches := splitArrayGeneric(valueChunks, maxParallel)

	for _, batch := range parallelBatches {
		type chunkResult struct {
			embeddings       []Embedding
			usage            EmbeddingModelUsage
			warnings         []Warning
			providerMetadata ProviderMetadata
			response         *EmbedResponseData
			err              error
		}

		results := make([]chunkResult, len(batch))
		var wg sync.WaitGroup
		wg.Add(len(batch))

		for i, chunk := range batch {
			go func(idx int, values []string) {
				defer wg.Done()
				res, err := model.DoEmbed(ctx, DoEmbedOptions{
					Values:          values,
					Headers:         opts.Headers,
					ProviderOptions: opts.ProviderOptions,
				})
				if err != nil {
					results[idx] = chunkResult{err: err}
					return
				}
				usage := EmbeddingModelUsage{}
				if res.Usage != nil {
					usage = *res.Usage
				}
				results[idx] = chunkResult{
					embeddings:       res.Embeddings,
					usage:            usage,
					warnings:         res.Warnings,
					providerMetadata: res.ProviderMetadata,
					response:         res.Response,
				}
			}(i, chunk)
		}
		wg.Wait()

		for _, r := range results {
			if r.err != nil {
				return nil, r.err
			}
			allEmbeddings = append(allEmbeddings, r.embeddings...)
			allWarnings = append(allWarnings, r.warnings...)
			allResponses = append(allResponses, r.response)
			totalTokens += r.usage.Tokens
			if r.providerMetadata != nil {
				if providerMeta == nil {
					providerMeta = make(ProviderMetadata)
					for k, v := range r.providerMetadata {
						providerMeta[k] = v
					}
				} else {
					for providerName, metadata := range r.providerMetadata {
						existing, ok := providerMeta[providerName]
						if !ok {
							providerMeta[providerName] = metadata
						} else {
							merged := make(map[string]any)
							for k, v := range existing {
								merged[k] = v
							}
							for k, v := range metadata {
								merged[k] = v
							}
							providerMeta[providerName] = merged
						}
					}
				}
			}
		}
	}

	return &EmbedManyResult{
		Values:           opts.Values,
		Embeddings:       allEmbeddings,
		Usage:            EmbeddingModelUsage{Tokens: totalTokens},
		Warnings:         allWarnings,
		ProviderMetadata: providerMeta,
		Responses:        allResponses,
	}, nil
}

// splitArray splits a slice into chunks of at most chunkSize.
func splitArray(arr []string, chunkSize int) [][]string {
	if chunkSize <= 0 {
		return [][]string{arr}
	}
	var chunks [][]string
	for i := 0; i < len(arr); i += chunkSize {
		end := i + chunkSize
		if end > len(arr) {
			end = len(arr)
		}
		chunks = append(chunks, arr[i:end])
	}
	return chunks
}

// splitArrayGeneric splits a slice of slices into batches.
func splitArrayGeneric[T any](arr []T, batchSize int) [][]T {
	if batchSize <= 0 {
		return [][]T{arr}
	}
	var batches [][]T
	for i := 0; i < len(arr); i += batchSize {
		end := i + batchSize
		if end > len(arr) {
			end = len(arr)
		}
		batches = append(batches, arr[i:end])
	}
	return batches
}
