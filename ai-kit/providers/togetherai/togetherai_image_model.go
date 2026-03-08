// Ported from: packages/togetherai/src/togetherai-image-model.ts
package togetherai

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// TogetherAIImageModelConfig holds the configuration for the Together AI image model.
type TogetherAIImageModelConfig struct {
	Provider        string
	BaseURL         string
	Headers         func() map[string]string
	Fetch           providerutils.FetchFunction
	CurrentDateFunc func() time.Time
}

// TogetherAIImageModel implements imagemodel.ImageModel for Together AI image generation.
type TogetherAIImageModel struct {
	modelID TogetherAIImageModelID
	config  TogetherAIImageModelConfig
}

// NewTogetherAIImageModel creates a new TogetherAIImageModel.
func NewTogetherAIImageModel(modelID TogetherAIImageModelID, config TogetherAIImageModelConfig) *TogetherAIImageModel {
	return &TogetherAIImageModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *TogetherAIImageModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *TogetherAIImageModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *TogetherAIImageModel) ModelID() string { return m.modelID }

// MaxImagesPerCall returns the maximum number of images per call (1).
func (m *TogetherAIImageModel) MaxImagesPerCall() (*int, error) {
	v := 1
	return &v, nil
}

// TogetherAIImageModelOptions contains provider-specific options for Together AI image generation.
type TogetherAIImageModelOptions struct {
	// Steps is the number of generation steps. Higher values can improve quality.
	Steps *int `json:"steps,omitempty"`

	// Guidance is the guidance scale for image generation.
	Guidance *float64 `json:"guidance,omitempty"`

	// NegativePrompt is a negative prompt to guide what to avoid.
	NegativePrompt *string `json:"negative_prompt,omitempty"`

	// DisableSafetyChecker disables the safety checker for image generation.
	// When true, the API will not reject images flagged as potentially NSFW.
	// Not available for Flux Schnell Free and Flux Pro models.
	DisableSafetyChecker *bool `json:"disable_safety_checker,omitempty"`
}

