package cross

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testCrossKitPublishReply — Kit A publishes to Kit B, gets reply.
// Ported from adversarial/crosskit_matrix_test.go:TestCrossKitMatrix_PublishReply.
// Requires Podman (needs real transport for cross-Kit).
func testCrossKitPublishReply(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)

	// Both nodes must share the SAME transport for cross-Kit communication
	sharedTF := transportFieldsForBackend(t, "nats")
	kitA := makeNodeWithConfig(t, env, "xk-a-suite", sharedTF)
	kitB := makeNodeWithConfig(t, env, "xk-b-suite", sharedTF)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Kit B handler
	testutil.Deploy(t, kitB, "xk-handler-suite.ts", `
		bus.on("ping", function(msg) { msg.reply({from: "kit-b", test: "suite"}); });
	`)

	// Kit A publishes to Kit B
	pr, err := sdk.PublishTo[messages.CustomMsg](kitA, ctx, "xk-b-suite",
		messages.CustomMsg{Topic: "ts.xk-handler-suite.ping", Payload: json.RawMessage(`{}`)})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, err := kitA.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	require.NoError(t, err)
	defer unsub()

	select {
	case p := <-ch:
		assert.Contains(t, string(p), "kit-b")
	case <-ctx.Done():
		t.Fatal("timeout on cross-Kit publish/reply")
	}
}

// testCrossKitErrorPropagation — error codes survive cross-Kit.
// Ported from adversarial/crosskit_matrix_test.go:TestCrossKitMatrix_ErrorPropagation.
// Requires Podman (needs real transport for cross-Kit).
func testCrossKitErrorPropagation(t *testing.T, env *suite.TestEnv) {
	env.RequirePodman(t)

	// Both nodes must share the SAME transport for cross-Kit communication
	sharedTF := transportFieldsForBackend(t, "nats")
	kitA := makeNodeWithConfig(t, env, "xe-a-suite", sharedTF)
	_ = makeNodeWithConfig(t, env, "xe-b-suite", sharedTF) // Kit B must exist to receive the call

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Call nonexistent tool on Kit B from Kit A
	pr, err := sdk.PublishTo[messages.ToolCallMsg](kitA, ctx, "xe-b-suite",
		messages.ToolCallMsg{Name: "ghost-cross-kit-tool-suite"})
	require.NoError(t, err)

	ch := make(chan json.RawMessage, 1)
	unsub, err := kitA.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) {
		ch <- json.RawMessage(m.Payload)
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case payload := <-ch:
		code := suite.ResponseCode(payload)
		assert.Equal(t, "NOT_FOUND", code, "error code should survive cross-Kit")
	case <-ctx.Done():
		t.Fatal("timeout on cross-Kit error propagation")
	}
}
