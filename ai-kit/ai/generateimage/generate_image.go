// Ported from: packages/ai/src/generate-image/generate-image.ts
package generateimage

import (
	"context"
	"fmt"
	"math"
	"sync"
)

// ImageModel is the interface for image generation models.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type ImageModel interface {
	// Provider returns the provider name.
	Provider() string
	// ModelID returns the model identifier.
	ModelID() string
	// MaxImagesPerCall returns the maximum images per call, or 0 if unlimited.
	MaxImagesPerCall() int
	// DoGenerate performs the image generation operation.
	DoGenerate(ctx context.Context, opts DoGenerateOptions) (*DoGenerateResult, error)
}

// DoGenerateOptions are the options passed to ImageModel.DoGenerate.
type DoGenerateOptions struct {
	Prompt          string
	N               int
	Size            string
	AspectRatio     string
	Seed            *int
	Headers         map[string]string
	ProviderOptions map[string]map[string]any
}

// DoGenerateResult is the result from ImageModel.DoGenerate.
type DoGenerateResult struct {
	Images           [][]byte
	Warnings         []Warning
	Response         ImageModelResponseMetadata
	ProviderMetadata ImageModelProviderMetadata
	Usage            *ImageModelUsage
}

// GenerateImageOptions are the options for the GenerateImage function.
type GenerateImageOptions struct {
	// Model is the image model to use.
	Model ImageModel

	// Prompt is the prompt that should be used to generate the image.
	Prompt string

	// N is the number of images to generate. Default: 1.
	N int

	// MaxImagesPerCall overrides the model's default max images per call.
	MaxImagesPerCall *int

	// Size of the images to generate. Format: "{width}x{height}".
	Size string

	// AspectRatio of the images to generate. Format: "{width}:{height}".
	AspectRatio string

	// Seed for the image generation.
	Seed *int

	// MaxRetries is the maximum number of retries. Default: 2.
	MaxRetries *int

	// Headers are additional headers to include in the request.
	Headers map[string]string

	// ProviderOptions are additional provider-specific options.
	ProviderOptions map[string]map[string]any
}

// GenerateImage generates images using an image model.
func GenerateImage(ctx context.Context, opts GenerateImageOptions) (*GenerateImageResult, error) {
	model := opts.Model

	n := opts.N
	if n <= 0 {
		n = 1
	}

	// Determine max images per call.
	maxPerCall := 1
	if opts.MaxImagesPerCall != nil {
		maxPerCall = *opts.MaxImagesPerCall
	} else if model.MaxImagesPerCall() > 0 {
		maxPerCall = model.MaxImagesPerCall()
	}

	// Parallelize calls to the model.
	callCount := int(math.Ceil(float64(n) / float64(maxPerCall)))
	callImageCounts := make([]int, callCount)
	for i := 0; i < callCount; i++ {
		if i < callCount-1 {
			callImageCounts[i] = maxPerCall
		} else {
			remainder := n % maxPerCall
			if remainder == 0 {
				callImageCounts[i] = maxPerCall
			} else {
				callImageCounts[i] = remainder
			}
		}
	}

	type callResult struct {
		result *DoGenerateResult
		err    error
	}

	results := make([]callResult, callCount)
	var wg sync.WaitGroup
	wg.Add(callCount)

	for i, count := range callImageCounts {
		go func(idx, imageCount int) {
			defer wg.Done()
			res, err := model.DoGenerate(ctx, DoGenerateOptions{
				Prompt:          opts.Prompt,
				N:               imageCount,
				Size:            opts.Size,
				AspectRatio:     opts.AspectRatio,
				Seed:            opts.Seed,
				Headers:         opts.Headers,
				ProviderOptions: opts.ProviderOptions,
			})
			results[idx] = callResult{result: res, err: err}
		}(i, count)
	}
	wg.Wait()

	// Collect results.
	var images []GeneratedFile
	var warnings []Warning
	var responses []ImageModelResponseMetadata
	providerMetadata := make(ImageModelProviderMetadata)
	totalUsage := ImageModelUsage{}

	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}

		for _, imgData := range r.result.Images {
			mediaType := detectImageMediaType(imgData)
			if mediaType == "" {
				mediaType = "image/png"
			}
			images = append(images, GeneratedFile{
				Data:      imgData,
				MediaType: mediaType,
			})
		}

		warnings = append(warnings, r.result.Warnings...)

		if r.result.Usage != nil {
			totalUsage = addImageModelUsage(totalUsage, *r.result.Usage)
		}

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
						merged[k] = v
					}
					providerMetadata[providerName] = merged
				}
			}
		}

		responses = append(responses, r.result.Response)
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("no image generated")
	}

	return &GenerateImageResult{
		Image:            images[0],
		Images:           images,
		Warnings:         warnings,
		Responses:        responses,
		ProviderMetadata: providerMetadata,
		Usage:            totalUsage,
	}, nil
}

// detectImageMediaType attempts to detect the media type from image data bytes.
func detectImageMediaType(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	// PNG: 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}
	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}
	// GIF: 47 49 46 38
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return "image/gif"
	}
	// WebP: 52 49 46 46 ... 57 45 42 50
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return "image/webp"
	}
	return ""
}

// addImageModelUsage adds two ImageModelUsage values together.
func addImageModelUsage(a, b ImageModelUsage) ImageModelUsage {
	result := ImageModelUsage{}
	if a.InputTokens != nil || b.InputTokens != nil {
		sum := 0
		if a.InputTokens != nil {
			sum += *a.InputTokens
		}
		if b.InputTokens != nil {
			sum += *b.InputTokens
		}
		result.InputTokens = &sum
	}
	if a.OutputTokens != nil || b.OutputTokens != nil {
		sum := 0
		if a.OutputTokens != nil {
			sum += *a.OutputTokens
		}
		if b.OutputTokens != nil {
			sum += *b.OutputTokens
		}
		result.OutputTokens = &sum
	}
	if a.TotalTokens != nil || b.TotalTokens != nil {
		sum := 0
		if a.TotalTokens != nil {
			sum += *a.TotalTokens
		}
		if b.TotalTokens != nil {
			sum += *b.TotalTokens
		}
		result.TotalTokens = &sum
	}
	return result
}
