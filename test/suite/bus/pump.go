package bus

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testPumpScheduleLatency(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	testutil.Deploy(t, env.Kit, "latency-test.ts", `bus.on("ping", (msg) => { msg.reply({ pong: true }); });`)

	latencies := make([]time.Duration, 10)
	for i := range latencies {
		start := time.Now()
		_, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](env.Kit, ctx, sdk.CustomMsg{
			Topic:   sdk.ResolveServiceTopic("latency-test.ts", "ping"),
			Payload: json.RawMessage(`{"x":true}`),
		})
		require.NoError(t, err)
		latencies[i] = time.Since(start)
	}

	for i := 0; i < len(latencies); i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[j] < latencies[i] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}
	median := latencies[len(latencies)/2]
	maxMedian := maxPumpScheduleMedian(env.Config.Transport)

	t.Logf("bus.on round-trip latencies: %v", latencies)
	t.Logf("p50: %v", median)

	assert.Less(t, median, maxMedian,
		"event-driven pump should deliver callbacks before the 100ms fallback tick; got %v on %s", median, env.Config.Transport)
}

func testPumpResponsiveAfterIdle(t *testing.T, env *suite.TestEnv) {
	time.Sleep(500 * time.Millisecond)

	result := testutil.EvalTS(t, env.Kit, "__idle_test.ts", `return "alive"`)
	assert.Equal(t, "alive", result)
}

func maxPumpScheduleMedian(backend string) time.Duration {
	switch backend {
	case "", "memory", "embedded":
		return 5 * time.Millisecond
	default:
		// Broker-backed campaigns measure Go -> broker -> Go -> JS -> broker
		// reply latency. Keep the ceiling below the 100ms fallback tick so this
		// still proves the event-driven pump wake is working.
		return 50 * time.Millisecond
	}
}
