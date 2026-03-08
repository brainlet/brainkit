// Ported from: packages/core/src/stream/base/output-format-handlers.ts
package base

import (
	"encoding/json"
	"reflect"
	"regexp"
	"strings"

	"github.com/brainlet/brainkit/agent-kit/core/stream"
)

// ---------------------------------------------------------------------------
// escapeUnescapedControlCharsInJsonStrings
// ---------------------------------------------------------------------------

// EscapeUnescapedControlCharsInJsonStrings escapes unescaped newlines,
// carriage returns, and tabs within JSON string values.
//
// LLMs often output actual newline characters inside JSON strings instead of
// properly escaped \n sequences, which breaks JSON parsing. This function
// fixes that by:
//  1. Tracking whether we're inside a JSON string (after an unescaped quote)
//  2. Replacing literal newlines/tabs with their escape sequences only inside strings
//  3. Preserving already-escaped sequences like \\n
func EscapeUnescapedControlCharsInJsonStrings(text string) string {
	var result strings.Builder
	result.Grow(len(text))
	inString := false
	i := 0

	for i < len(text) {
		ch := text[i]

		// Check for escape sequences
		if ch == '\\' && i+1 < len(text) {
			// This is an escape sequence - pass through both characters
			result.WriteByte(ch)
			result.WriteByte(text[i+1])
			i += 2
			continue
		}

		// Track string boundaries (unescaped quotes)
		if ch == '"' {
			inString = !inString
			result.WriteByte(ch)
			i++
			continue
		}

		// If inside a string, escape control characters
		if inString {
			if ch == '\n' {
				result.WriteString("\\n")
				i++
				continue
			}
			if ch == '\r' {
				result.WriteString("\\r")
				i++
				continue
			}
			if ch == '\t' {
				result.WriteString("\\t")
				i++
				continue
			}
		}

		result.WriteByte(ch)
		i++
	}

	return result.String()
}

// ---------------------------------------------------------------------------
// ProcessPartialChunkParams / ProcessPartialChunkResult
// ---------------------------------------------------------------------------

// ProcessPartialChunkParams holds the parameters for processing a partial chunk.
type ProcessPartialChunkParams struct {
	// AccumulatedText is text accumulated from streaming so far.
	AccumulatedText string
	// PreviousObject is the previously parsed object from last emission.
	PreviousObject any
	// PreviousResult is the previous processing result (handler-specific state).
	PreviousResult any
}

// ProcessPartialChunkResult is the result of processing a partial chunk.
type ProcessPartialChunkResult struct {
	// ShouldEmit indicates whether a new value should be emitted.
	ShouldEmit bool
	// EmitValue is the value to emit if ShouldEmit is true.
	EmitValue any
	// NewPreviousResult is the new previous result state for next iteration.
	NewPreviousResult any
}

// ---------------------------------------------------------------------------
// ValidateAndTransformFinalResult
// ---------------------------------------------------------------------------

// ValidateAndTransformFinalResult is the result of validating and transforming
// the final parsed value when streaming completes.
type ValidateAndTransformFinalResult struct {
	// Success indicates whether validation succeeded.
	Success bool
	// Value is the validated and transformed value if successful.
	Value any
	// Error is the error if validation failed.
	Error error
}

// ---------------------------------------------------------------------------
// FormatHandler interface
// ---------------------------------------------------------------------------

// FormatHandler is the interface for output format handlers.
// Each handler implements format-specific logic for processing partial chunks
// and validating final results.
type FormatHandler interface {
	// Type returns the format type: "object", "array", or "enum".
	Type() string
	// ProcessPartialChunk processes a partial chunk and determines if a new value should be emitted.
	ProcessPartialChunk(params ProcessPartialChunkParams) ProcessPartialChunkResult
	// ValidateAndTransformFinal validates and transforms the final parsed value.
	ValidateAndTransformFinal(finalValue string) ValidateAndTransformFinalResult
}

