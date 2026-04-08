package health

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

func testAliveWhenRunning(t *testing.T, env *suite.TestEnv) {
	assert.True(t, testutil.Alive(t, env.Kit))
}

func testReadyWhenRunning(t *testing.T, env *suite.TestEnv) {
	// Ready = kit responds to health and reports healthy
	assert.True(t, testutil.Alive(t, env.Kit))
}

func testReadyFalseWhenDraining(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	testutil.SetDraining(t, freshEnv.Kit, true)
	// After draining, health should report "draining" status
	health := queryHealth(t, freshEnv.Kit)
	assert.Equal(t, "draining", health.Status)
}

func testStatusRunning(t *testing.T, env *suite.TestEnv) {
	health := queryHealth(t, env.Kit)
	assert.True(t, health.Healthy)
	assert.Equal(t, "running", health.Status)
	assert.Greater(t, health.Uptime, time.Duration(0))
	assert.GreaterOrEqual(t, len(health.Checks), 4)

	var runtimeCheck *brainkit.HealthCheck
	for i := range health.Checks {
		if health.Checks[i].Name == "runtime" {
			runtimeCheck = &health.Checks[i]
			break
		}
	}
	require.NotNil(t, runtimeCheck)
	assert.True(t, runtimeCheck.Healthy)
	assert.Greater(t, runtimeCheck.Latency, time.Duration(0))
}

func testTransportProbe(t *testing.T, env *suite.TestEnv) {
	health := queryHealth(t, env.Kit)
	var transportCheck *brainkit.HealthCheck
	for i := range health.Checks {
		if health.Checks[i].Name == "transport" {
			transportCheck = &health.Checks[i]
			break
		}
	}
	require.NotNil(t, transportCheck)
	assert.True(t, transportCheck.Healthy)
}

func testStorageBridgeCheck(t *testing.T, env *suite.TestEnv) {
	health := queryHealth(t, env.Kit)
	var storageCheck *brainkit.HealthCheck
	for i := range health.Checks {
		if health.Checks[i].Name == "storage:default" {
			storageCheck = &health.Checks[i]
			break
		}
	}
	require.NotNil(t, storageCheck, "should have storage:default check")
	assert.True(t, storageCheck.Healthy)
}

func testStatusDraining(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	testutil.SetDraining(t, freshEnv.Kit, true)
	health := queryHealth(t, freshEnv.Kit)
	assert.Equal(t, "draining", health.Status)
}

func testDeploymentsCount(t *testing.T, env *suite.TestEnv) {
	health := queryHealth(t, env.Kit)
	var deploymentsCheck *brainkit.HealthCheck
	for i := range health.Checks {
		if health.Checks[i].Name == "deployments" {
			deploymentsCheck = &health.Checks[i]
			break
		}
	}
	require.NotNil(t, deploymentsCheck)
	assert.True(t, deploymentsCheck.Healthy)
}

// queryHealth queries health via the kit.health bus command and returns HealthStatus.
func queryHealth(t *testing.T, kit *brainkit.Kit) brainkit.HealthStatus {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, err := sdk.PublishKitHealth(kit, ctx, sdk.KitHealthMsg{})
	require.NoError(t, err)

	ch := make(chan json.RawMessage, 1)
	unsub, err := sdk.SubscribeKitHealthResp(kit, ctx, pr.ReplyTo,
		func(resp sdk.KitHealthResp, _ sdk.Message) {
			ch <- resp.Health
		})
	require.NoError(t, err)
	defer unsub()

	var raw json.RawMessage
	select {
	case raw = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout querying health")
	}

	var health brainkit.HealthStatus
	require.NoError(t, json.Unmarshal(raw, &health))
	return health
}
