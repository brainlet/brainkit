// Ported from: packages/provider/src/language-model/v3/language-model-v3-tool-choice.ts
package languagemodel

// ToolChoice specifies how the tool should be selected.
// This is a sealed interface (discriminated union in TS).
type ToolChoice interface {
	toolChoiceType() string
}

// ToolChoiceAuto means tool selection is automatic (can be no tool).
type ToolChoiceAuto struct{}

func (ToolChoiceAuto) toolChoiceType() string { return "auto" }

// ToolChoiceNone means no tool must be selected.
type ToolChoiceNone struct{}

func (ToolChoiceNone) toolChoiceType() string { return "none" }

// ToolChoiceRequired means one of the available tools must be selected.
type ToolChoiceRequired struct{}

func (ToolChoiceRequired) toolChoiceType() string { return "required" }

// ToolChoiceTool means a specific tool must be selected.
type ToolChoiceTool struct {
	// ToolName is the name of the tool that must be selected.
	ToolName string
}

func (ToolChoiceTool) toolChoiceType() string { return "tool" }
