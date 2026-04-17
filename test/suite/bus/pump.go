package bus

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPumpScheduleLatency(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, pkgDeployMsg("latency-test.ts", `bus.on("ping", (msg) => { msg.reply({ pong: true }); });`))
	require.NoError(t, err)
	deployCh := make(chan struct{}, 1)
	unsub, err := sdk.SubscribeTo[sdk.PackageDeployResp](env.Kit, ctx, pr.ReplyTo, func(_ sdk.PackageDeployResp, _ sdk.Message) {
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
		sendPR, err := sdk.SendToService(env.Kit, ctx, "latency-test.ts", "ping", map[string]bool{"x": true})
		require.NoError(t, err)

		done := make(chan time.Duration, 1)
		pongUnsub, err := env.Kit.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) {
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

func testPumpResponsiveAfterIdle(t *testing.T, env *suite.TestEnv) {
	time.Sleep(500 * time.Millisecond)

	result := testutil.EvalTS(t, env.Kit, "__idle_test.ts", `return "alive"`)
	assert.Equal(t, "alive", result)
}
