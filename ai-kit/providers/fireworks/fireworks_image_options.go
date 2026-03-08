// Ported from: packages/fireworks/src/fireworks-image-options.ts
package fireworks

// FireworksImageModelID represents a Fireworks image model identifier.
// https://fireworks.ai/models?type=image
type FireworksImageModelID = string

const (
	FireworksImageModelFlux1DevFp8                     FireworksImageModelID = "accounts/fireworks/models/flux-1-dev-fp8"
	FireworksImageModelFlux1SchnellFp8                 FireworksImageModelID = "accounts/fireworks/models/flux-1-schnell-fp8"
	FireworksImageModelFluxKontextPro                  FireworksImageModelID = "accounts/fireworks/models/flux-kontext-pro"
	FireworksImageModelFluxKontextMax                  FireworksImageModelID = "accounts/fireworks/models/flux-kontext-max"
	FireworksImageModelPlaygroundV2p5_1024pxAesthetic  FireworksImageModelID = "accounts/fireworks/models/playground-v2-5-1024px-aesthetic"
	FireworksImageModelJapaneseStableDiffusionXL       FireworksImageModelID = "accounts/fireworks/models/japanese-stable-diffusion-xl"
	FireworksImageModelPlaygroundV2_1024pxAesthetic    FireworksImageModelID = "accounts/fireworks/models/playground-v2-1024px-aesthetic"
	FireworksImageModelSSD1B                           FireworksImageModelID = "accounts/fireworks/models/SSD-1B"
	FireworksImageModelStableDiffusionXL1024V1_0       FireworksImageModelID = "accounts/fireworks/models/stable-diffusion-xl-1024-v1-0"
)
