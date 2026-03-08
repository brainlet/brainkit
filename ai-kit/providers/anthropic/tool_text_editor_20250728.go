// Ported from: packages/anthropic/src/tool/text-editor_20250728.ts
package anthropic

import "github.com/brainlet/brainkit/ai-kit/providerutils"

// TextEditor20250728ArgsSchema is the args schema for the text_editor_20250728 tool,
// used by prepareTools for validation.
type TextEditor20250728Args struct {
	MaxCharacters *int `json:"maxCharacters,omitempty"`
}

// TextEditor20250728Input is the input schema for the text_editor_20250728 tool.
type TextEditor20250728Input struct {
	Command    string  `json:"command"`
	Path       string  `json:"path"`
	FileText   *string `json:"file_text,omitempty"`
	InsertLine *int    `json:"insert_line,omitempty"`
	NewStr     *string `json:"new_str,omitempty"`
	InsertText *string `json:"insert_text,omitempty"`
	OldStr     *string `json:"old_str,omitempty"`
	ViewRange  []int   `json:"view_range,omitempty"`
}

// TextEditor20250728 is the provider tool factory for the text_editor_20250728 tool.
var TextEditor20250728 = providerutils.CreateProviderToolFactory(providerutils.ProviderToolConfig[TextEditor20250728Input]{
	ID: "anthropic.text_editor_20250728",
})
