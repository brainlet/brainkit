// Ported from: packages/openai/src/tool/web-search.ts
package openai

// WebSearchUserLocation contains user location information for search.
type WebSearchUserLocation struct {
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

// WebSearchFilters contains filter options for web search.
type WebSearchFilters struct {
	// AllowedDomains limits search to specific domains.
	// Subdomains of the provided domains are allowed as well.
	AllowedDomains []string `json:"allowedDomains,omitempty"`
}

// WebSearchOutputAction represents the action taken in a web search call.
type WebSearchOutputAction struct {
	// Type is "search", "openPage", or "findInPage".
	Type string `json:"type"`

	// Query is the search query (for "search" type).
	Query string `json:"query,omitempty"`

	// URL is the URL (for "openPage" and "findInPage" types).
	URL *string `json:"url,omitempty"`

	// Pattern is the text to search for within the page (for "findInPage" type).
	Pattern *string `json:"pattern,omitempty"`
}

// WebSearchOutputSource represents a source cited in web search results.
type WebSearchOutputSource struct {
	// Type is "url" or "api".
	Type string `json:"type"`

	// URL is the source URL (when Type is "url").
	URL string `json:"url,omitempty"`

	// Name is the API name (when Type is "api").
	Name string `json:"name,omitempty"`
}

// WebSearchOutput is the output schema for the web_search tool.
type WebSearchOutput struct {
	// Action describes the specific action taken in this web search call.
	Action *WebSearchOutputAction `json:"action,omitempty"`

	// Sources are the sources cited by the model.
	Sources []WebSearchOutputSource `json:"sources,omitempty"`
}

// WebSearchArgs contains configuration options for the web_search tool.
type WebSearchArgs struct {
	// ExternalWebAccess indicates whether to use external web access for fetching live content.
	ExternalWebAccess *bool `json:"externalWebAccess,omitempty"`

	// Filters contains search filters.
	Filters *WebSearchFilters `json:"filters,omitempty"`

	// SearchContextSize is the search context size: "low", "medium", or "high".
	SearchContextSize string `json:"searchContextSize,omitempty"`

	// UserLocation is user location information for geographically relevant results.
	UserLocation *WebSearchUserLocation `json:"userLocation,omitempty"`
}

// WebSearchToolID is the provider tool ID for web_search.
const WebSearchToolID = "openai.web_search"

// NewWebSearchTool creates a provider tool configuration for the web_search tool.
func NewWebSearchTool(args *WebSearchArgs) map[string]interface{} {
	result := map[string]interface{}{
		"type": "provider",
		"id":   WebSearchToolID,
	}
	if args == nil {
		return result
	}
	if args.ExternalWebAccess != nil {
		result["externalWebAccess"] = *args.ExternalWebAccess
	}
	if args.Filters != nil {
		filters := map[string]interface{}{}
		if args.Filters.AllowedDomains != nil {
			filters["allowedDomains"] = args.Filters.AllowedDomains
		}
		result["filters"] = filters
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