// ---------------------------------------------------------------------------
// preprocessText — shared helper
// ---------------------------------------------------------------------------

// preprocessText preprocesses accumulated text to handle LLMs that wrap JSON
// in code blocks and fix common JSON formatting issues like unescaped newlines
// in strings. Extracts content from the first complete valid ```json...```
// code block or removes opening ```json prefix if no complete code block
// is found (streaming chunks).
func preprocessText(accumulatedText string) string {
	processedText := accumulatedText

	// Some LLMs (e.g., LMStudio with jsonPromptInjection) wrap JSON in special tokens
	// Format: '<|channel|>final <|constrain|>JSON<|message|>{"key":"value"}'
	if strings.Contains(processedText, "<|message|>") {
		re := regexp.MustCompile(`<\|message\|>([\s\S]+)$`)
		matches := re.FindStringSubmatch(processedText)
		if len(matches) > 1 {
			processedText = matches[1]
		}
	}

	// Some LLMs wrap the JSON response in code blocks.
	if strings.Contains(processedText, "```json") {
		re := regexp.MustCompile("```json\\s*\\n?([\\s\\S]*?)\\n?\\s*```")
		matches := re.FindStringSubmatch(processedText)
		if len(matches) > 1 {
			// Complete code block found - use content between tags
			processedText = strings.TrimSpace(matches[1])
		} else {
			// No complete code block - just remove the opening ```json
			openRe := regexp.MustCompile("^```json\\s*\\n?")
			processedText = openRe.ReplaceAllString(processedText, "")
		}
	}

	// LLMs often output actual newlines/tabs inside JSON strings instead of
	// properly escaped \n sequences. Fix this before parsing.
	processedText = EscapeUnescapedControlCharsInJsonStrings(processedText)

	return processedText
}

// ---------------------------------------------------------------------------
// parsePartialJson — simplified partial JSON parser
// ---------------------------------------------------------------------------

// parsePartialJsonState indicates the state of a partial JSON parse.
type parsePartialJsonState string

const (
	parseStateSuccessful parsePartialJsonState = "successful-parse"
	parseStateRepaired   parsePartialJsonState = "repaired-parse"
	parseStateFailed     parsePartialJsonState = "failed-parse"
)

// parsePartialJsonResult holds the result of a partial JSON parse.
type parsePartialJsonResult struct {
	Value any
	State parsePartialJsonState
}

// parsePartialJson attempts to parse a (possibly incomplete) JSON string.
// If standard parsing fails, it tries progressively adding closing brackets.
// This mirrors the @internal/ai-sdk-v5 parsePartialJson function.
func parsePartialJson(text string) parsePartialJsonResult {
	text = strings.TrimSpace(text)
	if text == "" {
		return parsePartialJsonResult{Value: nil, State: parseStateFailed}
	}

	// Try standard parse first
	var result any
	if err := json.Unmarshal([]byte(text), &result); err == nil {
		return parsePartialJsonResult{Value: result, State: parseStateSuccessful}
	}

	// Try repair strategies: add closing brackets/braces
	closers := []string{"}", "]", "\"}", "\"]", "\"}]", "\"}", "}]", "]}"}
	for _, closer := range closers {
		candidate := text + closer
		var result any
		if err := json.Unmarshal([]byte(candidate), &result); err == nil {
			return parsePartialJsonResult{Value: result, State: parseStateRepaired}
		}
	}

	// Try more aggressive repair: remove trailing comma + close
	trimmed := strings.TrimRight(text, " \t\n\r,")
	aggressiveClosers := []string{"}", "]", "}]", "]}"}
	for _, closer := range aggressiveClosers {
		candidate := trimmed + closer
		var result any
		if err := json.Unmarshal([]byte(candidate), &result); err == nil {
			return parsePartialJsonResult{Value: result, State: parseStateRepaired}
		}
	}

	return parsePartialJsonResult{Value: nil, State: parseStateFailed}
}

