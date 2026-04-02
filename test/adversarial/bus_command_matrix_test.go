package adversarial_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cmdTest defines one bus command to exercise in the matrix.
type cmdTest struct {
	topic    string
	valid    messages.BrainkitMessage // valid input
	empty    messages.BrainkitMessage // empty/missing fields
	errCode  string                   // expected code on empty input ("" = no error expected)
	rbacOnly bool                     // needs RBAC-enabled kernel
	nodeOnly bool                     // needs Node, not standalone Kernel
}

func commandTable() []cmdTest {
	return []cmdTest{
		// Tools
		{"tools.call", messages.ToolCallMsg{Name: "echo", Input: map[string]any{"message": "test"}}, messages.ToolCallMsg{Name: ""}, "NOT_FOUND", false, false},
		{"tools.list", messages.ToolListMsg{}, messages.ToolListMsg{}, "", false, false},
		{"tools.resolve", messages.ToolResolveMsg{Name: "echo"}, messages.ToolResolveMsg{Name: "ghost-tool-xyz"}, "NOT_FOUND", false, false},

		// Agents
		{"agents.list", messages.AgentListMsg{}, messages.AgentListMsg{}, "", false, false},
		{"agents.get-status", messages.AgentGetStatusMsg{Name: "ghost"}, messages.AgentGetStatusMsg{Name: ""}, "VALIDATION_ERROR", false, false},
		{"agents.set-status", messages.AgentSetStatusMsg{Name: "ghost", Status: "idle"}, messages.AgentSetStatusMsg{Name: "", Status: ""}, "VALIDATION_ERROR", false, false},
		{"agents.discover", messages.AgentDiscoverMsg{}, messages.AgentDiscoverMsg{}, "", false, false},

		// Filesystem — removed (fs is now a jsbridge polyfill, not bus commands)

		// Kit lifecycle
		{"kit.list", messages.KitListMsg{}, messages.KitListMsg{}, "", false, false},
		{"kit.teardown", messages.KitTeardownMsg{Source: "ghost.ts"}, messages.KitTeardownMsg{Source: ""}, "", false, false},

		// Secrets
		{"secrets.set", messages.SecretsSetMsg{Name: "matrix-k", Value: "v"}, messages.SecretsSetMsg{Name: "", Value: "v"}, "VALIDATION_ERROR", false, false},
		{"secrets.get", messages.SecretsGetMsg{Name: "matrix-k"}, messages.SecretsGetMsg{Name: ""}, "VALIDATION_ERROR", false, false},
		{"secrets.delete", messages.SecretsDeleteMsg{Name: "ghost"}, messages.SecretsDeleteMsg{Name: ""}, "VALIDATION_ERROR", false, false},
		{"secrets.list", messages.SecretsListMsg{}, messages.SecretsListMsg{}, "", false, false},
		{"secrets.rotate", messages.SecretsRotateMsg{Name: "matrix-k", NewValue: "v2"}, messages.SecretsRotateMsg{Name: ""}, "VALIDATION_ERROR", false, false},

		// Registry
		{"registry.has", messages.RegistryHasMsg{Category: "provider", Name: "openai"}, messages.RegistryHasMsg{}, "", false, false},
		{"registry.list", messages.RegistryListMsg{Category: "provider"}, messages.RegistryListMsg{}, "", false, false},
		{"registry.resolve", messages.RegistryResolveMsg{Category: "provider", Name: "ghost"}, messages.RegistryResolveMsg{}, "", false, false},

		// Metrics
		{"metrics.get", messages.MetricsGetMsg{}, messages.MetricsGetMsg{}, "", false, false},

		// RBAC (needs RBAC kernel)
		{"rbac.assign", messages.RBACAssignMsg{Source: "test.ts", Role: "admin"}, messages.RBACAssignMsg{Source: "", Role: "admin"}, "VALIDATION_ERROR", true, false},
		{"rbac.revoke", messages.RBACRevokeMsg{Source: "test.ts"}, messages.RBACRevokeMsg{}, "", true, false},
		{"rbac.list", messages.RBACListMsg{}, messages.RBACListMsg{}, "", true, false},
		{"rbac.roles", messages.RBACRolesMsg{}, messages.RBACRolesMsg{}, "", true, false},

		// Packages
		{"packages.search", messages.PackagesSearchMsg{Query: "test"}, messages.PackagesSearchMsg{}, "", false, false},
		{"packages.list", messages.PackagesListMsg{}, messages.PackagesListMsg{}, "", false, false},
		{"packages.info", messages.PackagesInfoMsg{Name: "ghost"}, messages.PackagesInfoMsg{Name: ""}, "", false, false},

		// WASM
		{"wasm.list", messages.WasmListMsg{}, messages.WasmListMsg{}, "", false, false},
		{"wasm.get", messages.WasmGetMsg{Name: "ghost-module"}, messages.WasmGetMsg{Name: ""}, "", false, false},
		{"wasm.remove", messages.WasmRemoveMsg{Name: "ghost-module"}, messages.WasmRemoveMsg{Name: ""}, "", false, false},
		{"wasm.deploy", messages.WasmDeployMsg{Name: "ghost-module"}, messages.WasmDeployMsg{Name: ""}, "", false, false},
		{"wasm.undeploy", messages.WasmUndeployMsg{Name: "ghost-module"}, messages.WasmUndeployMsg{Name: ""}, "", false, false},
		{"wasm.describe", messages.WasmDescribeMsg{Name: "ghost-module"}, messages.WasmDescribeMsg{Name: ""}, "", false, false},

		// Workflow
		{"workflow.list", messages.WorkflowListMsg{}, messages.WorkflowListMsg{}, "", false, false},
		{"workflow.status", messages.WorkflowStatusMsg{RunID: "ghost-run"}, messages.WorkflowStatusMsg{RunID: ""}, "", false, false},
		{"workflow.cancel", messages.WorkflowCancelMsg{RunID: "ghost-run"}, messages.WorkflowCancelMsg{RunID: ""}, "", false, false},
		{"workflow.history", messages.WorkflowHistoryMsg{RunID: "ghost-run"}, messages.WorkflowHistoryMsg{RunID: ""}, "", false, false},

		// Package Deploy
		{"package.list", messages.PackageListDeployedMsg{}, messages.PackageListDeployedMsg{}, "", false, false},
		{"package.info", messages.PackageDeployInfoMsg{Name: "ghost"}, messages.PackageDeployInfoMsg{Name: ""}, "", false, false},
		{"package.teardown", messages.PackageTeardownMsg{Name: "ghost"}, messages.PackageTeardownMsg{Name: ""}, "", false, false},

		// Automation
		{"automation.list", messages.AutomationListMsg{}, messages.AutomationListMsg{}, "", false, false},
		{"automation.info", messages.AutomationInfoMsg{Name: "ghost"}, messages.AutomationInfoMsg{Name: ""}, "", false, false},
	}
}

