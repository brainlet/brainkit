// Ported from: packages/openai/src/tool/web-search-preview.ts
package openai

// WebSearchPreviewUserLocation contains user location information for search.
type WebSearchPreviewUserLocation struct {
	// Type is always "approximate".
	Type string `json:"type"`

	// Country is the two-letter ISO country code (e.g., "US", "GB").
	Country string `json:"country,omitempty"`

	// City is the city name (free text, e.g., "Minneapolis").
	City string `json:"city,omitempty"`

	// Region is the region name (free text, e.g., "Minnesota").
	Region string `json:"region,omitempty"`

	// Timezone is the IANA timezone (e.g., "America/Chicago").
	Timezone string `json:"timezone,omitempty"`
}

// WebSearchPreviewOutputAction represents the action taken in a web search call.
type WebSearchPreviewOutputAction struct {
	// Type is "search", "openPage", or "findInPage".
	Type string `json:"type"`

	// Query is the search query (for "search" type).
	Query string `json:"query,omitempty"`

	// URL is the URL (for "openPage" and "findInPage" types).
	URL *string `json:"url,omitempty"`

	// Pattern is the text to search for within the page (for "findInPage" type).
	Pattern *string `json:"pattern,omitempty"`
}

// WebSearchPreviewOutput is the output schema for the web_search_preview tool.
type WebSearchPreviewOutput struct {
	// Action describes the specific action taken in this web search call.
	Action *WebSearchPreviewOutputAction `json:"action,omitempty"`
}

// WebSearchPreviewArgs contains configuration options for the web_search_preview tool.
type WebSearchPreviewArgs struct {
	// SearchContextSize is the search context size: "low", "medium", or "high".
	SearchContextSize string `json:"searchContextSize,omitempty"`

	// UserLocation is user location information for geographically relevant results.
	UserLocation *WebSearchPreviewUserLocation `json:"userLocation,omitempty"`
}

// WebSearchPreviewToolID is the provider tool ID for web_search_preview.
const WebSearchPreviewToolID = "openai.web_search_preview"

// NewWebSearchPreviewTool creates a provider tool configuration for the web_search_preview tool.
func NewWebSearchPreviewTool(args *WebSearchPreviewArgs) map[string]interface{} {
	result := map[string]interface{}{
		"type": "provider",
		"id":   WebSearchPreviewToolID,
	}
	if args == nil {
		return result
	}
	if args.SearchContextSize != "" {
		result["searchContextSize"] = args.SearchContextSize
	}
	if args.UserLocation != nil {
		loc := map[string]interface{}{
			"type": "approximate",
		}
		if args.UserLocation.Country != "" {
			loc["country"] = args.UserLocation.Country
		}
		if args.UserLocation.City != "" {
			loc["city"] = args.UserLocation.City
		}
		if args.UserLocation.Region != "" {
			loc["region"] = args.UserLocation.Region
		}
		if args.UserLocation.Timezone != "" {
			loc["timezone"] = args.UserLocation.Timezone
		}
		result["userLocation"] = loc
	}
	return result
}
