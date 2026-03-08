// Ported from: packages/ai/src/generate-text/step-result.ts
package generatetext

import "strings"

// StepResult is the result of a single step in the generation process.
type StepResult struct {
	// StepNumber is the zero-based index of this step.
	StepNumber int

	// Model contains information about the model that produced this step.
	Model ModelInfo

	// FunctionID is an identifier from telemetry settings for grouping related operations.
	FunctionID string

	// Metadata is additional metadata from telemetry settings.
	Metadata map[string]interface{}

	// ExperimentalContext is a user-defined context object flowing through the generation.
	ExperimentalContext interface{}

	// Content is the content that was generated in this step.
	Content []ContentPart

	// FinishReason is the unified reason why the generation finished.
	FinishReason FinishReason

	// RawFinishReason is the raw reason from the provider.
	RawFinishReason string

	// Usage is the token usage for this step.
	Usage LanguageModelUsage

	// Warnings are warnings from the model provider.
	Warnings []CallWarning

	// Request contains additional request information.
	Request LanguageModelRequestMetadata

	// Response contains additional response information.
	Response StepResponseMetadata

	// ProviderMetadata contains additional provider-specific metadata.
	ProviderMetadata ProviderMetadata
}

// ModelInfo contains basic model information.
type ModelInfo struct {
	Provider string
	ModelID  string
}

// StepResponseMetadata extends LanguageModelResponseMetadata with messages and body.
type StepResponseMetadata struct {
	LanguageModelResponseMetadata

	// Messages are the response messages generated during the call.
	Messages []ResponseMessage

	// Body is the response body (available only for HTTP-based providers).
	Body interface{}
}

// Text returns the concatenated text from all text content parts.
func (s *StepResult) Text() string {
	var parts []string
	for _, part := range s.Content {
		if part.Type == "text" {
			parts = append(parts, part.Text)
		}
	}
	return strings.Join(parts, "")
}

// Reasoning returns all reasoning content parts as ReasoningParts.
func (s *StepResult) Reasoning() []ReasoningPart {
	var parts []ReasoningPart
	for _, part := range s.Content {
		if part.Type == "reasoning" {
			parts = append(parts, ReasoningPart{
				Type:             "reasoning",
				Text:             part.Text,
				ProviderMetadata: part.ProviderMetadata,
			})
		}
	}
	return parts
}

// ReasoningText returns the concatenated reasoning text.
// Returns empty string if no reasoning parts exist.
func (s *StepResult) ReasoningText() string {
	reasoning := s.Reasoning()
	if len(reasoning) == 0 {
		return ""
	}
	var texts []string
	for _, r := range reasoning {
		texts = append(texts, r.Text)
	}
	return strings.Join(texts, "")
}

// Files returns all generated files from file content parts.
func (s *StepResult) Files() []GeneratedFile {
	var files []GeneratedFile
	for _, part := range s.Content {
		if part.Type == "file" && part.File != nil {
			files = append(files, part.File)
		}
	}
	return files
}

// Sources returns all source references.
func (s *StepResult) Sources() []Source {
	var sources []Source
	for _, part := range s.Content {
		if part.Type == "source" && part.Source != nil {
			sources = append(sources, *part.Source)
		}
	}
	return sources
}

// ToolCalls returns all tool calls from content parts.
func (s *StepResult) ToolCalls() []ToolCall {
	var calls []ToolCall
	for _, part := range s.Content {
		if part.Type == "tool-call" && part.ToolCall != nil {
			calls = append(calls, *part.ToolCall)
		}
	}
	return calls
}

// StaticToolCalls returns tool calls that are not dynamic.
func (s *StepResult) StaticToolCalls() []ToolCall {
	var calls []ToolCall
	for _, tc := range s.ToolCalls() {
		if !tc.Dynamic {
			calls = append(calls, tc)
		}
	}
	return calls
}

// DynamicToolCalls returns tool calls that are dynamic.
func (s *StepResult) DynamicToolCalls() []ToolCall {
	var calls []ToolCall
	for _, tc := range s.ToolCalls() {
		if tc.Dynamic {
			calls = append(calls, tc)
		}
	}
	return calls
}

// ToolResults returns all tool results from content parts.
func (s *StepResult) ToolResults() []ToolResult {
	var results []ToolResult
	for _, part := range s.Content {
		if part.Type == "tool-result" && part.ToolResult != nil {
			results = append(results, *part.ToolResult)
		}
	}
	return results
}

// StaticToolResults returns tool results that are not dynamic.
func (s *StepResult) StaticToolResults() []ToolResult {
	var results []ToolResult
	for _, tr := range s.ToolResults() {
		if !tr.Dynamic {
			results = append(results, tr)
		}
	}
	return results
}

// DynamicToolResults returns tool results that are dynamic.
func (s *StepResult) DynamicToolResults() []ToolResult {
	var results []ToolResult
	for _, tr := range s.ToolResults() {
		if tr.Dynamic {
			results = append(results, tr)
		}
	}
	return results
}
