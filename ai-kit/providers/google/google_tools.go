// Ported from: packages/google/src/google-tools.ts
package google

// GoogleToolIDs contains all the tool IDs for Google provider tools.
// In Go, these are exported as constants in each tool_*.go file.
// This file provides a convenience struct that mirrors the TS googleTools object.

// GoogleTools groups all Google provider tool IDs.
var GoogleTools = struct {
	GoogleSearch       string
	EnterpriseWebSearch string
	GoogleMaps         string
	URLContext         string
	FileSearch         string
	CodeExecution      string
	VertexRagStore     string
}{
	GoogleSearch:        GoogleSearchToolID,
	EnterpriseWebSearch: EnterpriseWebSearchToolID,
	GoogleMaps:          GoogleMapsToolID,
	URLContext:          URLContextToolID,
	FileSearch:          FileSearchToolID,
	CodeExecution:       CodeExecutionToolID,
	VertexRagStore:      VertexRagStoreToolID,
}
