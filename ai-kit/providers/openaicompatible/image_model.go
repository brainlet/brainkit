// Ported from: packages/openai-compatible/src/image/openai-compatible-image-model.ts
package openaicompatible

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// ImageModelConfig holds the configuration for an image model.
type ImageModelConfig struct {
	// Provider is the provider identifier (e.g. "openai.image").
	Provider string

	// Headers returns the HTTP headers to send with each request.
	Headers func() map[string]string

	// URL builds the full API URL from the given path.
	URL func(path string) string

	// Fetch is an optional custom HTTP fetch function.
	Fetch providerutils.FetchFunction

	// ErrorStructure is the provider-specific error structure.
	// If nil, DefaultErrorStructure is used.
	ErrorStructure *ProviderErrorStructure[ErrorData]

	// CurrentDateFunc is an optional function to get the current date.
	// Used for testing; defaults to time.Now.
	CurrentDateFunc func() time.Time
}

// ImageModel implements imagemodel.ImageModel for OpenAI-compatible
// image generation endpoints.
type ImageModel struct {
	modelID ImageModelID
	config  ImageModelConfig
}

// NewImageModel creates a new ImageModel.
func NewImageModel(modelID string, config ImageModelConfig) *ImageModel {
	return &ImageModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *ImageModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *ImageModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *ImageModel) ModelID() string { return m.modelID }

// MaxImagesPerCall returns the max images per call (10).
func (m *ImageModel) MaxImagesPerCall() (*int, error) {
	v := 10
	return &v, nil
}

func (m *ImageModel) providerOptionsKey() string {
	return strings.TrimSpace(strings.SplitN(m.config.Provider, ".", 2)[0])
}

// getArgs extracts provider-specific arguments from providerOptions,
// merging both snake_case and camelCase keys.
func (m *ImageModel) getArgs(providerOptions shared.ProviderOptions) map[string]interface{} {
	result := make(map[string]interface{})
	key := m.providerOptionsKey()

	// TODO: deprecate non-camelCase keys and remove in future major version
	if providerOptions != nil {
		if opts, ok := providerOptions[key]; ok {
			for k, v := range opts {
				result[k] = v
			}
		}
		camelKey := toCamelCase(key)
		if camelKey != key {
			if opts, ok := providerOptions[camelKey]; ok {
				for k, v := range opts {
					result[k] = v
				}
			}
		}
	}

	return result
}

