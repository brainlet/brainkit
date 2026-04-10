package bus

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
)

type cmdTest struct {
	topic    string
	valid    sdk.BrainkitMessage
	empty    sdk.BrainkitMessage
	errCode  string
	rbacOnly bool
	nodeOnly bool
}

func busCommandTable() []cmdTest {
	return []cmdTest{
		{"tools.call", sdk.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "test"}}, sdk.ToolCallMsg{Name: ""}, "NOT_FOUND", false, false},
		{"tools.list", sdk.ToolListMsg{}, sdk.ToolListMsg{}, "", false, false},
		{"tools.resolve", sdk.ToolResolveMsg{Name: "echo"}, sdk.ToolResolveMsg{Name: "ghost-tool-xyz"}, "NOT_FOUND", false, false},
		{"agents.list", sdk.AgentListMsg{}, sdk.AgentListMsg{}, "", false, false},
		{"agents.get-status", sdk.AgentGetStatusMsg{Name: "ghost"}, sdk.AgentGetStatusMsg{Name: ""}, "VALIDATION_ERROR", false, false},
		{"agents.set-status", sdk.AgentSetStatusMsg{Name: "ghost", Status: "idle"}, sdk.AgentSetStatusMsg{Name: "", Status: ""}, "VALIDATION_ERROR", false, false},
		{"agents.discover", sdk.AgentDiscoverMsg{}, sdk.AgentDiscoverMsg{}, "", false, false},
		{"kit.list", sdk.KitListMsg{}, sdk.KitListMsg{}, "", false, false},
		{"kit.teardown", sdk.KitTeardownMsg{Source: "ghost.ts"}, sdk.KitTeardownMsg{Source: ""}, "", false, false},
		{"secrets.set", sdk.SecretsSetMsg{Name: "matrix-k", Value: "v"}, sdk.SecretsSetMsg{Name: "", Value: "v"}, "VALIDATION_ERROR", false, false},
		{"secrets.get", sdk.SecretsGetMsg{Name: "matrix-k"}, sdk.SecretsGetMsg{Name: ""}, "VALIDATION_ERROR", false, false},
		{"secrets.delete", sdk.SecretsDeleteMsg{Name: "ghost"}, sdk.SecretsDeleteMsg{Name: ""}, "VALIDATION_ERROR", false, false},
		{"secrets.list", sdk.SecretsListMsg{}, sdk.SecretsListMsg{}, "", false, false},
		{"secrets.rotate", sdk.SecretsRotateMsg{Name: "matrix-k", NewValue: "v2"}, sdk.SecretsRotateMsg{Name: ""}, "VALIDATION_ERROR", false, false},
		{"registry.has", sdk.RegistryHasMsg{Category: "provider", Name: "openai"}, sdk.RegistryHasMsg{}, "", false, false},
		{"registry.list", sdk.RegistryListMsg{Category: "provider"}, sdk.RegistryListMsg{}, "", false, false},
		{"registry.resolve", sdk.RegistryResolveMsg{Category: "provider", Name: "ghost"}, sdk.RegistryResolveMsg{}, "", false, false},
		{"metrics.get", sdk.MetricsGetMsg{}, sdk.MetricsGetMsg{}, "", false, false},
		{"packages.search", sdk.PackagesSearchMsg{Query: "test"}, sdk.PackagesSearchMsg{}, "", false, false},
		{"packages.list", sdk.PackagesListMsg{}, sdk.PackagesListMsg{}, "", false, false},
		{"packages.info", sdk.PackagesInfoMsg{Name: "ghost"}, sdk.PackagesInfoMsg{Name: ""}, "", false, false},
		{"package.list", sdk.PackageListDeployedMsg{}, sdk.PackageListDeployedMsg{}, "", false, false},
		{"package.info", sdk.PackageDeployInfoMsg{Name: "ghost"}, sdk.PackageDeployInfoMsg{Name: ""}, "", false, false},
		{"package.teardown", sdk.PackageTeardownMsg{Name: "ghost"}, sdk.PackageTeardownMsg{Name: ""}, "", false, false},
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
			if cmd.rbacOnly {
				t.Skip("RBAC has been removed")
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
			if cmd.rbacOnly {
				t.Skip("RBAC has been removed")
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

				freshEnv.Kit.PublishRaw(ctx, topic, g)

				time.Sleep(50 * time.Millisecond)
				assert.True(t, testutil.Alive(t, freshEnv.Kit), "kernel died after garbage to %s", topic)
			})
		}
	}
}

