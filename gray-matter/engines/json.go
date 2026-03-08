package engines

import (
	"encoding/json"
)

// JSON is a front-matter engine that parses and stringifies JSON.
type JSON struct{}

// Parse converts a JSON string to a map of key-value pairs.
func (j JSON) Parse(input string) (any, error) {
	var result any
	if err := json.Unmarshal([]byte(input), &result); err != nil {
		return nil, err
	}
	if result == nil {
		return map[string]any{}, nil
	}
	return result, nil
}

// Stringify converts a map of key-value pairs to a JSON string.
func (j JSON) Stringify(data any) (string, error) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}
