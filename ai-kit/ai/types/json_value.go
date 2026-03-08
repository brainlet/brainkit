// Ported from: packages/ai/src/types/json-value.ts
package aitypes

// JSONValue is a value that can be serialized and deserialized by JSON.
// It can be a string, number, boolean, object, array, or null.
//
// In Go, this is represented as any since Go's encoding/json handles
// the same set of types natively (string, float64, bool, map[string]any,
// []any, nil).
type JSONValue = any

// JSONObject is a JSON object with string keys and optional JSONValue values.
type JSONObject = map[string]JSONValue

// JSONArray is a JSON array of JSONValue elements.
type JSONArray = []JSONValue
