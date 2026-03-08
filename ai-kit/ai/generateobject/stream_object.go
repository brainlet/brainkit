// Ported from: packages/ai/src/generate-object/stream-object.ts
package generateobject

import (
	"context"
	"fmt"
)

// StreamLanguageModel is the interface for language models that support streaming.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type StreamLanguageModel interface {
	// Provider returns the provider name.
	Provider() string
	// ModelID returns the model identifier.
	ModelID() string
	// DoStream performs a streaming text generation operation.
	DoStream(ctx context.Context, opts DoStreamObjectOptions) (<-chan StreamChunk, error)
}

// DoStreamObjectOptions are the options passed to StreamLanguageModel.DoStream.
type DoStreamObjectOptions struct {
	// Mode is the generation mode: "json" or "tool".
	Mode string
	// Prompt is the prompt to send to the model.
	Prompt string
	// Schema is the JSON schema for the expected output.
	Schema any
	// SchemaName is the name of the schema.
	SchemaName string
	// SchemaDescription is the description of the schema.
	SchemaDescription string
	// Headers are additional headers for the request.
	Headers map[string]string
	// ProviderOptions are additional provider-specific options.
	ProviderOptions map[string]map[string]any
}

// StreamChunk represents a chunk in the streaming response.
type StreamChunk struct {
	// Type is the chunk type: "text-delta", "finish", "error", "stream-start".
	Type string
	// TextDelta is the text delta (when Type is "text-delta").
	TextDelta string
	// FinishReason is why the generation finished (when Type is "finish").
	FinishReason FinishReason
	// Usage is the token usage (when Type is "finish").
	Usage LanguageModelUsage
	// Response is the response metadata (when Type is "finish").
	Response LanguageModelResponseMetadata
	// ProviderMetadata is provider-specific metadata (when Type is "finish").
	ProviderMetadata ProviderMetadata
	// Error is the error (when Type is "error").
	Error error
	// Warnings from the model provider (when Type is "stream-start").
	Warnings []CallWarning
	// Request is request metadata (when Type is "stream-start").
	Request *LanguageModelRequestMetadata
}

// StreamObjectOnFinishCallback is called when the stream finishes.
type StreamObjectOnFinishCallback func(event StreamObjectOnFinishEvent)

// StreamObjectOnFinishEvent contains the data passed to the onFinish callback.
type StreamObjectOnFinishEvent struct {
	// Usage is the token usage of the generated response.
	Usage LanguageModelUsage
	// Object is the generated object. Nil if the final object does not match the schema.
	Object any
	// Error is set when the final object does not match the schema (e.g. TypeValidationError).
	Error error
	// Response is additional response information.
	Response LanguageModelResponseMetadata
	// Warnings from the model provider.
	Warnings []CallWarning
	// ProviderMetadata is additional provider-specific metadata.
	ProviderMetadata ProviderMetadata
}

// StreamObjectOnErrorCallback is called when an error occurs during streaming.
type StreamObjectOnErrorCallback func(event StreamObjectOnErrorEvent)

// StreamObjectOnErrorEvent contains the error event data.
type StreamObjectOnErrorEvent struct {
	Error error
}

// StreamObjectOptions are the options for the StreamObject function.
type StreamObjectOptions struct {
	// Model is the streaming language model to use.
	Model StreamLanguageModel

	// Output is the output type: "object", "array", "enum", "no-schema".
	// Default: "object".
	Output string

	// Schema is the JSON schema for the expected output.
	Schema any

	// SchemaName is an optional name for the schema.
	SchemaName string

	// SchemaDescription is an optional description for the schema.
	SchemaDescription string

	// EnumValues are the allowed values for enum output.
	EnumValues []string

	// Mode is the generation mode: "json" or "tool". Default: "json".
	Mode string

	// Prompt is the text prompt for generation.
	Prompt string

	// MaxRetries is the maximum number of retries. Default: 2.
	MaxRetries *int

	// Headers are additional headers for the request.
	Headers map[string]string

	// ProviderOptions are additional provider-specific options.
	ProviderOptions map[string]map[string]any

	// OnChunk is called for each stream chunk.
	OnChunk func(chunk ObjectStreamPart)

	// OnFinish is called when the stream finishes with the final result.
	OnFinish StreamObjectOnFinishCallback

	// OnError is called when an error occurs during streaming.
	OnError StreamObjectOnErrorCallback

	// RepairText is an optional function to repair malformed JSON output.
	RepairText RepairTextFunc
}

