// Ported from: packages/anthropic/src/tool/text-editor_20250124.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// TextEditor20250124Input is the input schema for the text_editor_20250124 tool.
type TextEditor20250124Input struct {
	Command    string  `json:"command"`
	Path       string  `json:"path"`
	FileText   *string `json:"file_text,omitempty"`
	InsertLine *int    `json:"insert_line,omitempty"`
	NewStr     *string `json:"new_str,omitempty"`
	InsertText *string `json:"insert_text,omitempty"`
	OldStr     *string `json:"old_str,omitempty"`
	ViewRange  []int   `json:"view_range,omitempty"`
}

// TextEditor20250124 is the provider tool factory for the text_editor_20250124 tool.
var TextEditor20250124 = providerutils.CreateProviderToolFactory(providerutils.ProviderToolConfig[TextEditor20250124Input]{
	ID: "anthropic.text_editor_20250124",
})