// isDeepEqualData compares two values for deep equality.
// Mirrors the @internal/ai-sdk-v5 isDeepEqualData function.
func isDeepEqualData(a, b any) bool {
	return reflect.DeepEqual(a, b)
}

// ---------------------------------------------------------------------------
// ObjectFormatHandler
// ---------------------------------------------------------------------------

// ObjectFormatHandler handles object format streaming. Emits parsed objects
// when they change during streaming. This is the simplest format - objects
// are parsed and emitted directly without wrapping.
type ObjectFormatHandler struct {
	schema OutputSchema
}

// NewObjectFormatHandler creates a new ObjectFormatHandler.
func NewObjectFormatHandler(schema OutputSchema) *ObjectFormatHandler {
	return &ObjectFormatHandler{schema: schema}
}

// Type returns "object".
func (h *ObjectFormatHandler) Type() string { return "object" }

// ProcessPartialChunk processes a partial chunk for object format.
func (h *ObjectFormatHandler) ProcessPartialChunk(params ProcessPartialChunkParams) ProcessPartialChunkResult {
	processedText := preprocessText(params.AccumulatedText)
	parsed := parsePartialJson(processedText)

	if parsed.Value != nil &&
		isMapOrStruct(parsed.Value) &&
		!isDeepEqualData(params.PreviousObject, parsed.Value) {
		shouldEmit := parsed.State == parseStateSuccessful || parsed.State == parseStateRepaired
		return ProcessPartialChunkResult{
			ShouldEmit:        shouldEmit,
			EmitValue:         parsed.Value,
			NewPreviousResult: parsed.Value,
		}
	}
	return ProcessPartialChunkResult{ShouldEmit: false}
}

// ValidateAndTransformFinal validates and transforms the final value for object format.
func (h *ObjectFormatHandler) ValidateAndTransformFinal(finalRawValue string) ValidateAndTransformFinalResult {
	if finalRawValue == "" {
		return ValidateAndTransformFinalResult{
			Success: false,
			Error:   newFormatError("No object generated: could not parse the response."),
		}
	}
	rawValue := preprocessText(finalRawValue)
	parsed := parsePartialJson(rawValue)

	if parsed.Value == nil {
		return ValidateAndTransformFinalResult{
			Success: false,
			Error:   newFormatError("No object generated: could not parse the response."),
		}
	}

	// Schema validation skipped — Zod runtime not applicable in Go; callers validate externally
	return ValidateAndTransformFinalResult{
		Success: true,
		Value:   parsed.Value,
	}
}

// ---------------------------------------------------------------------------
// ArrayFormatHandler
// ---------------------------------------------------------------------------

// ArrayFormatHandler handles array format streaming. Arrays are wrapped in
// {elements: [...]} objects by the LLM for better generation reliability.
// This handler unwraps them and filters incomplete elements.
// Emits progressive array states as elements are completed.
type ArrayFormatHandler struct {
	schema                    OutputSchema
	textPreviousFilteredArray []any
	hasEmittedInitialArray    bool
}

// NewArrayFormatHandler creates a new ArrayFormatHandler.
func NewArrayFormatHandler(schema OutputSchema) *ArrayFormatHandler {
	return &ArrayFormatHandler{
		schema:                    schema,
		textPreviousFilteredArray: []any{},
	}
}

// Type returns "array".
func (h *ArrayFormatHandler) Type() string { return "array" }

