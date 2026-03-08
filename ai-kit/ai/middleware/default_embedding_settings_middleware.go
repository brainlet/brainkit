// Ported from: packages/ai/src/middleware/default-embedding-settings-middleware.ts
package middleware

import (
	em "github.com/brainlet/brainkit/ai-kit/provider/embeddingmodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
	"github.com/brainlet/brainkit/ai-kit/ai/util"
)

// DefaultEmbeddingSettings holds the default settings to apply for an embedding model.
type DefaultEmbeddingSettings struct {
	Headers         map[string]string
	ProviderOptions map[string]map[string]any
}

// embeddingSettingsToMap converts DefaultEmbeddingSettings to a map for MergeObjects.
func embeddingSettingsToMap(s DefaultEmbeddingSettings) map[string]interface{} {
	m := make(map[string]interface{})
	if s.Headers != nil {
		hm := make(map[string]interface{})
		for k, v := range s.Headers {
			hm[k] = v
		}
		m["headers"] = hm
	}
	if s.ProviderOptions != nil {
		po := make(map[string]interface{})
		for k, v := range s.ProviderOptions {
			inner := make(map[string]interface{})
			for ik, iv := range v {
				inner[ik] = iv
			}
			po[k] = inner
		}
		m["providerOptions"] = po
	}
	return m
}

// embeddingCallOptionsToMap converts em.CallOptions to a map for MergeObjects.
func embeddingCallOptionsToMap(opts em.CallOptions) map[string]interface{} {
	m := make(map[string]interface{})
	if opts.Values != nil {
		m["values"] = opts.Values
	}
	if opts.Headers != nil {
		hm := make(map[string]interface{})
		for k, v := range opts.Headers {
			hm[k] = v
		}
		m["headers"] = hm
	}
	if opts.ProviderOptions != nil {
		po := make(map[string]interface{})
		for k, v := range opts.ProviderOptions {
			inner := make(map[string]interface{})
			for ik, iv := range v {
				inner[ik] = iv
			}
			po[k] = inner
		}
		m["providerOptions"] = po
	}
	if opts.Ctx != nil {
		m["ctx"] = opts.Ctx
	}
	return m
}

// mapToEmbeddingCallOptions converts a merged map back to em.CallOptions.
func mapToEmbeddingCallOptions(m map[string]interface{}) em.CallOptions {
	opts := em.CallOptions{}
	if v, ok := m["values"]; ok && v != nil {
		opts.Values = v.([]string)
	}
	if v, ok := m["headers"]; ok && v != nil {
		switch hv := v.(type) {
		case map[string]string:
			opts.Headers = hv
		case map[string]interface{}:
			hm := make(map[string]string)
			for hk, hval := range hv {
				if s, ok2 := hval.(string); ok2 {
					hm[hk] = s
				}
			}
			opts.Headers = hm
		}
	}
	if v, ok := m["providerOptions"]; ok && v != nil {
		if poMap, ok2 := v.(map[string]interface{}); ok2 {
			po := make(map[string]map[string]any)
			for k, val := range poMap {
				if innerMap, ok3 := val.(map[string]interface{}); ok3 {
					po[k] = innerMap
				}
			}
			opts.ProviderOptions = po
		}
	}
	return opts
}

// DefaultEmbeddingSettingsMiddleware applies default settings for an embedding model.
// The settings are used as defaults — user-provided params take precedence.
func DefaultEmbeddingSettingsMiddleware(settings DefaultEmbeddingSettings) mw.EmbeddingModelMiddleware {
	return mw.EmbeddingModelMiddleware{
		TransformParams: func(opts mw.EmbeddingTransformParamsOptions) (em.CallOptions, error) {
			base := embeddingSettingsToMap(settings)
			override := embeddingCallOptionsToMap(opts.Params)
			merged := util.MergeObjects(base, override)
			return mapToEmbeddingCallOptions(merged), nil
		},
	}
}
