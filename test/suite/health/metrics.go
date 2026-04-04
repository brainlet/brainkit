package health

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMetricsReflectsState(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx := context.Background()

	// Deploy something so deployments > 0
	err := freshEnv.Deploy("metrics-svc.ts", `bus.on("x", (msg) => msg.reply({}));`)
	require.NoError(t, err)

	// Create a schedule so schedules > 0
	_, err = freshEnv.Kernel.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "every 1h", Topic: "metrics.noop", Payload: json.RawMessage(`{}`), Source: "test",
	})
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	m := freshEnv.Kernel.Metrics()
	assert.Equal(t, 1, m.ActiveDeployments, "should have 1 deployment")
	assert.Equal(t, 1, m.ActiveSchedules, "should have 1 schedule")
	assert.True(t, m.PumpCycles > 0, "pump should have run at least once")
	assert.True(t, m.Uptime > 0, "uptime should be positive")
}
