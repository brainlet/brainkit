// Ported from: packages/ai/src/generate-text/output.ts
package generatetext

import (
	"encoding/json"
	"fmt"
)

// Output is an interface for structured output specifications.
type Output interface {
	// Name returns the name of the output mode.
	Name() string

	// ResponseFormat returns the response format to use for the model.
	ResponseFormat() (*ResponseFormat, error)

	// ParseCompleteOutput parses the complete output of the model.
	ParseCompleteOutput(text string, ctx OutputContext) (interface{}, error)

	// ParsePartialOutput parses partial output of the model.
	// Returns nil if parsing fails or input is undefined.
	ParsePartialOutput(text string) *PartialOutput
}

// OutputContext provides context for parsing complete output.
type OutputContext struct {
	Response     LanguageModelResponseMetadata
	Usage        LanguageModelUsage
	FinishReason FinishReason
}

// PartialOutput wraps a partially parsed output.
type PartialOutput struct {
	Partial interface{}
}

// --- Text output ---

type textOutput struct{}

// TextOutput returns an Output specification for text generation (the default mode).
func TextOutput() Output {
	return &textOutput{}
}

func (t *textOutput) Name() string { return "text" }

func (t *textOutput) ResponseFormat() (*ResponseFormat, error) {
	return &ResponseFormat{Type: "text"}, nil
}

func (t *textOutput) ParseCompleteOutput(text string, _ OutputContext) (interface{}, error) {
	return text, nil
}

func (t *textOutput) ParsePartialOutput(text string) *PartialOutput {
	return &PartialOutput{Partial: text}
}

// --- Object output ---

type objectOutput struct {
	schema      interface{} // JSON Schema
	name        string
	description string
}

// ObjectOutputOptions contains options for object output.
type ObjectOutputOptions struct {
	Schema      interface{} // JSON Schema for the object
	Name        string
	Description string
}

// ObjectOutput returns an Output specification for typed object generation using schemas.
func ObjectOutput(opts ObjectOutputOptions) Output {
	return &objectOutput{
		schema:      opts.Schema,
		name:        opts.Name,
		description: opts.Description,
	}
}

func (o *objectOutput) Name() string { return "object" }

func (o *objectOutput) ResponseFormat() (*ResponseFormat, error) {
	schema, ok := o.schema.(map[string]interface{})
	if !ok {
		schema = map[string]interface{}{}
	}
	rf := &ResponseFormat{
		Type:   "json",
		Schema: schema,
	}
	if o.name != "" {
		rf.Name = o.name
	}
	if o.description != "" {
		rf.Description = o.description
	}
	return rf, nil
}

func (o *objectOutput) ParseCompleteOutput(text string, ctx OutputContext) (interface{}, error) {
	var result interface{}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("no object generated: could not parse the response: %w", err)
	}
	// TODO: validate against schema
	return result, nil
}

func (o *objectOutput) ParsePartialOutput(text string) *PartialOutput {
	var result interface{}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil
	}
	return &PartialOutput{Partial: result}
}

// --- Array output ---

type arrayOutput struct {
	elementSchema interface{} // JSON Schema for the element
	name          string
	description   string
}

// ArrayOutputOptions contains options for array output.
type ArrayOutputOptions struct {
	Element     interface{} // JSON Schema for the array element
	Name        string
	Description string
}

// ArrayOutput returns an Output specification for array generation.
func ArrayOutput(opts ArrayOutputOptions) Output {
	return &arrayOutput{
		elementSchema: opts.Element,
		name:          opts.Name,
		description:   opts.Description,
	}
}

func (a *arrayOutput) Name() string { return "array" }

func (a *arrayOutput) ResponseFormat() (*ResponseFormat, error) {
	itemSchema, ok := a.elementSchema.(map[string]interface{})
	if !ok {
		itemSchema = map[string]interface{}{}
	}
	// Remove $schema from element schema
	cleanSchema := make(map[string]interface{})
	for k, v := range itemSchema {
		if k != "$schema" {
			cleanSchema[k] = v
		}
	}

	rf := &ResponseFormat{
		Type: "json",
		Schema: map[string]interface{}{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type":    "object",
			"properties": map[string]interface{}{
				"elements": map[string]interface{}{
					"type":  "array",
					"items": cleanSchema,
				},
			},
			"required":             []string{"elements"},
			"additionalProperties": false,
		},
	}
	if a.name != "" {
		rf.Name = a.name
	}
	if a.description != "" {
		rf.Description = a.description
	}
	return rf, nil
}

