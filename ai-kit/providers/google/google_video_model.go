// Ported from: packages/google/src/google-generative-ai-video-model.ts
package google

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/provider/videomodel"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// GoogleVideoModelOptions contains provider-specific options for video generation.
type GoogleVideoModelOptions struct {
	// PollIntervalMs is the polling interval for long-running operations.
	PollIntervalMs *int `json:"pollIntervalMs,omitempty"`

	// PollTimeoutMs is the maximum time to wait for video generation.
	PollTimeoutMs *int `json:"pollTimeoutMs,omitempty"`

	// PersonGeneration controls person generation in videos.
	PersonGeneration *string `json:"personGeneration,omitempty"`

	// NegativePrompt is a negative prompt for the video generation.
	NegativePrompt *string `json:"negativePrompt,omitempty"`

	// ReferenceImages are reference images for style/asset reference.
	ReferenceImages []GoogleVideoReferenceImage `json:"referenceImages,omitempty"`
}

// GoogleVideoReferenceImage is a reference image for video generation.
type GoogleVideoReferenceImage struct {
	BytesBase64Encoded *string `json:"bytesBase64Encoded,omitempty"`
	GcsURI             *string `json:"gcsUri,omitempty"`
}

// GoogleVideoModelOptionsSchema is the providerutils.Schema for video model options.
var GoogleVideoModelOptionsSchema = &providerutils.Schema[GoogleVideoModelOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[GoogleVideoModelOptions], error) {
		data, err := json.Marshal(value)
		if err != nil {
			return &providerutils.ValidationResult[GoogleVideoModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		var opts GoogleVideoModelOptions
		if err := json.Unmarshal(data, &opts); err != nil {
			return &providerutils.ValidationResult[GoogleVideoModelOptions]{
				Success: false,
				Error:   err,
			}, nil
		}
		return &providerutils.ValidationResult[GoogleVideoModelOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}

// GoogleVideoModelConfig configures a GoogleVideoModel.
type GoogleVideoModelConfig struct {
	Provider   string
	BaseURL    string
	Headers    func() map[string]string
	Fetch      providerutils.FetchFunction
	GenerateID providerutils.IdGenerator

	Internal *GoogleVideoModelInternal
}

// GoogleVideoModelInternal contains internal configuration for testing.
type GoogleVideoModelInternal struct {
	CurrentDate func() time.Time
}

// GoogleVideoModel implements videomodel.VideoModel for the Google Generative AI API.
type GoogleVideoModel struct {
	modelID string
	config  GoogleVideoModelConfig
}

// NewGoogleVideoModel creates a new GoogleVideoModel.
func NewGoogleVideoModel(modelID string, config GoogleVideoModelConfig) *GoogleVideoModel {
	return &GoogleVideoModel{
		modelID: modelID,
		config:  config,
	}
}

// SpecificationVersion returns the video model interface version.
func (m *GoogleVideoModel) SpecificationVersion() string {
	return "v3"
}

// Provider returns the provider ID.
func (m *GoogleVideoModel) Provider() string {
	return m.config.Provider
}

// ModelID returns the model ID.
func (m *GoogleVideoModel) ModelID() string {
	return m.modelID
}

// MaxVideosPerCall returns the maximum videos per call.
func (m *GoogleVideoModel) MaxVideosPerCall() (*int, error) {
	v := 4
	return &v, nil
}

// DoGenerate generates videos.
func (m *GoogleVideoModel) DoGenerate(options videomodel.CallOptions) (videomodel.GenerateResult, error) {
	currentDate := m.getCurrentDate()
	var warnings []shared.Warning

	googleOptions, err := providerutils.ParseProviderOptions(
		"google",
		toInterfaceMap(options.ProviderOptions),
		GoogleVideoModelOptionsSchema,
	)
	if err != nil {
		return videomodel.GenerateResult{}, err
	}

	instance := map[string]any{}

	if options.Prompt != nil {
		instance["prompt"] = *options.Prompt
	}

	// Handle image-to-video: convert image to base64
	if options.Image != nil {
		switch img := options.Image.(type) {
		case videomodel.VideoFileURL:
			warnings = append(warnings, shared.UnsupportedWarning{
				Feature: "URL-based image input",
				Details: strPtr("Google Generative AI video models require base64-encoded images. URL will be ignored."),
			})
		case videomodel.VideoFileData:
			var base64Data string
			switch d := img.Data.(type) {
			case videomodel.VideoFileDataString:
				base64Data = d.Value
			case videomodel.VideoFileDataBytes:
				base64Data = providerutils.ConvertBytesToBase64(d.Data)
			}

			mediaType := img.MediaType
			if mediaType == "" {
				mediaType = "image/png"
			}

			instance["image"] = map[string]any{
				"inlineData": map[string]any{
					"mimeType": mediaType,
					"data":     base64Data,
				},
			}
		}
	}

	// Reference images
	if googleOptions != nil && len(googleOptions.ReferenceImages) > 0 {
		refImages := make([]any, len(googleOptions.ReferenceImages))
		for i, refImg := range googleOptions.ReferenceImages {
			if refImg.BytesBase64Encoded != nil {
				refImages[i] = map[string]any{
					"inlineData": map[string]any{
						"mimeType": "image/png",
						"data":     *refImg.BytesBase64Encoded,
					},
				}
			} else if refImg.GcsURI != nil {
				refImages[i] = map[string]any{
					"gcsUri": *refImg.GcsURI,
				}
			} else {
				refImages[i] = refImg
			}
		}
		instance["referenceImages"] = refImages
	}

	parameters := map[string]any{
		"sampleCount": options.N,
	}

	if options.AspectRatio != nil {
		parameters["aspectRatio"] = *options.AspectRatio
	}

	if options.Resolution != nil {
		resolutionMap := map[string]string{
			"1280x720":  "720p",
			"1920x1080": "1080p",
			"3840x2160": "4k",
		}
		if mapped, ok := resolutionMap[*options.Resolution]; ok {
			parameters["resolution"] = mapped
		} else {
			parameters["resolution"] = *options.Resolution
		}
	}

	if options.Duration != nil {
		parameters["durationSeconds"] = *options.Duration
	}

	if options.Seed != nil {
		parameters["seed"] = *options.Seed
	}

	if googleOptions != nil {
		if googleOptions.PersonGeneration != nil {
			parameters["personGeneration"] = *googleOptions.PersonGeneration
		}
		if googleOptions.NegativePrompt != nil {
			parameters["negativePrompt"] = *googleOptions.NegativePrompt
		}
	}

	body := map[string]any{
		"instances":  []any{instance},
		"parameters": parameters,
	}

	headers := m.config.Headers()
	mergedHeaders := providerutils.CombineHeaders(headers, convertOptionalHeaders(options.Headers))

	operationResult, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[googleOperation]{
		URL:                       fmt.Sprintf("%s/models/%s:predictLongRunning", m.config.BaseURL, m.modelID),
		Headers:                   mergedHeaders,
		Body:                      body,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler[googleOperation](nil),
		FailedResponseHandler:     GoogleFailedResponseHandler,
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return videomodel.GenerateResult{}, err
	}

	operation := operationResult.Value
	if operation.Name == nil || *operation.Name == "" {
		return videomodel.GenerateResult{}, &errors.AISDKError{
			Name:    "GOOGLE_VIDEO_GENERATION_ERROR",
			Message: "No operation name returned from API",
		}
	}

	operationName := *operation.Name

	pollIntervalMs := 10000 // 10 seconds
	if googleOptions != nil && googleOptions.PollIntervalMs != nil {
		pollIntervalMs = *googleOptions.PollIntervalMs
	}

	pollTimeoutMs := 600000 // 10 minutes
	if googleOptions != nil && googleOptions.PollTimeoutMs != nil {
		pollTimeoutMs = *googleOptions.PollTimeoutMs
	}

	startTime := time.Now()
	finalOperation := operation
	var responseHeaders map[string]string

	for finalOperation.Done == nil || !*finalOperation.Done {
		if time.Since(startTime).Milliseconds() > int64(pollTimeoutMs) {
			return videomodel.GenerateResult{}, &errors.AISDKError{
				Name:    "GOOGLE_VIDEO_GENERATION_TIMEOUT",
				Message: fmt.Sprintf("Video generation timed out after %dms", pollTimeoutMs),
			}
		}

		err := providerutils.Delay(options.Ctx, time.Duration(pollIntervalMs)*time.Millisecond)
		if err != nil {
			return videomodel.GenerateResult{}, &errors.AISDKError{
				Name:    "GOOGLE_VIDEO_GENERATION_ABORTED",
				Message: "Video generation request was aborted",
			}
		}

		if options.Ctx != nil {
			select {
			case <-options.Ctx.Done():
				return videomodel.GenerateResult{}, &errors.AISDKError{
					Name:    "GOOGLE_VIDEO_GENERATION_ABORTED",
					Message: "Video generation request was aborted",
				}
			default:
			}
		}

		statusResult, err := providerutils.GetFromApi(providerutils.GetFromApiOptions[googleOperation]{
			URL:                       fmt.Sprintf("%s/%s", m.config.BaseURL, operationName),
			Headers:                   providerutils.CombineHeaders(headers, convertOptionalHeaders(options.Headers)),
			SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler[googleOperation](nil),
			FailedResponseHandler:     GoogleFailedResponseHandler,
			Ctx:                       options.Ctx,
			Fetch:                     m.config.Fetch,
		})
		if err != nil {
			return videomodel.GenerateResult{}, err
		}

		finalOperation = statusResult.Value
		responseHeaders = statusResult.ResponseHeaders
	}

	if finalOperation.Error != nil {
		return videomodel.GenerateResult{}, &errors.AISDKError{
			Name:    "GOOGLE_VIDEO_GENERATION_FAILED",
			Message: fmt.Sprintf("Video generation failed: %s", finalOperation.Error.Message),
		}
	}

	if finalOperation.Response == nil ||
		finalOperation.Response.GenerateVideoResponse == nil ||
		len(finalOperation.Response.GenerateVideoResponse.GeneratedSamples) == 0 {
		rawJSON, _ := json.Marshal(finalOperation)
		return videomodel.GenerateResult{}, &errors.AISDKError{
			Name:    "GOOGLE_VIDEO_GENERATION_ERROR",
			Message: fmt.Sprintf("No videos in response. Response: %s", string(rawJSON)),
		}
	}

	var videos []videomodel.VideoData
	var videoMetadata []map[string]any

	// Get API key from headers to append to download URLs
	resolvedHeaders := m.config.Headers()
	apiKey := resolvedHeaders["x-goog-api-key"]

	for _, sample := range finalOperation.Response.GenerateVideoResponse.GeneratedSamples {
		if sample.Video != nil && sample.Video.URI != nil && *sample.Video.URI != "" {
			uri := *sample.Video.URI

			// Append API key to URL for authentication during download
			urlWithAuth := uri
			if apiKey != "" {
				separator := "?"
				if containsChar(uri, '?') {
					separator = "&"
				}
				urlWithAuth = fmt.Sprintf("%s%skey=%s", uri, separator, apiKey)
			}

			videos = append(videos, videomodel.VideoDataURL{
				URL:       urlWithAuth,
				MediaType: "video/mp4",
			})
			videoMetadata = append(videoMetadata, map[string]any{
				"uri": uri,
			})
		}
	}

	if len(videos) == 0 {
		return videomodel.GenerateResult{}, &errors.AISDKError{
			Name:    "GOOGLE_VIDEO_GENERATION_ERROR",
			Message: "No valid videos in response",
		}
	}

	providerMeta := shared.ProviderMetadata{
		"google": map[string]any{
			"videos": videoMetadata,
		},
	}

	return videomodel.GenerateResult{
		Videos:           videos,
		Warnings:         warnings,
		ProviderMetadata: providerMeta,
		Response: videomodel.GenerateResultResponse{
			Timestamp: currentDate,
			ModelID:   m.modelID,
			Headers:   responseHeaders,
		},
	}, nil
}

func (m *GoogleVideoModel) getCurrentDate() time.Time {
	if m.config.Internal != nil && m.config.Internal.CurrentDate != nil {
		return m.config.Internal.CurrentDate()
	}
	return time.Now()
}

func containsChar(s string, c byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return true
		}
	}
	return false
}

// --- Response types for long-running operations ---

type googleOperation struct {
	Name     *string              `json:"name,omitempty"`
	Done     *bool                `json:"done,omitempty"`
	Error    *googleOperationError `json:"error,omitempty"`
	Response *googleOperationResponse `json:"response,omitempty"`
}

type googleOperationError struct {
	Code    *int   `json:"code,omitempty"`
	Message string `json:"message"`
	Status  *string `json:"status,omitempty"`
}

type googleOperationResponse struct {
	GenerateVideoResponse *googleGenerateVideoResponse `json:"generateVideoResponse,omitempty"`
}

type googleGenerateVideoResponse struct {
	GeneratedSamples []googleGeneratedSample `json:"generatedSamples,omitempty"`
}

type googleGeneratedSample struct {
	Video *googleVideoInfo `json:"video,omitempty"`
}

type googleVideoInfo struct {
	URI *string `json:"uri,omitempty"`
}
