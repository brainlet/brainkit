package tools

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testToolsList(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := sdk.Call[toolmsg.ToolListMsg, toolmsg.ToolListResp](env.Kit, ctx, toolmsg.ToolListMsg{})
	require.NoError(t, err)
	names := make(map[string]bool)
	for _, tool := range resp.Tools {
		names[tool.ShortName] = true
	}
	assert.True(t, names["echo"], "echo tool should be registered")
	assert.True(t, names["add"], "add tool should be registered")
}

func testToolsResolveEcho(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := sdk.Call[toolmsg.ToolResolveMsg, toolmsg.ToolResolveResp](env.Kit, ctx, toolmsg.ToolResolveMsg{Name: "echo"})
	require.NoError(t, err)
	assert.Equal(t, "echo", resp.ShortName)
	assert.Equal(t, "echoes the input message", resp.Description)
	assert.NotNil(t, resp.InputSchema)
}

func testToolsResolveNotFound(t *testing.T, env *suite.TestEnv) {
	payload, err := env.PublishAndWait(t, toolmsg.ToolResolveMsg{Name: "nonexistent"}, 10*time.Second)
	require.NoError(t, err)
	assert.True(t, suite.ResponseHasError(payload), "should have error in response")
}

func testToolsCallEcho(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := sdk.Call[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](env.Kit, ctx, toolmsg.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": "hello world"},
	})
	require.NoError(t, err)
	var result map[string]string
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, "hello world", result["echoed"])
}

func testToolsCallAdd(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := sdk.Call[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](env.Kit, ctx, toolmsg.ToolCallMsg{
		Name:  "add",
		Input: map[string]any{"a": 17, "b": 25},
	})
	require.NoError(t, err)
	var result map[string]int
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, 42, result["sum"])
}

func testToolsCallNotFound(t *testing.T, env *suite.TestEnv) {
	payload, err := env.PublishAndWait(t, toolmsg.ToolCallMsg{
		Name:  "nonexistent",
		Input: map[string]any{},
	}, 10*time.Second)
	require.NoError(t, err)
	assert.True(t, suite.ResponseHasError(payload), "should have error in response")
}
