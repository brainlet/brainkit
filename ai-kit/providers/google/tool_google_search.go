// Ported from: packages/google/src/tool/google-search.ts
package google

// GoogleSearchToolID is the tool ID for Google Search grounding.
const GoogleSearchToolID = "google.google_search"

// GoogleSearchToolArgs contains the arguments for the Google Search tool.
type GoogleSearchToolArgs struct {
	SearchTypes     *GoogleSearchTypes     `json:"searchTypes,omitempty"`
	TimeRangeFilter *GoogleTimeRangeFilter `json:"timeRangeFilter,omitempty"`
}

// GoogleSearchTypes specifies which search types to use.
type GoogleSearchTypes struct {
	WebSearch   *struct{} `json:"webSearch,omitempty"`
	ImageSearch *struct{} `json:"imageSearch,omitempty"`
}

// GoogleTimeRangeFilter specifies a time range filter for search results.
type GoogleTimeRangeFilter struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
}
