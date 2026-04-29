package tools

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testToolCallRoundtrip — tool call roundtrip (publish ToolCallMsg, receive reply).
// Ported from adversarial/backend_matrix_test.go:TestBackendMatrix_ToolCall.
func testToolCallRoundtrip(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := sdk.Call[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](env.Kit, ctx, toolmsg.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "roundtrip-suite"}})
	require.NoError(t, err)

	assert.Contains(t, string(resp.Result), "roundtrip-suite")
}
