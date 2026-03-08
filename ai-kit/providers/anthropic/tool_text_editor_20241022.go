// Ported from: packages/anthropic/src/tool/text-editor_20241022.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// TextEditor20241022Input is the input schema for the text_editor_20241022 tool.
type TextEditor20241022Input struct {
	Command    string  `json:"command"`
	Path       string  `json:"path"`
	FileText   *string `json:"file_text,omitempty"`
	InsertLine *int    `json:"insert_line,omitempty"`
	NewStr     *string `json:"new_str,omitempty"`
	InsertText *string `json:"insert_text,omitempty"`
	OldStr     *string `json:"old_str,omitempty"`
	ViewRange  []int   `json:"view_range,omitempty"`
}

// TextEditor20241022 is the provider tool factory for the text_editor_20241022 tool.
var TextEditor20241022 = providerutils.CreateProviderToolFactory(providerutils.ProviderToolConfig[TextEditor20241022Input]{
	ID: "anthropic.text_editor_20241022",
})
