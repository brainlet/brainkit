// Ported from: packages/xai/src/xai-chat-options.ts
package xai

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// XaiChatModelId is the type alias for xAI chat model identifiers.
// Known values include "grok-4-1-fast-reasoning", "grok-4-1-fast-non-reasoning",
// "grok-4-fast-non-reasoning", "grok-4-fast-reasoning", "grok-code-fast-1",
// "grok-4", "grok-4-0709", "grok-4-latest", "grok-3", "grok-3-latest",
// "grok-3-mini", "grok-3-mini-latest", "grok-2-vision-1212", "grok-2-vision",
// "grok-2-vision-latest", "grok-2-image-1212", "grok-2-image", "grok-2-image-latest",
// or any other string.
type XaiChatModelId = string

// SearchSource represents a search source configuration.
type SearchSource struct {
	Type               string   `json:"type"`                          // "web", "x", "news", "rss"
	Country            *string  `json:"country,omitempty"`             // web, news
	ExcludedWebsites   []string `json:"excludedWebsites,omitempty"`    // web, news
	AllowedWebsites    []string `json:"allowedWebsites,omitempty"`     // web
	SafeSearch         *bool    `json:"safeSearch,omitempty"`          // web, news
	ExcludedXHandles   []string `json:"excludedXHandles,omitempty"`    // x
	IncludedXHandles   []string `json:"includedXHandles,omitempty"`    // x
	PostFavoriteCount  *int     `json:"postFavoriteCount,omitempty"`   // x
	PostViewCount      *int     `json:"postViewCount,omitempty"`       // x
	XHandles           []string `json:"xHandles,omitempty"`            // x (deprecated)
	Links              []string `json:"links,omitempty"`               // rss
}

// SearchParameters contains xAI search configuration.
type SearchParameters struct {
	// Mode is the search mode: "off", "auto", or "on".
	Mode string `json:"mode"`

	// ReturnCitations indicates whether to return citations. Defaults to true.
	ReturnCitations *bool `json:"returnCitations,omitempty"`

	// FromDate is the start date for search data (ISO8601 format: YYYY-MM-DD).
	FromDate *string `json:"fromDate,omitempty"`

	// ToDate is the end date for search data (ISO8601 format: YYYY-MM-DD).
	ToDate *string `json:"toDate,omitempty"`

	// MaxSearchResults is the maximum number of search results. Defaults to 20.
	MaxSearchResults *int `json:"maxSearchResults,omitempty"`

	// Sources are the data sources to search from.
	Sources []SearchSource `json:"sources,omitempty"`
}

// XaiLanguageModelChatOptions represents xAI-specific provider options for chat models.
type XaiLanguageModelChatOptions struct {
	ReasoningEffort        *string           `json:"reasoningEffort,omitempty"`
	Logprobs               *bool             `json:"logprobs,omitempty"`
	TopLogprobs            *int              `json:"topLogprobs,omitempty"`
	ParallelFunctionCalling *bool            `json:"parallel_function_calling,omitempty"`
	SearchParameters       *SearchParameters `json:"searchParameters,omitempty"`
}

// xaiLanguageModelChatOptionsSchema is the schema for validating xAI chat options.
var xaiLanguageModelChatOptionsSchema = &providerutils.Schema[XaiLanguageModelChatOptions]{
	Validate: func(value interface{}) (*providerutils.ValidationResult[XaiLanguageModelChatOptions], error) {
		m, ok := value.(map[string]interface{})
		if !ok {
			return &providerutils.ValidationResult[XaiLanguageModelChatOptions]{Success: false}, nil
		}

		var opts XaiLanguageModelChatOptions

		if v, ok := m["reasoningEffort"].(string); ok {
			opts.ReasoningEffort = &v
		}
		if v, ok := m["logprobs"].(bool); ok {
			opts.Logprobs = &v
		}
		if v, ok := m["topLogprobs"]; ok {
			if n, ok := toInt(v); ok {
				opts.TopLogprobs = &n
			}
		}
		if v, ok := m["parallel_function_calling"].(bool); ok {
			opts.ParallelFunctionCalling = &v
		}
		if sp, ok := m["searchParameters"].(map[string]interface{}); ok {
			params := &SearchParameters{}
			if mode, ok := sp["mode"].(string); ok {
				params.Mode = mode
			}
			if v, ok := sp["returnCitations"].(bool); ok {
				params.ReturnCitations = &v
			}
			if v, ok := sp["fromDate"].(string); ok {
				params.FromDate = &v
			}
			if v, ok := sp["toDate"].(string); ok {
				params.ToDate = &v
			}
			if v, ok := sp["maxSearchResults"]; ok {
				if n, ok := toInt(v); ok {
					params.MaxSearchResults = &n
				}
			}
			if sources, ok := sp["sources"].([]interface{}); ok {
				for _, s := range sources {
					if sm, ok := s.(map[string]interface{}); ok {
						src := SearchSource{}
						if t, ok := sm["type"].(string); ok {
							src.Type = t
						}
						if v, ok := sm["country"].(string); ok {
							src.Country = &v
						}
						src.ExcludedWebsites = toStringSlice(sm["excludedWebsites"])
						src.AllowedWebsites = toStringSlice(sm["allowedWebsites"])
						if v, ok := sm["safeSearch"].(bool); ok {
							src.SafeSearch = &v
						}
						src.ExcludedXHandles = toStringSlice(sm["excludedXHandles"])
						src.IncludedXHandles = toStringSlice(sm["includedXHandles"])
						if v, ok := sm["postFavoriteCount"]; ok {
							if n, ok := toInt(v); ok {
								src.PostFavoriteCount = &n
							}
						}
						if v, ok := sm["postViewCount"]; ok {
							if n, ok := toInt(v); ok {
								src.PostViewCount = &n
							}
						}
						src.XHandles = toStringSlice(sm["xHandles"])
						src.Links = toStringSlice(sm["links"])
						params.Sources = append(params.Sources, src)
					}
				}
			}
			opts.SearchParameters = params
		}

		return &providerutils.ValidationResult[XaiLanguageModelChatOptions]{
			Success: true,
			Value:   opts,
		}, nil
	},
}

// toInt converts a numeric value to int.
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

// toStringSlice converts an interface{} to a []string.
func toStringSlice(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	result := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
