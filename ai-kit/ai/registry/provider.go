// Ported from: packages/ai/src/registry/ (provider interface for registry use)
package registry

import (
	"github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/rerankingmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/speechmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/transcriptionmodel"
	"github.com/brainlet/brainkit/ai-kit/provider/videomodel"
)

// Provider is the interface for all model types used within the registry.
// This corresponds to ProviderV4 in the TypeScript Vercel AI SDK and includes
// all model types: language, embedding, image, transcription, speech, reranking,
// and video models.
//
// In the TypeScript source, some methods (transcriptionModel, speechModel,
// rerankingModel, videoModel) are optional on ProviderV4. In Go, all methods
// are required on the interface, but implementations may return a
// NoSuchModelError for unsupported model types.
type Provider interface {
	// SpecificationVersion returns the provider specification version.
	SpecificationVersion() string

	// LanguageModel returns the language model with the given id.
	LanguageModel(modelID string) (languagemodel.LanguageModel, error)

	// EmbeddingModel returns the embedding model with the given id.
	EmbeddingModel(modelID string) (embeddingmodel.EmbeddingModel, error)

	// ImageModel returns the image model with the given id.
	ImageModel(modelID string) (imagemodel.ImageModel, error)

	// TranscriptionModel returns the transcription model with the given id.
	TranscriptionModel(modelID string) (transcriptionmodel.TranscriptionModel, error)

	// SpeechModel returns the speech model with the given id.
	SpeechModel(modelID string) (speechmodel.SpeechModel, error)

	// RerankingModel returns the reranking model with the given id.
	RerankingModel(modelID string) (rerankingmodel.RerankingModel, error)

	// VideoModel returns the video model with the given id.
	VideoModel(modelID string) (videomodel.VideoModel, error)
}
