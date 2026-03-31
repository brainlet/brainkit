package adversarial_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBackendMatrix_PublishReply tests publish+reply on every transport backend.
func TestBackendMatrix_PublishReply(t *testing.T) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			transport := testutil.CreateTestTransport(t, backend)
			tmpDir := t.TempDir()

			k, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test",
				CallerID:  "test-" + backend,
				FSRoot:    tmpDir,
				Transport: transport,
			})
			require.NoError(t, err)
			defer k.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Deploy a handler
			_, err = k.Deploy(ctx, "backend-test.ts", `
				bus.on("ping", function(msg) { msg.reply({backend: "works"}); });
			`)
			require.NoError(t, err)

			// Publish and wait for reply
			pr, err := sdk.Publish(k, ctx, messages.CustomMsg{
				Topic:   "ts.backend-test.ping",
				Payload: json.RawMessage(`{}`),
			})
			require.NoError(t, err)

			ch := make(chan []byte, 1)
			unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
			require.NoError(t, err)
			defer unsub()

			select {
			case p := <-ch:
				assert.Contains(t, string(p), "works")
			case <-ctx.Done():
				t.Fatalf("timeout on %s backend", backend)
			}
		})
	}
}

// TestBackendMatrix_ToolCall tests tool call roundtrip on every backend.
func TestBackendMatrix_ToolCall(t *testing.T) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			transport := testutil.CreateTestTransport(t, backend)
			tmpDir := t.TempDir()

			k, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test",
				CallerID:  "test-" + backend,
				FSRoot:    tmpDir,
				Transport: transport,
			})
			require.NoError(t, err)
			defer k.Close()

			type echoIn struct{ Message string `json:"message"` }
			brainkit.RegisterTool(k, "echo", registry.TypedTool[echoIn]{
				Description: "echoes",
				Execute: func(ctx context.Context, in echoIn) (any, error) {
					return map[string]string{"echoed": in.Message}, nil
				},
			})

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			pr, err := sdk.Publish(k, ctx, messages.ToolCallMsg{Name: "echo", Input: map[string]any{"message": backend}})
			require.NoError(t, err)

			ch := make(chan []byte, 1)
			unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
			require.NoError(t, err)
			defer unsub()

			select {
			case p := <-ch:
				assert.Contains(t, string(p), backend)
			case <-ctx.Done():
				t.Fatalf("timeout on %s backend", backend)
			}
		})
	}
}

// TestBackendMatrix_DeployPersistRestart tests deploy persistence on every backend.
func TestBackendMatrix_DeployPersistRestart(t *testing.T) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			transport := testutil.CreateTestTransport(t, backend)
			tmpDir := t.TempDir()
			storePath := filepath.Join(tmpDir, "store.db")
			store, err := brainkit.NewSQLiteStore(storePath)
			require.NoError(t, err)

			// Phase 1: Deploy
			k1, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test",
				CallerID:  "test-" + backend,
				FSRoot:    tmpDir,
				Transport: transport,
				Store:     store,
			})
			require.NoError(t, err)

			_, err = k1.Deploy(context.Background(), "persist-test.ts", `output("persisted");`)
			require.NoError(t, err)
			k1.Close()

			// Phase 2: Reopen with same store — deployment should be restored
			transport2 := testutil.CreateTestTransport(t, backend)
			store2, err := brainkit.NewSQLiteStore(storePath)
			require.NoError(t, err)

			k2, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test",
				CallerID:  "test-" + backend,
				FSRoot:    tmpDir,
				Transport: transport2,
				Store:     store2,
			})
			require.NoError(t, err)
			defer k2.Close()

			deps := k2.ListDeployments()
			found := false
			for _, d := range deps {
				if d.Source == "persist-test.ts" {
					found = true
				}
			}
			assert.True(t, found, "persist-test.ts should survive restart on %s", backend)
		})
	}
}

// TestBackendMatrix_ErrorCodeOnBus tests that error codes survive transport on every backend.
func TestBackendMatrix_ErrorCodeOnBus(t *testing.T) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			transport := testutil.CreateTestTransport(t, backend)
			tmpDir := t.TempDir()

			k, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test",
				CallerID:  "test-" + backend,
				FSRoot:    tmpDir,
				Transport: transport,
			})
			require.NoError(t, err)
			defer k.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Call nonexistent tool — should get NOT_FOUND with code field
			pr, err := sdk.Publish(k, ctx, messages.ToolCallMsg{Name: "ghost-backend-tool"})
			require.NoError(t, err)

			ch := make(chan json.RawMessage, 1)
			unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
				ch <- json.RawMessage(m.Payload)
			})
			require.NoError(t, err)
			defer unsub()

			select {
			case payload := <-ch:
				code := responseCode(payload)
				assert.Equal(t, "NOT_FOUND", code, "error code should survive %s transport", backend)
			case <-ctx.Done():
				t.Fatalf("timeout on %s backend", backend)
			}
		})
	}
}

// TestBackendMatrix_LargePayload tests 100KB message on every backend.
func TestBackendMatrix_LargePayload(t *testing.T) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			transport := testutil.CreateTestTransport(t, backend)
			tmpDir := t.TempDir()

			k, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test",
				CallerID:  "test-" + backend,
				FSRoot:    tmpDir,
				Transport: transport,
			})
			require.NoError(t, err)
			defer k.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			// Deploy handler that echoes payload size
			_, err = k.Deploy(ctx, "big-msg.ts", `
				bus.on("big", function(msg) {
					var size = JSON.stringify(msg.payload).length;
					msg.reply({size: size});
				});
			`)
			require.NoError(t, err)

			// Build 100KB payload
			big := make([]byte, 100000)
			for i := range big {
				big[i] = 'x'
			}
			payload, _ := json.Marshal(map[string]string{"data": string(big)})

			pr, err := sdk.Publish(k, ctx, messages.CustomMsg{
				Topic:   "ts.big-msg.big",
				Payload: payload,
			})
			require.NoError(t, err)

			ch := make(chan []byte, 1)
			unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
			require.NoError(t, err)
			defer unsub()

			select {
			case p := <-ch:
				assert.Contains(t, string(p), "size")
			case <-ctx.Done():
				t.Fatalf("timeout on %s backend with 100KB payload", backend)
			}
		})
	}
}

// TestBackendMatrix_DottedTopicNames tests topics with dots on every backend (sanitizer-sensitive).
func TestBackendMatrix_DottedTopicNames(t *testing.T) {
	for _, backend := range testutil.AllBackends(t) {
		t.Run(backend, func(t *testing.T) {
			transport := testutil.CreateTestTransport(t, backend)
			tmpDir := t.TempDir()

			k, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test",
				CallerID:  "test-" + backend,
				FSRoot:    tmpDir,
				Transport: transport,
			})
			require.NoError(t, err)
			defer k.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Deploy with a dotted source name — topic becomes ts.my.dotted.agent.ask
			_, err = k.Deploy(ctx, "my.dotted.agent.ts", `
				bus.on("ask", function(msg) { msg.reply({dotted: true}); });
			`)
			require.NoError(t, err)

			// Publish to the dotted topic
			pr, err := sdk.Publish(k, ctx, messages.CustomMsg{
				Topic:   "ts.my.dotted.agent.ask",
				Payload: json.RawMessage(`{}`),
			})
			require.NoError(t, err)

			ch := make(chan []byte, 1)
			unsub, err := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
			require.NoError(t, err)
			defer unsub()

			select {
			case p := <-ch:
				assert.Contains(t, string(p), "dotted")
			case <-ctx.Done():
				t.Fatalf("timeout on %s backend with dotted topic", backend)
			}
		})
	}
}
