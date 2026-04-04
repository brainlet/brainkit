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

func testMetricsReflectsState(t *testing.T, env *suite.TestEnv) {
	ctx := context.Background()

	// Deploy something so deployments > 0
	err := env.Deploy("metrics-svc.ts", `bus.on("x", (msg) => msg.reply({}));`)
	require.NoError(t, err)

	// Create a schedule so schedules > 0
	_, err = env.Kernel.Schedule(ctx, brainkit.ScheduleConfig{
		Expression: "every 1h", Topic: "metrics.noop", Payload: json.RawMessage(`{}`), Source: "test",
	})
	require.NoError(t, err)

	time.Sleep(200 * time.Millisecond)

	m := env.Kernel.Metrics()
	assert.GreaterOrEqual(t, m.ActiveDeployments, 1, "should have at least 1 deployment")
	assert.GreaterOrEqual(t, m.ActiveSchedules, 1, "should have at least 1 schedule")
	assert.Greater(t, m.PumpCycles, int64(0), "pump should have cycled")
	assert.Greater(t, m.Uptime, time.Duration(0), "uptime should be positive")
}
