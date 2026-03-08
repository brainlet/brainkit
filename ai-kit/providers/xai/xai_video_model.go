// Ported from: packages/xai/src/xai-video-model.ts
package xai

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/errors"
	"github.com/brainlet/brainkit/ai-kit/provider/shared"
	"github.com/brainlet/brainkit/ai-kit/provider/videomodel"
	"github.com/brainlet/brainkit/ai-kit/providerutils"
)

// XaiVideoModelConfig configures the xAI video model.
type XaiVideoModelConfig struct {
	Provider    string
	BaseURL     string
	Headers     func() map[string]string
	Fetch       providerutils.FetchFunction
	CurrentDate func() time.Time // internal, for testing
}

// resolutionMap maps standard resolution strings to xAI format.
var resolutionMap = map[string]string{
	"1280x720": "720p",
	"854x480":  "480p",
	"640x480":  "480p",
}

// xaiCreateVideoResponse is the response from creating a video generation request.
type xaiCreateVideoResponse struct {
	RequestID *string `json:"request_id,omitempty"`
}

// xaiVideoStatusResponse is the response from polling video generation status.
type xaiVideoStatusResponse struct {
	Status *string                    `json:"status,omitempty"`
	Video  *xaiVideoStatusVideoEntry `json:"video,omitempty"`
	Model  *string                   `json:"model,omitempty"`
}

// xaiVideoStatusVideoEntry contains video data in the status response.
type xaiVideoStatusVideoEntry struct {
	URL                string   `json:"url"`
	Duration           *float64 `json:"duration,omitempty"`
	RespectModeration  *bool    `json:"respect_moderation,omitempty"`
}

// xaiCreateVideoResponseSchema is the schema for video creation response validation.
var xaiCreateVideoResponseSchema = &providerutils.Schema[xaiCreateVideoResponse]{}

// xaiVideoStatusResponseSchema is the schema for video status response validation.
var xaiVideoStatusResponseSchema = &providerutils.Schema[xaiVideoStatusResponse]{}

// XaiVideoModel implements the VideoModel interface for xAI.
type XaiVideoModel struct {
	specificationVersion string
	maxVideosPerCall     int
	modelId              XaiVideoModelId
	config               XaiVideoModelConfig
}

// NewXaiVideoModel creates a new xAI video model.
func NewXaiVideoModel(modelId XaiVideoModelId, config XaiVideoModelConfig) *XaiVideoModel {
	return &XaiVideoModel{
		specificationVersion: "v3",
		maxVideosPerCall:     1,
		modelId:              modelId,
		config:               config,
	}
}

// SpecificationVersion returns the video model interface version.
func (m *XaiVideoModel) SpecificationVersion() string {
	return m.specificationVersion
}

// Provider returns the provider name.
func (m *XaiVideoModel) Provider() string {
	return m.config.Provider
}

// ModelID returns the model ID.
func (m *XaiVideoModel) ModelID() string {
	return m.modelId
}

// MaxVideosPerCall returns the maximum videos per call.
func (m *XaiVideoModel) MaxVideosPerCall() (*int, error) {
	return &m.maxVideosPerCall, nil
}

