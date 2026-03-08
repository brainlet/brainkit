// Ported from: packages/xai/src/xai-video-options.ts
package xai

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// XaiVideoModelOptions represents xAI-specific options for video generation.
type XaiVideoModelOptions struct {
	PollIntervalMs *int    `json:"pollIntervalMs,omitempty"`
	PollTimeoutMs  *int    `json:"pollTimeoutMs,omitempty"`
	Resolution     *string `json:"resolution,omitempty"` // "480p" or "720p"
	VideoURL       *string `json:"videoUrl,omitempty"`
}

// xaiVideoModelOptionsSchema is the schema for validating xAI video options.
var xaiVideoModelOptionsSchema = &providerutils.Schema[XaiVideoModelOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[XaiVideoModelOptions], error) {
		m, ok := value.(map[string]interface{})
		if !ok {
			return &providerutils.ValidationResult[XaiVideoModelOptions]{Success: false}, nil
		}

		var opts XaiVideoModelOptions

		if v, ok := m["pollIntervalMs"]; ok {
			if n, ok := toInt(v); ok {
				opts.PollIntervalMs = &n
			}
		}
		if v, ok := m["pollTimeoutMs"]; ok {
			if n, ok := toInt(v); ok {
				opts.PollTimeoutMs = &n
			}
		}
		if v, ok := m["resolution"].(string); ok {
			opts.Resolution = &v
		}
		if v, ok := m["videoUrl"].(string); ok {
			opts.VideoURL = &v
		}

		return &providerutils.ValidationResult[XaiVideoModelOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}
