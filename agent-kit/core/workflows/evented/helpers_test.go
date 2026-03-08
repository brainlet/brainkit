// Ported from: packages/core/src/workflows/evented/helpers.test.ts
package evented

import (
	"testing"
)

// ---------------------------------------------------------------------------
// IsTripwireChunk tests
// ---------------------------------------------------------------------------

func TestIsTripwireChunk(t *testing.T) {
	t.Run("should return false for nil", func(t *testing.T) {
		if IsTripwireChunk(nil) {
			t.Error("IsTripwireChunk(nil) = true, want false")
		}
	})

	t.Run("should return false for a string", func(t *testing.T) {
		if IsTripwireChunk("hello") {
			t.Error("IsTripwireChunk(string) = true, want false")
		}
	})

	t.Run("should return false for an int", func(t *testing.T) {
		if IsTripwireChunk(42) {
			t.Error("IsTripwireChunk(int) = true, want false")
		}
	})

	t.Run("should return false for a map without type field", func(t *testing.T) {
		m := map[string]any{"payload": map[string]any{}}
		if IsTripwireChunk(m) {
			t.Error("IsTripwireChunk(no type) = true, want false")
		}
	})

	t.Run("should return false for a map with wrong type", func(t *testing.T) {
		m := map[string]any{"type": "text-delta", "payload": map[string]any{}}
		if IsTripwireChunk(m) {
			t.Error("IsTripwireChunk(wrong type) = true, want false")
		}
	})

	t.Run("should return false for a map with type=tripwire but no payload", func(t *testing.T) {
		m := map[string]any{"type": "tripwire"}
		if IsTripwireChunk(m) {
			t.Error("IsTripwireChunk(no payload) = true, want false")
		}
	})

	t.Run("should return true for a valid tripwire chunk map", func(t *testing.T) {
		m := map[string]any{
			"type":    "tripwire",
			"payload": map[string]any{"reason": "test"},
		}
		if !IsTripwireChunk(m) {
			t.Error("IsTripwireChunk(valid map) = false, want true")
		}
	})

	t.Run("should return true for a typed TripwireChunk struct", func(t *testing.T) {
		chunk := TripwireChunk{
			Type:    "tripwire",
			Payload: TripwireChunkPayload{Reason: "test"},
		}
		if !IsTripwireChunk(chunk) {
			t.Error("IsTripwireChunk(struct) = false, want true")
		}
	})

	t.Run("should return true for a pointer to TripwireChunk struct", func(t *testing.T) {
		chunk := &TripwireChunk{
			Type:    "tripwire",
			Payload: TripwireChunkPayload{Reason: "test"},
		}
		if !IsTripwireChunk(chunk) {
			t.Error("IsTripwireChunk(*struct) = false, want true")
		}
	})

	t.Run("should return false for a TripwireChunk with wrong type", func(t *testing.T) {
		chunk := TripwireChunk{
			Type:    "not-tripwire",
			Payload: TripwireChunkPayload{Reason: "test"},
		}
		if IsTripwireChunk(chunk) {
			t.Error("IsTripwireChunk(wrong type struct) = true, want false")
		}
	})

	t.Run("should return false for a nil pointer to TripwireChunk", func(t *testing.T) {
		var chunk *TripwireChunk
		if IsTripwireChunk(chunk) {
			t.Error("IsTripwireChunk(nil pointer) = true, want false")
		}
	})
}

// ---------------------------------------------------------------------------
// TripWire tests
// ---------------------------------------------------------------------------

func TestTripWire_Error(t *testing.T) {
	t.Run("should implement error interface with message", func(t *testing.T) {
		tw := &TripWire{Message: "rate limit exceeded"}
		if tw.Error() != "rate limit exceeded" {
			t.Errorf("TripWire.Error() = %q, want %q", tw.Error(), "rate limit exceeded")
		}
	})

	t.Run("should return empty string for empty message", func(t *testing.T) {
		tw := &TripWire{}
		if tw.Error() != "" {
			t.Errorf("TripWire.Error() = %q, want empty string", tw.Error())
		}
	})
}

// ---------------------------------------------------------------------------
// CreateTripWireFromChunk tests
// ---------------------------------------------------------------------------

