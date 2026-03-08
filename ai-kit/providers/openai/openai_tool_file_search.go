// Ported from: packages/openai/src/tool/file-search.ts
package openai

// FileSearchComparisonFilter represents a comparison filter for file search.
type FileSearchComparisonFilter struct {
	// Key is the attribute key to filter on.
	Key string `json:"key"`

	// Type is the comparison operator: "eq", "ne", "gt", "gte", "lt", "lte", "in", "nin".
	Type string `json:"type"`

	// Value is the comparison value (string, number, boolean, or []string).
	Value interface{} `json:"value"`
}

// FileSearchCompoundFilter represents a compound (and/or) filter for file search.
type FileSearchCompoundFilter struct {
	// Type is the compound operator: "and" or "or".
	Type string `json:"type"`

	// Filters are the nested filters (can be comparison or compound).
	Filters []interface{} `json:"filters"`
}

// FileSearchRanking contains ranking options for the file search.
type FileSearchRanking struct {
	// Ranker is the ranker to use for the file search.
	Ranker string `json:"ranker,omitempty"`

	// ScoreThreshold is the score threshold for the file search, between 0 and 1.
	ScoreThreshold *float64 `json:"scoreThreshold,omitempty"`
}

// FileSearchResultItem represents a single file search result.
type FileSearchResultItem struct {
	// Attributes are key-value pairs attached to the object.
	Attributes map[string]interface{} `json:"attributes"`

	// FileID is the unique ID of the file.
	FileID string `json:"fileId"`

	// Filename is the name of the file.
	Filename string `json:"filename"`

	// Score is the relevance score between 0 and 1.
	Score float64 `json:"score"`

	// Text is the text retrieved from the file.
	Text string `json:"text"`
}

// FileSearchOutput is the output schema for the file_search tool.
type FileSearchOutput struct {
	// Queries are the search queries that were executed.
	Queries []string `json:"queries"`

	// Results are the search results, or nil if no results.
	Results []FileSearchResultItem `json:"results"`
}

// FileSearchArgs contains configuration options for the file_search tool.
type FileSearchArgs struct {
	// VectorStoreIds is a list of vector store IDs to search through.
	VectorStoreIds []string `json:"vectorStoreIds"`

	// MaxNumResults is the maximum number of search results to return.
	MaxNumResults *int `json:"maxNumResults,omitempty"`

	// Ranking contains ranking options for the search.
	Ranking *FileSearchRanking `json:"ranking,omitempty"`

	// Filters is an optional filter to apply (comparison or compound).
	Filters interface{} `json:"filters,omitempty"`
}

// FileSearchToolID is the provider tool ID for file_search.
const FileSearchToolID = "openai.file_search"

// NewFileSearchTool creates a provider tool configuration for the file_search tool.
func NewFileSearchTool(args FileSearchArgs) map[string]interface{} {
	result := map[string]interface{}{
		"type":           "provider",
		"id":             FileSearchToolID,
		"vectorStoreIds": args.VectorStoreIds,
	}
	if args.MaxNumResults != nil {
		result["maxNumResults"] = *args.MaxNumResults
	}
	if args.Ranking != nil {
		ranking := map[string]interface{}{}
		if args.Ranking.Ranker != "" {
			ranking["ranker"] = args.Ranking.Ranker
		}
		if args.Ranking.ScoreThreshold != nil {
			ranking["scoreThreshold"] = *args.Ranking.ScoreThreshold
		}
		result["ranking"] = ranking
	}
	if args.Filters != nil {
		result["filters"] = args.Filters
	}
	return result
}
