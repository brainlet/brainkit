// Ported from: packages/ai/src/util/prepare-retries.test.ts
package util

import (
	"context"
	"testing"
)

func TestPrepareRetries_DefaultValues(t *testing.T) {
	result, err := PrepareRetries(nil, context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MaxRetries != 2 {
		t.Fatalf("expected default maxRetries=2, got %d", result.MaxRetries)
	}
}
