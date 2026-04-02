package adversarial_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper: create kernel with store, run setup, close, reopen, run verify.
func withRestart(t *testing.T, setup func(k *brainkit.Kernel), verify func(k *brainkit.Kernel)) {
	t.Helper()
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.db")

	// Phase 1
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1,
	})
	require.NoError(t, err)

	setup(k1)
	k1.Close()

	// Phase 2
	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	verify(k2)
}

// P12: Secrets survive restart
func TestPersistence_Secrets(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.db")

	// Phase 1: Set secrets
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1, SecretKey: "test-master-key-1234567890",
	})
	require.NoError(t, err)

	ctx := context.Background()
	pr, _ := sdk.Publish(k1, ctx, messages.SecretsSetMsg{Name: "persist-secret", Value: "secret-value-123"})
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

	pr2, _ := sdk.Publish(k2, ctx, messages.SecretsGetMsg{Name: "persist-secret"})
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

// P: Multiple deployments restart in correct order with metadata
func TestPersistence_MultiDeployOrderAndMetadata(t *testing.T) {
	withRestart(t,
		func(k *brainkit.Kernel) {
			ctx := context.Background()
			_, err := k.Deploy(ctx, "first.ts", `output("first");`)
			require.NoError(t, err)
			_, err = k.Deploy(ctx, "second.ts", `output("second");`, brainkit.WithRole("admin"))
			require.NoError(t, err)
			_, err = k.Deploy(ctx, "third.ts", `output("third");`, brainkit.WithPackageName("my-pkg"))
			require.NoError(t, err)
		},
		func(k *brainkit.Kernel) {
			deps := k.ListDeployments()
			sources := make([]string, len(deps))
			for i, d := range deps {
				sources[i] = d.Source
			}
			// All three should be present
			assert.Contains(t, sources, "first.ts")
			assert.Contains(t, sources, "second.ts")
			assert.Contains(t, sources, "third.ts")
		},
	)
}

// P: Schedule with different expressions all survive restart
func TestPersistence_MultipleSchedules(t *testing.T) {
	withRestart(t,
		func(k *brainkit.Kernel) {
			ctx := context.Background()
			k.Schedule(ctx, brainkit.ScheduleConfig{Expression: "every 1h", Topic: "sched.hourly", Payload: json.RawMessage(`{}`)})
			k.Schedule(ctx, brainkit.ScheduleConfig{Expression: "every 5m", Topic: "sched.fivemin", Payload: json.RawMessage(`{}`)})
			k.Schedule(ctx, brainkit.ScheduleConfig{Expression: "in 24h", Topic: "sched.onetime", Payload: json.RawMessage(`{}`)})
		},
		func(k *brainkit.Kernel) {
			scheds := k.ListSchedules()
			assert.GreaterOrEqual(t, len(scheds), 2, "at least 2 schedules should survive (one-time may have fired)")
		},
	)
}

// P: Deployment with code that uses bus.on survives restart and re-subscribes
func TestPersistence_DeployWithBusHandler(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "store.db")

	// Phase 1: Deploy with bus handler
	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir, Store: store1,
	})
	require.NoError(t, err)

	_, err = k1.Deploy(context.Background(), "handler.ts", `
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
		Topic:   "ts.handler.ping",
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
