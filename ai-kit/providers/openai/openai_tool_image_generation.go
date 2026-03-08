// Ported from: packages/openai/src/tool/image-generation.ts
package openai

// ImageGenerationInputImageMask contains mask configuration for inpainting.
type ImageGenerationInputImageMask struct {
	// FileID is the file ID for the mask image.
	FileID string `json:"fileId,omitempty"`

	// ImageURL is the base64-encoded mask image URL.
	ImageURL string `json:"imageUrl,omitempty"`
}

// ImageGenerationOutput is the output schema for the image_generation tool.
type ImageGenerationOutput struct {
	// Result is the generated image encoded in base64.
	Result string `json:"result"`
}

// ImageGenerationArgs contains configuration options for the image_generation tool.
type ImageGenerationArgs struct {
	// Background type for the generated image. Default is "auto".
	Background string `json:"background,omitempty"`

	// InputFidelity for the generated image. Default is "low".
	InputFidelity string `json:"inputFidelity,omitempty"`

	// InputImageMask is an optional mask for inpainting.
	InputImageMask *ImageGenerationInputImageMask `json:"inputImageMask,omitempty"`

	// Model is the image generation model to use. Default: "gpt-image-1".
	Model string `json:"model,omitempty"`

	// Moderation level for the generated image. Default: "auto".
	Moderation string `json:"moderation,omitempty"`

	// OutputCompression is the compression level (0-100). Default: 100.
	OutputCompression *int `json:"outputCompression,omitempty"`

	// OutputFormat is the output format: "png", "jpeg", or "webp". Default: "png".
	OutputFormat string `json:"outputFormat,omitempty"`

	// PartialImages is the number of partial images (0-3). Default: 0.
	PartialImages *int `json:"partialImages,omitempty"`

	// Quality of the generated image: "auto", "low", "medium", "high". Default: "auto".
	Quality string `json:"quality,omitempty"`

	// Size of the generated image: "1024x1024", "1024x1536", "1536x1024", "auto". Default: "auto".
	Size string `json:"size,omitempty"`
}

// ImageGenerationToolID is the provider tool ID for image_generation.
const ImageGenerationToolID = "openai.image_generation"

// NewImageGenerationTool creates a provider tool configuration for the image_generation tool.
func NewImageGenerationTool(args *ImageGenerationArgs) map[string]interface{} {
	result := map[string]interface{}{
		"type": "provider",
		"id":   ImageGenerationToolID,
	}
	if args == nil {
		return result
	}
	if args.Background != "" {
		result["background"] = args.Background
	}
	if args.InputFidelity != "" {
		result["inputFidelity"] = args.InputFidelity
	}
	if args.InputImageMask != nil {
		mask := map[string]interface{}{}
		if args.InputImageMask.FileID != "" {
			mask["fileId"] = args.InputImageMask.FileID
		}
		if args.InputImageMask.ImageURL != "" {
			mask["imageUrl"] = args.InputImageMask.ImageURL
		}
		result["inputImageMask"] = mask
	}
	if args.Model != "" {
		result["model"] = args.Model
	}
	if args.Moderation != "" {
		result["moderation"] = args.Moderation
	}
	if args.OutputCompression != nil {
		result["outputCompression"] = *args.OutputCompression
	}
	if args.OutputFormat != "" {
		result["outputFormat"] = args.OutputFormat
	}
	if args.PartialImages != nil {
		result["partialImages"] = *args.PartialImages
	}
	if args.Quality != "" {
		result["quality"] = args.Quality
	}
	if args.Size != "" {
		result["size"] = args.Size
	}
	return result
}
