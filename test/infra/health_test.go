package infra_test

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealth_AliveWhenRunning(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	assert.True(t, k.Alive(ctx))
}

func TestHealth_ReadyWhenRunning(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assert.True(t, k.Ready(ctx))
}

func TestHealth_ReadyFalseWhenDraining(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	k.Kernel.SetDraining(true)
	assert.False(t, k.Ready(ctx))
	k.Kernel.Close()
}

func TestHealth_StatusRunning(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health := k.Health(ctx)
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

func TestHealth_TransportProbe(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health := k.Health(ctx)

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

func TestHealth_StorageBridgeCheck(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health := k.Health(ctx)

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

func TestHealth_StatusDraining(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	k.Kernel.SetDraining(true)
	health := k.Health(ctx)
	assert.Equal(t, "draining", health.Status)
	k.Kernel.Close()
}

func TestHealth_DeploymentsCount(t *testing.T) {
	k := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health := k.Health(ctx)

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
