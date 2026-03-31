package infra_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/kit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetrics_ReflectsState(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy something
	_, err := k.Deploy(ctx, "metrics-test.ts", `bus.on("x", (msg) => msg.reply({}));`)
	require.NoError(t, err)

	// Schedule something
	_, err = k.Schedule(ctx, kit.ScheduleConfig{
		Expression: "every 1h", Topic: "metrics.tick",
		Payload: json.RawMessage(`{}`), Source: "test",
	})
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond) // let pump run a few cycles

	m := k.Metrics()
	assert.Equal(t, 1, m.ActiveDeployments, "should have 1 deployment")
	assert.Equal(t, 1, m.ActiveSchedules, "should have 1 schedule")
	assert.True(t, m.PumpCycles > 0, "pump should have run at least once")
	assert.True(t, m.Uptime > 0, "uptime should be positive")
}
