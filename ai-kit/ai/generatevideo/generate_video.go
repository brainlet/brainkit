// Ported from: packages/ai/src/generate-video/generate-video.ts
package generatevideo

import (
	"context"
	"encoding/base64"
	"fmt"
	"math"
	"sync"
)

// VideoModel is the interface for video generation models.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type VideoModel interface {
	// Provider returns the provider name.
	Provider() string
	// ModelID returns the model identifier.
	ModelID() string
	// MaxVideosPerCall returns the maximum videos per call, or 0 if unlimited.
	MaxVideosPerCall() int
	// DoGenerate performs the video generation operation.
	DoGenerate(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error)
}

// VideoData represents a video returned by the model.
type VideoData struct {
	// Type is one of "url", "base64", or "binary".
	Type string
	// URL is the URL of the video (when Type is "url").
	URL string
	// Data is the base64-encoded or binary data.
	Data string
	// BinaryData is the raw binary data (when Type is "binary").
	BinaryData []byte
	// MediaType is the MIME type of the video.
	MediaType string
}

// DoGenerateOptions are the options passed to VideoModel.DoGenerate.
type DoGenerateOptions struct {
	Prompt          string
	N               int
	AspectRatio     string
	Resolution      string
	Duration        *float64
	FPS             *int
	Seed            *int
	Headers         map[string]string
	ProviderOptions map[string]map[string]any
}

// DoGenerateResult is the result from VideoModel.DoGenerate.
type DoGenerateResult struct {
	Videos           []VideoData
	Warnings         []Warning
	Response         VideoModelResponseMetadata
	ProviderMetadata VideoModelProviderMetadata
}

// DownloadFunc is a function that downloads data from a URL.
type DownloadFunc func(ctx context.Context, url string) (data []byte, mediaType string, err error)

// GenerateVideoOptions are the options for the GenerateVideo function.
type GenerateVideoOptions struct {
	// Model is the video model to use.
	Model VideoModel

	// Prompt is the prompt that should be used to generate the video.
	Prompt string

	// N is the number of videos to generate. Default: 1.
	N int

	// MaxVideosPerCall overrides the model's default max videos per call.
	MaxVideosPerCall *int

	// AspectRatio of the videos. Format: "{width}:{height}".
	AspectRatio string

	// Resolution of the videos. Format: "{width}x{height}".
	Resolution string

	// Duration of the video in seconds.
	Duration *float64

	// FPS is the frames per second for the video.
	FPS *int

	// Seed for the video generation.
	Seed *int

	// MaxRetries is the maximum number of retries. Default: 2.
	MaxRetries *int

	// Headers are additional headers to include in the request.
	Headers map[string]string

	// ProviderOptions are additional provider-specific options.
	ProviderOptions map[string]map[string]any

	// Download is a custom download function for fetching videos from URLs.
	Download DownloadFunc
}

