// Ported from: packages/togetherai/src/togetherai-image-settings.ts
package togetherai

// https://api.together.ai/models

// TogetherAIImageModelID is the type for Together AI image model identifiers.
// Known model IDs are provided as constants; any string is accepted.
type TogetherAIImageModelID = string

// Known Together AI image model IDs.
const (
	// Text-to-image models
	ImageModelStableDiffusionXLBase1_0 TogetherAIImageModelID = "stabilityai/stable-diffusion-xl-base-1.0"
	ImageModelFLUX1Dev                 TogetherAIImageModelID = "black-forest-labs/FLUX.1-dev"
	ImageModelFLUX1DevLora             TogetherAIImageModelID = "black-forest-labs/FLUX.1-dev-lora"
	ImageModelFLUX1Schnell             TogetherAIImageModelID = "black-forest-labs/FLUX.1-schnell"
	ImageModelFLUX1Canny               TogetherAIImageModelID = "black-forest-labs/FLUX.1-canny"
	ImageModelFLUX1Depth               TogetherAIImageModelID = "black-forest-labs/FLUX.1-depth"
	ImageModelFLUX1Redux               TogetherAIImageModelID = "black-forest-labs/FLUX.1-redux"
	ImageModelFLUX1_1Pro               TogetherAIImageModelID = "black-forest-labs/FLUX.1.1-pro"
	ImageModelFLUX1Pro                 TogetherAIImageModelID = "black-forest-labs/FLUX.1-pro"
	ImageModelFLUX1SchnellFree         TogetherAIImageModelID = "black-forest-labs/FLUX.1-schnell-Free"

	// FLUX Kontext models for image editing
	ImageModelFLUX1KontextPro TogetherAIImageModelID = "black-forest-labs/FLUX.1-kontext-pro"
	ImageModelFLUX1KontextMax TogetherAIImageModelID = "black-forest-labs/FLUX.1-kontext-max"
	ImageModelFLUX1KontextDev TogetherAIImageModelID = "black-forest-labs/FLUX.1-kontext-dev"
)
