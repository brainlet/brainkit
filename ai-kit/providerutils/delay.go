// Ported from: packages/provider-utils/src/delay.ts
package providerutils

import (
	"context"
	"time"
)

// Delay creates a delay that resolves after the specified duration.
// If duration is 0 or negative, it returns immediately.
// The context can be used to cancel the delay early.
func Delay(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}

	if ctx == nil {
		ctx = context.Background()
	}

	// Check if already cancelled
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
