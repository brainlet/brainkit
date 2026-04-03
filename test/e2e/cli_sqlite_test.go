package e2e_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLI_SQLiteTransport_SendReply simulates the real CLI scenario:
// - Node running on sql-sqlite transport
// - CLI client connecting to the same SQLite DB from a separate transport
// - CLI sends kit.send, Node processes, reply comes back
func TestCLI_SQLiteTransport_SendReply(t *testing.T) {
	testutil.LoadEnv(t)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "transport.db")

	// Create the Node with sql-sqlite transport
	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel: brainkit.KernelConfig{
			Namespace: "test",
			CallerID:  "test-node",
			FSRoot:    tmpDir,
		},
		Messaging: brainkit.MessagingConfig{
			Transport:  "sql-sqlite",
			SQLitePath: dbPath,
		},
	})
	require.NoError(t, err)
	require.NoError(t, node.Start(context.Background()))
	t.Cleanup(func() { node.Close() })

	// Deploy a service on the Node
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = node.Kernel.Deploy(ctx, "ping-svc.ts", `
		bus.on("ping", (msg) => {
			msg.reply({ pong: msg.payload.value });
		});
	`)
	require.NoError(t, err)

	// Create a separate BusClient (simulates CLI connecting to same DB)
	client, err := brainkit.NewClient(brainkit.NodeConfig{
		Kernel: brainkit.KernelConfig{
			Namespace: "test",
		},
		Messaging: brainkit.MessagingConfig{
			Transport:  "sql-sqlite",
			SQLitePath: dbPath,
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })

	// Send kit.send from the CLI client
	pr, err := sdk.Publish(client, ctx, messages.KitSendMsg{
		Topic:   "ts.ping-svc.ping",
		Payload: json.RawMessage(`{"value":"from-cli"}`),
	})
	require.NoError(t, err)

	replyCh := make(chan messages.Message, 1)
	unsub, err := client.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
		select {
		case replyCh <- msg:
		default:
		}
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case msg := <-replyCh:
		var resp messages.KitSendResp
		require.NoError(t, json.Unmarshal(msg.Payload, &resp))
		var payload struct {
			Pong string `json:"pong"`
		}
		require.NoError(t, json.Unmarshal(resp.Payload, &payload))
		assert.Equal(t, "from-cli", payload.Pong)
	case <-ctx.Done():
		t.Fatal("timeout — SQLite transport failed to deliver between Node and CLI client")
	}
}

// TestCLI_SQLiteTransport_MultipleCommands sends several commands rapidly
// to verify no SQLITE_BUSY under concurrent access.
func TestCLI_SQLiteTransport_MultipleCommands(t *testing.T) {
	testutil.LoadEnv(t)
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "transport.db")

	node, err := brainkit.NewNode(brainkit.NodeConfig{
		Kernel: brainkit.KernelConfig{
			Namespace: "test",
			CallerID:  "test-node",
			FSRoot:    tmpDir,
		},
		Messaging: brainkit.MessagingConfig{
			Transport:  "sql-sqlite",
			SQLitePath: dbPath,
		},
	})
	require.NoError(t, err)
	require.NoError(t, node.Start(context.Background()))
	t.Cleanup(func() { node.Close() })

	client, err := brainkit.NewClient(brainkit.NodeConfig{
		Kernel: brainkit.KernelConfig{
			Namespace: "test",
		},
		Messaging: brainkit.MessagingConfig{
			Transport:  "sql-sqlite",
			SQLitePath: dbPath,
		},
	})
	require.NoError(t, err)
	t.Cleanup(func() { client.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Send 5 health checks rapidly
	for i := 0; i < 5; i++ {
		pr, err := sdk.Publish(client, ctx, messages.KitHealthMsg{})
		require.NoError(t, err, "publish %d", i)

		ch := make(chan []byte, 1)
		unsub, err := client.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
			select {
			case ch <- msg.Payload:
			default:
			}
		})
		require.NoError(t, err, "subscribe %d", i)

		select {
		case payload := <-ch:
			var resp messages.KitHealthResp
			require.NoError(t, json.Unmarshal(payload, &resp), "unmarshal %d", i)
			assert.NotEmpty(t, resp.Health, "health %d", i)
		case <-ctx.Done():
			t.Fatalf("timeout on command %d", i)
		}
		unsub()
	}
}
