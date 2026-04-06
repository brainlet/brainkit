package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/messaging"
	toolreg "github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/require"
)

// TestPluginToolCallViaBusSQLite verifies that tools.call for a plugin-style
// tool works on SQLite transport with the pass-through replyTo protocol.
// The host forwards the caller's replyTo to the plugin, the plugin responds directly.
func TestPluginToolCallViaBusSQLite(t *testing.T) {
	dir := t.TempDir()
	transportDB := dir + "/transport.db"

	transport, err := messaging.NewTransportSet(messaging.TransportConfig{
		Type:       "sql-sqlite",
		SQLitePath: transportDB,
	})
	require.NoError(t, err)
	defer transport.Close()

	kernel, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test-plugin-bus",
		Transport: transport,
	})
	require.NoError(t, err)
	defer kernel.Close()

	// Simulate plugin side: subscribe to tool topic, respond.
	// Mirrors sdk/serve.go handler — checks for replyTo in metadata.
	fakeTopic := "fake.plugin.tool.echo"
	fakeResultTopic := fakeTopic + ".result"

	_, err = kernel.SubscribeRaw(context.Background(), fakeTopic, func(msg messages.Message) {
		correlationID := msg.Metadata["correlationId"]
		result := json.RawMessage(`{"echoed":"ok"}`)
		resp, _ := json.Marshal(messages.ToolCallResp{Result: result})

		replyMsg := message.NewMessage(watermill.NewUUID(), resp)
		replyMsg.Metadata.Set("correlationId", correlationID)

		// Pass-through: if host forwarded a replyTo, publish directly there.
		if replyTo := msg.Metadata["replyTo"]; replyTo != "" {
			transport.Publisher.Publish(replyTo, replyMsg)
			return
		}
		// Fallback: publish to .result topic
		ctx := messaging.ContextWithCorrelationID(context.Background(), correlationID)
		kernel.PublishRaw(ctx, fakeResultTopic, resp)
	})
	require.NoError(t, err)

	// Register tool with pass-through executor (mirrors processPluginManifest).
	kernel.Tools.Register(toolreg.RegisteredTool{
		Name:      "test/echo@0.1.0/echo",
		ShortName: "echo",
		Executor: &toolreg.GoFuncExecutor{
			Fn: func(callCtx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				// Pass-through: forward caller's replyTo to the plugin.
				// Uses PublishRawWithMeta to stamp replyTo directly (already resolved).
				callerReplyTo := messaging.ReplyToFromContext(callCtx)
				if callerReplyTo != "" {
					correlationID := messaging.CorrelationIDFromContext(callCtx)
					wmsg := message.NewMessage(watermill.NewUUID(), []byte(input))
					wmsg.Metadata.Set("replyTo", callerReplyTo)
					if correlationID != "" {
						wmsg.Metadata.Set("correlationId", correlationID)
					}
					resolvedTopic := transport.SanitizeTopic(messaging.NamespacedTopic("test-plugin-bus", fakeTopic))
					if err := transport.Publisher.Publish(resolvedTopic, wmsg); err != nil {
						return nil, err
					}
					return nil, nil
				}

				// Fallback: direct call — subscribe and wait.
				correlationID := fmt.Sprintf("%d", time.Now().UnixNano())
				waitCtx, cancel := context.WithCancel(callCtx)
				defer cancel()

				resultCh := make(chan messages.Message, 1)
				stop, subErr := kernel.SubscribeRaw(waitCtx, fakeResultTopic, func(msg messages.Message) {
					if msg.Metadata["correlationId"] == correlationID {
						resultCh <- msg
						cancel()
					}
				})
				if subErr != nil {
					return nil, subErr
				}
				defer stop()

				ctx := messaging.ContextWithCorrelationID(callCtx, correlationID)
				if _, pubErr := kernel.PublishRaw(ctx, fakeTopic, input); pubErr != nil {
					return nil, pubErr
				}

				select {
				case <-callCtx.Done():
					return nil, callCtx.Err()
				case msg := <-resultCh:
					var resp messages.ToolCallResp
					json.Unmarshal(msg.Payload, &resp)
					return resp.Result, nil
				}
			},
		},
	})

	// Test 1: direct executor call (fallback path — subscribe and wait)
	t.Run("direct_executor", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tool, err := kernel.Tools.Resolve("echo")
		require.NoError(t, err)
		result, err := tool.Executor.Call(ctx, "test", []byte(`"hello"`))
		require.NoError(t, err)
		t.Logf("direct: %s", string(result))
	})

	// Test 2: via bus tools.call command (pass-through path)
	t.Run("via_bus_command", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		replyTo := fmt.Sprintf("tools.call.reply.%d", time.Now().UnixNano())
		replyCh := make(chan json.RawMessage, 1)
		unsub, err := kernel.SubscribeRaw(ctx, replyTo, func(msg messages.Message) {
			replyCh <- msg.Payload
		})
		require.NoError(t, err)
		defer unsub()

		start := time.Now()
		_, err = sdk.Publish(kernel, ctx, messages.ToolCallMsg{
			Name:  "echo",
			Input: "test input",
		}, sdk.WithReplyTo(replyTo))
		require.NoError(t, err)

		select {
		case payload := <-replyCh:
			elapsed := time.Since(start)
			t.Logf("bus response in %s: %s", elapsed.Round(time.Millisecond), string(payload))
			require.Less(t, elapsed, 3*time.Second, "plugin tool call via bus must complete in <3s")
			var resp messages.ToolCallResp
			require.NoError(t, json.Unmarshal(payload, &resp))
			require.Empty(t, resp.Error, "plugin tool call must not return an error")
			require.True(t, len(resp.Result) > 0 && string(resp.Result) != "null",
				"plugin tool call must return a non-null result, got: %s", string(resp.Result))
		case <-ctx.Done():
			t.Fatal("REGRESSION: tools.call via bus for plugin-style tool times out on SQLite transport")
		}
	})
}
