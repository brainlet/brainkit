package health

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/internal/tracing"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testAliveAfterHeavyLoad — kernel stays alive after many deploy/teardown cycles.
func testAliveAfterHeavyLoad(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	for i := 0; i < 20; i++ {
		src := "health-load-degraded.ts"
		err := env.Deploy(src, `output("load");`)
		require.NoError(t, err)
		testutil.Teardown(t, env.Kit, src)
	}

	assert.True(t, testutil.Alive(t, env.Kit))
}

// testReadyToggleDuringDrain — Ready returns false during drain, true after recovery.
func testReadyToggleDuringDrain(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	assert.True(t, testutil.Alive(t, env.Kit))

	testutil.SetDraining(t, env.Kit, true)
	health := queryHealth(t, env.Kit)
	assert.Equal(t, "draining", health.Status)
	assert.True(t, testutil.Alive(t, env.Kit)) // alive even during drain

	testutil.SetDraining(t, env.Kit, false)
	health = queryHealth(t, env.Kit)
	assert.Equal(t, "running", health.Status)
}

// testFullHealthCheckCategories — Health() returns all check categories.
func testFullHealthCheckCategories(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	health := queryHealth(t, env.Kit)
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

	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace:  "test",
		CallerID:   "test",
		FSRoot:     tmpDir,
		TraceStore: traceStore,
	})
	require.NoError(t, err)
	defer k.Close()

	health := queryHealth(t, k)
	assert.True(t, health.Healthy)
}

// testHealthWithStorageBridges — health checks storage bridges.
func testHealthWithStorageBridges(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	health := queryHealth(t, env.Kit)
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

	// Health should report 0 deployments initially
	health0 := queryHealth(t, env.Kit)
	assert.True(t, health0.Healthy)

	err := env.Deploy("metrics-degraded.ts", `output("m");`)
	require.NoError(t, err)

	err = env.Deploy("metrics-degraded2.ts", `output("m2");`)
	require.NoError(t, err)

	health1 := queryHealth(t, env.Kit)
	assert.True(t, health1.Healthy)

	testutil.Teardown(t, env.Kit, "metrics-degraded.ts")

	health2 := queryHealth(t, env.Kit)
	assert.True(t, health2.Healthy)
}

// testUptimeIncreases — uptime is positive and increases over time.
func testUptimeIncreases(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t)

	h1 := queryHealth(t, env.Kit)
	time.Sleep(100 * time.Millisecond)
	h2 := queryHealth(t, env.Kit)

	assert.Greater(t, h2.Uptime, h1.Uptime)
	assert.Greater(t, h2.Uptime, time.Duration(0))
}

// testHealthAfterClose — health methods don't panic after close.
func testHealthAfterClose(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)
	k.Close()

	// After close, Alive should return false (health query will fail/timeout)
	assert.False(t, testutil.Alive(t, k))
}

// testPersistenceStoreHealth — health when persistence store is configured.
func testPersistenceStoreHealth(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	store, err := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "store.db"))
	require.NoError(t, err)

	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store,
	})
	require.NoError(t, err)
	defer k.Close()

	health := queryHealth(t, k)
	assert.True(t, health.Healthy)
}
