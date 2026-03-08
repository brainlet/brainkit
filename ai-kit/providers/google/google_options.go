// Ported from: packages/google/src/google-generative-ai-options.ts
package google

import (
	"encoding/json"

	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// GoogleLanguageModelOptions contains the provider-specific options for Google
// Generative AI language models.
type GoogleLanguageModelOptions struct {
	// ResponseModalities specifies the modalities of the response (e.g. TEXT, IMAGE).
	ResponseModalities []string `json:"responseModalities,omitempty"`

	// ThinkingConfig configures the model's thinking behavior.
	ThinkingConfig *ThinkingConfig `json:"thinkingConfig,omitempty"`

	// CachedContent is the name of the cached content used as context.
	// Format: cachedContents/{cachedContent}
	CachedContent *string `json:"cachedContent,omitempty"`

	// StructuredOutputs enables/disables structured output. Default is true.
	StructuredOutputs *bool `json:"structuredOutputs,omitempty"`

	// SafetySettings is a list of safety settings for blocking unsafe content.
	SafetySettings []SafetySetting `json:"safetySettings,omitempty"`

	// Threshold is a global safety threshold.
	Threshold *string `json:"threshold,omitempty"`

	// AudioTimestamp enables timestamp understanding for audio-only files.
	AudioTimestamp *bool `json:"audioTimestamp,omitempty"`

	// Labels defines labels used in billing reports. Available on Vertex AI only.
	Labels map[string]string `json:"labels,omitempty"`

	// MediaResolution specifies the media resolution.
	MediaResolution *string `json:"mediaResolution,omitempty"`

	// ImageConfig configures image generation aspect ratio for Gemini models.
	ImageConfig *ImageConfig `json:"imageConfig,omitempty"`

	// RetrievalConfig provides location context for Google Maps and Google Search grounding.
	RetrievalConfig *RetrievalConfig `json:"retrievalConfig,omitempty"`
}

// ThinkingConfig configures the model's thinking behavior.
type ThinkingConfig struct {
	ThinkingBudget  *int    `json:"thinkingBudget,omitempty"`
	IncludeThoughts *bool   `json:"includeThoughts,omitempty"`
	ThinkingLevel   *string `json:"thinkingLevel,omitempty"`
}

// SafetySetting represents a safety setting for blocking unsafe content.
type SafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

// ImageConfig configures image generation.
type ImageConfig struct {
	AspectRatio *string `json:"aspectRatio,omitempty"`
	ImageSize   *string `json:"imageSize,omitempty"`
}

// RetrievalConfig provides location context for grounding.
type RetrievalConfig struct {
	LatLng *LatLng `json:"latLng,omitempty"`
}

// LatLng represents a geographic coordinate.
type LatLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// GoogleLanguageModelOptionsSchema is the providerutils.Schema used to validate
// and parse GoogleLanguageModelOptions from provider options maps.
var GoogleLanguageModelOptionsSchema = &providerutils.Schema[GoogleLanguageModelOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[GoogleLanguageModelOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[GoogleLanguageModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts GoogleLanguageModelOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[GoogleLanguageModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[GoogleLanguageModelOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}
