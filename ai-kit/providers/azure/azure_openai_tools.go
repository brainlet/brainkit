// Ported from: packages/azure/src/azure-openai-tools.ts
package azure

import "github.com/brainlet/brainkit/ai-kit/providers/openai"

// AzureOpenAITools re-exports the OpenAI built-in tool constructors for use
// with Azure OpenAI deployments.
var AzureOpenAITools = struct {
	// CodeInterpreter creates a code interpreter tool configuration.
	CodeInterpreter func(args *openai.CodeInterpreterArgs) map[string]interface{}
	// FileSearch creates a file search tool configuration.
	FileSearch func(args openai.FileSearchArgs) map[string]interface{}
	// ImageGeneration creates an image generation tool configuration.
	ImageGeneration func(args *openai.ImageGenerationArgs) map[string]interface{}
	// WebSearchPreview creates a web search preview tool configuration.
	WebSearchPreview func(args *openai.WebSearchPreviewArgs) map[string]interface{}
}{
	CodeInterpreter:  openai.NewCodeInterpreterTool,
	FileSearch:       openai.NewFileSearchTool,
	ImageGeneration:  openai.NewImageGenerationTool,
	WebSearchPreview: openai.NewWebSearchPreviewTool,
}
