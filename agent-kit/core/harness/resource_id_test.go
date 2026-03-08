// Ported from: packages/core/src/harness/resource-id.test.ts
package harness

import (
	"testing"
)

func TestHarnessResourceID(t *testing.T) {
	// Helper to create a harness for resource ID tests
	createHarness := func(resourceID string) *Harness {
		h, err := New(HarnessConfig{
			ID:         "test-harness",
			ResourceID: resourceID,
			Modes: []HarnessMode{
				{ID: "default", Name: "Default", Default: true},
			},
		})
		if err != nil {
			t.Fatalf("failed to create harness: %v", err)
		}
		return h
	}

	t.Run("getDefaultResourceId", func(t *testing.T) {
		t.Run("returns the harness id when no explicit resourceId is configured", func(t *testing.T) {
			h := createHarness("")
			if got := h.GetDefaultResourceID(); got != "test-harness" {
				t.Errorf("GetDefaultResourceID() = %q, want %q", got, "test-harness")
			}
		})

		t.Run("returns the configured resourceId when one is provided", func(t *testing.T) {
			h := createHarness("custom-resource")
			if got := h.GetDefaultResourceID(); got != "custom-resource" {
				t.Errorf("GetDefaultResourceID() = %q, want %q", got, "custom-resource")
			}
		})

		t.Run("still returns the original default after SetResourceID is called", func(t *testing.T) {
			h := createHarness("original")
			h.SetResourceID("changed")
			if got := h.GetResourceID(); got != "changed" {
				t.Errorf("GetResourceID() = %q, want %q", got, "changed")
			}
			if got := h.GetDefaultResourceID(); got != "original" {
				t.Errorf("GetDefaultResourceID() = %q, want %q", got, "original")
			}
		})
	})

	t.Run("getKnownResourceIds", func(t *testing.T) {
		t.Skip("not yet implemented - requires getKnownResourceIds method and storage integration")
	})
}
