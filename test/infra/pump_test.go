package infra_test

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPump_ScheduleLatency measures the round-trip latency of a bus.on handler.
// With the event-driven pump, this should be well under 5ms.
// With the old 10ms polling pump, p50 was ~10ms.
func TestPump_ScheduleLatency(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(k, ctx, messages.KitDeployMsg{
		Source: "latency-test.ts",
		Code:   `bus.on("ping", (msg) => { msg.reply({ pong: true }); });`,
	})
	require.NoError(t, err)
	deployCh := make(chan struct{}, 1)
	unsub, err := sdk.SubscribeTo[messages.KitDeployResp](k, ctx, pr.ReplyTo, func(_ messages.KitDeployResp, _ messages.Message) {
		deployCh <- struct{}{}
	})
	require.NoError(t, err)
	defer unsub()
	select {
	case <-deployCh:
	case <-ctx.Done():
		t.Fatal("deploy timeout")
	}
	time.Sleep(100 * time.Millisecond)

	latencies := make([]time.Duration, 10)
	for i := range latencies {
		start := time.Now()
		sendPR, err := sdk.SendToService(k, ctx, "latency-test.ts", "ping", map[string]bool{"x": true})
		require.NoError(t, err)

		done := make(chan time.Duration, 1)
		pongUnsub, err := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
			done <- time.Since(start)
		})
		require.NoError(t, err)

		select {
		case latency := <-done:
			latencies[i] = latency
		case <-ctx.Done():
			t.Fatal("ping timeout")
		}
		pongUnsub()
	}

	// Sort for percentiles
	for i := 0; i < len(latencies); i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[j] < latencies[i] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}
	median := latencies[len(latencies)/2]

	t.Logf("bus.on round-trip latencies: %v", latencies)
	t.Logf("p50: %v", median)

	assert.Less(t, median, 5*time.Millisecond,
		"event-driven pump should deliver callbacks in <5ms; got %v", median)
}

// TestPump_ResponsiveAfterIdle verifies the pump stays alive after idle.
func TestPump_ResponsiveAfterIdle(t *testing.T) {
	k := testutil.NewTestKernelFull(t)

	time.Sleep(500 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result, err := k.EvalTS(ctx, "__idle_test.ts", `return "alive"`)
	require.NoError(t, err)
	assert.Equal(t, "alive", result)
}