// StreamObject streams a typed object using a language model.
// It returns a StreamObjectResult with channels for receiving partial objects.
func StreamObject(ctx context.Context, opts StreamObjectOptions) (*StreamObjectResult, error) {
	// Validate input.
	output := opts.Output
	if output == "" {
		output = "object"
	}

	if err := ValidateObjectGenerationInput(ValidateObjectGenerationInputOptions{
		Output:            output,
		Schema:            opts.Schema,
		SchemaName:        opts.SchemaName,
		SchemaDescription: opts.SchemaDescription,
		EnumValues:        opts.EnumValues,
	}); err != nil {
		return nil, err
	}

	// Get the output strategy.
	strategy, err := GetOutputStrategy(OutputType(output), opts.Schema, opts.EnumValues)
	if err != nil {
		return nil, err
	}

	// Get JSON schema from the strategy.
	jsonSchema, err := strategy.JSONSchema()
	if err != nil {
		return nil, fmt.Errorf("failed to get JSON schema: %w", err)
	}

	// Build the prompt with JSON instructions.
	mode := opts.Mode
	if mode == "" {
		mode = "json"
	}

	var prompt string
	if mode == "json" {
		prompt = InjectJsonInstruction(InjectJsonInstructionOptions{
			Prompt: opts.Prompt,
			Schema: jsonSchema,
		})
	} else {
		prompt = opts.Prompt
	}

	// Start streaming.
	streamCh, err := opts.Model.DoStream(ctx, DoStreamObjectOptions{
		Mode:              mode,
		Prompt:            prompt,
		Schema:            jsonSchema,
		SchemaName:        opts.SchemaName,
		SchemaDescription: opts.SchemaDescription,
		Headers:           opts.Headers,
		ProviderOptions:   opts.ProviderOptions,
	})
	if err != nil {
		return nil, err
	}

	// Create output channels.
	textStream := make(chan string, 100)
	fullStream := make(chan ObjectStreamPart, 100)

	result := &StreamObjectResult{
		TextStream: textStream,
		FullStream: fullStream,
	}

	// Process the stream in a goroutine.
	go func() {
		defer close(textStream)
		defer close(fullStream)

		var accumulatedText string
		var warnings []CallWarning
		var objectResult any
		var objectError error

		for chunk := range streamCh {
			switch chunk.Type {
			case "stream-start":
				// Capture warnings and request metadata from the stream-start chunk.
				warnings = chunk.Warnings
				result.Warnings = warnings
				if chunk.Request != nil {
					result.Request = *chunk.Request
				}

			case "text-delta":
				accumulatedText += chunk.TextDelta

				textStream <- chunk.TextDelta

				part := ObjectStreamPart{
					Type:      ObjectStreamPartTypeTextDelta,
					TextDelta: chunk.TextDelta,
				}
				fullStream <- part

				if opts.OnChunk != nil {
					opts.OnChunk(part)
				}

				// Try to parse partial JSON.
				parsed, parseErr := ParseJSON(accumulatedText)
				if parseErr == nil {
					objectPart := ObjectStreamPart{
						Type:   ObjectStreamPartTypeObject,
						Object: parsed,
					}
					fullStream <- objectPart

					if opts.OnChunk != nil {
						opts.OnChunk(objectPart)
					}
				}

			case "finish":
				result.FinishReason = chunk.FinishReason
				result.Usage = chunk.Usage
				result.Response = chunk.Response
				result.ProviderMetadata = chunk.ProviderMetadata

				// Parse and validate the final result, with optional repair.
				if accumulatedText != "" {
					if opts.RepairText != nil {
						obj, repairErr := ParseAndValidateObjectResultWithRepair(
							accumulatedText,
							strategy,
							opts.RepairText,
						)
						if repairErr != nil {
							objectError = repairErr
						} else {
							objectResult = obj
							result.Object = obj
						}
					} else {
						parsed, parseErr := ParseJSON(accumulatedText)
						if parseErr == nil {
							validationResult := strategy.ValidateFinalResult(parsed)
							if validationResult.Success {
								objectResult = validationResult.Value
								result.Object = validationResult.Value
							} else {
								objectError = validationResult.Error
							}
						} else {
							objectError = parseErr
						}
					}
				}

				finishPart := ObjectStreamPart{
					Type:             ObjectStreamPartTypeFinish,
					FinishReason:     chunk.FinishReason,
					Usage:            chunk.Usage,
					Response:         chunk.Response,
					ProviderMetadata: chunk.ProviderMetadata,
				}
				fullStream <- finishPart

				if opts.OnChunk != nil {
					opts.OnChunk(finishPart)
				}

			case "error":
				if opts.OnError != nil {
					opts.OnError(StreamObjectOnErrorEvent{Error: chunk.Error})
				}

				errorPart := ObjectStreamPart{
					Type:  ObjectStreamPartTypeError,
					Error: chunk.Error,
				}
				fullStream <- errorPart

				if opts.OnChunk != nil {
					opts.OnChunk(errorPart)
				}
			}
		}

		// Call onFinish after the stream ends.
		if opts.OnFinish != nil {
			opts.OnFinish(StreamObjectOnFinishEvent{
				Usage:            result.Usage,
				Object:           objectResult,
				Error:            objectError,
				Response:         result.Response,
				Warnings:         warnings,
				ProviderMetadata: result.ProviderMetadata,
			})
		}
	}()

	return result, nil
}
