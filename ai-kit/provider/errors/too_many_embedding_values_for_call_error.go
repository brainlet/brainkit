// Ported from: packages/provider/src/errors/too-many-embedding-values-for-call-error.ts
package errors

import "fmt"

// TooManyEmbeddingValuesForCallError indicates too many values were provided
// for a single embedding call.
type TooManyEmbeddingValuesForCallError struct {
	AISDKError

	// Provider is the name of the provider.
	Provider string

	// ModelID is the model identifier.
	ModelID string

	// MaxEmbeddingsPerCall is the maximum allowed embeddings per call.
	MaxEmbeddingsPerCall int

	// Values are the values that were provided.
	Values []any
}

// NewTooManyEmbeddingValuesForCallError creates a new TooManyEmbeddingValuesForCallError.
func NewTooManyEmbeddingValuesForCallError(provider, modelID string, maxEmbeddingsPerCall int, values []any) *TooManyEmbeddingValuesForCallError {
	return &TooManyEmbeddingValuesForCallError{
		AISDKError: AISDKError{
			Name: "AI_TooManyEmbeddingValuesForCallError",
			Message: fmt.Sprintf(
				"Too many values for a single embedding call. "+
					"The %s model %q can only embed up to %d values per call, but %d values were provided.",
				provider, modelID, maxEmbeddingsPerCall, len(values)),
		},
		Provider:             provider,
		ModelID:              modelID,
		MaxEmbeddingsPerCall: maxEmbeddingsPerCall,
		Values:               values,
	}
}

// IsTooManyEmbeddingValuesForCallError checks if an error is a TooManyEmbeddingValuesForCallError.
func IsTooManyEmbeddingValuesForCallError(err error) bool {
	var target *TooManyEmbeddingValuesForCallError
	return As(err, &target)
}
