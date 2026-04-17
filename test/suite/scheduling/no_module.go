package scheduling

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testNoModuleThrowsNotConfigured verifies that without the schedules module,
// .ts bus.schedule(...) calls surface a NOT_CONFIGURED error (instead of
// silently succeeding and then never firing).
func testNoModuleThrowsNotConfigured(t *testing.T, _ *suite.TestEnv) {
	// Build a Kit without the schedules module. We use Memory transport +
	// no modules at all so the only relevant state is the absent scheduler.
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test",
		CallerID:  "test",
		FSRoot:    t.TempDir(),
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Use kit.eval in "ts" mode to invoke bus.schedule in the runtime — the
	// response envelope carries the bridge-thrown error when no handler is
	// attached.
	pr, err := sdk.Publish(k, ctx, sdk.KitEvalMsg{
		Source: "no-sched-module.ts",
		Mode:   "ts",
		Code:   `bus.schedule("every 100ms", "tick", {});`,
	})
	require.NoError(t, err)

	ch := make(chan sdk.Message, 1)
	unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m })
	require.NoError(t, err)
	defer unsub()

	select {
	case m := <-ch:
		env, decodeErr := sdk.DecodeEnvelope(m.Payload)
		require.NoError(t, decodeErr)
		require.NotNil(t, env.Error, "eval should surface the bridge error, got: %s", string(m.Payload))
		// Bridge raised NOT_CONFIGURED; the eval path wraps JS exceptions
		// into INTERNAL_ERROR, but the human-readable message carries the
		// bridge's original "schedules not configured" surface.
		combined := strings.ToLower(env.Error.Code + " " + env.Error.Message)
		assert.Contains(t, combined, "not configured",
			"expected not-configured surface, got code=%q message=%q", env.Error.Code, env.Error.Message)
		assert.Contains(t, strings.ToLower(env.Error.Message), "schedules",
			"error should mention schedules: %s", env.Error.Message)
	case <-ctx.Done():
		t.Fatal("timeout waiting for deploy reply")
	}
}
