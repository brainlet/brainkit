package tools

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testToolCallRoundtrip — tool call roundtrip (publish ToolCallMsg, receive reply).
// Ported from adversarial/backend_matrix_test.go:TestBackendMatrix_ToolCall.
func testToolCallRoundtrip(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kernel, ctx, messages.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "roundtrip-suite"}})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, err := env.Kernel.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	require.NoError(t, err)
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "roundtrip-suite")
	case <-ctx.Done():
		t.Fatal("timeout on tool call roundtrip")
	}
}
