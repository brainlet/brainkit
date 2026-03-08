// Ported from: packages/ai/src/util/parse-partial-json.ts
package util

import "encoding/json"

// ParseState represents the result state of a partial JSON parse attempt.
type ParseState string

const (
	ParseStateUndefinedInput ParseState = "undefined-input"
	ParseStateSuccessfulParse ParseState = "successful-parse"
	ParseStateRepairedParse  ParseState = "repaired-parse"
	ParseStateFailedParse    ParseState = "failed-parse"
)

// ParsePartialJSONResult holds the result of parsing partial JSON.
type ParsePartialJSONResult struct {
	Value interface{}
	State ParseState
}

// ParsePartialJSON attempts to parse a JSON string, repairing it if necessary.
// If jsonText is nil (represented by passing a pointer), returns undefined-input state.
func ParsePartialJSON(jsonText *string) ParsePartialJSONResult {
	if jsonText == nil {
		return ParsePartialJSONResult{Value: nil, State: ParseStateUndefinedInput}
	}

	text := *jsonText

	// Try parsing as-is
	var value interface{}
	if err := json.Unmarshal([]byte(text), &value); err == nil {
		return ParsePartialJSONResult{Value: value, State: ParseStateSuccessfulParse}
	}

	// Try fixing and parsing
	fixed := FixJSON(text)
	if err := json.Unmarshal([]byte(fixed), &value); err == nil {
		return ParsePartialJSONResult{Value: value, State: ParseStateRepairedParse}
	}

	return ParsePartialJSONResult{Value: nil, State: ParseStateFailedParse}
}
