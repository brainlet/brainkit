// Ported from: packages/core/src/request-context/index.test.ts
package requestcontext

import (
	"encoding/json"
	"math"
	"testing"
)

func TestConstructor_FromMap(t *testing.T) {
	// "should construct from a plain object (e.g. deserialized from JSON)"
	original := NewRequestContext()
	original.Set("userTier", "free")
	original.Set("feature", "dark-mode")
	original.Set("count", 42)

	serialized := toJSON(t, original)
	restored := NewRequestContextFromMap(serialized)

	assertEqual(t, restored.Get("userTier"), "free")
	assertEqual(t, restored.Get("feature"), "dark-mode")
	// JSON round-trip turns int into float64; match that semantics.
	assertJSONNumber(t, serialized, "count", 42)
	assertEqual(t, restored.Size(), 3)
}

func TestConstructor_EmptyMap(t *testing.T) {
	// "should construct from an empty plain object"
	restored := NewRequestContextFromMap(map[string]any{})
	assertEqual(t, restored.Size(), 0)
}

func TestConstructor_Default(t *testing.T) {
	// "should still construct from undefined" (no args in Go)
	ctx := NewRequestContext()
	assertEqual(t, ctx.Size(), 0)
}

func TestConstructor_FromEntries(t *testing.T) {
	// "should still construct from an array of tuples"
	ctx := NewRequestContextFromEntries([]Entry{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
	})
	assertEqual(t, ctx.Get("key1"), "value1")
	assertEqual(t, ctx.Get("key2"), "value2")
}

func TestToJSON_SerializableValues(t *testing.T) {
	// "should correctly serialize serializable values"
	ctx := NewRequestContext()
	ctx.Set("string", "hello")
	ctx.Set("number", 42)
	ctx.Set("boolean", true)
	ctx.Set("null", nil)
	ctx.Set("object", map[string]any{"nested": "value"})
	ctx.Set("array", []int{1, 2, 3})

	j := toJSON(t, ctx)

	assertEqual(t, j["string"], "hello")
	assertJSONNumber(t, j, "number", 42)
	assertEqual(t, j["boolean"], true)
	if j["null"] != nil {
		t.Errorf("expected null to be nil, got %v", j["null"])
	}

	obj, ok := j["object"].(map[string]any)
	if !ok {
		t.Fatalf("expected object to be map[string]any, got %T", j["object"])
	}
	assertEqual(t, obj["nested"], "value")

	arr, ok := j["array"].([]any)
	if !ok {
		t.Fatalf("expected array to be []any, got %T", j["array"])
	}
	assertEqual(t, len(arr), 3)
}

func TestToJSON_SkipNonSerializable(t *testing.T) {
	// "should skip functions" — in Go, func values are not JSON-serializable.
	// Go's json.Marshal returns an error for func types, so isSerializable filters them.
	ctx := NewRequestContext()
	ctx.Set("serializable", "value")
	ctx.Set("func", func() string { return "function" })

	j := toJSON(t, ctx)

	assertEqual(t, j["serializable"], "value")
	if _, ok := j["func"]; ok {
		t.Error("expected 'func' key to be skipped in JSON output")
	}
}

func TestToJSON_SkipUnserializableChannels(t *testing.T) {
	// Go equivalent of "should skip symbols" — channels are not JSON-serializable.
	ctx := NewRequestContext()
	ctx.Set("serializable", "value")
	ctx.Set("channel", make(chan int))

	j := toJSON(t, ctx)

	assertEqual(t, j["serializable"], "value")
	if _, ok := j["channel"]; ok {
		t.Error("expected 'channel' key to be skipped in JSON output")
	}
}

func TestToJSON_SkipCircularReferences(t *testing.T) {
	// "should skip objects with circular references"
	// In Go, we simulate this with a value that json.Marshal rejects.
	ctx := NewRequestContext()
	ctx.Set("serializable", "value")
	// math.NaN() causes json.Marshal to return an unsupported value error.
	ctx.Set("circular", math.NaN())

	j := toJSON(t, ctx)

	assertEqual(t, j["serializable"], "value")
	if _, ok := j["circular"]; ok {
		t.Error("expected 'circular' key to be skipped in JSON output")
	}
}

func TestToJSON_SkipNonMarshallable(t *testing.T) {
	// Go equivalent of "should skip objects without toJSON method (e.g., RPC proxies)"
	// In Go, a type with a broken MarshalJSON method is the equivalent.
	ctx := NewRequestContext()
	ctx.Set("serializable", "value")
	ctx.Set("rpcProxy", brokenMarshaler{})

	j := toJSON(t, ctx)

	assertEqual(t, j["serializable"], "value")
	if _, ok := j["rpcProxy"]; ok {
		t.Error("expected 'rpcProxy' key to be skipped in JSON output")
	}
}

// brokenMarshaler simulates an RPC proxy that fails JSON serialization.
type brokenMarshaler struct{}

func (brokenMarshaler) MarshalJSON() ([]byte, error) {
	return nil, &json.UnsupportedTypeError{}
}

func TestToJSON_NilValues(t *testing.T) {
	// "should handle undefined values" — nil is Go's equivalent
	ctx := NewRequestContext()
	ctx.Set("defined", "value")
	ctx.Set("undefined", nil)

	j := toJSON(t, ctx)

	assertEqual(t, j["defined"], "value")
	if v, ok := j["undefined"]; !ok {
		t.Error("expected 'undefined' key to be present in JSON output")
	} else if v != nil {
		t.Errorf("expected 'undefined' to be nil, got %v", v)
	}
}

func TestToJSON_Empty(t *testing.T) {
	// "should return empty object for empty RequestContext"
	ctx := NewRequestContext()

	j := toJSON(t, ctx)

	assertEqual(t, len(j), 0)
}

func TestToJSON_MixedValues(t *testing.T) {
	// "should return only serializable values when mixed with non-serializable values"
	ctx := NewRequestContext()
	ctx.Set("userId", "user-123")
	ctx.Set("feature", "dark-mode")
	ctx.Set("callback", func() {})
	ctx.Set("badData", math.Inf(1)) // +Inf is not JSON-serializable

	j := toJSON(t, ctx)

	assertEqual(t, j["userId"], "user-123")
	assertEqual(t, j["feature"], "dark-mode")
	if _, ok := j["callback"]; ok {
		t.Error("expected 'callback' key to be skipped in JSON output")
	}
	if _, ok := j["badData"]; ok {
		t.Error("expected 'badData' key to be skipped in JSON output")
	}
}

func TestConstants(t *testing.T) {
	assertEqual(t, MastraResourceIDKey, "mastra__resourceId")
	assertEqual(t, MastraThreadIDKey, "mastra__threadId")
}

// --- helpers ---

// toJSON marshals then unmarshals through JSON to simulate the toJSON() round-trip,
// matching how the TS tests consume the result of toJSON().
func toJSON(t *testing.T, rc *RequestContext) map[string]any {
	t.Helper()
	data, err := rc.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	return result
}

func assertEqual(t *testing.T, got, want any) {
	t.Helper()
	if got != want {
		t.Errorf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}

// assertJSONNumber handles the fact that JSON unmarshaling produces float64 for numbers.
func assertJSONNumber(t *testing.T, m map[string]any, key string, want float64) {
	t.Helper()
	v, ok := m[key]
	if !ok {
		t.Fatalf("key %q not found in map", key)
	}
	f, ok := v.(float64)
	if !ok {
		t.Fatalf("expected float64 for key %q, got %T", key, v)
	}
	if f != want {
		t.Errorf("key %q: got %v, want %v", key, f, want)
	}
}
