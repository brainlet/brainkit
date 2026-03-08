// Ported from: packages/provider/src/provider/v3/provider-v3.ts
package provider

import (
	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
)

// Provider is the interface for language, text embedding, and image generation models.
type Provider interface {
	// SpecificationVersion returns the provider interface version. Must return "v3".
	SpecificationVersion() string

	// LanguageModel returns the language model with the given id.
	// Returns NoSuchModelError if no such model exists.
	LanguageModel(modelID string) (languagemodel.LanguageModel, error)

	// EmbeddingModel returns the text embedding model with the given id.
	// Returns NoSuchModelError if no such model exists.
	EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error)

	// ImageModel returns the image model with the given id.
	ImageModel(modelID string) (imagemodel.ImageModel, error)

	// TranscriptionModel returns the transcription model with the given id.
	// This method is optional; implementations may return an error.
	TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error)

	// SpeechModel returns the speech model with the given id.
	// This method is optional; implementations may return an error.
	SpeechModel(modelID string) (speechmodel.SpeechModel, error)

	// RerankingModel returns the reranking model with the given id.
	// Returns NoSuchModelError if no such model exists.
	// This method is optional; implementations may return an error.
	RerankingModel(modelID string) (rerankingmodel.RerankingModel, error)
}
