package cross

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
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

	resp, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kitA, ctx,
		sdk.CustomMsg{Topic: "ts.xk-handler-suite.ping", Payload: json.RawMessage(`{}`)},
		brainkit.WithCallTo("xk-b-suite"),
	)
	require.NoError(t, err)
	assert.Contains(t, string(resp), "kit-b")
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

	_, err := brainkit.Call[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](kitA, ctx,
		toolmsg.ToolCallMsg{Name: "ghost-cross-kit-tool-suite"},
		brainkit.WithCallTo("xe-b-suite"),
	)
	assertErrorCode(t, err, "NOT_FOUND")
}
