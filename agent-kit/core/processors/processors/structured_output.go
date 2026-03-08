// Ported from: packages/core/src/processors/processors/structured-output.ts
package concreteprocessors

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	processors "github.com/brainlet/brainkit/agent-kit/core/processors"
)

// ---------------------------------------------------------------------------
// Stub types for unported dependencies
// ---------------------------------------------------------------------------

// OutputSchema is a stub for ../../stream.OutputSchema.
// TODO: import from stream package once ported.
type OutputSchema = any

// StructuredOutputProcessorOptions configures the StructuredOutputProcessor.
// TODO: import from ../../agent/types once ported.
type StructuredOutputProcessorOptions struct {
	// Schema is the structured output schema (required).
	Schema OutputSchema

	// Model is the model configuration for the structuring agent (required).
	Model MastraModelConfig

	// ErrorStrategy: "strict", "warn", "fallback". Default: "strict".
	ErrorStrategy string

	// FallbackValue is the fallback value when errorStrategy is "fallback".
	FallbackValue any

	// Instructions is custom instructions for the structuring agent.
	Instructions string

	// JSONPromptInjection uses system prompt injection for JSON coercion.
	JSONPromptInjection bool

	// ProviderOptions are provider-specific options.
	ProviderOptions ProviderOptions

	// Logger is an optional structured logger.
	Logger StructuredOutputLogger
}

// StructuredOutputLogger is a minimal logger interface for structured output.
// TODO: import from logger package once ported.
type StructuredOutputLogger interface {
	Error(message string, args ...any)
	Warn(message string, args ...any)
	Info(message string, args ...any)
}

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

// StructuredOutputProcessorName is the canonical processor name.
const StructuredOutputProcessorName = "structured-output"

// ---------------------------------------------------------------------------
// StructuredOutputProcessor
// ---------------------------------------------------------------------------

// StructuredOutputProcessor transforms unstructured agent output into structured JSON
// using an internal structuring agent and provides real-time streaming support.
type StructuredOutputProcessor struct {
	processors.BaseProcessor
	schema                       OutputSchema
	structuringAgent             Agent
	errorStrategy                string
	fallbackValue                any
	isStructuringAgentStarted    bool
	jsonPromptInjection          bool
	providerOptions              ProviderOptions
	logger                       StructuredOutputLogger
}

// NewStructuredOutputProcessor creates a new StructuredOutputProcessor.
func NewStructuredOutputProcessor(opts StructuredOutputProcessorOptions) (*StructuredOutputProcessor, error) {
	if opts.Schema == nil {
		return nil, fmt.Errorf("StructuredOutputProcessor requires a schema to be provided")
	}
	if opts.Model == nil {
		return nil, fmt.Errorf("StructuredOutputProcessor requires a model to be provided")
	}

	errorStrategy := opts.ErrorStrategy
	if errorStrategy == "" {
		errorStrategy = "strict"
	}

	// TODO: Create actual structuring agent once Agent is ported.

	return &StructuredOutputProcessor{
		BaseProcessor:   processors.NewBaseProcessor(StructuredOutputProcessorName, "Structured Output"),
		schema:          opts.Schema,
		structuringAgent: nil,
		errorStrategy:   errorStrategy,
		fallbackValue:   opts.FallbackValue,
		jsonPromptInjection: opts.JSONPromptInjection,
		providerOptions: opts.ProviderOptions,
		logger:          opts.Logger,
	}, nil
}

// ProcessOutputStream processes stream chunks, intercepting the "finish" chunk
// to start the structuring agent stream.
func (sop *StructuredOutputProcessor) ProcessOutputStream(args processors.ProcessOutputStreamArgs) (*processors.ChunkType, error) {
	part := args.Part

	switch part.Type {
	case "finish":
		// The main stream is finished; start the structuring agent stream.
		if err := sop.processAndEmitStructuredOutput(args.StreamParts, args.Abort); err != nil {
			return nil, err
		}
		return &part, nil
	default:
		return &part, nil
	}
}

// processAndEmitStructuredOutput collects stream text and processes it via the structuring agent.
func (sop *StructuredOutputProcessor) processAndEmitStructuredOutput(
	streamParts []processors.ChunkType,
	abort func(string, *processors.TripWireOptions) error,
) error {
	if sop.isStructuringAgentStarted {
		return nil
	}
	sop.isStructuringAgentStarted = true

	structuringPrompt := sop.buildStructuringPrompt(streamParts)
	prompt := fmt.Sprintf("Extract and structure the key information from the following text according to the specified schema. Keep the original meaning and details:\n\n%s", structuringPrompt)

	// TODO: Once Agent is ported, use the structuring agent to stream structured output.
	_ = prompt
	log.Println("[StructuredOutputProcessor] Structuring agent not yet ported")

	return nil
}

