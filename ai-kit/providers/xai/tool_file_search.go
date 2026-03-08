// Ported from: packages/xai/src/tool/file-search.ts
package xai

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// FileSearchInput is the input schema for the file search tool (empty, args are passed via ProviderTool.Args).
type FileSearchInput struct{}

// FileSearchOutput is the output of the file search tool.
type FileSearchOutput struct {
	Queries []string                `json:"queries"`
	Results []FileSearchResultItem  `json:"results"`
}

// FileSearchResultItem is a single file search result.
type FileSearchResultItem struct {
	FileID   string  `json:"fileId"`
	Filename string  `json:"filename"`
	Score    float64 `json:"score"`
	Text     string  `json:"text"`
}

// FileSearchArgs are the arguments for the file search tool.
type FileSearchArgs struct {
	// VectorStoreIds is a list of vector store IDs (collection IDs) to search through.
	VectorStoreIds []string `json:"vectorStoreIds"`
	// MaxNumResults is the maximum number of search results to return.
	MaxNumResults *int `json:"maxNumResults,omitempty"`
}

// fileSearchToolFactory is the factory for the file search tool.
var fileSearchToolFactory = providerutils.CreateProviderToolFactoryWithOutputSchema(
	providerutils.ProviderToolWithOutputSchemaConfig[FileSearchInput, FileSearchOutput]{
		ID:           "xai.file_search",
		InputSchema:  &providerutils.Schema[FileSearchInput]{},
		OutputSchema: &providerutils.Schema[FileSearchOutput]{},
	},
)

// FileSearch creates a file search provider tool.
func FileSearch(opts providerutils.ProviderToolOptions[FileSearchInput, FileSearchOutput]) providerutils.ProviderTool[FileSearchInput, FileSearchOutput] {
	return fileSearchToolFactory(opts)
}
