// Ported from: packages/ai/src/middleware/default-settings-middleware.ts
package middleware

import (
	lm "github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	mw "github.com/brainlet/brainkit/ai-kit/provider/middleware"
	"github.com/brainlet/brainkit/ai-kit/ai/util"
)

// DefaultSettings holds the default settings to apply for a language model.
type DefaultSettings struct {
	MaxOutputTokens  *int
	Temperature      *float64
	StopSequences    []string
	TopP             *float64
	TopK             *int
	PresencePenalty  *float64
	FrequencyPenalty *float64
	ResponseFormat   lm.ResponseFormat
	Seed             *int
	Tools            []lm.Tool
	ToolChoice       lm.ToolChoice
	Headers          map[string]*string
	ProviderOptions  map[string]map[string]any
}

// settingsToMap converts DefaultSettings to a map for MergeObjects.
func settingsToMap(s DefaultSettings) map[string]interface{} {
	m := make(map[string]interface{})
	if s.MaxOutputTokens != nil {
		m["maxOutputTokens"] = s.MaxOutputTokens
	}
	if s.Temperature != nil {
		m["temperature"] = s.Temperature
	}
	if s.StopSequences != nil {
		m["stopSequences"] = s.StopSequences
	}
	if s.TopP != nil {
		m["topP"] = s.TopP
	}
	if s.TopK != nil {
		m["topK"] = s.TopK
	}
	if s.PresencePenalty != nil {
		m["presencePenalty"] = s.PresencePenalty
	}
	if s.FrequencyPenalty != nil {
		m["frequencyPenalty"] = s.FrequencyPenalty
	}
	if s.ResponseFormat != nil {
		m["responseFormat"] = s.ResponseFormat
	}
	if s.Seed != nil {
		m["seed"] = s.Seed
	}
	if s.Tools != nil {
		m["tools"] = s.Tools
	}
	if s.ToolChoice != nil {
		m["toolChoice"] = s.ToolChoice
	}
	if s.Headers != nil {
		hm := make(map[string]interface{})
		for k, v := range s.Headers {
			hm[k] = v
		}
		m["headers"] = hm
	}
	if s.ProviderOptions != nil {
		// Convert to map[string]interface{} for deep merging
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

// callOptionsToMap converts CallOptions to a map for MergeObjects.
func callOptionsToMap(opts lm.CallOptions) map[string]interface{} {
	m := make(map[string]interface{})
	// Always include prompt
	m["prompt"] = opts.Prompt
	if opts.MaxOutputTokens != nil {
		m["maxOutputTokens"] = opts.MaxOutputTokens
	}
	if opts.Temperature != nil {
		m["temperature"] = opts.Temperature
	}
	if opts.StopSequences != nil {
		m["stopSequences"] = opts.StopSequences
	}
	if opts.TopP != nil {
		m["topP"] = opts.TopP
	}
	if opts.TopK != nil {
		m["topK"] = opts.TopK
	}
	if opts.PresencePenalty != nil {
		m["presencePenalty"] = opts.PresencePenalty
	}
	if opts.FrequencyPenalty != nil {
		m["frequencyPenalty"] = opts.FrequencyPenalty
	}
	if opts.ResponseFormat != nil {
		m["responseFormat"] = opts.ResponseFormat
	}
	if opts.Seed != nil {
		m["seed"] = opts.Seed
	}
	if opts.Tools != nil {
		m["tools"] = opts.Tools
	}
	if opts.ToolChoice != nil {
		m["toolChoice"] = opts.ToolChoice
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
	if opts.IncludeRawChunks != nil {
		m["includeRawChunks"] = opts.IncludeRawChunks
	}
	return m
}

// mapToCallOptions converts a merged map back to CallOptions.
func mapToCallOptions(m map[string]interface{}) lm.CallOptions {
	opts := lm.CallOptions{}
	if v, ok := m["prompt"]; ok && v != nil {
		opts.Prompt = v.(lm.Prompt)
	}
	if v, ok := m["maxOutputTokens"]; ok && v != nil {
		opts.MaxOutputTokens = v.(*int)
	}
	if v, ok := m["temperature"]; ok && v != nil {
		opts.Temperature = v.(*float64)
	}
	if v, ok := m["stopSequences"]; ok && v != nil {
		opts.StopSequences = v.([]string)
	}
	if v, ok := m["topP"]; ok && v != nil {
		opts.TopP = v.(*float64)
	}
	if v, ok := m["topK"]; ok && v != nil {
		opts.TopK = v.(*int)
	}
	if v, ok := m["presencePenalty"]; ok && v != nil {
		opts.PresencePenalty = v.(*float64)
	}
	if v, ok := m["frequencyPenalty"]; ok && v != nil {
		opts.FrequencyPenalty = v.(*float64)
	}
	if v, ok := m["responseFormat"]; ok && v != nil {
		opts.ResponseFormat = v.(lm.ResponseFormat)
	}
	if v, ok := m["seed"]; ok && v != nil {
		opts.Seed = v.(*int)
	}
	if v, ok := m["tools"]; ok && v != nil {
		opts.Tools = v.([]lm.Tool)
	}
	if v, ok := m["toolChoice"]; ok && v != nil {
		opts.ToolChoice = v.(lm.ToolChoice)
	}
	if v, ok := m["headers"]; ok && v != nil {
		switch hv := v.(type) {
		case map[string]*string:
			opts.Headers = hv
		case map[string]interface{}:
			hm := make(map[string]*string)
			for hk, hval := range hv {
				if hval == nil {
					hm[hk] = nil
				} else if sp, ok2 := hval.(*string); ok2 {
					hm[hk] = sp
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

// DefaultSettingsMiddleware applies default settings for a language model.
// The settings are used as defaults — user-provided params take precedence.
func DefaultSettingsMiddleware(settings DefaultSettings) mw.LanguageModelMiddleware {
	return mw.LanguageModelMiddleware{
		TransformParams: func(opts mw.TransformParamsOptions) (lm.CallOptions, error) {
			base := settingsToMap(settings)
			override := callOptionsToMap(opts.Params)
			merged := util.MergeObjects(base, override)
			return mapToCallOptions(merged), nil
		},
	}
}
