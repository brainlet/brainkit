package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/modules/agents/agentmsg"
	metricsmod "github.com/brainlet/brainkit/modules/metrics"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	"github.com/brainlet/brainkit/modules/registry/registrymsg"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/modules/tools/toolmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

type cmdTest struct {
	topic    string
	valid    sdk.BrainkitMessage
	empty    sdk.BrainkitMessage
	errCode  string
	nodeOnly bool
}

func busCommandTable() []cmdTest {
	return []cmdTest{
		{"tools.call", toolmsg.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "test"}}, toolmsg.ToolCallMsg{Name: ""}, "NOT_FOUND", false},
		{"tools.list", toolmsg.ToolListMsg{}, toolmsg.ToolListMsg{}, "", false},
		{"tools.resolve", toolmsg.ToolResolveMsg{Name: "echo"}, toolmsg.ToolResolveMsg{Name: "ghost-tool-xyz"}, "NOT_FOUND", false},
		{"agents.list", agentmsg.AgentListMsg{}, agentmsg.AgentListMsg{}, "", false},
		{"agents.get-status", agentmsg.AgentGetStatusMsg{Name: "ghost"}, agentmsg.AgentGetStatusMsg{Name: ""}, "VALIDATION_ERROR", false},
		{"agents.set-status", agentmsg.AgentSetStatusMsg{Name: "ghost", Status: "idle"}, agentmsg.AgentSetStatusMsg{Name: "", Status: ""}, "VALIDATION_ERROR", false},
		{"agents.discover", agentmsg.AgentDiscoverMsg{}, agentmsg.AgentDiscoverMsg{}, "", false},
		{"secrets.set", secretmsg.SecretsSetMsg{Name: "matrix-k", Value: "v"}, secretmsg.SecretsSetMsg{Name: "", Value: "v"}, "VALIDATION_ERROR", false},
		{"secrets.get", secretmsg.SecretsGetMsg{Name: "matrix-k"}, secretmsg.SecretsGetMsg{Name: ""}, "VALIDATION_ERROR", false},
		{"secrets.delete", secretmsg.SecretsDeleteMsg{Name: "ghost"}, secretmsg.SecretsDeleteMsg{Name: ""}, "VALIDATION_ERROR", false},
		{"secrets.list", secretmsg.SecretsListMsg{}, secretmsg.SecretsListMsg{}, "", false},
		{"secrets.rotate", secretmsg.SecretsRotateMsg{Name: "matrix-k", NewValue: "v2"}, secretmsg.SecretsRotateMsg{Name: ""}, "VALIDATION_ERROR", false},
		{"registry.has", registrymsg.RegistryHasMsg{Category: "provider", Name: "openai"}, registrymsg.RegistryHasMsg{}, "", false},
		{"registry.list", registrymsg.RegistryListMsg{Category: "provider"}, registrymsg.RegistryListMsg{}, "", false},
		{"registry.resolve", registrymsg.RegistryResolveMsg{Category: "provider", Name: "ghost"}, registrymsg.RegistryResolveMsg{}, "", false},
		{"metrics.get", metricsmod.MetricsGetMsg{}, metricsmod.MetricsGetMsg{}, "", false},
		{"package.list", packagemsg.PackageListDeployedMsg{}, packagemsg.PackageListDeployedMsg{}, "", false},
		{"package.info", packagemsg.PackageDeployInfoMsg{Name: "ghost"}, packagemsg.PackageDeployInfoMsg{Name: ""}, "", false},
		{"package.teardown", packagemsg.PackageTeardownMsg{Name: "ghost"}, packagemsg.PackageTeardownMsg{Name: ""}, "", false},
	}
}

// testBusMatrixValidInput — every command with valid input gets a response (no hang, no panic).
func testBusMatrixValidInput(t *testing.T, _ *suite.TestEnv) {
	tkEnv := suite.Full(t)

	for _, cmd := range busCommandTable() {
		t.Run(cmd.topic, func(t *testing.T) {
			if cmd.nodeOnly {
				t.Skip("node-only")
				return
			}

			payload, ok := tkEnv.SendAndReceive(t, cmd.valid, 5*time.Second)
			if !ok {
				t.Fatalf("timeout — %s hung on valid input", cmd.topic)
			}
			_ = payload
		})
	}
}

// testBusMatrixEmptyInput — every command with empty input returns clean error or empty success.
func testBusMatrixEmptyInput(t *testing.T, _ *suite.TestEnv) {
	tkEnv := suite.Full(t)

	for _, cmd := range busCommandTable() {
		if cmd.errCode == "" {
			continue
		}
		t.Run(cmd.topic, func(t *testing.T) {
			if cmd.nodeOnly {
				t.Skip("node-only")
				return
			}

			payload, ok := tkEnv.SendAndReceive(t, cmd.empty, 5*time.Second)
			if !ok {
				t.Fatalf("timeout — %s hung on empty input", cmd.topic)
			}
			code := suite.ResponseCode(payload)
			assert.Equal(t, cmd.errCode, code, "%s: wrong error code on empty input (payload: %s)", cmd.topic, string(payload))
		})
	}
}

// testBusMatrixGarbagePayload — every command gets garbage JSON and doesn't panic.
func testBusMatrixGarbagePayload(t *testing.T, _ *suite.TestEnv) {
	freshEnv := suite.Full(t)

	garbage := []json.RawMessage{
		json.RawMessage(`{"garbage": true}`),
		json.RawMessage(`"just a string"`),
		json.RawMessage(`42`),
		json.RawMessage(`null`),
		json.RawMessage(`[]`),
		json.RawMessage(`{"deeply": {"nested": {"object": {"with": {"many": "levels"}}}}}`),
	}

	topics := []string{
		"tools.call", "tools.list", "tools.resolve",
		"agents.list", "agents.get-status", "agents.set-status", "agents.discover",
		"package.deploy",
		"secrets.set", "secrets.get", "secrets.delete", "secrets.list", "secrets.rotate",
		"registry.has", "registry.list", "registry.resolve",
		"metrics.get",
		"package.list", "package.info", "package.teardown",
	}

	for _, topic := range topics {
		for i, g := range garbage {
			t.Run(fmt.Sprintf("%s/garbage_%d", topic, i), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				freshEnv.Kit.PublishRaw(ctx, topic, g)

				time.Sleep(50 * time.Millisecond)
				assert.True(t, testutil.Alive(t, freshEnv.Kit), "kernel died after garbage to %s", topic)
			})
		}
	}
}
