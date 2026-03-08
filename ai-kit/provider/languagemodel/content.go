// Ported from: packages/provider/src/language-model/v3/language-model-v3-content.ts
package languagemodel

// Content is a sealed interface representing the types of content that a
// language model can generate. Implementations:
//   - Text
//   - Reasoning
//   - File
//   - ToolApprovalRequest
//   - Source (SourceURL, SourceDocument)
//   - ToolCall
//   - ToolResult
type Content interface {
	isContent()
}
