package adversarial_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/rbac"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// === RBAC Command Permission Matrix ===
// Tests role.Commands.AllowsCommand() for every command × every role.
// This is a unit test on the matcher — fast, no Kernel needed.

func TestRBACMatrix_CommandPermissions(t *testing.T) {
	commands := []string{
		"tools.call", "tools.list", "tools.resolve",
		"agents.list", "agents.discover", "agents.get-status", "agents.set-status",
		"kit.deploy", "kit.teardown", "kit.list", "kit.redeploy",
		"secrets.set", "secrets.get", "secrets.delete", "secrets.list", "secrets.rotate",
		"registry.has", "registry.list", "registry.resolve",
		"metrics.get",
		"rbac.assign", "rbac.revoke", "rbac.list", "rbac.roles",
		"packages.search", "packages.list", "packages.info", "packages.install", "packages.remove",
		"mcp.listTools", "mcp.callTool",
		"trace.get", "trace.list",
		"wasm.compile", "wasm.run", "wasm.deploy", "wasm.undeploy", "wasm.list", "wasm.get", "wasm.remove",
		"test.run",
		"workflow.run", "workflow.status", "workflow.cancel", "workflow.list", "workflow.history",
	}

	roles := []struct {
		name string
		role rbac.Role
	}{
		{"admin", rbac.RoleAdmin},
		{"service", rbac.RoleService},
		{"gateway", rbac.RoleGateway},
		{"observer", rbac.RoleObserver},
	}

	// Expected: admin allows everything
	t.Run("admin_allows_all", func(t *testing.T) {
		for _, cmd := range commands {
			assert.True(t, rbac.RoleAdmin.Commands.AllowsCommand(cmd), "admin should allow %s", cmd)
		}
	})

	// Service: explicit allowlist
	serviceAllowed := map[string]bool{
		"tools.call": true, "tools.list": true, "tools.resolve": true,
		"secrets.get": true,
	}
	t.Run("service", func(t *testing.T) {
		for _, cmd := range commands {
			expected := serviceAllowed[cmd]
			actual := rbac.RoleService.Commands.AllowsCommand(cmd)
			assert.Equal(t, expected, actual, "service/%s: expected %v got %v", cmd, expected, actual)
		}
	})

	// Gateway: nothing allowed
	t.Run("gateway_denies_all", func(t *testing.T) {
		for _, cmd := range commands {
			assert.False(t, rbac.RoleGateway.Commands.AllowsCommand(cmd), "gateway should deny %s", cmd)
		}
	})

	// Observer: specific allowlist
	observerAllowed := map[string]bool{
		"tools.list": true, "kit.list": true, "registry.list": true, "registry.has": true,
	}
	t.Run("observer", func(t *testing.T) {
		for _, cmd := range commands {
			expected := observerAllowed[cmd]
			actual := rbac.RoleObserver.Commands.AllowsCommand(cmd)
			assert.Equal(t, expected, actual, "observer/%s: expected %v got %v", cmd, expected, actual)
		}
	})

	// Cross-check: every command × every role in table form
	for _, role := range roles {
		for _, cmd := range commands {
			t.Run(fmt.Sprintf("%s/%s", role.name, cmd), func(t *testing.T) {
				// Just verify it doesn't panic — the specific allow/deny is tested above
				_ = role.role.Commands.AllowsCommand(cmd)
			})
		}
	}
}

// === RBAC Bus Permission Matrix ===
// Tests topic filter matching for publish/subscribe/emit.

func TestRBACMatrix_BusPublish(t *testing.T) {
	topics := []string{
		"incoming.test", "incoming.user.msg",
		"events.test", "events.deploy",
		"gateway.http.request", "gateway.ws.connect",
		"random.unknown.topic",
		"ts.my-agent.ask", // own mailbox handled separately
	}

	type expectation struct {
		admin, service, gateway, observer bool
	}
	expected := map[string]expectation{
		"incoming.test":         {true, true, true, false},
		"incoming.user.msg":     {true, true, true, false},
		"events.test":           {true, true, false, false},
		"events.deploy":         {true, true, false, false},
		"gateway.http.request":  {true, false, true, false},
		"gateway.ws.connect":    {true, false, true, false},
		"random.unknown.topic":  {true, false, false, false},
		"ts.my-agent.ask":       {true, false, false, false}, // not own mailbox in this test
	}

	for _, topic := range topics {
		exp := expected[topic]
		t.Run("admin/"+topic, func(t *testing.T) {
			assert.Equal(t, exp.admin, rbac.RoleAdmin.Bus.Publish.Allows(topic))
		})
		t.Run("service/"+topic, func(t *testing.T) {
			assert.Equal(t, exp.service, rbac.RoleService.Bus.Publish.Allows(topic))
		})
		t.Run("gateway/"+topic, func(t *testing.T) {
			assert.Equal(t, exp.gateway, rbac.RoleGateway.Bus.Publish.Allows(topic))
		})
		t.Run("observer/"+topic, func(t *testing.T) {
			assert.Equal(t, exp.observer, rbac.RoleObserver.Bus.Publish.Allows(topic))
		})
	}
}