func (a *arrayOutput) ParseCompleteOutput(text string, ctx OutputContext) (interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("no object generated: could not parse the response: %w", err)
	}

	elements, ok := result["elements"]
	if !ok {
		return nil, fmt.Errorf("no object generated: response did not match schema")
	}
	arr, ok := elements.([]interface{})
	if !ok {
		return nil, fmt.Errorf("no object generated: response did not match schema")
	}
	// TODO: validate each element against schema
	return arr, nil
}

func (a *arrayOutput) ParsePartialOutput(text string) *PartialOutput {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil
	}
	elements, ok := result["elements"]
	if !ok {
		return nil
	}
	arr, ok := elements.([]interface{})
	if !ok {
		return nil
	}
	return &PartialOutput{Partial: arr}
}

// --- Choice output ---

type choiceOutput struct {
	options     []string
	name        string
	description string
}

// ChoiceOutputOptions contains options for choice output.
type ChoiceOutputOptions struct {
	Options     []string
	Name        string
	Description string
}

// ChoiceOutput returns an Output specification for choice generation.
func ChoiceOutput(opts ChoiceOutputOptions) Output {
	return &choiceOutput{
		options:     opts.Options,
		name:        opts.Name,
		description: opts.Description,
	}
}

func (c *choiceOutput) Name() string { return "choice" }

func (c *choiceOutput) ResponseFormat() (*ResponseFormat, error) {
	opts := make([]interface{}, len(c.options))
	for i, o := range c.options {
		opts[i] = o
	}
	rf := &ResponseFormat{
		Type: "json",
		Schema: map[string]interface{}{
			"$schema": "http://json-schema.org/draft-07/schema#",
			"type":    "object",
			"properties": map[string]interface{}{
				"result": map[string]interface{}{
					"type": "string",
					"enum": opts,
				},
			},
			"required":             []string{"result"},
			"additionalProperties": false,
		},
	}
	if c.name != "" {
		rf.Name = c.name
	}
	if c.description != "" {
		rf.Description = c.description
	}
	return rf, nil
}

func (c *choiceOutput) ParseCompleteOutput(text string, ctx OutputContext) (interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("no object generated: could not parse the response: %w", err)
	}

	val, ok := result["result"]
	if !ok {
		return nil, fmt.Errorf("no object generated: response did not match schema")
	}
	str, ok := val.(string)
	if !ok {
		return nil, fmt.Errorf("no object generated: response did not match schema")
	}
	for _, opt := range c.options {
		if str == opt {
			return str, nil
		}
	}
	return nil, fmt.Errorf("no object generated: response did not match schema")
}

func (c *choiceOutput) ParsePartialOutput(text string) *PartialOutput {
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil
	}
	val, ok := result["result"]
	if !ok {
		return nil
	}
	str, ok := val.(string)
	if !ok {
		return nil
	}
	return &PartialOutput{Partial: str}
}

// --- JSON output ---

type jsonOutput struct {
	name        string
	description string
}

// JSONOutputOptions contains options for JSON output.
type JSONOutputOptions struct {
	Name        string
	Description string
}

// JSONOutput returns an Output specification for unstructured JSON generation.
func JSONOutput(opts JSONOutputOptions) Output {
	return &jsonOutput{
		name:        opts.Name,
		description: opts.Description,
	}
}

func (j *jsonOutput) Name() string { return "json" }

func (j *jsonOutput) ResponseFormat() (*ResponseFormat, error) {
	rf := &ResponseFormat{Type: "json"}
	if j.name != "" {
		rf.Name = j.name
	}
	if j.description != "" {
		rf.Description = j.description
	}
	return rf, nil
}

func (j *jsonOutput) ParseCompleteOutput(text string, ctx OutputContext) (interface{}, error) {
	var result interface{}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil, fmt.Errorf("no object generated: could not parse the response: %w", err)
	}
	return result, nil
}

func (j *jsonOutput) ParsePartialOutput(text string) *PartialOutput {
	var result interface{}
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		return nil
	}
	return &PartialOutput{Partial: result}
}
