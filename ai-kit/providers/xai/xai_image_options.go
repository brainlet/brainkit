// Ported from: packages/xai/src/xai-image-options.ts
package xai

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// XaiImageModelOptions represents xAI-specific options for image generation.
type XaiImageModelOptions struct {
	AspectRatio  *string `json:"aspect_ratio,omitempty"`
	OutputFormat *string `json:"output_format,omitempty"`
	SyncMode     *bool   `json:"sync_mode,omitempty"`
	Resolution   *string `json:"resolution,omitempty"` // "1k" or "2k"
}

// xaiImageModelOptionsSchema is the schema for validating xAI image options.
var xaiImageModelOptionsSchema = &providerutils.Schema[XaiImageModelOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[XaiImageModelOptions], error) {
		m, ok := value.(map[string]interface{})
		if !ok {
			return &providerutils.ValidationResult[XaiImageModelOptions]{Success: false}, nil
		}

		var opts XaiImageModelOptions

		if v, ok := m["aspect_ratio"].(string); ok {
			opts.AspectRatio = &v
		}
		if v, ok := m["output_format"].(string); ok {
			opts.OutputFormat = &v
		}
		if v, ok := m["sync_mode"].(bool); ok {
			opts.SyncMode = &v
		}
		if v, ok := m["resolution"].(string); ok {
			opts.Resolution = &v
		}

		return &providerutils.ValidationResult[XaiImageModelOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}
