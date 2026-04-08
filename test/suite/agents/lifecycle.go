package agents

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testListEmpty(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pr, err := sdk.Publish(env.Kit, ctx, sdk.AgentListMsg{})
	require.NoError(t, err)
	ch := make(chan sdk.AgentListResp, 1)
	unsub, err := sdk.SubscribeTo[sdk.AgentListResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.AgentListResp, m sdk.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()
	var resp sdk.AgentListResp
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

	pr, err := sdk.Publish(env.Kit, ctx, sdk.AgentDiscoverMsg{Capability: "teleportation"})
	require.NoError(t, err)
	ch := make(chan sdk.AgentDiscoverResp, 1)
	unsub, err := sdk.SubscribeTo[sdk.AgentDiscoverResp](env.Kit, ctx, pr.ReplyTo, func(r sdk.AgentDiscoverResp, m sdk.Message) { ch <- r })
	require.NoError(t, err)
	defer unsub()
	var resp sdk.AgentDiscoverResp
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

	pr, err := sdk.Publish(env.Kit, ctx, sdk.AgentGetStatusMsg{Name: "ghost-agent"})
	require.NoError(t, err)
	ch := make(chan string, 1)
	unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(msg sdk.Message) {
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

	pr, err := sdk.Publish(env.Kit, ctx, sdk.AgentSetStatusMsg{Name: "ghost-agent", Status: "busy"})
	require.NoError(t, err)
	ch := make(chan string, 1)
	unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(msg sdk.Message) {
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

	pr, err := sdk.Publish(env.Kit, ctx, sdk.AgentSetStatusMsg{Name: "any", Status: "flying"})
	require.NoError(t, err)
	ch := make(chan string, 1)
	unsub, _ := env.Kit.SubscribeRaw(ctx, pr.ReplyTo, func(msg sdk.Message) {
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
