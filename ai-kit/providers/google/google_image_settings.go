// Ported from: packages/google/src/google-generative-ai-image-settings.ts
package google

// GoogleImageSettings contains configuration for Google Generative AI image models.
type GoogleImageSettings struct {
	// MaxImagesPerCall overrides the maximum number of images per call (default 4).
	MaxImagesPerCall *int
}
