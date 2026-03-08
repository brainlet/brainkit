package engines

import (
	"gopkg.in/yaml.v3"
)

// YAML is a front-matter engine for parsing and stringifying YAML.
type YAML struct{}

// Parse converts a YAML front-matter string to a map of key-value pairs.
func (YAML) Parse(input string) (any, error) {
	var out any
	if err := yaml.Unmarshal([]byte(input), &out); err != nil {
		return nil, err
	}
	if out == nil {
		return map[string]any{}, nil
	}
	return out, nil
}

// Stringify converts a map of key-value pairs to a YAML string.
func (YAML) Stringify(data any) (string, error) {
	bytes, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
