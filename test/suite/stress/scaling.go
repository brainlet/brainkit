package stress

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/transport"
	"github.com/brainlet/brainkit/internal/types"
	tools "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func poolNodeConfig(t *testing.T) types.NodeConfig {
	t.Helper()
	return types.NodeConfig{
		Kernel: types.KernelConfig{
			Namespace: "pool-stress-test",
			CallerID:  "pool-stress-test",
			FSRoot:    t.TempDir(),
		},
		Messaging: types.MessagingConfig{
			Transport: "memory",
		},
	}
}

// testPoolSpawnAndKill creates a pool with 2 instances, verifies both
// are running, then kills the pool and verifies cleanup.
func testPoolSpawnAndKill(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	im := brainkit.NewInstanceManager()

	err := im.SpawnPool("stress-test-pool", brainkit.PoolConfig{
		Base:         poolNodeConfig(t),
		InitialCount: 2,
	})
	require.NoError(t, err)

	info, err := im.PoolInfo("stress-test-pool")
	require.NoError(t, err)
	assert.Equal(t, 2, info.Current)
	assert.Equal(t, "stress-test-pool", info.Name)

	pools := im.Pools()
	assert.Contains(t, pools, "stress-test-pool")

	err = im.KillPool("stress-test-pool")
	require.NoError(t, err)

	_, err = im.PoolInfo("stress-test-pool")
	var notFound *sdk.NotFoundError
	assert.True(t, errors.As(err, &notFound))
	assert.Equal(t, "pool", notFound.Resource)

	pools = im.Pools()
	assert.NotContains(t, pools, "stress-test-pool")
}

