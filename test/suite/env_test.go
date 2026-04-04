package suite

import (
	"testing"
	"time"

	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnv_Full_Smoke(t *testing.T) {
	env := Full(t)
	require.NotNil(t, env.Kernel)
	result, err := env.EvalTS(`return "hello"`)
	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestEnv_Full_ToolsRegistered(t *testing.T) {
	env := Full(t)
	// Verify echo and add tools are registered
	tools := env.Kernel.Tools.List("")
	names := make(map[string]bool)
	for _, tool := range tools {
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
	require.NotNil(t, env.Kernel)
	assert.Equal(t, "sqlite", env.Config.Persistence)
}

func TestEnv_Full_WithTracing(t *testing.T) {
	env := Full(t, WithTracing())
	require.NotNil(t, env.Kernel)
	assert.True(t, env.Config.Tracing)
}

func TestEnv_Minimal_Smoke(t *testing.T) {
	env := Minimal(t)
	require.NotNil(t, env.Kernel)
}

func TestEnv_Minimal_NoTools(t *testing.T) {
	env := Minimal(t)
	tools := env.Kernel.Tools.List("")
	assert.Empty(t, tools, "minimal env should have no tools")
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
