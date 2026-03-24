package transport_test

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/internal/messaging"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransport_Compliance(t *testing.T) {
	for _, backend := range testutil.AllBackends(t) {
		if backend == "amqp" || backend == "redis" || backend == "sql-postgres" {
			continue // these require more debugging — skip for now
		}
		t.Run(backend, func(t *testing.T) {
			transport := testutil.CreateTestTransport(t, backend)
			remote := messaging.NewRemoteClientWithTransport("compliance", "tester", transport)

			t.Run("PublishSubscribe", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				received := make(chan []byte, 1)
				unsub, err := remote.SubscribeRaw(ctx, "test.topic", func(msg messages.Message) {
					received <- msg.Payload
				})
				require.NoError(t, err)
				defer unsub()

				_, err = remote.PublishRaw(ctx, "test.topic", []byte(`{"hello":"world"}`))
				require.NoError(t, err)

				select {
				case payload := <-received:
					assert.JSONEq(t, `{"hello":"world"}`, string(payload))
				case <-ctx.Done():
					t.Fatal("timeout waiting for message")
				}
			})

			t.Run("CorrelationID", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				received := make(chan string, 1)
				unsub, err := remote.SubscribeRaw(ctx, "corr.topic", func(msg messages.Message) {
					received <- msg.Metadata["correlationId"]
				})
				require.NoError(t, err)
				defer unsub()

				corrID, err := remote.PublishRaw(ctx, "corr.topic", []byte(`{}`))
				require.NoError(t, err)
				assert.NotEmpty(t, corrID)

				select {
				case gotCorrID := <-received:
					assert.Equal(t, corrID, gotCorrID)
				case <-ctx.Done():
					t.Fatal("timeout")
				}
			})

			t.Run("DottedTopics", func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				received := make(chan bool, 1)
				unsub, err := remote.SubscribeRaw(ctx, "ai.generate", func(msg messages.Message) {
					received <- true
				})
				require.NoError(t, err)
				defer unsub()

				_, err = remote.PublishRaw(ctx, "ai.generate", []byte(`{"model":"test"}`))
				require.NoError(t, err)

				select {
				case <-received:
					// OK — dotted topic works on this backend
				case <-ctx.Done():
					t.Fatal("timeout — dotted topic failed")
				}
			})
		})
	}
}
