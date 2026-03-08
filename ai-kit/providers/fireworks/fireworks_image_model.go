// Ported from: packages/fireworks/src/fireworks-image-model.ts
package fireworks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

const (
	defaultPollIntervalMillis = 500
	defaultPollTimeoutMillis  = 120000 // 2 minutes for image generation
)

// imageModelBackendConfig describes the backend configuration for a specific image model.
type imageModelBackendConfig struct {
	urlFormat       string // "workflows", "workflows_async", or "image_generation"
	supportsSize    bool
	supportsEditing bool
}

// modelToBackendConfig maps known image model IDs to their backend configurations.
var modelToBackendConfig = map[FireworksImageModelID]*imageModelBackendConfig{
	"accounts/fireworks/models/flux-1-dev-fp8": {
		urlFormat: "workflows",
	},
	"accounts/fireworks/models/flux-1-schnell-fp8": {
		urlFormat: "workflows",
	},
	"accounts/fireworks/models/flux-kontext-pro": {
		urlFormat:       "workflows_async",
		supportsEditing: true,
	},
	"accounts/fireworks/models/flux-kontext-max": {
		urlFormat:       "workflows_async",
		supportsEditing: true,
	},
	"accounts/fireworks/models/playground-v2-5-1024px-aesthetic": {
		urlFormat:    "image_generation",
		supportsSize: true,
	},
	"accounts/fireworks/models/japanese-stable-diffusion-xl": {
		urlFormat:    "image_generation",
		supportsSize: true,
	},
	"accounts/fireworks/models/playground-v2-1024px-aesthetic": {
		urlFormat:    "image_generation",
		supportsSize: true,
	},
	"accounts/fireworks/models/stable-diffusion-xl-1024-v1-0": {
		urlFormat:    "image_generation",
		supportsSize: true,
	},
	"accounts/fireworks/models/SSD-1B": {
		urlFormat:    "image_generation",
		supportsSize: true,
	},
}

// getURLForModel returns the API URL for a given model.
func getURLForModel(baseURL string, modelID FireworksImageModelID) string {
	config := modelToBackendConfig[modelID]

	if config != nil {
		switch config.urlFormat {
		case "image_generation":
			return fmt.Sprintf("%s/image_generation/%s", baseURL, modelID)
		case "workflows_async":
			return fmt.Sprintf("%s/workflows/%s", baseURL, modelID)
		}
	}

	// Default: "workflows" or unknown
	return fmt.Sprintf("%s/workflows/%s/text_to_image", baseURL, modelID)
}

// getPollURLForModel returns the poll URL for an async model.
func getPollURLForModel(baseURL string, modelID FireworksImageModelID) string {
	return fmt.Sprintf("%s/workflows/%s/get_result", baseURL, modelID)
}

// FireworksImageModelConfig holds the configuration for a FireworksImageModel.
type FireworksImageModelConfig struct {
	Provider string
	BaseURL  string
	Headers  func() map[string]string
	Fetch    providerutils.FetchFunction

	// PollIntervalMillis is the poll interval in milliseconds between status checks
	// for async models. Defaults to 500ms.
	PollIntervalMillis *int

	// PollTimeoutMillis is the overall timeout in milliseconds for polling before
	// giving up. Defaults to 120000ms (2 minutes).
	PollTimeoutMillis *int

	// CurrentDateFunc is an optional function to get the current date.
	// Used for testing; defaults to time.Now.
	CurrentDateFunc func() time.Time
}

// FireworksImageModel implements imagemodel.ImageModel for Fireworks image generation.
type FireworksImageModel struct {
	modelID FireworksImageModelID
	config  FireworksImageModelConfig
}

// NewFireworksImageModel creates a new FireworksImageModel.
func NewFireworksImageModel(modelID FireworksImageModelID, config FireworksImageModelConfig) *FireworksImageModel {
	return &FireworksImageModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *FireworksImageModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *FireworksImageModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *FireworksImageModel) ModelID() string { return m.modelID }

// MaxImagesPerCall returns 1 (Fireworks generates one image per call).
func (m *FireworksImageModel) MaxImagesPerCall() (*int, error) {
	v := 1
	return &v, nil
}

