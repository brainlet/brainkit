// Ported from: packages/provider/src/video-model/v3/video-model-v3-call-options.ts
package videomodel

import (
	"context"

	"github.com/brainlet/brainkit/ai-kit/provider/shared"
)

// CallOptions contains the options for a video model call.
type CallOptions struct {
	// Prompt is the text prompt for the video generation.
	Prompt *string

	// N is the number of videos to generate. Default: 1.
	N int

	// AspectRatio of the videos to generate (e.g. "16:9", "9:16", "1:1").
	AspectRatio *string

	// Resolution of the video (e.g. "1280x720", "1920x1080").
	Resolution *string

	// Duration of the video in seconds.
	Duration *float64

	// FPS is the frames per second for the video.
	FPS *int

	// Seed for deterministic video generation.
	Seed *int

	// Image is an input image for image-to-video generation.
	Image File

	// ProviderOptions are additional provider-specific options.
	ProviderOptions shared.ProviderOptions

	// Ctx is the context for cancellation (replaces AbortSignal in TS).
	Ctx context.Context

	// Headers are additional HTTP headers to be sent with the request.
	Headers map[string]*string
}