func TestCreateTripWireFromChunk(t *testing.T) {
	t.Run("should create TripWire from chunk with reason", func(t *testing.T) {
		chunk := TripwireChunk{
			Type: "tripwire",
			Payload: TripwireChunkPayload{
				Reason: "rate limit exceeded",
			},
		}
		tw := CreateTripWireFromChunk(chunk)
		if tw.Message != "rate limit exceeded" {
			t.Errorf("tw.Message = %q, want %q", tw.Message, "rate limit exceeded")
		}
	})

	t.Run("should use default reason when reason is empty", func(t *testing.T) {
		chunk := TripwireChunk{
			Type:    "tripwire",
			Payload: TripwireChunkPayload{},
		}
		tw := CreateTripWireFromChunk(chunk)
		if tw.Message != "Agent tripwire triggered" {
			t.Errorf("tw.Message = %q, want %q", tw.Message, "Agent tripwire triggered")
		}
	})

	t.Run("should preserve retry flag", func(t *testing.T) {
		retry := true
		chunk := TripwireChunk{
			Type: "tripwire",
			Payload: TripwireChunkPayload{
				Reason: "test",
				Retry:  &retry,
			},
		}
		tw := CreateTripWireFromChunk(chunk)
		if tw.Options == nil {
			t.Fatal("tw.Options is nil")
		}
		if tw.Options.Retry == nil || *tw.Options.Retry != true {
			t.Errorf("tw.Options.Retry = %v, want true", tw.Options.Retry)
		}
	})

	t.Run("should preserve metadata", func(t *testing.T) {
		meta := map[string]any{"key": "value"}
		chunk := TripwireChunk{
			Type: "tripwire",
			Payload: TripwireChunkPayload{
				Reason:   "test",
				Metadata: meta,
			},
		}
		tw := CreateTripWireFromChunk(chunk)
		if tw.Options == nil {
			t.Fatal("tw.Options is nil")
		}
		if tw.Options.Metadata == nil {
			t.Fatal("tw.Options.Metadata is nil")
		}
	})

	t.Run("should preserve processorID", func(t *testing.T) {
		chunk := TripwireChunk{
			Type: "tripwire",
			Payload: TripwireChunkPayload{
				Reason:      "test",
				ProcessorID: "proc-1",
			},
		}
		tw := CreateTripWireFromChunk(chunk)
		if tw.ProcessorID != "proc-1" {
			t.Errorf("tw.ProcessorID = %q, want %q", tw.ProcessorID, "proc-1")
		}
	})
}

// ---------------------------------------------------------------------------
// GetTextDeltaFromChunk tests
// ---------------------------------------------------------------------------

func TestGetTextDeltaFromChunk(t *testing.T) {
	t.Run("should return empty for non-text-delta type", func(t *testing.T) {
		chunk := map[string]any{"type": "tripwire", "textDelta": "hello"}
		text, ok := GetTextDeltaFromChunk(chunk, false)
		if ok {
			t.Errorf("GetTextDeltaFromChunk should return ok=false for non-text-delta, got text=%q", text)
		}
	})

	t.Run("should return textDelta for V1 model", func(t *testing.T) {
		chunk := map[string]any{"type": "text-delta", "textDelta": "hello world"}
		text, ok := GetTextDeltaFromChunk(chunk, false)
		if !ok {
			t.Fatal("GetTextDeltaFromChunk returned ok=false, want true")
		}
		if text != "hello world" {
			t.Errorf("text = %q, want %q", text, "hello world")
		}
	})

	t.Run("should return payload.text for V2 model", func(t *testing.T) {
		chunk := map[string]any{
			"type": "text-delta",
			"payload": map[string]any{
				"text": "v2 text",
			},
		}
		text, ok := GetTextDeltaFromChunk(chunk, true)
		if !ok {
			t.Fatal("GetTextDeltaFromChunk returned ok=false, want true")
		}
		if text != "v2 text" {
			t.Errorf("text = %q, want %q", text, "v2 text")
		}
	})

	t.Run("should return false for V2 model without payload", func(t *testing.T) {
		chunk := map[string]any{"type": "text-delta"}
		_, ok := GetTextDeltaFromChunk(chunk, true)
		if ok {
			t.Error("GetTextDeltaFromChunk should return ok=false for V2 without payload")
		}
	})

	t.Run("should return false for V2 model with non-map payload", func(t *testing.T) {
		chunk := map[string]any{"type": "text-delta", "payload": "not-a-map"}
		_, ok := GetTextDeltaFromChunk(chunk, true)
		if ok {
			t.Error("GetTextDeltaFromChunk should return ok=false for V2 with non-map payload")
		}
	})

	t.Run("should return false for V1 model without textDelta", func(t *testing.T) {
		chunk := map[string]any{"type": "text-delta"}
		_, ok := GetTextDeltaFromChunk(chunk, false)
		if ok {
			t.Error("GetTextDeltaFromChunk should return ok=false for V1 without textDelta")
		}
	})

	t.Run("should return false for empty chunk without type", func(t *testing.T) {
		chunk := map[string]any{}
		_, ok := GetTextDeltaFromChunk(chunk, false)
		if ok {
			t.Error("GetTextDeltaFromChunk should return ok=false for empty chunk")
		}
	})
}