func TestRBACMatrix_BusSubscribe(t *testing.T) {
	topics := []string{
		"tools.call.reply.abc123",
		"events.test",
		"incoming.user.msg",
		"random.unknown.topic",
	}

	type expectation struct {
		admin, service, gateway, observer bool
	}
	// FIXED (bug #2): *.reply.* now matches — service/gateway CAN subscribe to reply topics.
	expected := map[string]expectation{
		"tools.call.reply.abc123": {true, true, true, true},   // *.reply.* now works for service/gateway
		"events.test":             {true, false, false, true},  // observer=* matches
		"incoming.user.msg":       {true, false, false, true},  // observer=* matches
		"random.unknown.topic":    {true, false, false, true},  // observer=* matches
	}

	for _, topic := range topics {
		exp := expected[topic]
		t.Run("admin/"+topic, func(t *testing.T) {
			assert.Equal(t, exp.admin, rbac.RoleAdmin.Bus.Subscribe.Allows(topic))
		})
		t.Run("service/"+topic, func(t *testing.T) {
			assert.Equal(t, exp.service, rbac.RoleService.Bus.Subscribe.Allows(topic))
		})
		t.Run("gateway/"+topic, func(t *testing.T) {
			assert.Equal(t, exp.gateway, rbac.RoleGateway.Bus.Subscribe.Allows(topic))
		})
		t.Run("observer/"+topic, func(t *testing.T) {
			assert.Equal(t, exp.observer, rbac.RoleObserver.Bus.Subscribe.Allows(topic))
		})
	}
}

func TestRBACMatrix_BusEmit(t *testing.T) {
	topics := []string{
		"events.test", "events.deploy",
		"gateway.http.request",
		"random.topic",
	}

	type expectation struct {
		admin, service, gateway, observer bool
	}
	expected := map[string]expectation{
		"events.test":          {true, true, false, false},
		"events.deploy":        {true, true, false, false},
		"gateway.http.request": {true, false, true, false},
		"random.topic":         {true, false, false, false},
	}

	for _, topic := range topics {
		exp := expected[topic]
		t.Run("admin/"+topic, func(t *testing.T) {
			assert.Equal(t, exp.admin, rbac.RoleAdmin.Bus.Emit.Allows(topic))
		})
		t.Run("service/"+topic, func(t *testing.T) {
			assert.Equal(t, exp.service, rbac.RoleService.Bus.Emit.Allows(topic))
		})
		t.Run("gateway/"+topic, func(t *testing.T) {
			assert.Equal(t, exp.gateway, rbac.RoleGateway.Bus.Emit.Allows(topic))
		})
		t.Run("observer/"+topic, func(t *testing.T) {
			assert.Equal(t, exp.observer, rbac.RoleObserver.Bus.Emit.Allows(topic))
		})
	}
}

// === RBAC Registration Matrix ===

func TestRBACMatrix_Registration(t *testing.T) {
	t.Run("admin/tools", func(t *testing.T) { assert.True(t, rbac.RoleAdmin.Registration.Tools) })
	t.Run("admin/agents", func(t *testing.T) { assert.True(t, rbac.RoleAdmin.Registration.Agents) })
	t.Run("service/tools", func(t *testing.T) { assert.True(t, rbac.RoleService.Registration.Tools) })
	t.Run("service/agents", func(t *testing.T) { assert.False(t, rbac.RoleService.Registration.Agents) })
	t.Run("gateway/tools", func(t *testing.T) { assert.False(t, rbac.RoleGateway.Registration.Tools) })
	t.Run("gateway/agents", func(t *testing.T) { assert.False(t, rbac.RoleGateway.Registration.Agents) })
	t.Run("observer/tools", func(t *testing.T) { assert.False(t, rbac.RoleObserver.Registration.Tools) })
	t.Run("observer/agents", func(t *testing.T) { assert.False(t, rbac.RoleObserver.Registration.Agents) })
}

// === RBAC Own Mailbox ===

func TestRBACMatrix_OwnMailbox(t *testing.T) {
	cases := []struct {
		source string
		topic  string
		expect bool
	}{
		{"my-agent.ts", "ts.my-agent.ask", true},
		{"my-agent.ts", "ts.my-agent.reply.abc", true},
		{"nested/path/svc.ts", "ts.nested.path.svc.ask", true}, // slashes→dots
		{"gateway.ts", "ts.gateway.status", true},
		{"agent-a.ts", "ts.agent-b.ask", false},                // different source
		{"", "ts.anything.ask", false},                          // empty source
		{"my-agent.ts", "events.something", false},              // not own mailbox
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("%s→%s", tc.source, tc.topic), func(t *testing.T) {
			assert.Equal(t, tc.expect, rbac.IsOwnMailbox(tc.source, tc.topic))
		})
	}

	// Non-matching
	assert.False(t, rbac.IsOwnMailbox("agent-a.ts", "ts.agent-b.ask"), "different source should not match")
	assert.False(t, rbac.IsOwnMailbox("", "ts.anything.ask"), "empty source never matches")
}