// TestBusMatrix_ValidInput — every command with valid input gets a response (no hang, no panic).
func TestBusMatrix_ValidInput(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	tkRBAC := testutil.NewTestKernelWithRBAC(t, map[string]rbac.Role{"admin": rbac.RoleAdmin}, "admin")

	for _, cmd := range commandTable() {
		t.Run(cmd.topic, func(t *testing.T) {
			if cmd.nodeOnly {
				t.Skip("node-only")
				return
			}
			rt := sdk.Runtime(tk)
			if cmd.rbacOnly {
				rt = tkRBAC
			}

			payload, ok := sendAndReceive(t, rt, cmd.valid, 5*time.Second)
			if !ok {
				t.Fatalf("timeout — %s hung on valid input", cmd.topic)
			}
			// Got a response. May be success or expected error (e.g. fs.read nonexistent).
			// The key: no hang, no panic.
			_ = payload
		})
	}
}

// TestBusMatrix_EmptyInput — every command with empty input returns clean error or empty success.
func TestBusMatrix_EmptyInput(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	tkRBAC := testutil.NewTestKernelWithRBAC(t, map[string]rbac.Role{"admin": rbac.RoleAdmin}, "admin")

	for _, cmd := range commandTable() {
		if cmd.errCode == "" {
			continue // no error expected on empty input
		}
		t.Run(cmd.topic, func(t *testing.T) {
			if cmd.nodeOnly {
				t.Skip("node-only")
				return
			}
			rt := sdk.Runtime(tk)
			if cmd.rbacOnly {
				rt = tkRBAC
			}

			payload, ok := sendAndReceive(t, rt, cmd.empty, 5*time.Second)
			if !ok {
				t.Fatalf("timeout — %s hung on empty input", cmd.topic)
			}
			code := responseCode(payload)
			assert.Equal(t, cmd.errCode, code, "%s: wrong error code on empty input (payload: %s)", cmd.topic, string(payload))
		})
	}
}

