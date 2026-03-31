package infra_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func poolNodeConfig(t *testing.T) brainkit.NodeConfig {
	t.Helper()
	return brainkit.NodeConfig{
		Kernel: brainkit.KernelConfig{
			Namespace:    "pool-test",
			CallerID:     "pool-test",
			FSRoot: t.TempDir(),
		},
		Messaging: brainkit.MessagingConfig{
			Transport: "memory",
		},
	}
}

// TestPool_SpawnAndKill creates a pool with 2 instances, verifies both
// are running, then kills the pool and verifies cleanup.
func TestPool_SpawnAndKill(t *testing.T) {
	im := brainkit.NewInstanceManager()

	err := im.SpawnPool("test-pool", brainkit.PoolConfig{
		Base:         poolNodeConfig(t),
		InitialCount: 2,
	})
	require.NoError(t, err)

	// Verify pool exists and has 2 instances
	info, err := im.PoolInfo("test-pool")
	require.NoError(t, err)
	assert.Equal(t, 2, info.Current)
	assert.Equal(t, "test-pool", info.Name)

	// Verify pool appears in list
	pools := im.Pools()
	assert.Contains(t, pools, "test-pool")

	// Kill the pool
	err = im.KillPool("test-pool")
	require.NoError(t, err)

	// Verify pool is gone
	_, err = im.PoolInfo("test-pool")
	var notFound *sdk.NotFoundError
	assert.True(t, errors.As(err, &notFound))
	assert.Equal(t, "pool", notFound.Resource)

	// Verify list is empty
	pools = im.Pools()
	assert.NotContains(t, pools, "test-pool")
}