// testPoolScaleUpDown creates a pool with 1 instance, scales to 3,
// verifies all 3 work, scales back to 1, verifies cleanup.
func testPoolScaleUpDown(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	im := brainkit.NewInstanceManager()

	err := im.SpawnPool("stress-scale-pool", brainkit.PoolConfig{
		Base:         poolNodeConfig(t),
		InitialCount: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { im.KillPool("stress-scale-pool") })

	info, _ := im.PoolInfo("stress-scale-pool")
	assert.Equal(t, 1, info.Current)

	err = im.Scale("stress-scale-pool", 2)
	require.NoError(t, err)

	info, _ = im.PoolInfo("stress-scale-pool")
	assert.Equal(t, 3, info.Current)

	err = im.Scale("stress-scale-pool", -2)
	require.NoError(t, err)

	info, _ = im.PoolInfo("stress-scale-pool")
	assert.Equal(t, 1, info.Current)

	err = im.Scale("stress-scale-pool", -10)
	require.NoError(t, err)

	info, _ = im.PoolInfo("stress-scale-pool")
	assert.Equal(t, 0, info.Current)
}

// testPoolDuplicateAndNotFound verifies error types for edge cases.
func testPoolDuplicateAndNotFound(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	im := brainkit.NewInstanceManager()

	err := im.SpawnPool("stress-dup-pool", brainkit.PoolConfig{
		Base:         poolNodeConfig(t),
		InitialCount: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { im.KillPool("stress-dup-pool") })

	err = im.SpawnPool("stress-dup-pool", brainkit.PoolConfig{
		Base:         poolNodeConfig(t),
		InitialCount: 1,
	})
	var exists *sdk.AlreadyExistsError
	require.True(t, errors.As(err, &exists))
	assert.Equal(t, "pool", exists.Resource)
	assert.Equal(t, "stress-dup-pool", exists.Name)

	err = im.Scale("nonexistent-stress", 1)
	var notFound *sdk.NotFoundError
	require.True(t, errors.As(err, &notFound))

	err = im.KillPool("nonexistent-stress")
	require.True(t, errors.As(err, &notFound))

	_, err = im.PoolInfo("nonexistent-stress")
	require.True(t, errors.As(err, &notFound))
}

// testPoolSharedTools verifies that tools registered on the shared
// registry are accessible from all instances in the pool.
func testPoolSharedTools(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	im := brainkit.NewInstanceManager()

	cfg := poolNodeConfig(t)
	sharedTools := tools.New()
	cfg.Kernel.SharedTools = sharedTools

	tools.Register(sharedTools, "stress-pool-echo", tools.TypedTool[struct {
		Msg string `json:"msg"`
	}]{
		Description: "pool echo tool",
		Execute: func(ctx context.Context, input struct {
			Msg string `json:"msg"`
		}) (any, error) {
			return map[string]string{"echo": input.Msg}, nil
		},
	})

	err := im.SpawnPool("stress-tools-pool", brainkit.PoolConfig{
		Base:         cfg,
		InitialCount: 2,
	})
	require.NoError(t, err)
	t.Cleanup(func() { im.KillPool("stress-tools-pool") })

	info, err := im.PoolInfo("stress-tools-pool")
	require.NoError(t, err)
	assert.Equal(t, 2, info.Current)

	tool, err := sharedTools.Resolve("stress-pool-echo")
	require.NoError(t, err)
	assert.Equal(t, "stress-pool-echo", tool.ShortName)
}

// testStrategyStatic verifies StaticStrategy returns correct decisions.
func testStrategyStatic(t *testing.T, env *suite.TestEnv) {
	s := brainkit.NewStaticStrategy(3)
	metrics := transport.MetricsSnapshot{
		Published: make(map[string]int),
		Handled:   make(map[string]int),
		Errors:    make(map[string]int),
	}

	d := s.Evaluate(metrics, brainkit.PoolInfo{Current: 1})
	assert.Equal(t, "scale-up", d.Action)
	assert.Equal(t, 2, d.Delta)

	d = s.Evaluate(metrics, brainkit.PoolInfo{Current: 3})
	assert.Equal(t, "none", d.Action)

	d = s.Evaluate(metrics, brainkit.PoolInfo{Current: 5})
	assert.Equal(t, "scale-down", d.Action)
	assert.Equal(t, 2, d.Delta)
}

// testStrategyThreshold verifies ThresholdStrategy respects min/max bounds.
func testStrategyThreshold(t *testing.T, env *suite.TestEnv) {
	s := brainkit.NewThresholdStrategy(10, 2)
	metrics := transport.MetricsSnapshot{
		Published: make(map[string]int),
		Handled:   make(map[string]int),
		Errors:    make(map[string]int),
	}

	d := s.Evaluate(metrics, brainkit.PoolInfo{Current: 2, Pending: 15, Min: 1, Max: 5})
	assert.Equal(t, "scale-up", d.Action)
	assert.Equal(t, 1, d.Delta)

	d = s.Evaluate(metrics, brainkit.PoolInfo{Current: 5, Pending: 15, Min: 1, Max: 5})
	assert.Equal(t, "none", d.Action)

	d = s.Evaluate(metrics, brainkit.PoolInfo{Current: 3, Pending: 1, Min: 1, Max: 5})
	assert.Equal(t, "scale-down", d.Action)
	assert.Equal(t, 1, d.Delta)

	d = s.Evaluate(metrics, brainkit.PoolInfo{Current: 1, Pending: 0, Min: 1, Max: 5})
	assert.Equal(t, "none", d.Action)

	d = s.Evaluate(metrics, brainkit.PoolInfo{Current: 2, Pending: 5, Min: 1, Max: 5})
	assert.Equal(t, "none", d.Action)

	s2 := &brainkit.ThresholdStrategy{ScaleUpThreshold: 5, ScaleUpStep: 10, ScaleDownThreshold: 0, ScaleDownStep: 1}
	d = s2.Evaluate(metrics, brainkit.PoolInfo{Current: 3, Pending: 20, Min: 1, Max: 5})
	assert.Equal(t, "scale-up", d.Action)
	assert.Equal(t, 2, d.Delta)

	s3 := &brainkit.ThresholdStrategy{ScaleUpThreshold: 100, ScaleDownThreshold: 5, ScaleUpStep: 1, ScaleDownStep: 10}
	d = s3.Evaluate(metrics, brainkit.PoolInfo{Current: 3, Pending: 0, Min: 2, Max: 10})
	assert.Equal(t, "scale-down", d.Action)
	assert.Equal(t, 1, d.Delta)
}

// testPoolEvaluateAndScale verifies the automatic scaling loop applies strategy decisions.
func testPoolEvaluateAndScale(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	im := brainkit.NewInstanceManager()

	err := im.SpawnPool("stress-auto-pool", brainkit.PoolConfig{
		Base:         poolNodeConfig(t),
		InitialCount: 1,
		Strategy:     brainkit.NewStaticStrategy(3),
	})
	require.NoError(t, err)
	t.Cleanup(func() { im.KillPool("stress-auto-pool") })

	info, _ := im.PoolInfo("stress-auto-pool")
	assert.Equal(t, 1, info.Current)

	im.EvaluateAndScale()

	info, _ = im.PoolInfo("stress-auto-pool")
	assert.Equal(t, 3, info.Current)

	im.EvaluateAndScale()

	info, _ = im.PoolInfo("stress-auto-pool")
	assert.Equal(t, 3, info.Current)
}

// testPoolInstancesProcessMessages verifies that pool instances can actually
// process bus messages.
func testPoolInstancesProcessMessages(t *testing.T, env *suite.TestEnv) {
	if testing.Short() {
		t.Skip("skipped in short mode")
	}

	im := brainkit.NewInstanceManager()

	cfg := poolNodeConfig(t)
	sharedTools := tools.New()
	cfg.Kernel.SharedTools = sharedTools

	tools.Register(sharedTools, "stress-ping", tools.TypedTool[struct{}]{
		Description: "ping tool",
		Execute: func(ctx context.Context, input struct{}) (any, error) {
			return map[string]string{"pong": "ok"}, nil
		},
	})

	err := im.SpawnPool("stress-msg-pool", brainkit.PoolConfig{
		Base:         cfg,
		InitialCount: 1,
	})
	require.NoError(t, err)
	t.Cleanup(func() { im.KillPool("stress-msg-pool") })

	info, err := im.PoolInfo("stress-msg-pool")
	require.NoError(t, err)
	assert.Equal(t, 1, info.Current)

	tool, err := sharedTools.Resolve("stress-ping")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := tool.Executor.Call(ctx, "test", json.RawMessage(`{}`))
	require.NoError(t, err)

	var parsed map[string]string
	json.Unmarshal(result, &parsed)
	assert.Equal(t, "ok", parsed["pong"])
}
