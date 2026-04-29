package agents

import (
	"context"
	"testing"
	"time"

	"github.com/brainlet/brainkit/modules/agents/agentmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testListEmpty(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := sdk.Call[agentmsg.AgentListMsg, agentmsg.AgentListResp](env.Kit, ctx, agentmsg.AgentListMsg{})
	require.NoError(t, err)
	assert.Empty(t, resp.Agents)
}

func testDiscoverNoMatch(t *testing.T, env *suite.TestEnv) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := sdk.Call[agentmsg.AgentDiscoverMsg, agentmsg.AgentDiscoverResp](env.Kit, ctx, agentmsg.AgentDiscoverMsg{Capability: "teleportation"})
	require.NoError(t, err)
	assert.Empty(t, resp.Agents)
}

func testGetStatusNotFound(t *testing.T, env *suite.TestEnv) {
	payload, err := env.PublishAndWait(t, agentmsg.AgentGetStatusMsg{Name: "ghost-agent"}, 15*time.Second)
	require.NoError(t, err)
	assert.NotEmpty(t, suite.ResponseErrorMessage(payload))
}

func testSetStatusNotFound(t *testing.T, env *suite.TestEnv) {
	payload, err := env.PublishAndWait(t, agentmsg.AgentSetStatusMsg{Name: "ghost-agent", Status: "busy"}, 15*time.Second)
	require.NoError(t, err)
	assert.NotEmpty(t, suite.ResponseErrorMessage(payload))
}

func testSetStatusInvalid(t *testing.T, env *suite.TestEnv) {
	payload, err := env.PublishAndWait(t, agentmsg.AgentSetStatusMsg{Name: "any", Status: "flying"}, 15*time.Second)
	require.NoError(t, err)
	assert.NotEmpty(t, suite.ResponseErrorMessage(payload))
}