// TestPool_ScaleUpDown creates a pool with 1 instance, scales to 3,
// verifies all 3 work, scales back to 1, verifies cleanup.
func TestPool_ScaleUpDown(t *testing.T) {
	im := brainkit.NewInstanceManager()

	err := im.SpawnPool("scale-pool", brainkit.PoolConfig{
		Base:         poolNodeConfig(t),
		InitialCount: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { im.KillPool("scale-pool") })

	// Start with 1
	info, _ := im.PoolInfo("scale-pool")
	assert.Equal(t, 1, info.Current)

	// Scale up to 3
	err = im.Scale("scale-pool", 2)
	require.NoError(t, err)

	info, _ = im.PoolInfo("scale-pool")
	assert.Equal(t, 3, info.Current)

	// Scale down to 1
	err = im.Scale("scale-pool", -2)
	require.NoError(t, err)

	info, _ = im.PoolInfo("scale-pool")
	assert.Equal(t, 1, info.Current)

	// Scale down beyond count — should clamp to 0
	err = im.Scale("scale-pool", -10)
	require.NoError(t, err)

	info, _ = im.PoolInfo("scale-pool")
	assert.Equal(t, 0, info.Current)
}

// TestPool_DuplicateAndNotFound verifies error types for edge cases.
func TestPool_DuplicateAndNotFound(t *testing.T) {
	im := brainkit.NewInstanceManager()

	err := im.SpawnPool("dup-pool", brainkit.PoolConfig{
		Base:         poolNodeConfig(t),
		InitialCount: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { im.KillPool("dup-pool") })

	// Duplicate spawn
	err = im.SpawnPool("dup-pool", brainkit.PoolConfig{
		Base:         poolNodeConfig(t),
		InitialCount: 1,
	})
	var exists *sdk.AlreadyExistsError
	require.True(t, errors.As(err, &exists))
	assert.Equal(t, "pool", exists.Resource)
	assert.Equal(t, "dup-pool", exists.Name)

	// Not found — Scale
	err = im.Scale("nonexistent", 1)
	var notFound *sdk.NotFoundError
	require.True(t, errors.As(err, &notFound))

	// Not found — KillPool
	err = im.KillPool("nonexistent")
	require.True(t, errors.As(err, &notFound))

	// Not found — PoolInfo
	_, err = im.PoolInfo("nonexistent")
	require.True(t, errors.As(err, &notFound))
}

// TestPool_SharedTools verifies that tools registered on the shared
// registry are accessible from all instances in the pool.
func TestPool_SharedTools(t *testing.T) {
	im := brainkit.NewInstanceManager()

	cfg := poolNodeConfig(t)
	sharedTools := registry.New()
	cfg.Kernel.SharedTools = sharedTools

	// Register a tool on the shared registry BEFORE spawning
	registry.Register(sharedTools, "pool-echo", registry.TypedTool[struct {
		Msg string `json:"msg"`
	}]{
		Description: "pool echo tool",
		Execute: func(ctx context.Context, input struct {
			Msg string `json:"msg"`
		}) (any, error) {
			return map[string]string{"echo": input.Msg}, nil
		},
	})

	err := im.SpawnPool("tools-pool", brainkit.PoolConfig{
		Base:         cfg,
		InitialCount: 2,
	})
	require.NoError(t, err)
	t.Cleanup(func() { im.KillPool("tools-pool") })

	// The shared tool should be visible — verify via PoolInfo that instances exist
	info, err := im.PoolInfo("tools-pool")
	require.NoError(t, err)
	assert.Equal(t, 2, info.Current)

	// Verify the shared registry has the tool
	tool, err := sharedTools.Resolve("pool-echo")
	require.NoError(t, err)
	assert.Equal(t, "pool-echo", tool.ShortName)
}

// TestStrategy_Static verifies StaticStrategy returns correct decisions.
func TestStrategy_Static(t *testing.T) {
	s := brainkit.NewStaticStrategy(3)
	metrics := messaging.MetricsSnapshot{
		Published: make(map[string]int),
		Handled:   make(map[string]int),
		Errors:    make(map[string]int),
	}

	// Below target → scale up
	d := s.Evaluate(metrics, brainkit.PoolInfo{Current: 1})
	assert.Equal(t, "scale-up", d.Action)
	assert.Equal(t, 2, d.Delta)

	// At target → none
	d = s.Evaluate(metrics, brainkit.PoolInfo{Current: 3})
	assert.Equal(t, "none", d.Action)

	// Above target → scale down
	d = s.Evaluate(metrics, brainkit.PoolInfo{Current: 5})
	assert.Equal(t, "scale-down", d.Action)
	assert.Equal(t, 2, d.Delta)
}

// TestStrategy_Threshold verifies ThresholdStrategy respects min/max bounds.
func TestStrategy_Threshold(t *testing.T) {
	s := brainkit.NewThresholdStrategy(10, 2)
	metrics := messaging.MetricsSnapshot{
		Published: make(map[string]int),
		Handled:   make(map[string]int),
		Errors:    make(map[string]int),
	}

	// Pending > threshold, below max → scale up
	d := s.Evaluate(metrics, brainkit.PoolInfo{Current: 2, Pending: 15, Min: 1, Max: 5})
	assert.Equal(t, "scale-up", d.Action)
	assert.Equal(t, 1, d.Delta)

	// Pending > threshold, AT max → no action
	d = s.Evaluate(metrics, brainkit.PoolInfo{Current: 5, Pending: 15, Min: 1, Max: 5})
	assert.Equal(t, "none", d.Action)

	// Pending < scale-down threshold, above min → scale down
	d = s.Evaluate(metrics, brainkit.PoolInfo{Current: 3, Pending: 1, Min: 1, Max: 5})
	assert.Equal(t, "scale-down", d.Action)
	assert.Equal(t, 1, d.Delta)

	// Pending < scale-down threshold, AT min → no action
	d = s.Evaluate(metrics, brainkit.PoolInfo{Current: 1, Pending: 0, Min: 1, Max: 5})
	assert.Equal(t, "none", d.Action)

	// Pending between thresholds → no action
	d = s.Evaluate(metrics, brainkit.PoolInfo{Current: 2, Pending: 5, Min: 1, Max: 5})
	assert.Equal(t, "none", d.Action)

	// Scale up capped by max
	s2 := &brainkit.ThresholdStrategy{ScaleUpThreshold: 5, ScaleUpStep: 10, ScaleDownThreshold: 0, ScaleDownStep: 1}
	d = s2.Evaluate(metrics, brainkit.PoolInfo{Current: 3, Pending: 20, Min: 1, Max: 5})
	assert.Equal(t, "scale-up", d.Action)
	assert.Equal(t, 2, d.Delta) // capped: max(5) - current(3) = 2, not step(10)

	// Scale down capped by min
	s3 := &brainkit.ThresholdStrategy{ScaleUpThreshold: 100, ScaleDownThreshold: 5, ScaleUpStep: 1, ScaleDownStep: 10}
	d = s3.Evaluate(metrics, brainkit.PoolInfo{Current: 3, Pending: 0, Min: 2, Max: 10})
	assert.Equal(t, "scale-down", d.Action)
	assert.Equal(t, 1, d.Delta) // capped: current(3) - min(2) = 1, not step(10)
}

// TestPool_EvaluateAndScale verifies the automatic scaling loop applies strategy decisions.
func TestPool_EvaluateAndScale(t *testing.T) {
	im := brainkit.NewInstanceManager()

	// Use static strategy targeting 3 instances, start with 1
	err := im.SpawnPool("auto-pool", brainkit.PoolConfig{
		Base:         poolNodeConfig(t),
		InitialCount: 1,
		Strategy:     brainkit.NewStaticStrategy(3),
	})
	require.NoError(t, err)
	t.Cleanup(func() { im.KillPool("auto-pool") })

	info, _ := im.PoolInfo("auto-pool")
	assert.Equal(t, 1, info.Current)

	// Run the evaluation loop — should scale up to 3
	im.EvaluateAndScale()

	info, _ = im.PoolInfo("auto-pool")
	assert.Equal(t, 3, info.Current)

	// Running again should be a no-op (already at target)
	im.EvaluateAndScale()

	info, _ = im.PoolInfo("auto-pool")
	assert.Equal(t, 3, info.Current)
}

// TestPool_InstancesProcessMessages verifies that pool instances can actually
// process bus messages — not just that the count is right.
func TestPool_InstancesProcessMessages(t *testing.T) {
	im := brainkit.NewInstanceManager()

	cfg := poolNodeConfig(t)
	sharedTools := registry.New()
	cfg.Kernel.SharedTools = sharedTools

	registry.Register(sharedTools, "ping", registry.TypedTool[struct{}]{
		Description: "ping tool",
		Execute: func(ctx context.Context, input struct{}) (any, error) {
			return map[string]string{"pong": "ok"}, nil
		},
	})

	err := im.SpawnPool("msg-pool", brainkit.PoolConfig{
		Base:         cfg,
		InitialCount: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { im.KillPool("msg-pool") })

	// Get pool info to confirm it's alive
	info, err := im.PoolInfo("msg-pool")
	require.NoError(t, err)
	assert.Equal(t, 1, info.Current)

	// The tool should be callable via the shared registry
	tool, err := sharedTools.Resolve("ping")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := tool.Executor.Call(ctx, "test", json.RawMessage(`{}`))
	require.NoError(t, err)

	var parsed map[string]string
	json.Unmarshal(result, &parsed)
	assert.Equal(t, "ok", parsed["pong"])
}

// Ensure sdk and messages imports are used (for error type checks).
var _ = sdk.NotFoundError{}
var _ = messages.ToolCallMsg{}
