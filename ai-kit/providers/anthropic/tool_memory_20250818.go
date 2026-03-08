// Ported from: packages/anthropic/src/tool/memory_20250818.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// Memory20250818Input is the input schema for the memory_20250818 tool.
// This is a discriminated union; the Command field indicates the variant.
type Memory20250818Input struct {
	Command    string   `json:"command"`
	Path       *string  `json:"path,omitempty"`
	ViewRange  *[2]int  `json:"view_range,omitempty"`
	FileText   *string  `json:"file_text,omitempty"`
	OldStr     *string  `json:"old_str,omitempty"`
	NewStr     *string  `json:"new_str,omitempty"`
	InsertLine *int     `json:"insert_line,omitempty"`
	InsertText *string  `json:"insert_text,omitempty"`
	OldPath    *string  `json:"old_path,omitempty"`
	NewPath    *string  `json:"new_path,omitempty"`
}

// Memory20250818 is the provider tool factory for the memory_20250818 tool.
var Memory20250818 = providerutils.CreateProviderToolFactory(providerutils.ProviderToolConfig[Memory20250818Input]{
	ID: "anthropic.memory_20250818",
})
