// Ported from: packages/core/src/utils.ts
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// Delay pauses execution for the given duration.
// This is the Go equivalent of: const delay = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));
func Delay(d time.Duration) {
	time.Sleep(d)
}

// IsPlainObject checks if a value is a plain map (not an array, function, etc.).
// In Go this checks for map[string]any specifically.
func IsPlainObject(value any) bool {
	if value == nil {
		return false
	}
	v := reflect.ValueOf(value)
	return v.Kind() == reflect.Map && v.Type().Key().Kind() == reflect.String
}

// DeepMerge deep-merges source into target, recursively merging nested maps.
// Arrays, functions, and other non-map values are replaced (not merged).
func DeepMerge(target, source map[string]any) map[string]any {
	output := make(map[string]any)
	for k, v := range target {
		output[k] = v
	}

	if source == nil {
		return output
	}

	for k, sourceVal := range source {
		targetVal, exists := output[k]
		targetMap, targetIsMap := targetVal.(map[string]any)
		sourceMap, sourceIsMap := sourceVal.(map[string]any)

		if exists && targetIsMap && sourceIsMap {
			output[k] = DeepMerge(targetMap, sourceMap)
		} else if sourceVal != nil {
			output[k] = sourceVal
		}
	}

	return output
}

// DeepEqual performs a deep equality comparison for comparing two values.
// Handles primitives, slices, maps, and time.Time instances.
func DeepEqual(a, b any) bool {
	return reflect.DeepEqual(a, b)
}

// GenerateEmptyFromSchema generates an empty object from a JSON schema string.
// Returns a map with default zero-values for each property.
func GenerateEmptyFromSchema(schema string) map[string]any {
	var parsed map[string]any
	if err := json.Unmarshal([]byte(schema), &parsed); err != nil {
		return map[string]any{}
	}

	schemaType, _ := parsed["type"].(string)
	if schemaType != "object" {
		return map[string]any{}
	}

	properties, ok := parsed["properties"].(map[string]any)
	if !ok {
		return map[string]any{}
	}

	typeDefaults := map[string]any{
		"string":  "",
		"array":   []any{},
		"object":  map[string]any{},
		"number":  float64(0),
		"integer": float64(0),
		"boolean": false,
	}

	obj := make(map[string]any)
	for key, propRaw := range properties {
		prop, ok := propRaw.(map[string]any)
		if !ok {
			obj[key] = nil
			continue
		}
		propType, _ := prop["type"].(string)
		if defVal, found := typeDefaults[propType]; found {
			obj[key] = defVal
		} else {
			obj[key] = nil
		}
	}

	return obj
}

// TagMaskOptions holds optional callbacks for the MaskStreamTags function.
type TagMaskOptions struct {
	// OnStart is called when masking begins (opening tag detected).
	OnStart func()
	// OnEnd is called when masking ends (closing tag detected).
	OnEnd func()
	// OnMask is called for each chunk that is masked (between tags).
	OnMask func(chunk string)
}

// MaskStreamTags transforms a channel-based stream by masking content between XML tags.
// tag is the tag name to mask between (e.g. for <foo>...</foo>, use "foo").
func MaskStreamTags(stream <-chan string, tag string, options TagMaskOptions) <-chan string {
	out := make(chan string)

	go func() {
		defer close(out)

		openTag := "<" + tag + ">"
		closeTag := "</" + tag + ">"

		var buffer string
		var fullContent string
		isMasking := false
		isBuffering := false

		for chunk := range stream {
			fullContent += chunk

			if isBuffering {
				buffer += chunk
			}

			chunkHasTag := strings.HasPrefix(strings.TrimSpace(chunk), openTag)
			bufferHasTag := !chunkHasTag && isBuffering && strings.HasPrefix(openTag, strings.TrimSpace(buffer))

			// Check if we should start masking
			if !isMasking && (chunkHasTag || bufferHasTag) {
				isMasking = true
				isBuffering = false
				buffer = ""
				if options.OnStart != nil {
					options.OnStart()
				}
			}

			// Check if we should start buffering
			if !isMasking && !isBuffering && strings.HasPrefix(openTag, strings.TrimSpace(chunk)) && strings.TrimSpace(chunk) != "" {
				isBuffering = true
				buffer += chunk
				continue
			}

			// Buffering deviation check
			if isBuffering && buffer != "" && !strings.HasPrefix(openTag, strings.TrimSpace(buffer)) {
				out <- buffer
				buffer = ""
				isBuffering = false
				continue
			}

			// Check if we should stop masking
			if isMasking && strings.Contains(fullContent, closeTag) {
				if options.OnMask != nil {
					options.OnMask(chunk)
				}
				if options.OnEnd != nil {
					options.OnEnd()
				}
				isMasking = false
				fullContent = ""
				continue
			}

			// Currently masking
			if isMasking {
				if options.OnMask != nil {
					options.OnMask(chunk)
				}
				continue
			}

			out <- chunk
		}
	}()

	return out
}

