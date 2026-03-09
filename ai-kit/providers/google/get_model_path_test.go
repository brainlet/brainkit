// Ported from: packages/google/src/get-model-path.test.ts
package google

import "testing"

func TestGetModelPath(t *testing.T) {
	t.Run("should pass through model path for models/*", func(t *testing.T) {
		result := GetModelPath("models/some-model")
		if result != "models/some-model" {
			t.Errorf("expected %q, got %q", "models/some-model", result)
		}
	})

	t.Run("should pass through model path for tunedModels/*", func(t *testing.T) {
		result := GetModelPath("tunedModels/some-model")
		if result != "tunedModels/some-model" {
			t.Errorf("expected %q, got %q", "tunedModels/some-model", result)
		}
	})

	t.Run("should add model path prefix to models without slash", func(t *testing.T) {
		result := GetModelPath("some-model")
		if result != "models/some-model" {
			t.Errorf("expected %q, got %q", "models/some-model", result)
		}
	})
}
