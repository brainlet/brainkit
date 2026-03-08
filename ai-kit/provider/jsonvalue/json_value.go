// Ported from: packages/provider/src/json-value/json-value.ts
package jsonvalue

// JSONValue can be a string, number, boolean, object, array, or null.
// JSON values can be serialized and deserialized by encoding/json.
// In Go, we represent this as any since encoding/json naturally handles
// the same set of types (nil, string, float64, bool, map[string]any, []any).
type JSONValue = any

// JSONObject is a JSON object: a map from string keys to JSON values.
type JSONObject = map[string]JSONValue

// JSONArray is a JSON array: a slice of JSON values.
type JSONArray = []JSONValue
