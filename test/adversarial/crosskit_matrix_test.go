package adversarial_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// crossKitBackends returns backends available for cross-Kit tests.
// FIXED (bug #5): all backends now work — the root cause was missing replyTo
// pre-declaration in PublishRawToNamespace (AMQP fanout discarded responses).
func crossKitBackends(t *testing.T) []string {
	t.Helper()
	if !testutil.PodmanAvailable() {
		t.Skip("cross-Kit tests need Podman")
	}
	return []string{"nats", "amqp", "redis", "sql-postgres"}
}

func messagingCfgForBackend(t *testing.T, backend string) brainkit.MessagingConfig {
	t.Helper()
	tcfg := testutil.TransportConfigForBackend(t, backend)
	return brainkit.MessagingConfig{
		Transport:   tcfg.Type,
		NATSURL:     tcfg.NATSURL,
		NATSName:    tcfg.NATSName,
		AMQPURL:     tcfg.AMQPURL,
		RedisURL:    tcfg.RedisURL,
		PostgresURL: tcfg.PostgresURL,
		SQLitePath:  tcfg.SQLitePath,
	}
}

// TestCrossKitMatrix_PublishReply — Kit A publishes to Kit B, gets reply, on every backend.
func TestCrossKitMatrix_PublishReply(t *testing.T) {
	for _, backend := range crossKitBackends(t) {
		t.Run(backend, func(t *testing.T) {
			msgCfg := messagingCfgForBackend(t, backend)
			tmpA := t.TempDir()
			tmpB := t.TempDir()

			nodeA, err := brainkit.NewNode(brainkit.NodeConfig{
				Kernel:    brainkit.KernelConfig{Namespace: "xk-a", CallerID: "xk-a", FSRoot: tmpA},
				Messaging: msgCfg,
			})
			require.NoError(t, err)
			defer nodeA.Close()

			nodeB, err := brainkit.NewNode(brainkit.NodeConfig{
				Kernel:    brainkit.KernelConfig{Namespace: "xk-b", CallerID: "xk-b", FSRoot: tmpB},
				Messaging: msgCfg,
			})
			require.NoError(t, err)
			defer nodeB.Close()

			require.NoError(t, nodeA.Start(context.Background()))
			require.NoError(t, nodeB.Start(context.Background()))

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Kit B handler
			_, err = nodeB.Kernel.Deploy(ctx, "xk-handler.ts", `
				bus.on("ping", function(msg) { msg.reply({from: "kit-b", backend: "`+backend+`"}); });
			`)
			require.NoError(t, err)

			// Kit A publishes to Kit B
			pr, err := sdk.PublishTo[messages.CustomMsg](nodeA, ctx, "xk-b",
				messages.CustomMsg{Topic: "ts.xk-handler.ping", Payload: json.RawMessage(`{}`)})
			require.NoError(t, err)

			ch := make(chan []byte, 1)
			unsub, err := nodeA.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
			require.NoError(t, err)
			defer unsub()

			select {
			case p := <-ch:
				assert.Contains(t, string(p), "kit-b")
			case <-ctx.Done():
				t.Fatalf("timeout on cross-Kit via %s", backend)
			}
		})
	}
}

// TestCrossKitMatrix_ErrorPropagation — error codes survive cross-Kit on every backend.
func TestCrossKitMatrix_ErrorPropagation(t *testing.T) {
	for _, backend := range crossKitBackends(t) {
		t.Run(backend, func(t *testing.T) {
			msgCfg := messagingCfgForBackend(t, backend)
			tmpA := t.TempDir()
			tmpB := t.TempDir()

			nodeA, err := brainkit.NewNode(brainkit.NodeConfig{
				Kernel:    brainkit.KernelConfig{Namespace: "xe-a", CallerID: "xe-a", FSRoot: tmpA},
				Messaging: msgCfg,
			})
			require.NoError(t, err)
			defer nodeA.Close()

			nodeB, err := brainkit.NewNode(brainkit.NodeConfig{
				Kernel:    brainkit.KernelConfig{Namespace: "xe-b", CallerID: "xe-b", FSRoot: tmpB},
				Messaging: msgCfg,
			})
			require.NoError(t, err)
			defer nodeB.Close()

			require.NoError(t, nodeA.Start(context.Background()))
			require.NoError(t, nodeB.Start(context.Background()))

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Call nonexistent tool on Kit B from Kit A
			pr, err := sdk.PublishTo[messages.ToolCallMsg](nodeA, ctx, "xe-b",
				messages.ToolCallMsg{Name: "ghost-cross-kit-tool"})
			require.NoError(t, err)

			ch := make(chan json.RawMessage, 1)
			unsub, err := nodeA.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
				ch <- json.RawMessage(m.Payload)
			})
			require.NoError(t, err)
			defer unsub()

			select {
			case payload := <-ch:
				code := responseCode(payload)
				assert.Equal(t, "NOT_FOUND", code, "error code should survive cross-Kit on %s", backend)
			case <-ctx.Done():
				t.Fatalf("timeout on cross-Kit error propagation via %s", backend)
			}
		})
	}
}
