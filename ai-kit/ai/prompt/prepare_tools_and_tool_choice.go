// Ported from: packages/ai/src/prompt/prepare-tools-and-tool-choice.ts
package prompt

import "encoding/json"

// LanguageModelV4FunctionTool represents a function tool for the language model.
// TODO: import from brainlink/experiments/ai-kit/provider once it exists
type LanguageModelV4FunctionTool struct {
	Type            string          `json:"type"` // always "function"
	Name            string          `json:"name"`
	Description     *string         `json:"description,omitempty"`
	InputSchema     json.RawMessage `json:"inputSchema,omitempty"`
	InputExamples   []interface{}   `json:"inputExamples,omitempty"`
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`
	Strict          *bool           `json:"strict,omitempty"`
}

// LanguageModelV4ProviderTool represents a provider-specific tool.
// TODO: import from brainlink/experiments/ai-kit/provider once it exists
type LanguageModelV4ProviderTool struct {
	Type string      `json:"type"` // always "provider"
	Name string      `json:"name"`
	ID   string      `json:"id"`
	Args interface{} `json:"args,omitempty"`
}

// LanguageModelV4ToolChoice represents the tool choice for the language model.
// TODO: import from brainlink/experiments/ai-kit/provider once it exists
type LanguageModelV4ToolChoice struct {
	Type     string  `json:"type"` // "auto", "none", "required", "tool"
	ToolName *string `json:"toolName,omitempty"`
}

// ToolType specifies the type of a tool.
type ToolType string

const (
	ToolTypeFunction ToolType = "function"
	ToolTypeDynamic  ToolType = "dynamic"
	ToolTypeProvider ToolType = "provider"
)

// Tool represents a tool definition.
// TODO: import from brainlink/experiments/ai-kit/providerutils once fully ported
type Tool struct {
	Type            ToolType        `json:"type,omitempty"`
	Description     *string         `json:"description,omitempty"`
	InputSchema     json.RawMessage `json:"inputSchema,omitempty"`
	InputExamples   []interface{}   `json:"inputExamples,omitempty"`
	ProviderOptions ProviderOptions `json:"providerOptions,omitempty"`
	Strict          *bool           `json:"strict,omitempty"`
	// For provider tools:
	ID   string      `json:"id,omitempty"`
	Args interface{} `json:"args,omitempty"`
}

// ToolChoice represents tool choice configuration.
// Can be a string ("auto", "none", "required") or a specific tool reference.
type ToolChoice struct {
	// Type is the tool choice type when it's a string choice.
	Type string `json:"type,omitempty"`
	// ToolName is the specific tool name when type is "tool".
	ToolName *string `json:"toolName,omitempty"`
}

// ToolSet is a map of tool names to tool definitions.
type ToolSet = map[string]Tool

// PrepareToolsAndToolChoiceResult holds the prepared tools and tool choice.
type PrepareToolsAndToolChoiceResult struct {
	Tools      []interface{}              `json:"tools,omitempty"`      // []LanguageModelV4FunctionTool | []LanguageModelV4ProviderTool
	ToolChoice *LanguageModelV4ToolChoice `json:"toolChoice,omitempty"`
}

// PrepareToolsAndToolChoice prepares tools and tool choice for the language model.
func PrepareToolsAndToolChoice(
	tools ToolSet,
	toolChoice *ToolChoice,
	activeTools []string,
) PrepareToolsAndToolChoiceResult {
	if len(tools) == 0 {
		return PrepareToolsAndToolChoiceResult{
			Tools:      nil,
			ToolChoice: nil,
		}
	}

	// Filter tools by activeTools if provided
	type toolEntry struct {
		name string
		tool Tool
	}

	var filteredTools []toolEntry
	if activeTools != nil {
		activeSet := make(map[string]bool)
		for _, name := range activeTools {
			activeSet[name] = true
		}
		for name, tool := range tools {
			if activeSet[name] {
				filteredTools = append(filteredTools, toolEntry{name, tool})
			}
		}
	} else {
		for name, tool := range tools {
			filteredTools = append(filteredTools, toolEntry{name, tool})
		}
	}

	var languageModelTools []interface{}
	for _, entry := range filteredTools {
		toolType := entry.tool.Type

		switch toolType {
		case "", ToolTypeDynamic, ToolTypeFunction:
			ft := LanguageModelV4FunctionTool{
				Type:            "function",
				Name:            entry.name,
				Description:     entry.tool.Description,
				InputSchema:     entry.tool.InputSchema,
				ProviderOptions: entry.tool.ProviderOptions,
			}
			if entry.tool.InputExamples != nil {
				ft.InputExamples = entry.tool.InputExamples
			}
			if entry.tool.Strict != nil {
				ft.Strict = entry.tool.Strict
			}
			languageModelTools = append(languageModelTools, ft)

		case ToolTypeProvider:
			pt := LanguageModelV4ProviderTool{
				Type: "provider",
				Name: entry.name,
				ID:   entry.tool.ID,
				Args: entry.tool.Args,
			}
			languageModelTools = append(languageModelTools, pt)
		}
	}

	var tc *LanguageModelV4ToolChoice
	if toolChoice == nil {
		tc = &LanguageModelV4ToolChoice{Type: "auto"}
	} else if toolChoice.ToolName != nil {
		tc = &LanguageModelV4ToolChoice{
			Type:     "tool",
			ToolName: toolChoice.ToolName,
		}
	} else {
		tc = &LanguageModelV4ToolChoice{Type: toolChoice.Type}
	}

	return PrepareToolsAndToolChoiceResult{
		Tools:      languageModelTools,
		ToolChoice: tc,
	}
}
