package suite

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/rbac"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnv_Full_Smoke(t *testing.T) {
	env := Full(t)
	require.NotNil(t, env.Kit)
	result, err := env.EvalTS(`return "hello"`)
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestEnv_Full_ToolsRegistered(t *testing.T) {
	env := Full(t)
	// Verify echo and add tools via bus command
	payload := testutil.PublishAndWait(t, env.Kit, messages.ToolListMsg{}, 5*time.Second)
	var resp messages.ToolListResp
	require.NoError(t, json.Unmarshal(payload, &resp))
	names := make(map[string]bool)
	for _, tool := range resp.Tools {
		names[tool.ShortName] = true
	}
	assert.True(t, names["echo"], "echo tool should be registered")
	assert.True(t, names["add"], "add tool should be registered")
}

func TestEnv_Full_WithRBAC(t *testing.T) {
	env := Full(t, WithRBAC(map[string]rbac.Role{
		"admin":   rbac.RoleAdmin,
		"service": rbac.RoleService,
	}, "service"), WithPersistence())
	require.NotNil(t, env.Kit)
	assert.Equal(t, "sqlite", env.Config.Persistence)
}

func TestEnv_Full_WithTracing(t *testing.T) {
	env := Full(t, WithTracing())
	require.NotNil(t, env.Kit)
	assert.True(t, env.Config.Tracing)
}

func TestEnv_Minimal_Smoke(t *testing.T) {
	env := Minimal(t)
	require.NotNil(t, env.Kit)
}

func TestEnv_Minimal_NoTools(t *testing.T) {
	env := Minimal(t)
	payload := testutil.PublishAndWait(t, env.Kit, messages.ToolListMsg{}, 5*time.Second)
	var resp messages.ToolListResp
	require.NoError(t, json.Unmarshal(payload, &resp))
	assert.Empty(t, resp.Tools, "minimal env should have no tools")
}

func TestEnv_SendAndReceive(t *testing.T) {
	env := Full(t)
	// Deploy a simple echo handler
	err := env.Deploy("echo-handler.ts", `
		bus.on("test-echo", (msg) => {
			msg.reply({ echoed: msg.payload.text });
		});
	`)
	require.NoError(t, err)

	// Use SendAndReceive helper
	payload, ok := env.SendAndReceive(t, messages.CustomMsg{
		Topic:   "ts.echo-handler.test-echo",
		Payload: []byte(`{"text":"hello"}`),
	}, 5*time.Second)
	require.True(t, ok, "should receive response")
	assert.Contains(t, string(payload), "echoed")
}
