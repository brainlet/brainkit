// Ported from: packages/openai/src/image/openai-image-model.ts
package openai

import (
	"fmt"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// OpenAIImageModelConfig extends OpenAIConfig with internal options.
type OpenAIImageModelConfig struct {
	OpenAIConfig

	// Internal contains optional internal configuration.
	Internal *OpenAIImageModelInternal
}

// OpenAIImageModelInternal contains internal options for OpenAIImageModel.
type OpenAIImageModelInternal struct {
	// CurrentDate returns the current date for testing purposes.
	CurrentDate func() time.Time
}

// OpenAIImageModel implements imagemodel.ImageModel for the OpenAI
// image generation endpoint.
type OpenAIImageModel struct {
	modelID OpenAIImageModelID
	config  OpenAIImageModelConfig
}

// NewOpenAIImageModel creates a new OpenAIImageModel.
func NewOpenAIImageModel(modelID OpenAIImageModelID, config OpenAIImageModelConfig) *OpenAIImageModel {
	return &OpenAIImageModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns "v3".
func (m *OpenAIImageModel) SpecificationVersion() string { return "v3" }

// Provider returns the provider identifier.
func (m *OpenAIImageModel) Provider() string { return m.config.Provider }

// ModelID returns the model identifier.
func (m *OpenAIImageModel) ModelID() string { return m.modelID }

// MaxImagesPerCall returns the maximum number of images per call for this model.
func (m *OpenAIImageModel) MaxImagesPerCall() (*int, error) {
	if v, ok := ModelMaxImagesPerCall[m.modelID]; ok {
		return &v, nil
	}
	v := 1
	return &v, nil
}

// DoGenerate generates images.
func (m *OpenAIImageModel) DoGenerate(options imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
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
	if m.config.Internal != nil && m.config.Internal.CurrentDate != nil {
		currentDate = m.config.Internal.CurrentDate()
	}

	// Extract openai provider options
	openaiOpts := getProviderOptionsMap(options.ProviderOptions, "openai")

	if len(options.Files) > 0 {
		return m.doGenerateEdit(options, openaiOpts, warnings, currentDate)
	}

	return m.doGenerateStandard(options, openaiOpts, warnings, currentDate)
}

func (m *OpenAIImageModel) doGenerateStandard(
	options imagemodel.CallOptions,
	openaiOpts map[string]interface{},
	warnings []shared.Warning,
	currentDate time.Time,
) (imagemodel.GenerateResult, error) {
	body := map[string]interface{}{
		"model": m.modelID,
		"n":     options.N,
	}
	if options.Prompt != nil {
		body["prompt"] = *options.Prompt
	}
	if options.Size != nil {
		body["size"] = *options.Size
	}
	// Merge provider-specific options
	for k, v := range openaiOpts {
		body[k] = v
	}
	// Add response_format if model doesn't have a default
	if !HasDefaultResponseFormat(m.modelID) {
		body["response_format"] = "b64_json"
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[openaiImageResponse]{
		URL:                       m.config.URL(struct{ ModelID string; Path string }{Path: "/images/generations", ModelID: m.modelID}),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), convertHeadersPtrMap(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     openaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(openaiImageResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return imagemodel.GenerateResult{}, err
	}

	return m.buildImageResult(result, warnings, currentDate)
}

func (m *OpenAIImageModel) doGenerateEdit(
	options imagemodel.CallOptions,
	openaiOpts map[string]interface{},
	warnings []shared.Warning,
	currentDate time.Time,
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
	}
	if options.Prompt != nil {
		formInput["prompt"] = *options.Prompt
	}
	if options.N > 0 {
		formInput["n"] = options.N
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

	// Merge provider-specific options
	for k, v := range openaiOpts {
		formInput[k] = v
	}

	formResult, err := providerutils.ConvertToFormData(formInput, nil)
	if err != nil {
		return imagemodel.GenerateResult{}, fmt.Errorf("failed to create form data: %w", err)
	}

	headers := providerutils.CombineHeaders(m.config.Headers(), convertHeadersPtrMap(options.Headers))
	headers["Content-Type"] = formResult.ContentType

	result, err := providerutils.PostToApi(providerutils.PostToApiOptions[openaiImageResponse]{
		URL:     m.config.URL(struct{ ModelID string; Path string }{Path: "/images/edits", ModelID: m.modelID}),
		Headers: headers,
		Body: providerutils.PostToApiBody{
			Content: formResult.Body,
			Values:  formResult.Values,
		},
		FailedResponseHandler:     openaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(openaiImageResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return imagemodel.GenerateResult{}, err
	}

	return m.buildImageResult(&providerutils.PostToApiResult[openaiImageResponse]{
		Value:           result.Value,
		RawValue:        result.RawValue,
		ResponseHeaders: result.ResponseHeaders,
	}, warnings, currentDate)
}

func (m *OpenAIImageModel) buildImageResult(
	result *providerutils.PostToApiResult[openaiImageResponse],
	warnings []shared.Warning,
	currentDate time.Time,
) (imagemodel.GenerateResult, error) {
	response := result.Value

	images := make([]string, len(response.Data))
	for i, item := range response.Data {
		images[i] = item.B64JSON
	}

	var usage *imagemodel.Usage
	if response.Usage != nil {
		usage = &imagemodel.Usage{
			InputTokens:  response.Usage.InputTokens,
			OutputTokens: response.Usage.OutputTokens,
			TotalTokens:  response.Usage.TotalTokens,
		}
	}

	// Build provider metadata
	providerImages := make(jsonvalue.JSONArray, len(response.Data))
	for i, item := range response.Data {
		imgMeta := map[string]interface{}{}
		if item.RevisedPrompt != nil {
			imgMeta["revisedPrompt"] = *item.RevisedPrompt
		}
		if response.Created != nil {
			imgMeta["created"] = *response.Created
		}
		if response.Size != nil {
			imgMeta["size"] = *response.Size
		}
		if response.Quality != nil {
			imgMeta["quality"] = *response.Quality
		}
		if response.Background != nil {
			imgMeta["background"] = *response.Background
		}
		if response.OutputFormat != nil {
			imgMeta["outputFormat"] = *response.OutputFormat
		}
		tokenDetails := distributeTokenDetails(response.Usage, i, len(response.Data))
		for k, v := range tokenDetails {
			imgMeta[k] = v
		}
		providerImages[i] = imgMeta
	}

	return imagemodel.GenerateResult{
		Images:   imagemodel.ImageDataStrings{Values: images},
		Warnings: warnings,
		Usage:    usage,
		Response: imagemodel.GenerateResultResponse{
			Timestamp: currentDate,
			ModelID:   m.modelID,
			Headers:   result.ResponseHeaders,
		},
		ProviderMetadata: imagemodel.ProviderMetadata{
			"openai": imagemodel.ImageProviderMetadataEntry{
				Images: providerImages,
			},
		},
	}, nil
}

// distributeTokenDetails distributes input token details evenly across images,
// with the remainder assigned to the last image so that summing across all
// entries gives the exact total.
func distributeTokenDetails(usage *openaiImageResponseUsage, index, total int) map[string]interface{} {
	if usage == nil || usage.InputTokensDetails == nil {
		return nil
	}
	details := usage.InputTokensDetails
	result := make(map[string]interface{})

	if details.ImageTokens != nil {
		base := *details.ImageTokens / total
		remainder := *details.ImageTokens - base*(total-1)
		if index == total-1 {
			result["imageTokens"] = remainder
		} else {
			result["imageTokens"] = base
		}
	}

	if details.TextTokens != nil {
		base := *details.TextTokens / total
		remainder := *details.TextTokens - base*(total-1)
		if index == total-1 {
			result["textTokens"] = remainder
		} else {
			result["textTokens"] = base
		}
	}

	return result
}

// imageFileToBytes converts an imagemodel.File to raw bytes.
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

// getProviderOptionsMap extracts a specific provider's options as a map.
func getProviderOptionsMap(opts shared.ProviderOptions, provider string) map[string]interface{} {
	if opts == nil {
		return nil
	}
	providerOpts, ok := opts[provider]
	if !ok || providerOpts == nil {
		return nil
	}
	result := make(map[string]interface{}, len(providerOpts))
	for k, v := range providerOpts {
		result[k] = v
	}
	return result
}
