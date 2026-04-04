package health

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testAliveWhenRunning(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	assert.True(t, env.Kernel.Alive(ctx))
}

func testReadyWhenRunning(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assert.True(t, env.Kernel.Ready(ctx))
}

func testReadyFalseWhenDraining(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	freshEnv.Kernel.SetDraining(true)
	assert.False(t, freshEnv.Kernel.Ready(ctx))
}

func testStatusRunning(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health := env.Kernel.Health(ctx)
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health := env.Kernel.Health(ctx)
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health := env.Kernel.Health(ctx)
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	freshEnv.Kernel.SetDraining(true)
	health := freshEnv.Kernel.Health(ctx)
	assert.Equal(t, "draining", health.Status)
}

func testDeploymentsCount(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health := env.Kernel.Health(ctx)
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