// TestBusMatrix_GarbagePayload — every command gets raw garbage JSON and doesn't panic.
func TestBusMatrix_GarbagePayload(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)

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
		"wasm.list", "wasm.get", "wasm.remove", "wasm.deploy", "wasm.undeploy", "wasm.describe",
		"workflow.list", "workflow.status", "workflow.cancel", "workflow.history",
		"package.list", "package.info", "package.teardown",
		"automation.list", "automation.info",
	}

	for _, topic := range topics {
		for i, g := range garbage {
			t.Run(fmt.Sprintf("%s/garbage_%d", topic, i), func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				// Publish garbage directly via PublishRaw
				tk.PublishRaw(ctx, topic, g)

				// Kernel must still be alive after processing garbage
				time.Sleep(50 * time.Millisecond)
				assert.True(t, tk.Alive(ctx), "kernel died after garbage to %s", topic)
			})
		}
	}
}

// TestBusMatrix_RBAC_DeniedFromTS — observer role .ts deployment tries bus operations it shouldn't access.
// NOTE: Compartments don't have __go_brainkit_request. They use bus.publish/emit which ARE endowments.
// RBAC on bus operations is enforced at the bridge level (checkBusPermission).
func TestBusMatrix_RBAC_DeniedFromTS(t *testing.T) {
	tkRBAC := testutil.NewTestKernelWithRBAC(t, map[string]rbac.Role{
		"observer": rbac.RoleObserver,
	}, "observer")

	ctx := context.Background()

	// Observer cannot publish to arbitrary topics (only subscribe)
	t.Run("bus.publish/denied", func(t *testing.T) {
		_, err := tkRBAC.Deploy(ctx, "rbac-bus-deny.ts", `
			var caught = "none";
			try { bus.publish("forbidden.topic", {}); }
			catch(e) { caught = "DENIED:" + (e.message || ""); }
			output(caught);
		`, brainkit.WithRole("observer"))
		require.NoError(t, err)
		defer tkRBAC.Teardown(ctx, "rbac-bus-deny.ts")

		result, _ := tkRBAC.EvalTS(ctx, "__rbac_bus_result.ts", `return String(globalThis.__module_result || "");`)
		assert.Contains(t, result, "DENIED", "observer should be denied bus.publish to forbidden topic")
	})

	// Observer CAN subscribe (observer role allows subscribe to *)
	t.Run("bus.subscribe/allowed", func(t *testing.T) {
		_, err := tkRBAC.Deploy(ctx, "rbac-bus-allow.ts", `
			var caught = "none";
			try {
				var subId = bus.subscribe("events.anything", function() {});
				bus.unsubscribe(subId);
				caught = "ALLOWED";
			} catch(e) { caught = "DENIED:" + (e.message || ""); }
			output(caught);
		`, brainkit.WithRole("observer"))
		require.NoError(t, err)
		defer tkRBAC.Teardown(ctx, "rbac-bus-allow.ts")

		result, _ := tkRBAC.EvalTS(ctx, "__rbac_sub_result.ts", `return String(globalThis.__module_result || "");`)
		assert.Equal(t, "ALLOWED", result, "observer should be allowed bus.subscribe")
	})

	// RBAC denial via Go SDK (Go surface works correctly)
	t.Run("rbac/go-sdk", func(t *testing.T) {
		// Deploy a .ts with observer role, then try to call tools.call via bus
		// The Go SDK publish itself isn't RBAC-checked (it's the Go developer's code),
		// but the command handler checks RBAC on the callerID.
		// This is tested in TestRBAC_CommandMatrix in test/infra already.
	})
}
