package health

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/brainlet/brainkit/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAliveAfterHeavyLoad — kernel stays alive after many deploy/teardown cycles.
func testAliveAfterHeavyLoad(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx := context.Background()

	for i := 0; i < 20; i++ {
		src := "health-load-degraded.ts"
		err := env.Deploy(src, `output("load");`)
		require.NoError(t, err)
		_, err = env.Kernel.Teardown(ctx, src)
		require.NoError(t, err)
	}

	assert.True(t, env.Kernel.Alive(ctx))
	assert.True(t, env.Kernel.Ready(ctx))
}

// testReadyToggleDuringDrain — Ready returns false during drain, true after recovery.
func testReadyToggleDuringDrain(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx := context.Background()

	assert.True(t, env.Kernel.Ready(ctx))

	env.Kernel.SetDraining(true)
	assert.False(t, env.Kernel.Ready(ctx))
	assert.True(t, env.Kernel.Alive(ctx)) // alive even during drain

	env.Kernel.SetDraining(false)
	assert.True(t, env.Kernel.Ready(ctx))
}

// testFullHealthCheckCategories — Health() returns all check categories.
func testFullHealthCheckCategories(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx := context.Background()

	health := env.Kernel.Health(ctx)
	assert.True(t, health.Healthy)
	assert.Equal(t, "running", health.Status)
	assert.Greater(t, len(health.Checks), 0)

	checkNames := make(map[string]bool)
	for _, c := range health.Checks {
		checkNames[c.Name] = true
	}
	assert.True(t, checkNames["runtime"], "should have runtime check")
	assert.True(t, checkNames["transport"], "should have transport check")
}

// testHealthWithTracingStore — health works when tracing is configured.
func testHealthWithTracingStore(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	traceStore := tracing.NewMemoryTraceStore(1000)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace:  "test",
		CallerID:   "test",
		FSRoot:     tmpDir,
		TraceStore: traceStore,
	})
	require.NoError(t, err)
	defer k.Close()

	health := k.Health(context.Background())
	assert.True(t, health.Healthy)
}

// testHealthWithStorageBridges — health checks storage bridges.
func testHealthWithStorageBridges(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx := context.Background()

	health := env.Kernel.Health(ctx)
	hasStorage := false
	for _, c := range health.Checks {
		if len(c.Name) > 8 && c.Name[:8] == "storage:" {
			hasStorage = true
		}
	}
	assert.True(t, hasStorage, "health should include storage checks")
}

// testMetricsReflectDeployments — metrics count tracks deployments.
func testMetricsReflectDeployments(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)
	ctx := context.Background()

	m0 := env.Kernel.Metrics()
	assert.Equal(t, 0, m0.ActiveDeployments)

	err := env.Deploy("metrics-degraded.ts", `output("m");`)
	require.NoError(t, err)
	m1 := env.Kernel.Metrics()
	assert.Equal(t, 1, m1.ActiveDeployments)

	err = env.Deploy("metrics-degraded2.ts", `output("m2");`)
	require.NoError(t, err)
	m2 := env.Kernel.Metrics()
	assert.Equal(t, 2, m2.ActiveDeployments)

	_, err = env.Kernel.Teardown(ctx, "metrics-degraded.ts")
	require.NoError(t, err)
	m3 := env.Kernel.Metrics()
	assert.Equal(t, 1, m3.ActiveDeployments)
}

// testUptimeIncreases — uptime is positive and increases over time.
func testUptimeIncreases(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	m1 := env.Kernel.Metrics()
	time.Sleep(100 * time.Millisecond)
	m2 := env.Kernel.Metrics()

	assert.Greater(t, m2.Uptime, m1.Uptime)
	assert.Greater(t, m2.Uptime, time.Duration(0))
}

// testHealthAfterClose — health methods don't panic after close.
func testHealthAfterClose(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)
	k.Close()

	// These should not panic — just return false/unhealthy
	assert.False(t, k.Alive(context.Background()))
	assert.False(t, k.Ready(context.Background()))
}

// testPersistenceStoreHealth — health when persistence store is configured.
func testPersistenceStoreHealth(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	store, err := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "store.db"))
	require.NoError(t, err)

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store,
	})
	require.NoError(t, err)
	defer k.Close()

	health := k.Health(context.Background())
	assert.True(t, health.Healthy)
}
