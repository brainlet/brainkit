// Ported from: packages/provider/src/errors/no-such-model-error.ts
package errors

import "fmt"

// ModelType represents the type of model that was not found.
type ModelType string

const (
	ModelTypeLanguage      ModelType = "languageModel"
	ModelTypeEmbedding     ModelType = "embeddingModel"
	ModelTypeImage         ModelType = "imageModel"
	ModelTypeTranscription ModelType = "transcriptionModel"
	ModelTypeSpeech        ModelType = "speechModel"
	ModelTypeReranking     ModelType = "rerankingModel"
	ModelTypeVideo         ModelType = "videoModel"
)

// NoSuchModelError indicates that the requested model does not exist.
type NoSuchModelError struct {
	AISDKError

	// ModelID is the ID of the model that was not found.
	ModelID string

	// ModelType is the type of the model that was not found.
	ModelType ModelType
}

// NoSuchModelErrorOptions are the options for creating a NoSuchModelError.
type NoSuchModelErrorOptions struct {
	ErrorName string
	ModelID   string
	ModelType ModelType
	Message   string
}

// NewNoSuchModelError creates a new NoSuchModelError.
func NewNoSuchModelError(opts NoSuchModelErrorOptions) *NoSuchModelError {
	errorName := opts.ErrorName
	if errorName == "" {
		errorName = "AI_NoSuchModelError"
	}
	message := opts.Message
	if message == "" {
		message = fmt.Sprintf("No such %s: %s", opts.ModelType, opts.ModelID)
	}
	return &NoSuchModelError{
		AISDKError: AISDKError{
			Name:    errorName,
			Message: message,
		},
		ModelID:   opts.ModelID,
		ModelType: opts.ModelType,
	}
}

// IsNoSuchModelError checks if an error is a NoSuchModelError.
func IsNoSuchModelError(err error) bool {
	var target *NoSuchModelError
	return As(err, &target)
}