// ProcessPartialChunk processes a partial chunk for array format.
func (h *ArrayFormatHandler) ProcessPartialChunk(params ProcessPartialChunkParams) ProcessPartialChunkResult {
	processedText := preprocessText(params.AccumulatedText)
	parsed := parsePartialJson(processedText)

	if parsed.Value != nil && !isDeepEqualData(params.PreviousObject, parsed.Value) {
		// For arrays, extract and filter elements
		var rawElements []any
		if m, ok := parsed.Value.(map[string]any); ok {
			if elems, ok := m["elements"].([]any); ok {
				rawElements = elems
			}
		}

		var filteredElements []any
		for i, element := range rawElements {
			// Skip the last element if it's incomplete (unless this is the final parse)
			if i == len(rawElements)-1 && parsed.State != parseStateSuccessful {
				if isNonEmptyObject(element) {
					filteredElements = append(filteredElements, element)
				}
			} else {
				if isNonEmptyObject(element) {
					filteredElements = append(filteredElements, element)
				}
			}
		}

		// Emit initial empty array if this is the first time we see any JSON structure
		if !h.hasEmittedInitialArray {
			h.hasEmittedInitialArray = true
			if len(filteredElements) == 0 {
				h.textPreviousFilteredArray = []any{}
				return ProcessPartialChunkResult{
					ShouldEmit:        true,
					EmitValue:         []any{},
					NewPreviousResult: parsed.Value,
				}
			}
		}

		// Only emit if the filtered array has actually changed
		if !isDeepEqualData(h.textPreviousFilteredArray, filteredElements) {
			h.textPreviousFilteredArray = make([]any, len(filteredElements))
			copy(h.textPreviousFilteredArray, filteredElements)
			return ProcessPartialChunkResult{
				ShouldEmit:        true,
				EmitValue:         filteredElements,
				NewPreviousResult: parsed.Value,
			}
		}
	}

	return ProcessPartialChunkResult{ShouldEmit: false}
}

// ValidateAndTransformFinal validates and transforms the final value for array format.
func (h *ArrayFormatHandler) ValidateAndTransformFinal(_ string) ValidateAndTransformFinalResult {
	resultValue := h.textPreviousFilteredArray
	if resultValue == nil {
		return ValidateAndTransformFinalResult{
			Success: false,
			Error:   newFormatError("No object generated: could not parse the response."),
		}
	}
	// Schema validation skipped — Zod runtime not applicable in Go; callers validate externally
	return ValidateAndTransformFinalResult{
		Success: true,
		Value:   resultValue,
	}
}

// ---------------------------------------------------------------------------
// EnumFormatHandler
// ---------------------------------------------------------------------------

// EnumFormatHandler handles enum format streaming. Enums are wrapped in
// {result: ""} objects by the LLM for better generation reliability.
// This handler unwraps them and provides partial matching.
type EnumFormatHandler struct {
	schema                 OutputSchema
	textPreviousEnumResult string
}

// NewEnumFormatHandler creates a new EnumFormatHandler.
func NewEnumFormatHandler(schema OutputSchema) *EnumFormatHandler {
	return &EnumFormatHandler{schema: schema}
}

// Type returns "enum".
func (h *EnumFormatHandler) Type() string { return "enum" }

// findBestEnumMatch finds the best matching enum value for a partial result string.
// If multiple values match, returns the partial string. If only one matches, returns that value.
func (h *EnumFormatHandler) findBestEnumMatch(partialResult string) (string, bool) {
	if h.schema == nil {
		return "", false
	}

	// Try to extract enum values from the schema
	var enumValues []any
	if js, ok := h.schema.(*JSONSchema7); ok {
		enumValues = js.Enum
	}

	if len(enumValues) == 0 {
		return "", false
	}

	var possibleValues []string
	for _, v := range enumValues {
		if s, ok := v.(string); ok {
			if strings.HasPrefix(s, partialResult) {
				possibleValues = append(possibleValues, s)
			}
		}
	}

	if len(possibleValues) == 0 {
		return "", false
	}

	// Emit the most specific result
	if len(possibleValues) == 1 {
		return possibleValues[0], true
	}
	return partialResult, true
}

