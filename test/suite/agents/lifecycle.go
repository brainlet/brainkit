package agents

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

func testListEmpty(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kernel, ctx, messages.AgentListMsg{})
	require.NoError(t, err)
	ch := make(chan messages.AgentListResp, 1)
	unsub, err := sdk.SubscribeTo[messages.AgentListResp](env.Kernel, ctx, pr.ReplyTo, func(r messages.AgentListResp, m messages.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()
	var resp messages.AgentListResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.Empty(t, resp.Agents)
}

func testDiscoverNoMatch(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kernel, ctx, messages.AgentDiscoverMsg{Capability: "teleportation"})
	require.NoError(t, err)
	ch := make(chan messages.AgentDiscoverResp, 1)
	unsub, err := sdk.SubscribeTo[messages.AgentDiscoverResp](env.Kernel, ctx, pr.ReplyTo, func(r messages.AgentDiscoverResp, m messages.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()
	var resp messages.AgentDiscoverResp
	select {
	case resp = <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}
	assert.Empty(t, resp.Agents)
}

func testGetStatusNotFound(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kernel, ctx, messages.AgentGetStatusMsg{Name: "ghost-agent"})
	require.NoError(t, err)
	ch := make(chan string, 1)
	unsub, _ := env.Kernel.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
		var r struct{ Error string `json:"error"` }
		json.Unmarshal(msg.Payload, &r)
		ch <- r.Error
	})
	defer unsub()
	select {
	case errMsg := <-ch:
		assert.NotEmpty(t, errMsg)
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func testSetStatusNotFound(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kernel, ctx, messages.AgentSetStatusMsg{Name: "ghost-agent", Status: "busy"})
	require.NoError(t, err)
	ch := make(chan string, 1)
	unsub, _ := env.Kernel.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
		var r struct{ Error string `json:"error"` }
		json.Unmarshal(msg.Payload, &r)
		ch <- r.Error
	})
	defer unsub()
	select {
	case errMsg := <-ch:
		assert.NotEmpty(t, errMsg)
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func testSetStatusInvalid(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kernel, ctx, messages.AgentSetStatusMsg{Name: "any", Status: "flying"})
	require.NoError(t, err)
	ch := make(chan string, 1)
	unsub, _ := env.Kernel.SubscribeRaw(ctx, pr.ReplyTo, func(msg messages.Message) {
		var r struct{ Error string `json:"error"` }
		json.Unmarshal(msg.Payload, &r)
		ch <- r.Error
	})
	defer unsub()
	select {
	case errMsg := <-ch:
		assert.NotEmpty(t, errMsg)
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}
