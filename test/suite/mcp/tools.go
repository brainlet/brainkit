package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/mcp/mcpmsg"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testListTools(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := sdk.Call[mcpmsg.McpListToolsMsg, mcpmsg.McpListToolsResp](env.Kit, ctx, mcpmsg.McpListToolsMsg{})
	require.NoError(t, err)

	found := false
	for _, tool := range resp.Tools {
		if tool.Name == "echo" && tool.Server == "testmcp" {
			found = true
		}
	}
	assert.True(t, found, "testmcp echo tool should be listed")
}

func testCallTool(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := sdk.Call[mcpmsg.McpCallToolMsg, mcpmsg.McpCallToolResp](env.Kit, ctx, mcpmsg.McpCallToolMsg{
		Server: "testmcp",
		Tool:   "echo",
		Args:   map[string]any{"message": "hello from mcp test"},
	})
	require.NoError(t, err)

	var result map[string]string
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, "hello from mcp test", result["echoed"])
	assert.Equal(t, "testmcp", result["server"])
}

func testCallToolViaRegistry(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := sdk.Call[toolmsg.ToolCallMsg, toolmsg.ToolCallResp](env.Kit, ctx, toolmsg.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": "via registry"},
	})
	require.NoError(t, err)

	var result map[string]string
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, "via registry", result["echoed"])
}
