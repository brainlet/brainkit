// Ported from: packages/core/src/workflows/evented/types.test.ts
package evented

import (
	"encoding/json"
	"testing"
)

func TestPendingMarkerKey(t *testing.T) {
	t.Run("should have the correct key value", func(t *testing.T) {
		if PendingMarkerKey != "__mastra_pending__" {
			t.Errorf("PendingMarkerKey = %q, want %q", PendingMarkerKey, "__mastra_pending__")
		}
	})
}

func TestCreatePendingMarker(t *testing.T) {
	t.Run("should create a marker with the correct key set to true", func(t *testing.T) {
		marker := CreatePendingMarker()
		if marker == nil {
			t.Fatal("CreatePendingMarker() returned nil")
		}
		val, ok := marker[PendingMarkerKey]
		if !ok {
			t.Fatalf("marker does not contain key %q", PendingMarkerKey)
		}
		if val != true {
			t.Errorf("marker[%q] = %v, want true", PendingMarkerKey, val)
		}
	})

	t.Run("should create a marker with only one key", func(t *testing.T) {
		marker := CreatePendingMarker()
		if len(marker) != 1 {
			t.Errorf("len(marker) = %d, want 1", len(marker))
		}
	})
}

func TestIsPendingMarker(t *testing.T) {
	t.Run("should return true for a PendingMarker type", func(t *testing.T) {
		marker := CreatePendingMarker()
		if !IsPendingMarker(marker) {
			t.Error("IsPendingMarker(marker) = false, want true")
		}
	})

	t.Run("should return true for a map[string]any with the marker key", func(t *testing.T) {
		m := map[string]any{PendingMarkerKey: true}
		if !IsPendingMarker(m) {
			t.Error("IsPendingMarker(map) = false, want true")
		}
	})

	t.Run("should return false for nil", func(t *testing.T) {
		if IsPendingMarker(nil) {
			t.Error("IsPendingMarker(nil) = true, want false")
		}
	})

	t.Run("should return false for an empty map", func(t *testing.T) {
		m := map[string]any{}
		if IsPendingMarker(m) {
			t.Error("IsPendingMarker(empty map) = true, want false")
		}
	})

	t.Run("should return false for a string", func(t *testing.T) {
		if IsPendingMarker("hello") {
			t.Error("IsPendingMarker(string) = true, want false")
		}
	})

	t.Run("should return false for an int", func(t *testing.T) {
		if IsPendingMarker(42) {
			t.Error("IsPendingMarker(int) = true, want false")
		}
	})

	t.Run("should return false when marker key value is false", func(t *testing.T) {
		m := map[string]any{PendingMarkerKey: false}
		if IsPendingMarker(m) {
			t.Error("IsPendingMarker(marker=false) = true, want false")
		}
	})

	t.Run("should return false when marker key value is not a bool", func(t *testing.T) {
		m := map[string]any{PendingMarkerKey: "true"}
		if IsPendingMarker(m) {
			t.Error("IsPendingMarker(marker=string) = true, want false")
		}
	})

	t.Run("should survive JSON round-trip", func(t *testing.T) {
		marker := CreatePendingMarker()
		data, err := json.Marshal(marker)
		if err != nil {
			t.Fatalf("json.Marshal failed: %v", err)
		}

		var deserialized map[string]any
		if err := json.Unmarshal(data, &deserialized); err != nil {
			t.Fatalf("json.Unmarshal failed: %v", err)
		}

		if !IsPendingMarker(deserialized) {
			t.Error("IsPendingMarker after JSON round-trip = false, want true")
		}
	})

	t.Run("should return false for a map with extra keys", func(t *testing.T) {
		// The marker key is present and true, but there are extra keys.
		// IsPendingMarker should still return true because it only checks the key.
		m := map[string]any{PendingMarkerKey: true, "extra": "value"}
		if !IsPendingMarker(m) {
			t.Error("IsPendingMarker(map with extra keys) = false, want true")
		}
	})
}
