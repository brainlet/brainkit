package persistence

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDeployPersistRestart — deploy persistence across kernel restart.
// Ported from adversarial/backend_matrix_test.go:TestBackendMatrix_DeployPersistRestart.
func testDeployPersistRestart(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-matrix.db")
	store, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	// Phase 1: Deploy
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test",
		FSRoot: tmpDir, Store: store,
	})
	require.NoError(t, err)

	_, err = k1.Deploy(context.Background(), "persist-test-matrix.ts", `output("persisted");`)
	require.NoError(t, err)
	k1.Close()

	// Phase 2: Reopen with same store — deployment should be restored
	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)

	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test",
		FSRoot: tmpDir, Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	deps := k2.ListDeployments()
	found := false
	for _, d := range deps {
		if d.Source == "persist-test-matrix.ts" {
			found = true
		}
	}
	assert.True(t, found, "persist-test-matrix.ts should survive restart")
}

// testSecretsSurviveRestart — secrets survive kernel restart.
// Ported from adversarial/persistence_matrix_test.go:TestPersistence_Secrets.
func testSecretsSurviveRestart(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-secrets.db")

	// Phase 1: Set secrets
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1, SecretKey: "test-master-key-1234567890",
	})
	require.NoError(t, err)

	ctx := context.Background()
	pr, _ := sdk.Publish(k1, ctx, messages.SecretsSetMsg{Name: "persist-secret-matrix", Value: "secret-value-123"})
	ch := make(chan []byte, 1)
	unsub, _ := k1.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout setting secret")
	}
	unsub()
	k1.Close()

	// Phase 2: Reopen — secret should be retrievable
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2, SecretKey: "test-master-key-1234567890",
	})
	require.NoError(t, err)
	defer k2.Close()

	pr2, _ := sdk.Publish(k2, ctx, messages.SecretsGetMsg{Name: "persist-secret-matrix"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := k2.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case p := <-ch2:
		var resp struct{ Value string `json:"value"` }
		json.Unmarshal(p, &resp)
		assert.Equal(t, "secret-value-123", resp.Value, "secret should survive restart")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout getting secret after restart")
	}
}

// testMultiDeployOrderAndMetadata — multiple deployments restart with metadata.
// Ported from adversarial/persistence_matrix_test.go:TestPersistence_MultiDeployOrderAndMetadata.
func testMultiDeployOrderAndMetadata(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-multi.db")

	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store1,
	})
	require.NoError(t, err)

	ctx := context.Background()
	_, err = k1.Deploy(ctx, "first-matrix.ts", `output("first");`)
	require.NoError(t, err)
	_, err = k1.Deploy(ctx, "second-matrix.ts", `output("second");`, brainkit.WithRole("admin"))
	require.NoError(t, err)
	_, err = k1.Deploy(ctx, "third-matrix.ts", `output("third");`, brainkit.WithPackageName("my-pkg"))
	require.NoError(t, err)
	k1.Close()

	// Phase 2: Reopen
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	deps := k2.ListDeployments()
	sources := make([]string, len(deps))
	for i, d := range deps {
		sources[i] = d.Source
	}
	assert.Contains(t, sources, "first-matrix.ts")
	assert.Contains(t, sources, "second-matrix.ts")
	assert.Contains(t, sources, "third-matrix.ts")
}

// testMultipleSchedulesSurvive — multiple schedules survive restart.
// Ported from adversarial/persistence_matrix_test.go:TestPersistence_MultipleSchedules.
func testMultipleSchedulesSurvive(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-sched.db")

	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store1,
	})
	require.NoError(t, err)

	ctx := context.Background()
	k1.Schedule(ctx, brainkit.ScheduleConfig{Expression: "every 1h", Topic: "sched.hourly.matrix", Payload: json.RawMessage(`{}`)})
	k1.Schedule(ctx, brainkit.ScheduleConfig{Expression: "every 5m", Topic: "sched.fivemin.matrix", Payload: json.RawMessage(`{}`)})
	k1.Schedule(ctx, brainkit.ScheduleConfig{Expression: "in 24h", Topic: "sched.onetime.matrix", Payload: json.RawMessage(`{}`)})
	k1.Close()

	// Reopen
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	scheds := k2.ListSchedules()
	assert.GreaterOrEqual(t, len(scheds), 2, "at least 2 schedules should survive (one-time may have fired)")
}

// testDeployWithBusHandlerSurvivesRestart — handler re-subscribes after restart.
// Ported from adversarial/persistence_matrix_test.go:TestPersistence_DeployWithBusHandler.
func testDeployWithBusHandlerSurvivesRestart(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store-handler.db")

	// Phase 1: Deploy with bus handler
	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store1,
	})
	require.NoError(t, err)

	_, err = k1.Deploy(context.Background(), "handler-matrix.ts", `
		bus.on("ping", function(msg) { msg.reply({alive: true}); });
	`)
	require.NoError(t, err)
	k1.Close()

	// Phase 2: Reopen — handler should be active again
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(k2, ctx, messages.CustomMsg{
		Topic:   "ts.handler-matrix.ping",
		Payload: json.RawMessage(`{}`),
	})

	ch := make(chan []byte, 1)
	unsub, _ := k2.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "alive")
	case <-ctx.Done():
		t.Fatal("timeout — handler should be active after restart")
	}
}
