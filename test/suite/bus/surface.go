package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/rbac"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

type cmdTest struct {
	topic    string
	valid    messages.BrainkitMessage
	empty    messages.BrainkitMessage
	errCode  string
	rbacOnly bool
	nodeOnly bool
}

func busCommandTable() []cmdTest {
	return []cmdTest{
		{"tools.call", messages.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "test"}}, messages.ToolCallMsg{Name: ""}, "NOT_FOUND", false, false},
		{"tools.list", messages.ToolListMsg{}, messages.ToolListMsg{}, "", false, false},
		{"tools.resolve", messages.ToolResolveMsg{Name: "echo"}, messages.ToolResolveMsg{Name: "ghost-tool-xyz"}, "NOT_FOUND", false, false},
		{"agents.list", messages.AgentListMsg{}, messages.AgentListMsg{}, "", false, false},
		{"agents.get-status", messages.AgentGetStatusMsg{Name: "ghost"}, messages.AgentGetStatusMsg{Name: ""}, "VALIDATION_ERROR", false, false},
		{"agents.set-status", messages.AgentSetStatusMsg{Name: "ghost", Status: "idle"}, messages.AgentSetStatusMsg{Name: "", Status: ""}, "VALIDATION_ERROR", false, false},
		{"agents.discover", messages.AgentDiscoverMsg{}, messages.AgentDiscoverMsg{}, "", false, false},
		{"kit.list", messages.KitListMsg{}, messages.KitListMsg{}, "", false, false},
		{"kit.teardown", messages.KitTeardownMsg{Source: "ghost.ts"}, messages.KitTeardownMsg{Source: ""}, "", false, false},
		{"secrets.set", messages.SecretsSetMsg{Name: "matrix-k", Value: "v"}, messages.SecretsSetMsg{Name: "", Value: "v"}, "VALIDATION_ERROR", false, false},
		{"secrets.get", messages.SecretsGetMsg{Name: "matrix-k"}, messages.SecretsGetMsg{Name: ""}, "VALIDATION_ERROR", false, false},
		{"secrets.delete", messages.SecretsDeleteMsg{Name: "ghost"}, messages.SecretsDeleteMsg{Name: ""}, "VALIDATION_ERROR", false, false},
		{"secrets.list", messages.SecretsListMsg{}, messages.SecretsListMsg{}, "", false, false},
		{"secrets.rotate", messages.SecretsRotateMsg{Name: "matrix-k", NewValue: "v2"}, messages.SecretsRotateMsg{Name: ""}, "VALIDATION_ERROR", false, false},
		{"registry.has", messages.RegistryHasMsg{Category: "provider", Name: "openai"}, messages.RegistryHasMsg{}, "", false, false},
		{"registry.list", messages.RegistryListMsg{Category: "provider"}, messages.RegistryListMsg{}, "", false, false},
		{"registry.resolve", messages.RegistryResolveMsg{Category: "provider", Name: "ghost"}, messages.RegistryResolveMsg{}, "", false, false},
		{"metrics.get", messages.MetricsGetMsg{}, messages.MetricsGetMsg{}, "", false, false},
		{"rbac.assign", messages.RBACAssignMsg{Source: "test.ts", Role: "admin"}, messages.RBACAssignMsg{Source: "", Role: "admin"}, "VALIDATION_ERROR", true, false},
		{"rbac.revoke", messages.RBACRevokeMsg{Source: "test.ts"}, messages.RBACRevokeMsg{}, "", true, false},
		{"rbac.list", messages.RBACListMsg{}, messages.RBACListMsg{}, "", true, false},
		{"rbac.roles", messages.RBACRolesMsg{}, messages.RBACRolesMsg{}, "", true, false},
		{"packages.search", messages.PackagesSearchMsg{Query: "test"}, messages.PackagesSearchMsg{}, "", false, false},
		{"packages.list", messages.PackagesListMsg{}, messages.PackagesListMsg{}, "", false, false},
		{"packages.info", messages.PackagesInfoMsg{Name: "ghost"}, messages.PackagesInfoMsg{Name: ""}, "", false, false},
		{"package.list", messages.PackageListDeployedMsg{}, messages.PackageListDeployedMsg{}, "", false, false},
		{"package.info", messages.PackageDeployInfoMsg{Name: "ghost"}, messages.PackageDeployInfoMsg{Name: ""}, "", false, false},
		{"package.teardown", messages.PackageTeardownMsg{Name: "ghost"}, messages.PackageTeardownMsg{Name: ""}, "", false, false},
	}
}

// testBusMatrixValidInput — every command with valid input gets a response (no hang, no panic).
func testBusMatrixValidInput(t *testing.T, _ *suite.TestEnv) {
	tkEnv := suite.Full(t)
	tkRBACEnv := suite.Full(t, suite.WithRBAC(map[string]rbac.Role{"admin": rbac.RoleAdmin}, "admin"), suite.WithPersistence())

	for _, cmd := range busCommandTable() {
		t.Run(cmd.topic, func(t *testing.T) {
			if cmd.nodeOnly {
				t.Skip("node-only")
				return
			}
			env := tkEnv
			if cmd.rbacOnly {
				env = tkRBACEnv
			}

			payload, ok := env.SendAndReceive(t, cmd.valid, 5*time.Second)
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
	tkRBACEnv := suite.Full(t, suite.WithRBAC(map[string]rbac.Role{"admin": rbac.RoleAdmin}, "admin"), suite.WithPersistence())

	for _, cmd := range busCommandTable() {
		if cmd.errCode == "" {
			continue
		}
		t.Run(cmd.topic, func(t *testing.T) {
			if cmd.nodeOnly {
				t.Skip("node-only")
				return
			}
			env := tkEnv
			if cmd.rbacOnly {
				env = tkRBACEnv
			}

			payload, ok := env.SendAndReceive(t, cmd.empty, 5*time.Second)
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
		"kit.list", "kit.deploy", "kit.teardown",
		"secrets.set", "secrets.get", "secrets.delete", "secrets.list", "secrets.rotate",
		"registry.has", "registry.list", "registry.resolve",
		"metrics.get",
		"packages.search", "packages.list", "packages.info",
		"package.list", "package.info", "package.teardown",
	}

	for _, topic := range topics {
		for i, g := range garbage {
			t.Run(fmt.Sprintf("%s/garbage_%d", topic, i), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				freshEnv.Kernel.PublishRaw(ctx, topic, g)

				time.Sleep(50 * time.Millisecond)
				assert.True(t, freshEnv.Kernel.Alive(ctx), "kernel died after garbage to %s", topic)
			})
		}
	}
}