// ---------------------------------------------------------------------------
// ResolveCurrentState tests
// ---------------------------------------------------------------------------

func TestResolveCurrentState(t *testing.T) {
	t.Run("should return stepResult.__state when available", func(t *testing.T) {
		expected := map[string]any{"key": "from-step-result"}
		result := ResolveCurrentState(ResolveStateParams{
			StepResult: map[string]any{
				"__state": expected,
			},
			StepResults: map[string]any{
				"__state": map[string]any{"key": "from-step-results"},
			},
			State: map[string]any{"key": "from-state"},
		})
		if result["key"] != "from-step-result" {
			t.Errorf("result[key] = %v, want from-step-result", result["key"])
		}
	})

	t.Run("should fall back to stepResults.__state", func(t *testing.T) {
		expected := map[string]any{"key": "from-step-results"}
		result := ResolveCurrentState(ResolveStateParams{
			StepResult: nil,
			StepResults: map[string]any{
				"__state": expected,
			},
			State: map[string]any{"key": "from-state"},
		})
		if result["key"] != "from-step-results" {
			t.Errorf("result[key] = %v, want from-step-results", result["key"])
		}
	})

	t.Run("should fall back to state", func(t *testing.T) {
		expected := map[string]any{"key": "from-state"}
		result := ResolveCurrentState(ResolveStateParams{
			StepResult:  nil,
			StepResults: nil,
			State:       expected,
		})
		if result["key"] != "from-state" {
			t.Errorf("result[key] = %v, want from-state", result["key"])
		}
	})

	t.Run("should return empty map when all sources are nil", func(t *testing.T) {
		result := ResolveCurrentState(ResolveStateParams{})
		if result == nil {
			t.Fatal("result is nil, want empty map")
		}
		if len(result) != 0 {
			t.Errorf("len(result) = %d, want 0", len(result))
		}
	})

	t.Run("should skip stepResult when __state is not a map", func(t *testing.T) {
		expected := map[string]any{"key": "from-state"}
		result := ResolveCurrentState(ResolveStateParams{
			StepResult: map[string]any{
				"__state": "not-a-map",
			},
			State: expected,
		})
		if result["key"] != "from-state" {
			t.Errorf("result[key] = %v, want from-state", result["key"])
		}
	})

	t.Run("should skip stepResult when it is not a map", func(t *testing.T) {
		expected := map[string]any{"key": "from-state"}
		result := ResolveCurrentState(ResolveStateParams{
			StepResult: "not-a-map",
			State:      expected,
		})
		if result["key"] != "from-state" {
			t.Errorf("result[key] = %v, want from-state", result["key"])
		}
	})

	t.Run("should skip stepResults when __state is not a map", func(t *testing.T) {
		expected := map[string]any{"key": "from-state"}
		result := ResolveCurrentState(ResolveStateParams{
			StepResults: map[string]any{
				"__state": 42,
			},
			State: expected,
		})
		if result["key"] != "from-state" {
			t.Errorf("result[key] = %v, want from-state", result["key"])
		}
	})
}
