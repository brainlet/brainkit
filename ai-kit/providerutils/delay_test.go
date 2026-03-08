// Ported from: packages/provider-utils/src/delay.test.ts
package providerutils

import (
	"context"
	"testing"
	"time"
)

func TestDelay_ResolveAfterDelay(t *testing.T) {
	start := time.Now()
	err := Delay(nil, 50*time.Millisecond)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("delay was too short: %v", elapsed)
	}
}

func TestDelay_ZeroDuration(t *testing.T) {
	err := Delay(nil, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelay_AlreadyCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Delay(ctx, 1*time.Second)
	if err == nil {
		t.Fatal("expected error for already canceled context")
	}
}

func TestDelay_CancelDuringDelay(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()
	err := Delay(ctx, 1*time.Second)
	if err == nil {
		t.Fatal("expected error from context cancellation")
	}
}