// convertFileToDataURI converts an imagemodel.File to a data URI string.
func convertFileToDataURI(file imagemodel.File) string {
	switch f := file.(type) {
	case imagemodel.FileURL:
		return f.URL
	case imagemodel.FileData:
		switch d := f.Data.(type) {
		case imagemodel.ImageFileDataString:
			return fmt.Sprintf("data:%s;base64,%s", f.MediaType, d.Value)
		case imagemodel.ImageFileDataBytes:
			return fmt.Sprintf("data:%s;base64,%s", f.MediaType, providerutils.ConvertBytesToBase64(d.Data))
		default:
			return ""
		}
	default:
		return ""
	}
}

// DoGenerate generates images using the Fireworks API.
func (m *FireworksImageModel) DoGenerate(options imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
	var warnings []shared.Warning

	backendConfig := modelToBackendConfig[m.modelID]

	if backendConfig == nil || !backendConfig.supportsSize {
		if options.Size != nil {
			details := "This model does not support the `size` option. Use `aspectRatio` instead."
			warnings = append(warnings, shared.UnsupportedWarning{
				Feature: "size",
				Details: &details,
			})
		}
	}

	// Use supportsSize as a proxy for whether the model does not support
	// aspectRatio. This invariant holds for the current set of models.
	if backendConfig != nil && backendConfig.supportsSize {
		if options.AspectRatio != nil {
			details := "This model does not support the `aspectRatio` option."
			warnings = append(warnings, shared.UnsupportedWarning{
				Feature: "aspectRatio",
				Details: &details,
			})
		}
	}

	// Handle files for image editing
	hasInputImage := len(options.Files) > 0
	var inputImage string

	if hasInputImage {
		inputImage = convertFileToDataURI(options.Files[0])

		if len(options.Files) > 1 {
			warnings = append(warnings, shared.OtherWarning{
				Message: "Fireworks only supports a single input image. Additional images are ignored.",
			})
		}
	}

	// Warn about mask - Fireworks Kontext models don't support explicit masks
	if options.Mask != nil {
		details := "Fireworks Kontext models do not support explicit masks. Use the prompt to describe the areas to edit."
		warnings = append(warnings, shared.UnsupportedWarning{
			Feature: "mask",
			Details: &details,
		})
	}

	currentDate := time.Now()
	if m.config.CurrentDateFunc != nil {
		currentDate = m.config.CurrentDateFunc()
	}

	combinedHeaders := providerutils.CombineHeaders(m.config.Headers(), convertHeadersPtrMap(options.Headers))

	body := map[string]interface{}{}
	if options.Prompt != nil {
		body["prompt"] = *options.Prompt
	}
	if options.AspectRatio != nil {
		body["aspect_ratio"] = *options.AspectRatio
	}
	if options.Seed != nil {
		body["seed"] = *options.Seed
	}
	body["samples"] = options.N

	if inputImage != "" {
		body["input_image"] = inputImage
	}

	if options.Size != nil {
		splitSize := strings.Split(*options.Size, "x")
		if len(splitSize) == 2 {
			body["width"] = splitSize[0]
			body["height"] = splitSize[1]
		}
	}

	// Merge fireworks-specific provider options
	if options.ProviderOptions != nil {
		if fireworksOpts, ok := options.ProviderOptions["fireworks"]; ok {
			for k, v := range fireworksOpts {
				body[k] = v
			}
		}
	}

	// Handle async models that require polling (e.g., flux-kontext-*)
	if backendConfig != nil && backendConfig.urlFormat == "workflows_async" {
		return m.doGenerateAsync(options.Ctx, body, combinedHeaders, warnings, currentDate)
	}

	// Handle sync models that return binary directly
	failedResponseHandler := providerutils.ResponseHandler[error](func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
		result, err := providerutils.CreateStatusCodeErrorResponseHandler()(opts)
		if err != nil {
			return nil, err
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

	response, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[[]byte]{
		URL:                       getURLForModel(m.config.BaseURL, m.modelID),
		Headers:                   combinedHeaders,
		Body:                      body,
		FailedResponseHandler:     failedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateBinaryResponseHandler(),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return imagemodel.GenerateResult{}, err
	}

	return imagemodel.GenerateResult{
		Images:   imagemodel.ImageDataBytes{Values: [][]byte{response.Value}},
		Warnings: warnings,
		Response: imagemodel.GenerateResultResponse{
			Timestamp: currentDate,
			ModelID:   m.modelID,
			Headers:   response.ResponseHeaders,
		},
	}, nil
}

// doGenerateAsync handles async image generation for models like flux-kontext-*
// that return a request_id and require polling for results.
func (m *FireworksImageModel) doGenerateAsync(
	ctx context.Context,
	body map[string]interface{},
	headers map[string]string,
	warnings []shared.Warning,
	currentDate time.Time,
) (imagemodel.GenerateResult, error) {
	failedResponseHandler := providerutils.ResponseHandler[error](func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
		result, err := providerutils.CreateStatusCodeErrorResponseHandler()(opts)
		if err != nil {
			return nil, err
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

	// Submit the generation request
	submitResponse, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[asyncSubmitResponse]{
		URL:                       getURLForModel(m.config.BaseURL, m.modelID),
		Headers:                   headers,
		Body:                      body,
		FailedResponseHandler:     failedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(asyncSubmitResponseSchema),
		Ctx:                       ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return imagemodel.GenerateResult{}, err
	}

	requestID := submitResponse.Value.RequestID

	// Poll for the result
	imageURL, err := m.pollForImageURL(ctx, requestID, headers)
	if err != nil {
		return imagemodel.GenerateResult{}, err
	}

	// Download the image from the URL
	imageResult, err := providerutils.GetFromApi(providerutils.GetFromApiOptions[[]byte]{
		URL:                       imageURL,
		Headers:                   headers,
		FailedResponseHandler:     failedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateBinaryResponseHandler(),
		Ctx:                       ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return imagemodel.GenerateResult{}, err
	}

	return imagemodel.GenerateResult{
		Images:   imagemodel.ImageDataBytes{Values: [][]byte{imageResult.Value}},
		Warnings: warnings,
		Response: imagemodel.GenerateResultResponse{
			Timestamp: currentDate,
			ModelID:   m.modelID,
			Headers:   imageResult.ResponseHeaders,
		},
	}, nil
}

// pollForImageURL polls the get_result endpoint until the image is ready.
func (m *FireworksImageModel) pollForImageURL(
	ctx context.Context,
	requestID string,
	headers map[string]string,
) (string, error) {
	pollIntervalMillis := defaultPollIntervalMillis
	if m.config.PollIntervalMillis != nil {
		pollIntervalMillis = *m.config.PollIntervalMillis
	}

	pollTimeoutMillis := defaultPollTimeoutMillis
	if m.config.PollTimeoutMillis != nil {
		pollTimeoutMillis = *m.config.PollTimeoutMillis
	}

	effectiveInterval := pollIntervalMillis
	if effectiveInterval < 1 {
		effectiveInterval = 1
	}
	maxPollAttempts := (pollTimeoutMillis + effectiveInterval - 1) / effectiveInterval

	pollURL := getPollURLForModel(m.config.BaseURL, m.modelID)

	failedResponseHandler := providerutils.ResponseHandler[error](func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
		result, err := providerutils.CreateStatusCodeErrorResponseHandler()(opts)
		if err != nil {
			return nil, err
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

	for i := 0; i < maxPollAttempts; i++ {
		pollResponse, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[asyncPollResponse]{
			URL:                       pollURL,
			Headers:                   headers,
			Body:                      map[string]interface{}{"id": requestID},
			FailedResponseHandler:     failedResponseHandler,
			SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(asyncPollResponseSchema),
			Ctx:                       ctx,
			Fetch:                     m.config.Fetch,
		})
		if err != nil {
			return "", err
		}

		status := pollResponse.Value.Status

		if status == "Ready" {
			if pollResponse.Value.Result != nil && pollResponse.Value.Result.Sample != nil {
				return *pollResponse.Value.Result.Sample, nil
			}
			return "", fmt.Errorf("Fireworks poll response is Ready but missing result.sample")
		}

		if status == "Error" || status == "Failed" {
			return "", fmt.Errorf("Fireworks image generation failed with status: %s", status)
		}

		// Wait before next poll attempt
		if err := providerutils.Delay(ctx, time.Duration(pollIntervalMillis)*time.Millisecond); err != nil {
			return "", err
		}
	}

	return "", fmt.Errorf("Fireworks image generation timed out after %dms", pollTimeoutMillis)
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