// buildStructuringPrompt builds a structured markdown prompt from stream parts.
func (sop *StructuredOutputProcessor) buildStructuringPrompt(streamParts []processors.ChunkType) string {
	var textChunks []string
	var reasoningChunks []string

	for _, part := range streamParts {
		switch part.Type {
		case "text-delta":
			if payload, ok := part.Payload.(map[string]any); ok {
				if text, ok := payload["text"].(string); ok {
					textChunks = append(textChunks, text)
				}
			}
		case "reasoning-delta":
			if payload, ok := part.Payload.(map[string]any); ok {
				if text, ok := payload["text"].(string); ok {
					reasoningChunks = append(reasoningChunks, text)
				}
			}
		case "tool-call":
			// Handled below with tool-result.
		case "tool-result":
			// Handled below.
		}
	}

	var sections []string

	if len(reasoningChunks) > 0 {
		sections = append(sections, fmt.Sprintf("# Assistant Reasoning\n%s", strings.Join(reasoningChunks, "")))
	}

	// Collect tool calls and results.
	var toolCallTexts []string
	var toolResultTexts []string
	for _, part := range streamParts {
		if part.Type == "tool-call" {
			if payload, ok := part.Payload.(map[string]any); ok {
				toolName, _ := payload["toolName"].(string)
				var argsStr string
				if args, ok := payload["args"]; ok {
					jsonBytes, err := json.Marshal(args)
					if err == nil {
						argsStr = string(jsonBytes)
					}
				}
				var outputStr string
				if output, ok := payload["output"]; ok && output != nil {
					jsonBytes, err := json.Marshal(output)
					if err == nil {
						outputStr = string(jsonBytes)
					}
				}
				entry := fmt.Sprintf("## %s\n### Input: %s", toolName, argsStr)
				if outputStr != "" {
					entry += fmt.Sprintf("\n### Output: %s", outputStr)
				}
				toolCallTexts = append(toolCallTexts, entry)
			}
		}
		if part.Type == "tool-result" {
			if payload, ok := part.Payload.(map[string]any); ok {
				toolName, _ := payload["toolName"].(string)
				result := payload["result"]
				var resultStr string
				if result == nil {
					resultStr = "null"
				} else {
					jsonBytes, err := json.Marshal(result)
					if err == nil {
						resultStr = string(jsonBytes)
					} else {
						resultStr = fmt.Sprintf("%v", result)
					}
				}
				toolResultTexts = append(toolResultTexts, fmt.Sprintf("%s: %s", toolName, resultStr))
			}
		}
	}

	if len(toolCallTexts) > 0 {
		sections = append(sections, fmt.Sprintf("# Tool Calls\n%s", strings.Join(toolCallTexts, "\n")))
	}
	if len(toolResultTexts) > 0 {
		sections = append(sections, fmt.Sprintf("# Tool Results\n%s", strings.Join(toolResultTexts, "\n")))
	}
	if len(textChunks) > 0 {
		sections = append(sections, fmt.Sprintf("# Assistant Response\n%s", strings.Join(textChunks, "")))
	}

	return strings.Join(sections, "\n\n")
}

// handleError handles errors based on the configured strategy.
func (sop *StructuredOutputProcessor) handleError(context, errMsg string, abort func(string, *processors.TripWireOptions) error) {
	message := fmt.Sprintf("[StructuredOutputProcessor] %s: %s", context, errMsg)

	switch sop.errorStrategy {
	case "strict":
		if sop.logger != nil {
			sop.logger.Error(message)
		}
		if abort != nil {
			_ = abort(message, nil)
		}
	case "warn":
		if sop.logger != nil {
			sop.logger.Warn(message)
		}
	case "fallback":
		if sop.logger != nil {
			sop.logger.Info(fmt.Sprintf("%s (using fallback)", message))
		}
	}
}

// ProcessInput is not implemented for this processor.
func (sop *StructuredOutputProcessor) ProcessInput(args processors.ProcessInputArgs) ([]processors.MastraDBMessage, *processors.MessageList, *processors.ProcessInputResultWithSystemMessages, error) {
	return nil, nil, nil, nil
}

// ProcessInputStep is not implemented for this processor.
func (sop *StructuredOutputProcessor) ProcessInputStep(args processors.ProcessInputStepArgs) (*processors.ProcessInputStepResult, []processors.MastraDBMessage, error) {
	return nil, nil, nil
}

// ProcessOutputResult is not implemented for this processor.
func (sop *StructuredOutputProcessor) ProcessOutputResult(args processors.ProcessOutputResultArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// ProcessOutputStep is not implemented for this processor.
func (sop *StructuredOutputProcessor) ProcessOutputStep(args processors.ProcessOutputStepArgs) ([]processors.MastraDBMessage, *processors.MessageList, error) {
	return nil, nil, nil
}

// GetSchema returns the processor's output schema.
func (sop *StructuredOutputProcessor) GetSchema() OutputSchema {
	return sop.schema
}
