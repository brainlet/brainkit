package tools

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

func testToolsList(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, messages.ToolListMsg{})
	require.NoError(t, err)
	ch := make(chan messages.ToolListResp, 1)
	unsub, err := sdk.SubscribeTo[messages.ToolListResp](env.Kit, ctx, pr.ReplyTo, func(r messages.ToolListResp, m messages.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()
	var resp messages.ToolListResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
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

	pr, err := sdk.Publish(env.Kit, ctx, messages.ToolResolveMsg{Name: "echo"})
	require.NoError(t, err)
	ch := make(chan messages.ToolResolveResp, 1)
	unsub, err := sdk.SubscribeTo[messages.ToolResolveResp](env.Kit, ctx, pr.ReplyTo, func(r messages.ToolResolveResp, m messages.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()
	var resp messages.ToolResolveResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.Equal(t, "echo", resp.ShortName)
	assert.Equal(t, "echoes the input message", resp.Description)
	assert.NotNil(t, resp.InputSchema)
}

func testToolsResolveNotFound(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, messages.ToolResolveMsg{Name: "nonexistent"})
	require.NoError(t, err)
	ch := make(chan messages.ToolResolveResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.ToolResolveResp](env.Kit, ctx, pr.ReplyTo, func(r messages.ToolResolveResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		assert.NotEmpty(t, resp.Error, "should have error in response")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func testToolsCallEcho(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, messages.ToolCallMsg{
		Name:  "echo",
		Input: map[string]any{"message": "hello world"},
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
	assert.Equal(t, "hello world", result["echoed"])
}

func testToolsCallAdd(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, messages.ToolCallMsg{
		Name:  "add",
		Input: map[string]any{"a": 17, "b": 25},
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
	var result map[string]int
	json.Unmarshal(resp.Result, &result)
	assert.Equal(t, 42, result["sum"])
}

func testToolsCallNotFound(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, messages.ToolCallMsg{
		Name:  "nonexistent",
		Input: map[string]any{},
	})
	require.NoError(t, err)
	ch := make(chan messages.ToolCallResp, 1)
	unsub, _ := sdk.SubscribeTo[messages.ToolCallResp](env.Kit, ctx, pr.ReplyTo, func(r messages.ToolCallResp, m messages.Message) { ch <- r })
	defer unsub()
	select {
	case resp := <-ch:
		assert.NotEmpty(t, resp.Error, "should have error in response")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}