// ProcessPartialChunk processes a partial chunk for enum format.
func (h *EnumFormatHandler) ProcessPartialChunk(params ProcessPartialChunkParams) ProcessPartialChunkResult {
	processedText := preprocessText(params.AccumulatedText)
	parsed := parsePartialJson(processedText)

	if parsed.Value != nil {
		if m, ok := parsed.Value.(map[string]any); ok {
			if result, ok := m["result"].(string); ok {
				if !isDeepEqualData(params.PreviousObject, parsed.Value) {
					bestMatch, found := h.findBestEnumMatch(result)
					if len(result) > 0 && found && bestMatch != h.textPreviousEnumResult {
						h.textPreviousEnumResult = bestMatch
						return ProcessPartialChunkResult{
							ShouldEmit:        true,
							EmitValue:         bestMatch,
							NewPreviousResult: parsed.Value,
						}
					}
				}
			}
		}
	}

	return ProcessPartialChunkResult{ShouldEmit: false}
}

// ValidateAndTransformFinal validates and transforms the final value for enum format.
func (h *EnumFormatHandler) ValidateAndTransformFinal(rawFinalValue string) ValidateAndTransformFinalResult {
	processedValue := preprocessText(rawFinalValue)
	parsed := parsePartialJson(processedValue)

	if parsed.Value == nil {
		return ValidateAndTransformFinalResult{
			Success: false,
			Error:   newFormatError("Invalid enum format: expected object with result property"),
		}
	}

	m, ok := parsed.Value.(map[string]any)
	if !ok {
		return ValidateAndTransformFinalResult{
			Success: false,
			Error:   newFormatError("Invalid enum format: expected object with result property"),
		}
	}

	result, ok := m["result"].(string)
	if !ok {
		return ValidateAndTransformFinalResult{
			Success: false,
			Error:   newFormatError("Invalid enum format: expected object with result property"),
		}
	}

	// Schema validation skipped — Zod runtime not applicable in Go; callers validate externally
	return ValidateAndTransformFinalResult{
		Success: true,
		Value:   result,
	}
}

// ---------------------------------------------------------------------------
// CreateOutputHandler factory
// ---------------------------------------------------------------------------

// CreateOutputHandler is a factory function to create the appropriate output format
// handler based on schema. Analyzes the transformed schema format and returns
// the corresponding handler instance.
func CreateOutputHandler(schema OutputSchema) FormatHandler {
	transformedSchema := GetTransformedSchema(schema)
	if transformedSchema != nil {
		switch transformedSchema.OutputFormat {
		case "array":
			return NewArrayFormatHandler(schema)
		case "enum":
			return NewEnumFormatHandler(schema)
		}
	}
	return NewObjectFormatHandler(schema)
}

// ---------------------------------------------------------------------------
// StructuredOutputOptions
// ---------------------------------------------------------------------------

// StructuredOutputOptions configures structured output processing.
// Stub: simplified shape — real agent type has additional fields and model reference.
type StructuredOutputOptions struct {
	Schema              OutputSchema
	Model               any    // optional model for processor mode
	ErrorStrategy       string // "warn", "fallback", or "" (default: error)
	FallbackValue       any
	JsonPromptInjection bool
}

// ---------------------------------------------------------------------------
// CreateObjectStreamTransformer
// ---------------------------------------------------------------------------

// ObjectStreamTransformer transforms raw text-delta chunks into structured
// object chunks for JSON mode streaming.
//
// For JSON response formats, this transformer:
//   - Accumulates text deltas and parses them as partial JSON
//   - Emits 'object' chunks when the parsed structure changes
//   - For arrays: filters incomplete elements and unwraps from {elements: [...]} wrapper
//   - For objects: emits the parsed object directly
//   - For enums: unwraps from {result: ""} wrapper and provides partial matching
//   - Always passes through original chunks for downstream processing
type ObjectStreamTransformer struct {
	handler          FormatHandler
	structuredOutput *StructuredOutputOptions
	accumulatedText  string
	previousObject   any
	currentRunID     string
	finalResult      *ValidateAndTransformFinalResult
}

