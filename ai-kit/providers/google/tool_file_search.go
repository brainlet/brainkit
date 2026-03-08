// Ported from: packages/google/src/tool/file-search.ts
package google

// FileSearchToolID is the tool ID for file search.
const FileSearchToolID = "google.file_search"

// FileSearchToolArgs contains the arguments for the file search tool.
type FileSearchToolArgs struct {
	// FileSearchStoreNames are the names of the file_search_stores to retrieve from.
	// Example: fileSearchStores/my-file-search-store-123
	FileSearchStoreNames []string `json:"fileSearchStoreNames"`

	// TopK is the number of file search retrieval chunks to retrieve.
	TopK *int `json:"topK,omitempty"`

	// MetadataFilter is a filter expression to restrict the files that can be retrieved.
	// See https://google.aip.dev/160 for the syntax.
	MetadataFilter *string `json:"metadataFilter,omitempty"`
}
