package tools

import (
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testInputAbuseCallNonexistent — calling a nonexistent tool returns NOT_FOUND.
func testInputAbuseCallNonexistent(t *testing.T, env *suite.TestEnv) {
	payload, ok := env.SendAndReceive(t, toolmsg.ToolCallMsg{Name: "absolutely-does-not-exist-tool-adv"}, 5*time.Second)
	require.True(t, ok, "should receive a response, not timeout")
	code := suite.ResponseCode(payload)
	assert.Equal(t, "NOT_FOUND", code)
}

// testInputAbuseWrongInputType — calling a tool with wrong input shape doesn't hang.
func testInputAbuseWrongInputType(t *testing.T, env *suite.TestEnv) {
	payload, err := env.PublishAndWait(t, toolmsg.ToolCallMsg{Name: "echo", Input: "not-an-object"}, 5*time.Second)
	require.NoError(t, err)
	assert.NotEmpty(t, payload)
}

// testInputAbuseEmptyToolName — calling with empty tool name returns error.
func testInputAbuseEmptyToolName(t *testing.T, env *suite.TestEnv) {
	payload, ok := env.SendAndReceive(t, toolmsg.ToolCallMsg{Name: "", Input: map[string]any{}}, 5*time.Second)
	require.True(t, ok, "should receive a response, not timeout")
	// Should return an error of some kind
	assert.True(t, suite.ResponseHasError(payload) || suite.ResponseCode(payload) != "", "empty name should produce error response")
}

// testInputAbuseOversizedInput — calling a tool with a very large input doesn't crash.
func testInputAbuseOversizedInput(t *testing.T, env *suite.TestEnv) {
	// 100KB input value
	big := make([]byte, 100000)
	for i := range big {
		big[i] = 'x'
	}

	payload, err := env.PublishAndWait(t, toolmsg.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": string(big)},
	}, 10*time.Second)
	require.NoError(t, err)
	assert.NotEmpty(t, payload)
}
