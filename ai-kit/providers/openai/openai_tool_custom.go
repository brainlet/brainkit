// Ported from: packages/openai/src/tool/custom.ts
package openai

// CustomToolFormat represents the output format specification for a custom tool.
type CustomToolFormat interface {
	customToolFormatType() string
}

// CustomToolFormatGrammar is a grammar-based format specification.
type CustomToolFormatGrammar struct {
	// Syntax is the grammar syntax type: "regex" or "lark".
	Syntax string `json:"syntax"`

	// Definition is the grammar definition.
	Definition string `json:"definition"`
}

func (CustomToolFormatGrammar) customToolFormatType() string { return "grammar" }

// CustomToolFormatText is a plain text format specification.
type CustomToolFormatText struct{}

func (CustomToolFormatText) customToolFormatType() string { return "text" }

// CustomToolArgs contains configuration options for a custom tool.
type CustomToolArgs struct {
	// Name is the name of the custom tool, used to identify it in the API.
	Name string `json:"name"`

	// Description is an optional description of what the tool does.
	Description string `json:"description,omitempty"`

	// Format is the optional output format specification for the tool.
	// Omit for unconstrained text output.
	Format CustomToolFormat `json:"format,omitempty"`
}

// CustomToolID is the provider tool ID for custom tools.
const CustomToolID = "openai.custom"

// NewCustomTool creates a provider tool configuration for a custom tool.
func NewCustomTool(args CustomToolArgs) map[string]interface{} {
	result := map[string]interface{}{
		"type": "provider",
		"id":   CustomToolID,
		"name": args.Name,
	}
	if args.Description != "" {
		result["description"] = args.Description
	}
	if args.Format != nil {
		switch f := args.Format.(type) {
		case CustomToolFormatGrammar:
			result["format"] = map[string]interface{}{
				"type":       "grammar",
				"syntax":     f.Syntax,
				"definition": f.Definition,
			}
		case CustomToolFormatText:
			result["format"] = map[string]interface{}{
				"type": "text",
			}
		}
	}
	return result
}
