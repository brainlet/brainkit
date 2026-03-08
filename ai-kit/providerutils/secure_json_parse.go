// Ported from: packages/provider-utils/src/secure-json-parse.ts
// Licensed under BSD-3-Clause (this file only)
// Code adapted from https://github.com/fastify/secure-json-parse
package providerutils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var suspectProtoRx = regexp.MustCompile(`"(?:_|\\u005[Ff])(?:_|\\u005[Ff])(?:p|\\u0070)(?:r|\\u0072)(?:o|\\u006[Ff])(?:t|\\u0074)(?:o|\\u006[Ff])(?:_|\\u005[Ff])(?:_|\\u005[Ff])"\s*:`)
var suspectConstructorRx = regexp.MustCompile(`"(?:c|\\u0063)(?:o|\\u006[Ff])(?:n|\\u006[Ee])(?:s|\\u0073)(?:t|\\u0074)(?:r|\\u0072)(?:u|\\u0075)(?:c|\\u0063)(?:t|\\u0074)(?:o|\\u006[Ff])(?:r|\\u0072)"\s*:`)

// SecureJsonParse parses a JSON string, checking for prototype pollution attacks.
func SecureJsonParse(text string) (interface{}, error) {
	var obj interface{}
	if err := json.Unmarshal([]byte(text), &obj); err != nil {
		return nil, err
	}

	// Ignore null and non-objects
	if obj == nil {
		return obj, nil
	}
	if _, isMap := obj.(map[string]interface{}); !isMap {
		if _, isSlice := obj.([]interface{}); !isSlice {
			return obj, nil
		}
	}

	if !suspectProtoRx.MatchString(text) && !suspectConstructorRx.MatchString(text) {
		return obj, nil
	}

	// Scan result for proto keys
	if err := filterProto(obj); err != nil {
		return nil, err
	}

	return obj, nil
}

func filterProto(obj interface{}) error {
	next := []interface{}{obj}

	for len(next) > 0 {
		nodes := next
		next = nil

		for _, node := range nodes {
			m, ok := node.(map[string]interface{})
			if !ok {
				continue
			}

			if _, has := m["__proto__"]; has {
				return fmt.Errorf("Object contains forbidden prototype property")
			}

			if constructor, has := m["constructor"]; has {
				if constructorMap, isMap := constructor.(map[string]interface{}); isMap {
					if _, hasProt := constructorMap["prototype"]; hasProt {
						return fmt.Errorf("Object contains forbidden prototype property")
					}
				}
			}

			for _, v := range m {
				switch child := v.(type) {
				case map[string]interface{}:
					next = append(next, child)
				case []interface{}:
					for _, item := range child {
						if _, isMap := item.(map[string]interface{}); isMap {
							next = append(next, item)
						}
					}
				}
			}
		}
	}

	return nil
}

// SecureJsonParseString is a convenience function that parses a JSON string
// and returns a string representation error message on failure.
func SecureJsonParseString(text string) (interface{}, string) {
	result, err := SecureJsonParse(text)
	if err != nil {
		return nil, err.Error()
	}
	return result, ""
}

// IsParsableJson checks whether the input string is valid JSON.
func IsParsableJson(input string) bool {
	_, err := SecureJsonParse(input)
	return err == nil
}

// MustSecureJsonParse parses JSON or panics. For internal use.
func MustSecureJsonParse(text string) interface{} {
	result, err := SecureJsonParse(text)
	if err != nil {
		panic(err)
	}
	return result
}

// Note: Go's encoding/json already handles __proto__ correctly since
// Go maps don't have prototype chains. However, we keep this for
// 1:1 fidelity with the TypeScript implementation and to detect
// suspicious JSON that might be used for attacks on other systems.
var _ = strings.Contains // suppress unused import
