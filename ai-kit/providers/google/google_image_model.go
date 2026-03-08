// Ported from: packages/google/src/google-generative-ai-image-model.ts
package google

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// GoogleImageModelConfig configures a GoogleImageModel.
type GoogleImageModelConfig struct {
	Provider   string
	BaseURL    string
	Headers    func() map[string]string
	Fetch      providerutils.FetchFunction
	GenerateID providerutils.IdGenerator

	Internal *GoogleImageModelInternal
}

// GoogleImageModelInternal contains internal configuration for testing.
type GoogleImageModelInternal struct {
	CurrentDate func() time.Time
}

// GoogleImageModelOptions contains provider-specific options for image generation.
type GoogleImageModelOptions struct {
	PersonGeneration *string `json:"personGeneration,omitempty"`
	AspectRatio      *string `json:"aspectRatio,omitempty"`
}

// GoogleImageModel implements imagemodel.ImageModel for the Google Generative AI API.
type GoogleImageModel struct {
	modelID  string
	settings GoogleImageSettings
	config   GoogleImageModelConfig
}

// NewGoogleImageModel creates a new GoogleImageModel.
func NewGoogleImageModel(modelID string, settings GoogleImageSettings, config GoogleImageModelConfig) *GoogleImageModel {
	return &GoogleImageModel{
		modelID:  modelID,
		settings: settings,
		config:   config,
	}
}

// SpecificationVersion returns the image model interface version.
func (m *GoogleImageModel) SpecificationVersion() string {
	return "v3"
}

// Provider returns the provider ID.
func (m *GoogleImageModel) Provider() string {
	return m.config.Provider
}

// ModelID returns the model ID.
func (m *GoogleImageModel) ModelID() string {
	return m.modelID
}

// MaxImagesPerCall returns the maximum images per call.
func (m *GoogleImageModel) MaxImagesPerCall() (*int, error) {
	if m.settings.MaxImagesPerCall != nil {
		return m.settings.MaxImagesPerCall, nil
	}
	var v int
	if isGeminiModel(m.modelID) {
		v = 10
	} else {
		v = 4
	}
	return &v, nil
}

// DoGenerate generates images.
func (m *GoogleImageModel) DoGenerate(options imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
	if isGeminiModel(m.modelID) {
		return m.doGenerateGemini(options)
	}
	return m.doGenerateImagen(options)
}

func (m *GoogleImageModel) doGenerateImagen(options imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
	var warnings []shared.Warning

	// Imagen API endpoints do not support image editing
	if len(options.Files) > 0 {
		return imagemodel.GenerateResult{}, fmt.Errorf(
			"Google Generative AI does not support image editing with Imagen models. " +
				"Use Google Vertex AI (@ai-sdk/google-vertex) for image editing capabilities.",
		)
	}

	if options.Mask != nil {
		return imagemodel.GenerateResult{}, fmt.Errorf(
			"Google Generative AI does not support image editing with masks. " +
				"Use Google Vertex AI (@ai-sdk/google-vertex) for image editing capabilities.",
		)
	}

	if options.Size != nil {
		warnings = append(warnings, shared.UnsupportedWarning{
			Feature: "size",
			Details: strPtr("This model does not support the `size` option. Use `aspectRatio` instead."),
		})
	}

	if options.Seed != nil {
		warnings = append(warnings, shared.UnsupportedWarning{
			Feature: "seed",
			Details: strPtr("This model does not support the `seed` option through this provider."),
		})
	}

	// Parse provider options
	googleOptions, err := providerutils.ParseProviderOptions(
		"google",
		toInterfaceMap(options.ProviderOptions),
		GoogleImageModelOptionsSchema,
	)
	if err != nil {
		return imagemodel.GenerateResult{}, err
	}

	currentDate := m.getCurrentDate()

	n := options.N
	if n == 0 {
		n = 1
	}

	parameters := map[string]any{
		"sampleCount": n,
	}

	aspectRatio := "1:1"
	if options.AspectRatio != nil {
		aspectRatio = *options.AspectRatio
	}
	parameters["aspectRatio"] = aspectRatio

	if googleOptions != nil {
		if googleOptions.PersonGeneration != nil {
			parameters["personGeneration"] = *googleOptions.PersonGeneration
		}
		if googleOptions.AspectRatio != nil {
			parameters["aspectRatio"] = *googleOptions.AspectRatio
		}
	}

	prompt := ""
	if options.Prompt != nil {
		prompt = *options.Prompt
	}

	body := map[string]any{
		"instances": []map[string]any{
			{"prompt": prompt},
		},
		"parameters": parameters,
	}

	headers := m.config.Headers()
	mergedHeaders := providerutils.CombineHeaders(headers, convertOptionalHeaders(options.Headers))

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[googleImageResponse]{
		URL:                       fmt.Sprintf("%s/models/%s:predict", m.config.BaseURL, m.modelID),
		Headers:                   mergedHeaders,
		Body:                      body,
		FailedResponseHandler:     GoogleFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler[googleImageResponse](nil),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return imagemodel.GenerateResult{}, err
	}

	images := make([]string, len(result.Value.Predictions))
	for i, p := range result.Value.Predictions {
		images[i] = p.BytesBase64Encoded
	}

	imagesMetadata := make([]any, len(result.Value.Predictions))
	for i := range result.Value.Predictions {
		imagesMetadata[i] = map[string]any{}
		_ = i
	}

	return imagemodel.GenerateResult{
		Images: imagemodel.ImageDataStrings{Values: images},
		Warnings: warnings,
		ProviderMetadata: imagemodel.ProviderMetadata{
			"google": {
				Images: imagesMetadata,
			},
		},
		Response: imagemodel.GenerateResultResponse{
			Timestamp: currentDate,
			ModelID:   m.modelID,
			Headers:   result.ResponseHeaders,
		},
	}, nil
}

