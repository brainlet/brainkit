package tools

import (
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testInputAbuseCallNonexistent — calling a nonexistent tool returns NOT_FOUND.
func testInputAbuseCallNonexistent(t *testing.T, env *suite.TestEnv) {
	payload, ok := env.SendAndReceive(t, sdk.ToolCallMsg{Name: "absolutely-does-not-exist-tool-adv"}, 5*time.Second)
	require.True(t, ok, "should receive a response, not timeout")
	code := suite.ResponseCode(payload)
	assert.Equal(t, "NOT_FOUND", code)
}

// testInputAbuseWrongInputType — calling a tool with wrong input shape doesn't hang.
func testInputAbuseWrongInputType(t *testing.T, env *suite.TestEnv) {
	ctx := env.T.Context()
	pr, err := sdk.Publish(env.Kit, ctx, sdk.ToolCallMsg{Name: "echo", Input: "not-an-object"})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case payload := <-ch:
		// Should get a response (possibly error) — not a hang
		assert.NotEmpty(t, payload)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout — tool call with malformed input hung")
	}
}

// testInputAbuseEmptyToolName — calling with empty tool name returns error.
func testInputAbuseEmptyToolName(t *testing.T, env *suite.TestEnv) {
	payload, ok := env.SendAndReceive(t, sdk.ToolCallMsg{Name: "", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok, "should receive a response, not timeout")
	// Should return an error of some kind
	assert.True(t, suite.ResponseHasError(payload) || suite.ResponseCode(payload) != "", "empty name should produce error response")
}

// testInputAbuseOversizedInput — calling a tool with a very large input doesn't crash.
func testInputAbuseOversizedInput(t *testing.T, env *suite.TestEnv) {
	ctx := env.T.Context()

	// 100KB input value
	big := make([]byte, 100000)
	for i := range big {
		big[i] = 'x'
	}

	pr, err := sdk.Publish(env.Kit, ctx, sdk.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": string(big)},
	})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case payload := <-ch:
		// Should succeed or error cleanly — never crash
		assert.NotEmpty(t, payload)
	case <-time.After(10 * time.Second):
		t.Fatal("timeout — tool call with oversized input hung")
	}
}