// CheckEvalStorageFields checks that required evaluation storage fields are present.
// Returns true if all required fields are present, false otherwise.
func CheckEvalStorageFields(traceObject map[string]any) bool {
	requiredFields := []string{"input", "output", "agentName", "metricName", "instructions", "globalRunId", "runId"}
	var missingFields []string

	for _, field := range requiredFields {
		val, exists := traceObject[field]
		if !exists || val == nil || val == "" {
			missingFields = append(missingFields, field)
		}
	}

	if len(missingFields) > 0 {
		fmt.Printf("Skipping evaluation storage due to missing required fields: %v\n", missingFields)
		return false
	}

	return true
}

// sqlIdentifierPattern validates SQL identifiers.
var sqlIdentifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ParseSQLIdentifier parses and returns a valid SQL identifier (table or column name).
// The identifier must start with a letter or underscore, contain only letters,
// numbers, or underscores, and be at most 63 characters long.
func ParseSQLIdentifier(name, kind string) (string, error) {
	if kind == "" {
		kind = "identifier"
	}
	if !sqlIdentifierPattern.MatchString(name) || len(name) > 63 {
		return "", fmt.Errorf(
			"invalid %s: %s. Must start with a letter or underscore, contain only letters, numbers, or underscores, and be at most 63 characters long",
			kind, name,
		)
	}
	return name, nil
}

// ParseFieldKey parses and returns a valid dot-separated SQL field key (e.g. "user.profile.name").
// Each segment must be a valid SQL identifier.
func ParseFieldKey(key string) (string, error) {
	if key == "" {
		return "", fmt.Errorf("field key cannot be empty")
	}
	segments := strings.Split(key, ".")
	for _, segment := range segments {
		if !sqlIdentifierPattern.MatchString(segment) || len(segment) > 63 {
			return "", fmt.Errorf("invalid field key segment: %s in %s", segment, key)
		}
	}
	return key, nil
}

// OmitKeys removes specific keys from a map.
func OmitKeys(obj map[string]any, keysToOmit []string) map[string]any {
	omitSet := make(map[string]bool, len(keysToOmit))
	for _, k := range keysToOmit {
		omitSet[k] = true
	}

	result := make(map[string]any)
	for k, v := range obj {
		if !omitSet[k] {
			result[k] = v
		}
	}
	return result
}

// SelectFields selectively extracts specific fields from an object using dot notation.
// Does not error if fields don't exist - simply omits them from the result.
func SelectFields(obj map[string]any, fields []string) map[string]any {
	if obj == nil {
		return nil
	}

	result := make(map[string]any)
	for _, field := range fields {
		value := GetNestedValue(obj, field)
		if value != nil {
			SetNestedValue(result, field, value)
		}
	}
	return result
}

// GetNestedValue gets a nested value from a map using dot notation.
func GetNestedValue(obj map[string]any, path string) any {
	keys := strings.Split(path, ".")
	var current any = obj

	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = m[key]
	}
	return current
}

// SetNestedValue sets a nested value in a map using dot notation.
func SetNestedValue(obj map[string]any, path string, value any) {
	keys := strings.Split(path, ".")
	if len(keys) == 0 {
		return
	}

	lastKey := keys[len(keys)-1]
	current := obj

	for _, key := range keys[:len(keys)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			next = make(map[string]any)
			current[key] = next
		}
		current = next
	}

	current[lastKey] = value
}

// RemoveUndefinedValues removes entries with nil values from a map.
// This is the Go equivalent of removing undefined values.
func RemoveUndefinedValues(obj map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range obj {
		if v != nil {
			result[k] = v
		}
	}
	return result
}

// CreateDeterministicID creates a deterministic hash-based ID from input.
// Returns the first 8 characters of the SHA-256 hex digest.
func CreateDeterministicID(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:])[:8]
}
