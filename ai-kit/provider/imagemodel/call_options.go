// Ported from: packages/provider/src/image-model/v3/image-model-v3-call-options.ts
package imagemodel

import (
	"context"

	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// CallOptions contains the options for an image model call.
type CallOptions struct {
	// Prompt for the image generation. Some operations, like upscaling, may not require a prompt.
	Prompt *string

	// N is the number of images to generate.
	N int

	// Size of the images to generate. Must have the format "{width}x{height}".
	// nil will use the provider's default size.
	Size *string

	// AspectRatio of the images to generate. Must have the format "{width}:{height}".
	// nil will use the provider's default aspect ratio.
	AspectRatio *string

	// Seed for the image generation.
	// nil will use the provider's default seed.
	Seed *int

	// Files is an array of images for image editing or variation generation.
	Files []File

	// Mask is a mask image for inpainting operations.
	Mask File

	// ProviderOptions are additional provider-specific options passed through
	// as body parameters.
	ProviderOptions shared.ProviderOptions

	// Ctx is the context for cancellation (replaces AbortSignal in TS).
	Ctx context.Context

	// Headers are additional HTTP headers to be sent with the request.
	Headers map[string]*string
}