// DoGenerate generates images. Supports two modes:
// - Standard generation: POST JSON to /images/generations
// - Image editing: POST form data to /images/edits (when files are provided)
func (m *ImageModel) DoGenerate(options imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
	var warnings []shared.Warning

	if options.AspectRatio != nil {
		details := "This model does not support aspect ratio. Use `size` instead."
		warnings = append(warnings, shared.UnsupportedWarning{
			Feature: "aspectRatio",
			Details: &details,
		})
	}

	if options.Seed != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "seed"})
	}

	currentDate := time.Now()
	if m.config.CurrentDateFunc != nil {
		currentDate = m.config.CurrentDateFunc()
	}

	args := m.getArgs(options.ProviderOptions)

	errorStructure := m.config.ErrorStructure
	if errorStructure == nil {
		es := DefaultErrorStructure
		errorStructure = &es
	}

	failedResponseHandler := providerutils.CreateJsonErrorResponseHandler(
		errorStructure.ErrorSchema,
		errorStructure.ErrorToMessage,
		errorStructure.IsRetryable,
	)
	wrappedHandler := providerutils.ResponseHandler[error](func(opts providerutils.ResponseHandlerOptions) (*providerutils.ResponseHandlerResult[error], error) {
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

	// Image editing mode -- use form data and /images/edits endpoint
	if len(options.Files) > 0 {
		return m.doGenerateEdit(options, args, warnings, currentDate, wrappedHandler)
	}

	// Standard image generation mode -- use JSON and /images/generations endpoint
	return m.doGenerateStandard(options, args, warnings, currentDate, wrappedHandler)
}

func (m *ImageModel) doGenerateStandard(
	options imagemodel.CallOptions,
	args map[string]interface{},
	warnings []shared.Warning,
	currentDate time.Time,
	failedResponseHandler providerutils.ResponseHandler[error],
) (imagemodel.GenerateResult, error) {
	body := map[string]interface{}{
		"model":           m.modelID,
		"n":               options.N,
		"response_format": "b64_json",
	}
	if options.Prompt != nil {
		body["prompt"] = *options.Prompt
	}
	if options.Size != nil {
		body["size"] = *options.Size
	}
	// Merge provider-specific args
	for k, v := range args {
		body[k] = v
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[imageResponse]{
		URL:                       m.config.URL("/images/generations"),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertHeadersPtrMap(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     failedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(imageResponseSchema),
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

func (m *ImageModel) doGenerateEdit(
	options imagemodel.CallOptions,
	args map[string]interface{},
	warnings []shared.Warning,
	currentDate time.Time,
	failedResponseHandler providerutils.ResponseHandler[error],
) (imagemodel.GenerateResult, error) {
	// Convert files to byte data for form data
	var imageBlobs [][]byte
	for _, file := range options.Files {
		blob, err := imageFileToBytes(file)
		if err != nil {
			return imagemodel.GenerateResult{}, fmt.Errorf("failed to convert image file: %w", err)
		}
		imageBlobs = append(imageBlobs, blob)
	}

	formInput := map[string]interface{}{
		"model": m.modelID,
		"n":     options.N,
	}
	if options.Prompt != nil {
		formInput["prompt"] = *options.Prompt
	}
	if options.Size != nil {
		formInput["size"] = *options.Size
	}

	// Add image(s)
	if len(imageBlobs) == 1 {
		formInput["image"] = imageBlobs[0]
	} else {
		blobsAsAny := make([]interface{}, len(imageBlobs))
		for i, b := range imageBlobs {
			blobsAsAny[i] = b
		}
		formInput["image"] = blobsAsAny
	}

	// Add mask if provided
	if options.Mask != nil {
		maskBlob, err := imageFileToBytes(options.Mask)
		if err != nil {
			return imagemodel.GenerateResult{}, fmt.Errorf("failed to convert mask file: %w", err)
		}
		formInput["mask"] = maskBlob
	}

	// Merge provider-specific args
	for k, v := range args {
		formInput[k] = v
	}

	formResult, err := providerutils.ConvertToFormData(formInput, nil)
	if err != nil {
		return imagemodel.GenerateResult{}, fmt.Errorf("failed to create form data: %w", err)
	}

	// Post form data
	headers := providerutils.CombineHeaders(m.config.Headers(), convertHeadersPtrMap(options.Headers))
	headers["Content-Type"] = formResult.ContentType

	result, err := providerutils.PostToApi(providerutils.PostToApiOptions[imageResponse]{
		URL:     m.config.URL("/images/edits"),
		Headers: headers,
		Body: providerutils.PostToApiBody{
			Content: formResult.Body,
			Values:  formResult.Values,
		},
		FailedResponseHandler:     failedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(imageResponseSchema),
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

type imageResponse struct {
	Data []imageResponseData `json:"data"`
}

type imageResponseData struct {
	B64JSON string `json:"b64_json"`
}

var imageResponseSchema = &providerutils.Schema[imageResponse]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[imageResponse], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[imageResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		var resp imageResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			return &providerutils.ValidationResult[imageResponse]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[imageResponse]{
			Success: true,
			Value:   resp,
		}, nil
	},
}

// --- Helper functions ---

// imageFileToBytes converts an imagemodel.File to raw bytes.
// For URL files, it downloads the data. For data files, it decodes base64 if needed.
func imageFileToBytes(file imagemodel.File) ([]byte, error) {
	switch f := file.(type) {
	case imagemodel.FileURL:
		result, err := providerutils.DownloadBlob(f.URL, nil)
		if err != nil {
			return nil, err
		}
		return result.Data, nil
	case imagemodel.FileData:
		switch d := f.Data.(type) {
		case imagemodel.ImageFileDataBytes:
			return d.Data, nil
		case imagemodel.ImageFileDataString:
			return providerutils.ConvertBase64ToBytes(d.Value)
		default:
			return nil, fmt.Errorf("unsupported image file data type: %T", f.Data)
		}
	default:
		return nil, fmt.Errorf("unsupported image file type: %T", file)
	}
}

// toCamelCase converts a snake_case or kebab-case string to camelCase.
func toCamelCase(s string) string {
	var result strings.Builder
	capitalizeNext := false
	for i, r := range s {
		if r == '_' || r == '-' {
			capitalizeNext = true
			continue
		}
		if capitalizeNext && i > 0 {
			result.WriteRune(unicode.ToUpper(r))
			capitalizeNext = false
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
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
