// Ported from: packages/ai/src/generate-object/generate-object.ts
package generateobject

import (
	"context"
	"encoding/json"
	"fmt"
)

// LanguageModel is the interface for language models used in object generation.
// TODO: import from brainlink/experiments/ai-kit/types once ported
type LanguageModel interface {
	// Provider returns the provider name.
	Provider() string
	// ModelID returns the model identifier.
	ModelID() string
	// DoGenerate performs the text generation operation.
	DoGenerate(ctx context.Context, opts DoGenerateObjectOptions) (*DoGenerateObjectResult, error)
}

// DoGenerateObjectOptions are the options passed to LanguageModel.DoGenerate.
type DoGenerateObjectOptions struct {
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

// DoGenerateObjectResult is the result from LanguageModel.DoGenerate.
type DoGenerateObjectResult struct {
	// Text is the raw text response from the model.
	Text string
	// FinishReason is why the generation finished.
	FinishReason FinishReason
	// Usage is the token usage.
	Usage LanguageModelUsage
	// Warnings from the model provider.
	Warnings []CallWarning
	// Request is request metadata.
	Request LanguageModelRequestMetadata
	// Response is response metadata.
	Response LanguageModelResponseMetadata
	// ProviderMetadata is additional provider-specific metadata.
	ProviderMetadata ProviderMetadata
	// Reasoning is the reasoning text if available.
	Reasoning string
}

// GenerateObjectOptions are the options for the GenerateObject function.
type GenerateObjectOptions struct {
	// Model is the language model to use.
	Model LanguageModel

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

	// RepairText is an optional function to repair malformed JSON output.
	RepairText RepairTextFunc
}

// GenerateObject generates a typed object using a language model.
func GenerateObject(ctx context.Context, opts GenerateObjectOptions) (*GenerateObjectResult, error) {
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

	// Call the model.
	result, err := opts.Model.DoGenerate(ctx, DoGenerateObjectOptions{
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

	// Parse and validate the result.
	object, err := ParseAndValidateObjectResultWithRepair(
		result.Text,
		strategy,
		opts.RepairText,
	)
	if err != nil {
		return nil, err
	}

	return &GenerateObjectResult{
		Object:           object,
		Reasoning:        result.Reasoning,
		FinishReason:     result.FinishReason,
		Usage:            result.Usage,
		Warnings:         result.Warnings,
		Request:          result.Request,
		Response:         result.Response,
		ProviderMetadata: result.ProviderMetadata,
	}, nil
}

// ToJSONResponse converts the result to a JSON byte slice.
func (r *GenerateObjectResult) ToJSONResponse() ([]byte, error) {
	return json.Marshal(r.Object)
}
