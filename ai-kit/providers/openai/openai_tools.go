// Ported from: packages/openai/src/openai-tools.ts
package openai

// OpenAITools provides access to OpenAI-specific tool constructors.
type OpenAITools struct{}

// NewOpenAITools creates a new OpenAITools instance.
func NewOpenAITools() OpenAITools {
	return OpenAITools{}
}

// ApplyPatch creates a provider tool configuration for the apply_patch tool.
func (t OpenAITools) ApplyPatch() map[string]interface{} {
	return NewApplyPatchTool()
}

// CustomTool creates a provider tool configuration for a custom tool.
func (t OpenAITools) CustomTool(args CustomToolArgs) map[string]interface{} {
	return NewCustomTool(args)
}

// CodeInterpreter creates a provider tool configuration for the code_interpreter tool.
func (t OpenAITools) CodeInterpreter(args *CodeInterpreterArgs) map[string]interface{} {
	return NewCodeInterpreterTool(args)
}

// FileSearch creates a provider tool configuration for the file_search tool.
func (t OpenAITools) FileSearch(args FileSearchArgs) map[string]interface{} {
	return NewFileSearchTool(args)
}

// ImageGeneration creates a provider tool configuration for the image_generation tool.
func (t OpenAITools) ImageGeneration(args *ImageGenerationArgs) map[string]interface{} {
	return NewImageGenerationTool(args)
}

// LocalShell creates a provider tool configuration for the local_shell tool.
func (t OpenAITools) LocalShell() map[string]interface{} {
	return NewLocalShellTool()
}

// Shell creates a provider tool configuration for the shell tool.
func (t OpenAITools) Shell(args *ShellArgs) map[string]interface{} {
	return NewShellTool(args)
}

// WebSearchPreview creates a provider tool configuration for the web_search_preview tool.
func (t OpenAITools) WebSearchPreview(args *WebSearchPreviewArgs) map[string]interface{} {
	return NewWebSearchPreviewTool(args)
}

// WebSearch creates a provider tool configuration for the web_search tool.
func (t OpenAITools) WebSearch(args *WebSearchArgs) map[string]interface{} {
	return NewWebSearchTool(args)
}

// MCP creates a provider tool configuration for the MCP tool.
func (t OpenAITools) MCP(args MCPArgs) map[string]interface{} {
	return NewMCPTool(args)
}