// togetheraiImageModelOptionsSchema validates TogetherAIImageModelOptions.
var togetheraiImageModelOptionsSchema = &providerutils.Schema[TogetherAIImageModelOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[TogetherAIImageModelOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[TogetherAIImageModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts TogetherAIImageModelOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[TogetherAIImageModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[TogetherAIImageModelOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}

// DoGenerate generates images using the Together AI image generation API.
func (m *TogetherAIImageModel) DoGenerate(options imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
	var warnings []shared.Warning

	if options.Mask != nil {
		return imagemodel.GenerateResult{}, fmt.Errorf(
			"Together AI does not support mask-based image editing. " +
				"Use FLUX Kontext models (e.g., black-forest-labs/FLUX.1-kontext-pro) " +
				"with a reference image and descriptive prompt instead.",
		)
	}

	if options.Size != nil {
		details := "This model does not support the `aspectRatio` option. Use `size` instead."
		warnings = append(warnings, shared.UnsupportedWarning{
			Feature: "aspectRatio",
			Details: &details,
		})
	}

	currentDate := time.Now()
	if m.config.CurrentDateFunc != nil {
		currentDate = m.config.CurrentDateFunc()
	}

	// Parse provider-specific options
	providerOptsMap := toInterfaceMap(options.ProviderOptions)
	togetheraiOptions, err := providerutils.ParseProviderOptions(
		"togetherai",
		providerOptsMap,
		togetheraiImageModelOptionsSchema,
	)
	if err != nil {
		return imagemodel.GenerateResult{}, err
	}

	// Handle image input from files
	var imageURL *string
	if len(options.Files) > 0 {
		dataURI := providerutils.ConvertImageModelFileToDataUri(providerutils.ImageModelFile{
			Type:      fileType(options.Files[0]),
			URL:       fileURL(options.Files[0]),
			MediaType: fileMediaType(options.Files[0]),
			Data:      fileData(options.Files[0]),
		})
		imageURL = &dataURI

		if len(options.Files) > 1 {
			warnings = append(warnings, shared.OtherWarning{
				Message: "Together AI only supports a single input image. Additional images are ignored.",
			})
		}
	}

	// Build request body
	// https://docs.together.ai/reference/post_images-generations
	body := map[string]interface{}{
		"model":           m.modelID,
		"response_format": "base64",
	}
	if options.Prompt != nil {
		body["prompt"] = *options.Prompt
	}
	if options.Seed != nil {
		body["seed"] = *options.Seed
	}
	if options.N > 1 {
		body["n"] = options.N
	}

	if options.Size != nil {
		splitSize := strings.SplitN(*options.Size, "x", 2)
		if len(splitSize) == 2 {
			if w, err := strconv.Atoi(splitSize[0]); err == nil {
				body["width"] = w
			}
			if h, err := strconv.Atoi(splitSize[1]); err == nil {
				body["height"] = h
			}
		}
	}

	if imageURL != nil {
		body["image_url"] = *imageURL
	}

	// Merge provider-specific options
	if togetheraiOptions != nil {
		if togetheraiOptions.Steps != nil {
			body["steps"] = *togetheraiOptions.Steps
		}
		if togetheraiOptions.Guidance != nil {
			body["guidance"] = *togetheraiOptions.Guidance
		}
		if togetheraiOptions.NegativePrompt != nil {
			body["negative_prompt"] = *togetheraiOptions.NegativePrompt
		}
		if togetheraiOptions.DisableSafetyChecker != nil {
			body["disable_safety_checker"] = *togetheraiOptions.DisableSafetyChecker
		}
	}

	// Build the failed response handler
	failedResponseHandler := providerutils.CreateJsonErrorResponseHandler(
		togetheraiErrorSchema,
		func(data togetheraiErrorData) string {
			return data.Error.Message
		},
		nil,
	)
	wrappedFailedHandler := providerutils.ResponseHandler[error](func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
		result, handlerErr := failedResponseHandler(opts)
		if handlerErr != nil {
			return nil, handlerErr
		}
		if result == nil {
			return nil, nil
		}
		return &providerutils.ResponseHandlerResult[error]{
			Value:           result.Value,
			RawValue:        result.RawValue,
			ResponseHeaders: result.ResponseHeaders,
		}, nil
	})

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[togetheraiImageResponse]{
		URL:                       fmt.Sprintf("%s/images/generations", m.config.BaseURL),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertHeadersPtrMap(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     wrappedFailedHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(togetheraiImageResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return imagemodel.GenerateResult{}, err
	}

	images := make([]string, len(result.Value.Data))
	for i, item := range result.Value.Data {
		images[i] = item.B64JSON
	}

	return imagemodel.GenerateResult{
		Images:   imagemodel.ImageDataStrings{Values: images},
		Warnings: warnings,
		Response: imagemodel.GenerateResultResponse{
			Timestamp: currentDate,
			ModelID:   m.modelID,
			Headers:   result.ResponseHeaders,
		},
	}, nil
}

// --- Response schemas ---

type togetheraiImageResponseData struct {
	B64JSON string `json:"b64_json"`
}

type togetheraiImageResponse struct {
	Data []togetheraiImageResponseData `json:"data"`
}

var togetheraiImageResponseSchema = &providerutils.Schema[togetheraiImageResponse]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[togetheraiImageResponse], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[togetheraiImageResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		var resp togetheraiImageResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return &providerutils.ValidationResult[togetheraiImageResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[togetheraiImageResponse]{
			Success: true,
			Value:   resp,
		}, nil
	},
}

// --- File helper functions ---

// fileType returns the type string for an imagemodel.File.
func fileType(f imagemodel.File) string {
	switch f.(type) {
	case imagemodel.FileURL:
		return "url"
	case imagemodel.FileData:
		return "data"
	default:
		return ""
	}
}

// fileURL returns the URL from a FileURL or empty string.
func fileURL(f imagemodel.File) string {
	if fu, ok := f.(imagemodel.FileURL); ok {
		return fu.URL
	}
	return ""
}

// fileMediaType returns the media type from a FileData or empty string.
func fileMediaType(f imagemodel.File) string {
	if fd, ok := f.(imagemodel.FileData); ok {
		return fd.MediaType
	}
	return ""
}

// fileData returns the data from a FileData or nil.
func fileData(f imagemodel.File) interface{} {
	if fd, ok := f.(imagemodel.FileData); ok {
		switch d := fd.Data.(type) {
		case imagemodel.ImageFileDataString:
			return d.Value
		case imagemodel.ImageFileDataBytes:
			return d.Data
		}
	}
	return nil
}

// convertHeadersPtrMap converts map[string]*string to map[string]string,
// dropping nil values.
func convertHeadersPtrMap(headers map[string]*string) map[string]string {
	if headers == nil {
		return nil
	}
	result := make(map[string]string, len(headers))
	for k, v := range headers {
		if v != nil {
			result[k] = *v
		}
	}
	return result
}
