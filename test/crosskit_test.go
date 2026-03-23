package test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrossKit_BasicRoundTrip(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, kitB := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Kit A publishes to Kit B's namespace, Kit B subscribes
			received := make(chan []byte, 1)
			unsub, err := kitB.SubscribeRaw(ctx, "crosskit.ping", func(msg messages.Message) {
				received <- msg.Payload
			})
			require.NoError(t, err)
			defer unsub()

			// Kit A publishes to Kit B's namespace via CrossNamespaceRuntime
			xrtA, ok := kitA.(sdk.CrossNamespaceRuntime)
			require.True(t, ok, "Kit A should implement CrossNamespaceRuntime")

			_, err = xrtA.PublishRawTo(ctx, "kit-b", "crosskit.ping", json.RawMessage(`{"from":"kit-a"}`))
			require.NoError(t, err)

			select {
			case payload := <-received:
				var msg map[string]string
				json.Unmarshal(payload, &msg)
				assert.Equal(t, "kit-a", msg["from"])
			case <-ctx.Done():
				t.Fatal("timeout waiting for cross-Kit message")
			}
		})
	}
}

func TestCrossKit_BidirectionalToolCall(t *testing.T) {
	for _, backend := range allBackends(t) {
		t.Run(backend, func(t *testing.T) {
			kitA, kitB := newTestKernelPair(t, backend)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			// Both Kits have "echo" tool registered (from newKitWithNamespace)
			// Kit A calls Kit B's echo tool via CrossNamespaceRuntime
			xrtA, ok := kitA.(sdk.CrossNamespaceRuntime)
			require.True(t, ok)

			// Subscribe to Kit B's tools.call.result in Kit B's namespace
			resultCh := make(chan messages.Message, 1)
			unsub, err := xrtA.SubscribeRawTo(ctx, "kit-b", "tools.call.result", func(msg messages.Message) {
				resultCh <- msg
			})
			require.NoError(t, err)
			defer unsub()

			// Publish tools.call to Kit B's namespace
			corrID, err := xrtA.PublishRawTo(ctx, "kit-b", "tools.call", json.RawMessage(`{"name":"echo","input":{"message":"cross-kit hello"}}`))
			require.NoError(t, err)
			assert.NotEmpty(t, corrID)

			// Wait for result from Kit B
			select {
			case msg := <-resultCh:
				if msg.Metadata["correlationId"] == corrID {
					var resp messages.ToolCallResp
					json.Unmarshal(msg.Payload, &resp)
					var result map[string]string
					json.Unmarshal(resp.Result, &result)
					assert.Equal(t, "cross-kit hello", result["echoed"])
				}
			case <-ctx.Done():
				t.Fatal("timeout waiting for cross-Kit tool result")
			}

			// Reverse: Kit B calls Kit A
			xrtB, ok := kitB.(sdk.CrossNamespaceRuntime)
			require.True(t, ok)

			resultCh2 := make(chan messages.Message, 1)
			unsub2, err := xrtB.SubscribeRawTo(ctx, "kit-a", "tools.call.result", func(msg messages.Message) {
				resultCh2 <- msg
			})
			require.NoError(t, err)
			defer unsub2()

			corrID2, err := xrtB.PublishRawTo(ctx, "kit-a", "tools.call", json.RawMessage(`{"name":"echo","input":{"message":"reverse call"}}`))
			require.NoError(t, err)

			select {
			case msg := <-resultCh2:
				if msg.Metadata["correlationId"] == corrID2 {
					var resp messages.ToolCallResp
					json.Unmarshal(msg.Payload, &resp)
					var result map[string]string
					json.Unmarshal(resp.Result, &result)
					assert.Equal(t, "reverse call", result["echoed"])
				}
			case <-ctx.Done():
				t.Fatal("timeout waiting for reverse cross-Kit tool result")
			}
		})
	}
}