func (m *GoogleImageModel) doGenerateGemini(options imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
	var warnings []shared.Warning

	// Gemini does not support mask-based inpainting
	if options.Mask != nil {
		return imagemodel.GenerateResult{}, fmt.Errorf(
			"Gemini image models do not support mask-based image editing.",
		)
	}

	// Gemini does not support generating multiple images per call
	if options.N > 1 {
		return imagemodel.GenerateResult{}, fmt.Errorf(
			"Gemini image models do not support generating a set number of images per call. Use n=1 or omit the n parameter.",
		)
	}

	if options.Size != nil {
		warnings = append(warnings, shared.UnsupportedWarning{
			Feature: "size",
			Details: strPtr("This model does not support the `size` option. Use `aspectRatio` instead."),
		})
	}

	// Build user message content for language model
	var userContent []languagemodel.UserMessagePart

	if options.Prompt != nil {
		userContent = append(userContent, languagemodel.TextPart{
			Text: *options.Prompt,
		})
	}

	// Add input images for editing
	for _, file := range options.Files {
		switch f := file.(type) {
		case imagemodel.FileURL:
			userContent = append(userContent, languagemodel.FilePart{
				Data:      languagemodel.DataContentString{Value: f.URL},
				MediaType: "image/*",
			})
		case imagemodel.FileData:
			switch d := f.Data.(type) {
			case imagemodel.ImageFileDataString:
				userContent = append(userContent, languagemodel.FilePart{
					Data:      languagemodel.DataContentString{Value: d.Value},
					MediaType: f.MediaType,
				})
			case imagemodel.ImageFileDataBytes:
				userContent = append(userContent, languagemodel.FilePart{
					Data:      languagemodel.DataContentBytes{Data: d.Data},
					MediaType: f.MediaType,
				})
			}
		}
	}

	languageModelPrompt := languagemodel.Prompt{
		languagemodel.UserMessage{Content: userContent},
	}

	// Instantiate language model
	genID := m.config.GenerateID
	if genID == nil {
		genID = providerutils.GenerateId
	}

	lm := NewGoogleLanguageModel(m.modelID, GoogleLanguageModelConfig{
		Provider:   m.config.Provider,
		BaseURL:    m.config.BaseURL,
		Headers:    m.config.Headers,
		Fetch:      m.config.Fetch,
		GenerateID: genID,
	})

	// Build provider options with responseModalities set to IMAGE
	googleProviderOpts := map[string]any{
		"responseModalities": []string{"IMAGE"},
	}
	if options.AspectRatio != nil {
		googleProviderOpts["imageConfig"] = map[string]any{
			"aspectRatio": *options.AspectRatio,
		}
	}

	// Merge user's google provider options (excluding responseModalities and imageConfig)
	if options.ProviderOptions != nil {
		if userGoogleMap, ok := options.ProviderOptions["google"]; ok && userGoogleMap != nil {
			for k, v := range userGoogleMap {
				if k != "responseModalities" && k != "imageConfig" {
					googleProviderOpts[k] = v
				}
			}
		}
	}

	lmResult, err := lm.DoGenerate(languagemodel.CallOptions{
		Prompt: languageModelPrompt,
		Seed:   options.Seed,
		ProviderOptions: shared.ProviderOptions{
			"google": googleProviderOpts,
		},
		Headers: options.Headers,
		Ctx:     options.Ctx,
	})
	if err != nil {
		return imagemodel.GenerateResult{}, err
	}

	currentDate := m.getCurrentDate()

	// Extract images from language model response
	var images []string
	for _, part := range lmResult.Content {
		if filePart, ok := part.(languagemodel.File); ok {
			if strings.HasPrefix(filePart.MediaType, "image/") {
				if ds, ok := filePart.Data.(languagemodel.FileDataString); ok {
					images = append(images, ds.Value)
				} else if db, ok := filePart.Data.(languagemodel.FileDataBytes); ok {
					images = append(images, providerutils.ConvertBytesToBase64(db.Data))
				}
			}
		}
	}

	imagesMetadata := make([]any, len(images))
	for i := range images {
		imagesMetadata[i] = map[string]any{}
	}

	var usage *imagemodel.Usage
	if lmResult.Usage.InputTokens.Total != nil || lmResult.Usage.OutputTokens.Total != nil {
		usage = &imagemodel.Usage{}
		if lmResult.Usage.InputTokens.Total != nil {
			usage.InputTokens = lmResult.Usage.InputTokens.Total
		}
		if lmResult.Usage.OutputTokens.Total != nil {
			usage.OutputTokens = lmResult.Usage.OutputTokens.Total
		}
		inputTotal := 0
		outputTotal := 0
		if lmResult.Usage.InputTokens.Total != nil {
			inputTotal = *lmResult.Usage.InputTokens.Total
		}
		if lmResult.Usage.OutputTokens.Total != nil {
			outputTotal = *lmResult.Usage.OutputTokens.Total
		}
		total := inputTotal + outputTotal
		usage.TotalTokens = &total
	}

	var responseHeaders map[string]string
	if lmResult.Response != nil {
		responseHeaders = lmResult.Response.Headers
	}

	return imagemodel.GenerateResult{
		Images:   imagemodel.ImageDataStrings{Values: images},
		Warnings: warnings,
		ProviderMetadata: imagemodel.ProviderMetadata{
			"google": {
				Images: imagesMetadata,
			},
		},
		Response: imagemodel.GenerateResultResponse{
			Timestamp: currentDate,
			ModelID:   m.modelID,
			Headers:   responseHeaders,
		},
		Usage: usage,
	}, nil
}

func (m *GoogleImageModel) getCurrentDate() time.Time {
	if m.config.Internal != nil && m.config.Internal.CurrentDate != nil {
		return m.config.Internal.CurrentDate()
	}
	return time.Now()
}

func isGeminiModel(modelID string) bool {
	return strings.HasPrefix(modelID, "gemini-")
}

// GoogleImageModelOptionsSchema is the providerutils.Schema for image model options.
var GoogleImageModelOptionsSchema = &providerutils.Schema[GoogleImageModelOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[GoogleImageModelOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[GoogleImageModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts GoogleImageModelOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[GoogleImageModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[GoogleImageModelOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}

// --- Response types ---

type googleImageResponse struct {
	Predictions []googleImagePrediction `json:"predictions"`
}

type googleImagePrediction struct {
	BytesBase64Encoded string `json:"bytesBase64Encoded"`
}

func strPtr(s string) *string {
	return &s
}