// DoGenerate generates a video.
func (m *XaiVideoModel) DoGenerate(options videomodel.CallOptions) (videomodel.GenerateResult, error) {
	currentDate := time.Now()
	if m.config.CurrentDate != nil {
		currentDate = m.config.CurrentDate()
	}

	var warnings []shared.Warning

	// Parse xAI-specific options
	xaiOpts, err := providerutils.ParseProviderOptions("xai", providerOptionsToMap(options.ProviderOptions), xaiVideoModelOptionsSchema)
	if err != nil {
		return videomodel.GenerateResult{}, err
	}

	isEdit := xaiOpts != nil && xaiOpts.VideoURL != nil

	if options.FPS != nil {
		detail := "xAI video models do not support custom FPS."
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "fps", Details: &detail})
	}

	if options.Seed != nil {
		detail := "xAI video models do not support seed."
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "seed", Details: &detail})
	}

	if options.N > 1 {
		detail := "xAI video models do not support generating multiple videos per call. Only 1 video will be generated."
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "n", Details: &detail})
	}

	if isEdit && options.Duration != nil {
		detail := "xAI video editing does not support custom duration."
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "duration", Details: &detail})
	}

	if isEdit && options.AspectRatio != nil {
		detail := "xAI video editing does not support custom aspect ratio."
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "aspectRatio", Details: &detail})
	}

	if isEdit && ((xaiOpts != nil && xaiOpts.Resolution != nil) || options.Resolution != nil) {
		detail := "xAI video editing does not support custom resolution."
		warnings = append(warnings, shared.UnsupportedWarning{Feature: "resolution", Details: &detail})
	}

	body := map[string]interface{}{
		"model":  m.modelId,
		"prompt": options.Prompt,
	}

	if !isEdit && options.Duration != nil {
		body["duration"] = *options.Duration
	}

	if !isEdit && options.AspectRatio != nil {
		body["aspect_ratio"] = *options.AspectRatio
	}

	if !isEdit {
		if xaiOpts != nil && xaiOpts.Resolution != nil {
			body["resolution"] = *xaiOpts.Resolution
		} else if options.Resolution != nil {
			if mapped, ok := resolutionMap[*options.Resolution]; ok {
				body["resolution"] = mapped
			} else {
				detail := fmt.Sprintf("Unrecognized resolution %q. Use providerOptions.xai.resolution with \"480p\" or \"720p\" instead.", *options.Resolution)
				warnings = append(warnings, shared.UnsupportedWarning{Feature: "resolution", Details: &detail})
			}
		}
	}

	// Video editing: pass source video URL
	if xaiOpts != nil && xaiOpts.VideoURL != nil {
		body["video"] = map[string]interface{}{
			"url": *xaiOpts.VideoURL,
		}
	}

	// Image-to-video: convert SDK image to nested image object
	if options.Image != nil {
		switch img := options.Image.(type) {
		case videomodel.VideoFileURL:
			body["image"] = map[string]interface{}{
				"url": img.URL,
			}
		case videomodel.VideoFileData:
			var base64Data string
			switch d := img.Data.(type) {
			case videomodel.VideoFileDataString:
				base64Data = d.Value
			case videomodel.VideoFileDataBytes:
				base64Data = providerutils.ConvertBytesToBase64(d.Data)
			}
			body["image"] = map[string]interface{}{
				"url": fmt.Sprintf("data:%s;base64,%s", img.MediaType, base64Data),
			}
		}
	}

	baseURL := m.config.BaseURL
	if baseURL == "" {
		baseURL = "https://api.x.ai/v1"
	}

	// Step 1: Create video generation/edit request
	genEndpoint := "generations"
	if isEdit {
		genEndpoint = "edits"
	}

	createResult, err := providerutils.PostJsonToApi(providerutils.PostJsonToApiOptions[xaiCreateVideoResponse]{
		URL:                       fmt.Sprintf("%s/videos/%s", baseURL, genEndpoint),
		Headers:                   providerutils.CombineHeaders(m.config.Headers(), headersToStringMap(options.Headers)),
		Body:                      body,
		FailedResponseHandler:     xaiFailedResponseHandler,
		SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(xaiCreateVideoResponseSchema),
		Ctx:                       options.Ctx,
		Fetch:                     m.config.Fetch,
	})
	if err != nil {
		return videomodel.GenerateResult{}, err
	}

	requestID := ""
	if createResult.Value.RequestID != nil {
		requestID = *createResult.Value.RequestID
	}
	if requestID == "" {
		rawJSON, _ := json.Marshal(createResult.Value)
		return videomodel.GenerateResult{}, &errors.AISDKError{
			Name:    "XAI_VIDEO_GENERATION_ERROR",
			Message: fmt.Sprintf("No request_id returned from xAI API. Response: %s", string(rawJSON)),
		}
	}

	// Step 2: Poll for completion
	pollIntervalMs := 5000
	if xaiOpts != nil && xaiOpts.PollIntervalMs != nil {
		pollIntervalMs = *xaiOpts.PollIntervalMs
	}
	pollTimeoutMs := 600000
	if xaiOpts != nil && xaiOpts.PollTimeoutMs != nil {
		pollTimeoutMs = *xaiOpts.PollTimeoutMs
	}

	startTime := time.Now()
	var responseHeaders map[string]string

	for {
		err := providerutils.Delay(options.Ctx, time.Duration(pollIntervalMs)*time.Millisecond)
		if err != nil {
			return videomodel.GenerateResult{}, err
		}

		if time.Since(startTime) > time.Duration(pollTimeoutMs)*time.Millisecond {
			return videomodel.GenerateResult{}, &errors.AISDKError{
				Name:    "XAI_VIDEO_GENERATION_TIMEOUT",
				Message: fmt.Sprintf("Video generation timed out after %dms", pollTimeoutMs),
			}
		}

		statusResult, err := providerutils.GetFromApi(providerutils.GetFromApiOptions[xaiVideoStatusResponse]{
			URL:                       fmt.Sprintf("%s/videos/%s", baseURL, requestID),
			Headers:                   providerutils.CombineHeaders(m.config.Headers(), headersToStringMap(options.Headers)),
			SuccessfulResponseHandler: providerutils.CreateJsonResponseHandler(xaiVideoStatusResponseSchema),
			FailedResponseHandler:     xaiFailedResponseHandler,
			Ctx:                       options.Ctx,
			Fetch:                     m.config.Fetch,
		})
		if err != nil {
			return videomodel.GenerateResult{}, err
		}

		responseHeaders = statusResult.ResponseHeaders
		statusResponse := statusResult.Value

		isDone := (statusResponse.Status != nil && *statusResponse.Status == "done") ||
			(statusResponse.Status == nil && statusResponse.Video != nil && statusResponse.Video.URL != "")

		if isDone {
			if statusResponse.Video == nil || statusResponse.Video.URL == "" {
				return videomodel.GenerateResult{}, &errors.AISDKError{
					Name:    "XAI_VIDEO_GENERATION_ERROR",
					Message: "Video generation completed but no video URL was returned.",
				}
			}

			providerMeta := map[string]interface{}{
				"requestId": requestID,
				"videoUrl":  statusResponse.Video.URL,
			}
			if statusResponse.Video.Duration != nil {
				providerMeta["duration"] = *statusResponse.Video.Duration
			}

			return videomodel.GenerateResult{
				Videos: []videomodel.VideoData{
					videomodel.VideoDataURL{
						URL:       statusResponse.Video.URL,
						MediaType: "video/mp4",
					},
				},
				Warnings: warnings,
				Response: videomodel.GenerateResultResponse{
					Timestamp: currentDate,
					ModelID:   m.modelId,
					Headers:   responseHeaders,
				},
				ProviderMetadata: shared.ProviderMetadata{
					"xai": providerMeta,
				},
			}, nil
		}

		if statusResponse.Status != nil && *statusResponse.Status == "expired" {
			return videomodel.GenerateResult{}, &errors.AISDKError{
				Name:    "XAI_VIDEO_GENERATION_EXPIRED",
				Message: "Video generation request expired.",
			}
		}

		// "pending" -> continue polling
	}
}