// === RBAC Integration: Deploy with Role, Verify Enforcement ===
// These test that RBAC actually enforces at the bridge level (not just matcher unit tests).

func TestRBACMatrix_Integration_ObserverDeniedPublish(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles:       map[string]rbac.Role{"observer": rbac.RoleObserver},
		DefaultRole: "observer",
		Storages:    map[string]brainkit.StorageConfig{"default": brainkit.InMemoryStorage()},
	})
	require.NoError(t, err)
	defer k.Close()

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(k, "echo", registry.TypedTool[echoIn]{
		Description: "echoes",
		Execute:     func(ctx context.Context, in echoIn) (any, error) { return map[string]string{"echoed": in.Message}, nil },
	})

	ctx := context.Background()

	// Observer deploys .ts that tries bus.publish to forbidden topic
	_, err = k.Deploy(ctx, "obs-pub.ts", `
		var caught = "ALLOWED";
		try { bus.publish("forbidden.topic", {}); }
		catch(e) { caught = "DENIED"; }
		output(caught);
	`, brainkit.WithRole("observer"))
	require.NoError(t, err)
	defer k.Teardown(ctx, "obs-pub.ts")

	result, _ := k.EvalTS(ctx, "__obs_result.ts", `return String(globalThis.__module_result || "");`)
	assert.Equal(t, "DENIED", result)
}

func TestRBACMatrix_Integration_ServiceAllowedToolCall(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles:       map[string]rbac.Role{"service": rbac.RoleService},
		DefaultRole: "service",
		Storages:    map[string]brainkit.StorageConfig{"default": brainkit.InMemoryStorage()},
	})
	require.NoError(t, err)
	defer k.Close()

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(k, "echo", registry.TypedTool[echoIn]{
		Description: "echoes",
		Execute:     func(ctx context.Context, in echoIn) (any, error) { return map[string]string{"echoed": in.Message}, nil },
	})

	ctx := context.Background()

	// Service deploys .ts that calls tools.call via the tools endowment
	_, err = k.Deploy(ctx, "svc-tool.ts", `
		var caught = "ALLOWED";
		try { await tools.call("echo", {message: "from service"}); }
		catch(e) { caught = "DENIED:" + (e.message || ""); }
		output(caught);
	`, brainkit.WithRole("service"))
	require.NoError(t, err)
	defer k.Teardown(ctx, "svc-tool.ts")

	result, _ := k.EvalTS(ctx, "__svc_result.ts", `return String(globalThis.__module_result || "");`)
	assert.Equal(t, "ALLOWED", result)
}

func TestRBACMatrix_Integration_GatewayDeniedEverything(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles:       map[string]rbac.Role{"gateway": rbac.RoleGateway},
		DefaultRole: "gateway",
		Storages:    map[string]brainkit.StorageConfig{"default": brainkit.InMemoryStorage()},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	// Gateway deploys .ts — can only publish to incoming.*/gateway.* topics
	_, err = k.Deploy(ctx, "gw-test.ts", `
		var results = {};

		// Should be allowed: gateway.* publish
		try { bus.publish("gateway.test", {}); results.gwPub = "ALLOWED"; }
		catch(e) { results.gwPub = "DENIED"; }

		// Should be allowed: incoming.* publish
		try { bus.publish("incoming.test", {}); results.incPub = "ALLOWED"; }
		catch(e) { results.incPub = "DENIED"; }

		// Should be denied: random topic publish
		try { bus.publish("random.topic", {}); results.randPub = "ALLOWED"; }
		catch(e) { results.randPub = "DENIED"; }

		// Should be denied: events.* emit (gateway can't emit events.*)
		try { bus.emit("events.test", {}); results.evtEmit = "ALLOWED"; }
		catch(e) { results.evtEmit = "DENIED"; }

		// Should be allowed: gateway.* emit
		try { bus.emit("gateway.status", {}); results.gwEmit = "ALLOWED"; }
		catch(e) { results.gwEmit = "DENIED"; }

		output(results);
	`, brainkit.WithRole("gateway"))
	require.NoError(t, err)
	defer k.Teardown(ctx, "gw-test.ts")

	result, _ := k.EvalTS(ctx, "__gw_result.ts", `
		var r = globalThis.__module_result;
		if (typeof r === "string") return r;
		return JSON.stringify(r || {});
	`)

	assert.Contains(t, result, `"gwPub":"ALLOWED"`)
	assert.Contains(t, result, `"incPub":"ALLOWED"`)
	assert.Contains(t, result, `"randPub":"DENIED"`)
	assert.Contains(t, result, `"evtEmit":"DENIED"`)
	assert.Contains(t, result, `"gwEmit":"ALLOWED"`)
}
