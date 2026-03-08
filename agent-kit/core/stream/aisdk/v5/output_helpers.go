// Ported from: packages/core/src/stream/aisdk/v5/output-helpers.ts
package v5

import (
	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

// ---------------------------------------------------------------------------
// Stub types for packages not yet ported
// ---------------------------------------------------------------------------

// StepTripwireData mirrors the TS StepTripwireData from stream/types.
// Stub: redefined locally to match v5-specific usage; real type is in stream.StepTripwireData.
type StepTripwireData struct {
	Name    string `json:"name,omitempty"`
	Message string `json:"message,omitempty"`
}

// ContentPart represents a single part of the model response content.
// This is a union type in TS; in Go we use a struct with a Type discriminator.
// ai-kit only ported V3 (@ai-sdk/provider-v6). V5 internal types remain local stubs.
type ContentPart struct {
	Type string `json:"type"`

	// text part fields
	Text string `json:"text,omitempty"`

	// reasoning part fields
	// Text is reused for reasoning text

	// file part fields
	File *DefaultGeneratedFile `json:"file,omitempty"`

	// source part fields
	SourceType       string         `json:"sourceType,omitempty"`
	ID               string         `json:"id,omitempty"`
	URL              string         `json:"url,omitempty"`
	Title            string         `json:"title,omitempty"`
	ProviderMetadata map[string]any `json:"providerMetadata,omitempty"`

	// tool-call part fields
	ToolCallID string `json:"toolCallId,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	Input      any    `json:"input,omitempty"`
	Dynamic    *bool  `json:"dynamic,omitempty"`

	// tool-result part fields
	Result any   `json:"result,omitempty"`
	Output any   `json:"output,omitempty"`
	IsErr  *bool `json:"isError,omitempty"`
}

// StepResultData holds the raw data for constructing a DefaultStepResult.
type StepResultData struct {
	Content          []ContentPart
	FinishReason     string
	Usage            stream.LanguageModelUsage
	Warnings         []any
	Request          any
	Response         any
	ProviderMetadata map[string]any
	Tripwire         *StepTripwireData
}

// ---------------------------------------------------------------------------
// DefaultStepResult
// ---------------------------------------------------------------------------

// DefaultStepResult implements the StepResult interface with computed getters.
// It holds the content parts from a model response step and provides
// convenience methods to extract text, reasoning, files, sources, tool calls,
// and tool results.
//
// In TS this is a class with getter properties that filter content parts.
type DefaultStepResult struct {
	Content          []ContentPart
	FinishReason     string
	Usage            stream.LanguageModelUsage
	Warnings         []any
	Request          any
	Response         any
	ProviderMetadata map[string]any
	// Tripwire data if this step was rejected by a processor.
	Tripwire *StepTripwireData
}

// NewDefaultStepResult creates a new DefaultStepResult from the given data.
func NewDefaultStepResult(data StepResultData) *DefaultStepResult {
	return &DefaultStepResult{
		Content:          data.Content,
		FinishReason:     data.FinishReason,
		Usage:            data.Usage,
		Warnings:         data.Warnings,
		Request:          data.Request,
		Response:         data.Response,
		ProviderMetadata: data.ProviderMetadata,
		Tripwire:         data.Tripwire,
	}
}

// Text returns concatenated text from all text content parts.
// Returns empty string if this step was rejected by a tripwire.
func (s *DefaultStepResult) Text() string {
	if s.Tripwire != nil {
		return ""
	}
	var result string
	for _, part := range s.Content {
		if part.Type == "text" {
			result += part.Text
		}
	}
	return result
}

// Reasoning returns all reasoning content parts.
func (s *DefaultStepResult) Reasoning() []ContentPart {
	var result []ContentPart
	for _, part := range s.Content {
		if part.Type == "reasoning" {
			result = append(result, part)
		}
	}
	return result
}

// ReasoningText returns concatenated text from all reasoning parts.
// Returns nil (empty string) if there are no reasoning parts.
func (s *DefaultStepResult) ReasoningText() string {
	reasoning := s.Reasoning()
	if len(reasoning) == 0 {
		return ""
	}
	var result string
	for _, part := range reasoning {
		result += part.Text
	}
	return result
}

// Files returns all file content from file parts.
func (s *DefaultStepResult) Files() []*DefaultGeneratedFile {
	var result []*DefaultGeneratedFile
	for _, part := range s.Content {
		if part.Type == "file" && part.File != nil {
			result = append(result, part.File)
		}
	}
	return result
}

// Sources returns all source content parts.
func (s *DefaultStepResult) Sources() []ContentPart {
	var result []ContentPart
	for _, part := range s.Content {
		if part.Type == "source" {
			result = append(result, part)
		}
	}
	return result
}

// ToolCalls returns all tool-call content parts.
func (s *DefaultStepResult) ToolCalls() []ContentPart {
	var result []ContentPart
	for _, part := range s.Content {
		if part.Type == "tool-call" {
			result = append(result, part)
		}
	}
	return result
}

// StaticToolCalls returns tool calls where Dynamic is false.
func (s *DefaultStepResult) StaticToolCalls() []ContentPart {
	var result []ContentPart
	for _, part := range s.ToolCalls() {
		if part.Dynamic != nil && !*part.Dynamic {
			result = append(result, part)
		}
	}
	return result
}

// DynamicToolCalls returns tool calls where Dynamic is true.
func (s *DefaultStepResult) DynamicToolCalls() []ContentPart {
	var result []ContentPart
	for _, part := range s.ToolCalls() {
		if part.Dynamic != nil && *part.Dynamic {
			result = append(result, part)
		}
	}
	return result
}

// ToolResults returns all tool-result content parts.
func (s *DefaultStepResult) ToolResults() []ContentPart {
	var result []ContentPart
	for _, part := range s.Content {
		if part.Type == "tool-result" {
			result = append(result, part)
		}
	}
	return result
}

// StaticToolResults returns tool results where Dynamic is false.
func (s *DefaultStepResult) StaticToolResults() []ContentPart {
	var result []ContentPart
	for _, part := range s.ToolResults() {
		if part.Dynamic != nil && !*part.Dynamic {
			result = append(result, part)
		}
	}
	return result
}

// DynamicToolResults returns tool results where Dynamic is true.
func (s *DefaultStepResult) DynamicToolResults() []ContentPart {
	var result []ContentPart
	for _, part := range s.ToolResults() {
		if part.Dynamic != nil && *part.Dynamic {
			result = append(result, part)
		}
	}
	return result
}