// NewObjectStreamTransformer creates a new ObjectStreamTransformer.
func NewObjectStreamTransformer(structuredOutput *StructuredOutputOptions) *ObjectStreamTransformer {
	var schema OutputSchema
	if structuredOutput != nil {
		schema = structuredOutput.Schema
	}
	return &ObjectStreamTransformer{
		handler:          CreateOutputHandler(schema),
		structuredOutput: structuredOutput,
	}
}

// Transform transforms an input chunk, optionally emitting additional object chunks.
// Returns the chunks to emit (may include the original plus 'object' chunks).
func (t *ObjectStreamTransformer) Transform(chunk stream.ChunkType) []stream.ChunkType {
	var out []stream.ChunkType

	if chunk.RunID != "" {
		t.currentRunID = chunk.RunID
	}

	if chunk.Type == "text-delta" {
		if payload, ok := chunk.Payload.(map[string]any); ok {
			if text, ok := payload["text"].(string); ok {
				t.accumulatedText += text
			}
		} else if payload, ok := chunk.Payload.(*stream.TextDeltaPayload); ok {
			t.accumulatedText += payload.Text
		} else if payload, ok := chunk.Payload.(stream.TextDeltaPayload); ok {
			t.accumulatedText += payload.Text
		}

		result := t.handler.ProcessPartialChunk(ProcessPartialChunkParams{
			AccumulatedText: t.accumulatedText,
			PreviousObject:  t.previousObject,
		})

		if result.ShouldEmit {
			if result.NewPreviousResult != nil {
				t.previousObject = result.NewPreviousResult
			}
			objectChunk := stream.ChunkType{
				BaseChunkType: stream.BaseChunkType{
					RunID: chunk.RunID,
					From:  chunk.From,
				},
				Type:   "object",
				Object: result.EmitValue,
			}
			out = append(out, objectChunk)
		}
	}

	// Validate and resolve object when text generation completes
	if chunk.Type == "text-end" {
		out = append(out, chunk)

		if strings.TrimSpace(t.accumulatedText) != "" && t.finalResult == nil {
			result := t.handler.ValidateAndTransformFinal(t.accumulatedText)
			t.finalResult = &result
			if result.Success {
				out = append(out, stream.ChunkType{
					BaseChunkType: stream.BaseChunkType{
						RunID: t.currentRunID,
						From:  stream.ChunkFromAgent,
					},
					Type:   "object-result",
					Object: result.Value,
				})
			}
		}
		return out
	}

	// Always pass through the original chunk for downstream processing
	out = append(out, chunk)
	return out
}

// Flush is called when the stream ends. Returns any remaining chunks.
func (t *ObjectStreamTransformer) Flush() []stream.ChunkType {
	var out []stream.ChunkType

	if t.finalResult != nil && !t.finalResult.Success {
		out = append(out, t.handleValidationError(t.finalResult.Error)...)
	}

	// Safety net: If text-end was never emitted, validate now as fallback
	if strings.TrimSpace(t.accumulatedText) != "" && t.finalResult == nil {
		result := t.handler.ValidateAndTransformFinal(t.accumulatedText)
		t.finalResult = &result
		if result.Success {
			out = append(out, stream.ChunkType{
				BaseChunkType: stream.BaseChunkType{
					RunID: t.currentRunID,
					From:  stream.ChunkFromAgent,
				},
				Type:   "object-result",
				Object: result.Value,
			})
		} else {
			out = append(out, t.handleValidationError(result.Error)...)
		}
	}

	return out
}