// GenerateVideo generates videos using a video model.
// This is the Go equivalent of experimental_generateVideo.
func GenerateVideo(ctx context.Context, opts GenerateVideoOptions) (*GenerateVideoResult, error) {
	model := opts.Model

	n := opts.N
	if n <= 0 {
		n = 1
	}

	// Determine max videos per call.
	maxPerCall := 1
	if opts.MaxVideosPerCall != nil {
		maxPerCall = *opts.MaxVideosPerCall
	} else if model.MaxVideosPerCall() > 0 {
		maxPerCall = model.MaxVideosPerCall()
	}

	// Parallelize calls to the model.
	callCount := int(math.Ceil(float64(n) / float64(maxPerCall)))
	callVideoCounts := make([]int, callCount)
	for i := 0; i < callCount; i++ {
		remaining := n - i*maxPerCall
		if remaining > maxPerCall {
			callVideoCounts[i] = maxPerCall
		} else {
			callVideoCounts[i] = remaining
		}
	}

	type callResult struct {
		result *DoGenerateResult
		err    error
	}

	results := make([]callResult, callCount)
	var wg sync.WaitGroup
	wg.Add(callCount)

	providerOpts := opts.ProviderOptions
	if providerOpts == nil {
		providerOpts = map[string]map[string]any{}
	}

	for i, count := range callVideoCounts {
		go func(idx, videoCount int) {
			defer wg.Done()
			res, err := model.DoGenerate(ctx, DoGenerateOptions{
				Prompt:          opts.Prompt,
				N:               videoCount,
				AspectRatio:     opts.AspectRatio,
				Resolution:      opts.Resolution,
				Duration:        opts.Duration,
				FPS:             opts.FPS,
				Seed:            opts.Seed,
				Headers:         opts.Headers,
				ProviderOptions: providerOpts,
			})
			results[idx] = callResult{result: res, err: err}
		}(i, count)
	}
	wg.Wait()

	// Collect results.
	var videos []GeneratedFile
	var warnings []Warning
	var responses []VideoModelResponseMetadata
	providerMetadata := make(VideoModelProviderMetadata)

	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}

		for _, videoData := range r.result.Videos {
			switch videoData.Type {
			case "url":
				if opts.Download != nil {
					data, downloadedMediaType, err := opts.Download(ctx, videoData.URL)
					if err != nil {
						return nil, err
					}
					mediaType := videoData.MediaType
					if mediaType == "" || mediaType == "application/octet-stream" {
						if downloadedMediaType != "" && downloadedMediaType != "application/octet-stream" {
							mediaType = downloadedMediaType
						} else {
							mediaType = detectVideoMediaType(data)
							if mediaType == "" {
								mediaType = "video/mp4"
							}
						}
					}
					videos = append(videos, GeneratedFile{
						Data:      data,
						MediaType: mediaType,
					})
				} else {
					// Without a download function, we cannot fetch URL videos.
					return nil, fmt.Errorf("download function required for URL video data")
				}

			case "base64":
				data, err := base64.StdEncoding.DecodeString(videoData.Data)
				if err != nil {
					return nil, fmt.Errorf("failed to decode base64 video data: %w", err)
				}
				mediaType := videoData.MediaType
				if mediaType == "" {
					mediaType = "video/mp4"
				}
				videos = append(videos, GeneratedFile{
					Data:      data,
					MediaType: mediaType,
				})

			case "binary":
				mediaType := videoData.MediaType
				if mediaType == "" {
					mediaType = detectVideoMediaType(videoData.BinaryData)
					if mediaType == "" {
						mediaType = "video/mp4"
					}
				}
				videos = append(videos, GeneratedFile{
					Data:      videoData.BinaryData,
					MediaType: mediaType,
				})
			}
		}

		warnings = append(warnings, r.result.Warnings...)

		responses = append(responses, VideoModelResponseMetadata{
			Timestamp:        r.result.Response.Timestamp,
			ModelID:          r.result.Response.ModelID,
			Headers:          r.result.Response.Headers,
			ProviderMetadata: mapToAny(r.result.ProviderMetadata),
		})

		if r.result.ProviderMetadata != nil {
			for providerName, metadata := range r.result.ProviderMetadata {
				existing, ok := providerMetadata[providerName]
				if !ok {
					providerMetadata[providerName] = metadata
				} else {
					merged := make(map[string]any)
					for k, v := range existing {
						merged[k] = v
					}
					for k, v := range metadata {
						// Merge videos arrays if both exist
						if k == "videos" {
							existingVideos, existOk := existing["videos"]
							if existOk {
								if existArr, ok1 := existingVideos.([]any); ok1 {
									if newArr, ok2 := v.([]any); ok2 {
										merged[k] = append(existArr, newArr...)
										continue
									}
								}
							}
						}
						merged[k] = v
					}
					providerMetadata[providerName] = merged
				}
			}
		}
	}

	if len(videos) == 0 {
		return nil, fmt.Errorf("no video generated")
	}

	return &GenerateVideoResult{
		Video:            videos[0],
		Videos:           videos,
		Warnings:         warnings,
		Responses:        responses,
		ProviderMetadata: providerMetadata,
	}, nil
}

// detectVideoMediaType attempts to detect the media type from video data bytes.
func detectVideoMediaType(data []byte) string {
	if len(data) < 12 {
		return ""
	}
	// MP4: ... ftyp at offset 4
	if data[4] == 0x66 && data[5] == 0x74 && data[6] == 0x79 && data[7] == 0x70 {
		return "video/mp4"
	}
	// WebM: 1A 45 DF A3
	if data[0] == 0x1A && data[1] == 0x45 && data[2] == 0xDF && data[3] == 0xA3 {
		return "video/webm"
	}
	return ""
}

// mapToAny converts a VideoModelProviderMetadata to map[string]any.
func mapToAny(m VideoModelProviderMetadata) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any)
	for k, v := range m {
		result[k] = v
	}
	return result
}
