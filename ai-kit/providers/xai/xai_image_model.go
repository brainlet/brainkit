// Ported from: packages/xai/src/xai-image-model.ts
package xai

import (
	"fmt"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/imagemodel"
	"github.com/brainlet/brainkit/ai-kit/provider/jsonvalue"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// XaiImageModelConfig configures the xAI image model.
type XaiImageModelConfig struct {
	Provider    string
	BaseURL     string
	Headers     func() map[string]string
	Fetch       providerutils.FetchFunction
	CurrentDate func() time.Time // internal, for testing
}

// xaiImageResponse is the response structure from xAI image generation.
type xaiImageResponse struct {
	Data []xaiImageResponseItem `json:"data"`
}

// xaiImageResponseItem is a single image in the response.
type xaiImageResponseItem struct {
	URL           string  `json:"url"`
	RevisedPrompt *string `json:"revised_prompt,omitempty"`
}

// xaiImageResponseSchema is the schema for image response validation.
var xaiImageResponseSchema = &providerutils.Schema[xaiImageResponse]{}

// XaiImageModel implements the ImageModel interface for xAI.
type XaiImageModel struct {
	specificationVersion string
	maxImagesPerCall     int
	modelId              XaiImageModelId
	config               XaiImageModelConfig
}

// NewXaiImageModel creates a new xAI image model.
func NewXaiImageModel(modelId XaiImageModelId, config XaiImageModelConfig) *XaiImageModel {
	return &XaiImageModel{
		specificationVersion: "v3",
		maxImagesPerCall:     1,
		modelId:              modelId,
		config:               config,
	}
}

// SpecificationVersion returns the image model interface version.
func (m *XaiImageModel) SpecificationVersion() string {
	return m.specificationVersion
}

// Provider returns the provider name.
func (m *XaiImageModel) Provider() string {
	return m.config.Provider
}

// ModelID returns the model ID.
func (m *XaiImageModel) ModelID() string {
	return m.modelId
}

// MaxImagesPerCall returns the maximum images per call.
func (m *XaiImageModel) MaxImagesPerCall() (*int, error) {
	return &m.maxImagesPerCall, nil
}

// DoGenerate generates images.
func (m *XaiImageModel) DoGenerate(options imagemodel.CallOptions) (imagemodel.GenerateResult, error) {
	var warnings []shared.Warning

	if options.Size != nil {
		detail := "This model does not support the `size` option. Use `aspectRatio` instead."
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "size", Details: &detail})
	}

	if options.Seed != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "seed"})
	}

	if options.Mask != nil {
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "mask"})
	}

	// Parse provider options
	xaiOpts, err := providerutils.ParseProviderOptions("xai", providerOptionsToMap(options.ProviderOptions), xaiImageModelOptionsSchema)
	if err != nil {
		return imagemodel.GenerateResult{}, err
	}

	hasFiles := len(options.Files) > 0
	var imageURL string

	if hasFiles {
		imageURL = convertImageModelFileToDataUri(options.Files[0])

		if len(options.Files) > 1 {
			warnings = append(warnings, shared.OtherWarning{
				Message: "xAI only supports a single input image. Additional images are ignored.",
			})
		}
	}

	endpoint := "/images/generations"
	if hasFiles {
		endpoint = "/images/edits"
	}

	body := map[string]interface{}{
		"model":           m.modelId,
		"prompt":          options.Prompt,
		"n":               options.N,
		"response_format": "url",
	}

	if options.AspectRatio != nil {
		body["aspect_ratio"] = *options.AspectRatio
	}

	if xaiOpts != nil {
		if xaiOpts.OutputFormat != nil {
			body["output_format"] = *xaiOpts.OutputFormat
		}
		if xaiOpts.SyncMode != nil {
			body["sync_mode"] = *xaiOpts.SyncMode
		}
		if xaiOpts.AspectRatio != nil && options.AspectRatio == nil {
			body["aspect_ratio"] = *xaiOpts.AspectRatio
		}
		if xaiOpts.Resolution != nil {
			body["resolution"] = *xaiOpts.Resolution
		}
	}

	if imageURL != "" {
		body["image"] = map[string]interface{}{
			"url":  imageURL,
			"type": "image_url",
		}
	}

	baseURL := m.config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.x.ai/v1"
	}

	currentDate := time.Now()
	if m.config.CurrentDate != nil {
		currentDate = m.config.CurrentDate()
	}

	result, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[xaiImageResponse]{
		URL:                       fmt.Sprintf("%s%s", baseURL, endpoint),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), headersToStringMap(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     xaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(xaiImageResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return imagemodel.GenerateResult{}, err
	}

	response := result.Value

	// Download images
	var downloadedImages [][]byte
	for _, img := range response.Data {
		imgBytes, err := m.downloadImage(img.URL, options)
		if err != nil {
			return imagemodel.GenerateResult{}, err
		}
		downloadedImages = append(downloadedImages, imgBytes)
	}

	// Build provider metadata
	var imagesMetadata jsonvalue.JSONArray
	for _, item := range response.Data {
		entry := make(map[string]interface{})
		if item.RevisedPrompt != nil {
			entry["revisedPrompt"] = *item.RevisedPrompt
		}
		imagesMetadata = append(imagesMetadata, entry)
	}

	return imagemodel.GenerateResult{
		Images: imagemodel.ImageDataBytes{Values: downloadedImages},
		Warnings: warnings,
		Response: imagemodel.GenerateResultResponse{
			Timestamp: currentDate,
			ModelID:   m.modelId,
			Headers:   result.ResponseHeaders,
		},
		ProviderMetadata: imagemodel.ProviderMetadata{
			"xai": imagemodel.ImageProviderMetadataEntry{
				Images: imagesMetadata,
			},
		},
	}, nil
}

// downloadImage downloads an image from a URL.
func (m *XaiImageModel) downloadImage(url string, options imagemodel.CallOptions) ([]byte, error) {
	result, err := providerutils.GetFromApi(providerutils.GetFromApiOptions[[]byte]{
		URL:                       url,
		Ctx:                       options.Ctx,
		FailedResponseHandler:     statusCodeErrorResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateBinaryResponseHandler(),
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return nil, err
	}
	return result.Value, nil
}

// convertImageModelFileToDataUri converts an imagemodel.File interface to a data URI.
func convertImageModelFileToDataUri(file imagemodel.File) string {
	switch f := file.(type) {
	case imagemodel.FileURL:
		return f.URL
	case imagemodel.FileData:
		switch d := f.Data.(type) {
		case imagemodel.ImageFileDataString:
			return fmt.Sprintf("data:%s;base64,%s", f.MediaType, d.Value)
		case imagemodel.ImageFileDataBytes:
			return fmt.Sprintf("data:%s;base64,%s", f.MediaType, providerutils.ConvertBytesToBase64(d.Data))
		}
	}
	return ""
}