// handleValidationError handles validation errors based on error strategy.
func (t *ObjectStreamTransformer) handleValidationError(err error) []stream.ChunkType {
	if t.structuredOutput != nil {
		switch t.structuredOutput.ErrorStrategy {
		case "warn":
			// In Go, caller should handle logging; we just skip emission
			return nil
		case "fallback":
			return []stream.ChunkType{{
				BaseChunkType: stream.BaseChunkType{
					RunID: t.currentRunID,
					From:  stream.ChunkFromAgent,
				},
				Type:   "object-result",
				Object: t.structuredOutput.FallbackValue,
			}}
		}
	}

	return []stream.ChunkType{{
		BaseChunkType: stream.BaseChunkType{
			RunID: t.currentRunID,
			From:  stream.ChunkFromAgent,
		},
		Type: "error",
		Payload: map[string]any{
			"error": err,
		},
	}}
}

// ---------------------------------------------------------------------------
// CreateJsonTextStreamTransformer
// ---------------------------------------------------------------------------

// JsonTextStreamTransformer transforms object chunks into JSON text chunks
// for streaming.
//
// This transformer:
//   - For arrays: emits opening bracket, new elements, and closing bracket
//   - For objects/no-schema: emits the object as JSON
type JsonTextStreamTransformer struct {
	previousArrayLength int
	hasStartedArray     bool
	chunkCount          int
	outputSchema        *TransformedSchemaResult
}

// NewJsonTextStreamTransformer creates a new JsonTextStreamTransformer.
func NewJsonTextStreamTransformer(schema OutputSchema) *JsonTextStreamTransformer {
	return &JsonTextStreamTransformer{
		outputSchema: GetTransformedSchema(schema),
	}
}

// Transform transforms an object chunk into JSON text strings.
// Returns the text strings to emit (may be empty if chunk is not an object chunk).
func (t *JsonTextStreamTransformer) Transform(chunk stream.ChunkType) []string {
	if chunk.Type != "object" || chunk.Object == nil {
		return nil
	}

	if t.outputSchema != nil && t.outputSchema.OutputFormat == "array" {
		if arr, ok := chunk.Object.([]any); ok {
			t.chunkCount++
			var out []string

			// If this is the first chunk, decide between complete vs incremental streaming
			if t.chunkCount == 1 {
				if len(arr) > 0 {
					data, _ := json.Marshal(arr)
					out = append(out, string(data))
					t.previousArrayLength = len(arr)
					t.hasStartedArray = true
					return out
				}
			}

			// Incremental streaming mode (multiple chunks)
			if !t.hasStartedArray {
				out = append(out, "[")
				t.hasStartedArray = true
			}

			// Emit new elements that were added
			for i := t.previousArrayLength; i < len(arr); i++ {
				elementJSON, _ := json.Marshal(arr[i])
				if i > 0 {
					out = append(out, ","+string(elementJSON))
				} else {
					out = append(out, string(elementJSON))
				}
			}
			t.previousArrayLength = len(arr)
			return out
		}
	}

	// For non-array objects, just emit as JSON
	data, _ := json.Marshal(chunk.Object)
	return []string{string(data)}
}

// Flush is called when the stream ends. Returns any remaining text.
func (t *JsonTextStreamTransformer) Flush() []string {
	// Close the array when the stream ends (only for incremental streaming)
	if t.hasStartedArray && t.outputSchema != nil && t.outputSchema.OutputFormat == "array" && t.chunkCount > 1 {
		return []string{"]"}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isMapOrStruct checks if a value is a map or non-nil object (not an array/slice).
func isMapOrStruct(v any) bool {
	if v == nil {
		return false
	}
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Map
}

// isNonEmptyObject checks if a value is a non-nil map with at least one key.
func isNonEmptyObject(v any) bool {
	if v == nil {
		return false
	}
	m, ok := v.(map[string]any)
	if !ok {
		return false
	}
	return len(m) > 0
}

// formatError is a simple error type for format handler errors.
type formatError struct {
	message string
}

func (e *formatError) Error() string { return e.message }

func newFormatError(message string) error {
	return &formatError{message: message}
}
