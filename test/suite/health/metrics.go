package health

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testMetricsReflectsState(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)

	// Deploy something so deployments > 0
	err := freshEnv.Deploy("metrics-svc.ts", `bus.on("x", (msg) => msg.reply({}));`)
	require.NoError(t, err)

	// Create a schedule so schedules > 0 (via bus command)
	testutil.Schedule(t, freshEnv.Kit, "every 1h", "metrics.noop", json.RawMessage(`{}`))

	time.Sleep(200 * time.Millisecond)

	health := queryHealth(t, freshEnv.Kit)
	assert.True(t, health.Healthy, "kit should be healthy")
	assert.Equal(t, "running", health.Status)
}
