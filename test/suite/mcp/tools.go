package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testListTools(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, messages.McpListToolsMsg{})
	require.NoError(t, err)
	ch := make(chan messages.McpListToolsResp, 1)
	unsub, err := sdk.SubscribeTo[messages.McpListToolsResp](env.Kit, ctx, pr.ReplyTo, func(r messages.McpListToolsResp, m messages.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()
	var resp messages.McpListToolsResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

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

	pr, err := sdk.Publish(env.Kit, ctx, messages.McpCallToolMsg{
		Server: "testmcp",
		Tool:   "echo",
		Args:   map[string]any{"message": "hello from mcp test"},
	})
	require.NoError(t, err)
	ch := make(chan messages.McpCallToolResp, 1)
	unsub, err := sdk.SubscribeTo[messages.McpCallToolResp](env.Kit, ctx, pr.ReplyTo, func(r messages.McpCallToolResp, m messages.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()
	var resp messages.McpCallToolResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	var result map[string]string
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, "hello from mcp test", result["echoed"])
	assert.Equal(t, "testmcp", result["server"])
}

func testCallToolViaRegistry(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, messages.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": "via registry"},
	})
	require.NoError(t, err)
	ch := make(chan messages.ToolCallResp, 1)
	unsub, err := sdk.SubscribeTo[messages.ToolCallResp](env.Kit, ctx, pr.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()
	var resp messages.ToolCallResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	var result map[string]string
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, "via registry", result["echoed"])
}
