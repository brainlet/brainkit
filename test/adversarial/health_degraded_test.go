package adversarial_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/tracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealth_AliveAfterHeavyLoad — kernel stays alive after many deploy/teardown cycles.
func TestHealth_AliveAfterHeavyLoad(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	for i := 0; i < 20; i++ {
		src := "health-load.ts"
		tk.Deploy(ctx, src, `output("load");`)
		tk.Teardown(ctx, src)
	}

	assert.True(t, tk.Alive(ctx))
	assert.True(t, tk.Ready(ctx))
}

// TestHealth_ReadyFalseDuringDrain — Ready returns false during drain.
func TestHealth_ReadyFalseDuringDrain(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	assert.True(t, tk.Ready(ctx))

	tk.SetDraining(true)
	assert.False(t, tk.Ready(ctx))
	assert.True(t, tk.Alive(ctx)) // alive even during drain

	tk.SetDraining(false)
	assert.True(t, tk.Ready(ctx))
}

// TestHealth_FullHealthCheck — Health() returns all check categories.
func TestHealth_FullHealthCheck(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	health := tk.Health(ctx)
	assert.True(t, health.Healthy)
	assert.Equal(t, "running", health.Status)
	assert.Greater(t, len(health.Checks), 0)

	// Should have runtime and transport checks at minimum
	checkNames := make(map[string]bool)
	for _, c := range health.Checks {
		checkNames[c.Name] = true
	}
	assert.True(t, checkNames["runtime"], "should have runtime check")
	assert.True(t, checkNames["transport"], "should have transport check")
}

// TestHealth_WithTracingStore — health works when tracing is configured.
func TestHealth_WithTracingStore(t *testing.T) {
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

// TestHealth_WithStorage — health checks storage bridges.
func TestHealth_WithStorage(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	health := tk.Health(ctx)
	// Should include storage checks
	hasStorage := false
	for _, c := range health.Checks {
		if len(c.Name) > 8 && c.Name[:8] == "storage:" {
			hasStorage = true
		}
	}
	assert.True(t, hasStorage, "health should include storage checks")
}

// TestHealth_MetricsReflectDeployments — metrics count tracks deployments.
func TestHealth_MetricsReflectDeployments(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	m0 := tk.Metrics()
	assert.Equal(t, 0, m0.ActiveDeployments)

	tk.Deploy(ctx, "metrics-test.ts", `output("m");`)
	m1 := tk.Metrics()
	assert.Equal(t, 1, m1.ActiveDeployments)

	tk.Deploy(ctx, "metrics-test2.ts", `output("m2");`)
	m2 := tk.Metrics()
	assert.Equal(t, 2, m2.ActiveDeployments)

	tk.Teardown(ctx, "metrics-test.ts")
	m3 := tk.Metrics()
	assert.Equal(t, 1, m3.ActiveDeployments)
}

// TestHealth_UptimeIncreases — uptime is positive and increases.
func TestHealth_UptimeIncreases(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)

	m1 := tk.Metrics()
	time.Sleep(100 * time.Millisecond)
	m2 := tk.Metrics()

	assert.Greater(t, m2.Uptime, m1.Uptime)
	assert.Greater(t, m2.Uptime, time.Duration(0))
}

// TestHealth_AfterClose — health methods don't panic after close.
func TestHealth_AfterClose(t *testing.T) {
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

// TestHealth_PersistenceStoreHealth — health when store is configured.
func TestHealth_PersistenceStoreHealth(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "store.db"))

	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store,
	})
	require.NoError(t, err)
	defer k.Close()

	health := k.Health(context.Background())
	assert.True(t, health.Healthy)
}
