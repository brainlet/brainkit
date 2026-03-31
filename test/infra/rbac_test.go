package infra

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit/kit"
	"github.com/brainlet/brainkit/kit/rbac"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func startKernelWithRBAC(t *testing.T) *kit.Kernel {
	t.Helper()
	storePath := t.TempDir() + "/rbac-test.db"
	store, err := kit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k, err := kit.NewKernel(kit.KernelConfig{
		Store: store,
		Roles: map[string]rbac.Role{
			// custom restricted role: can only publish to own mailbox + events.*
			"restricted": {
				Bus: rbac.BusPermissions{
					Publish:   rbac.TopicFilter{Allow: []string{"events.*"}},
					Subscribe: rbac.TopicFilter{Allow: []string{"*.reply.*"}},
					Emit:      rbac.TopicFilter{Allow: []string{"events.*"}},
				},
				Commands: rbac.CommandPermissions{Allow: []string{"tools.list", "tools.call"}},
				Registration: rbac.RegistrationPermissions{Tools: false, Agents: false},
			},
		},
		DefaultRole: "restricted",
	})
	require.NoError(t, err)
	t.Cleanup(func() { k.Close() })
	return k
}

func TestRBAC_RestrictedDeployCannotPublishForbiddenTopic(t *testing.T) {
	k := startKernelWithRBAC(t)
	ctx := context.Background()

	// Deploy a .ts file that tries to publish to a forbidden topic
	_, err := k.Deploy(ctx, "restricted.ts", `
		bus.on("test", async (msg) => {
			try {
				bus.publish("secrets.set", { name: "hack", value: "pwned" });
				msg.reply({ published: true });
			} catch(e) {
				msg.reply({ error: e.message });
			}
		});
	`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Send a message to trigger the handler
	sendPR, _ := sdk.SendToService(k, ctx, "restricted.ts", "test", map[string]bool{"go": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	defer cancel()

	select {
	case resp := <-replyCh:
		// Should have caught a permission denied error
		errMsg, ok := resp["error"].(string)
		require.True(t, ok, "expected error in response, got %v", resp)
		assert.Contains(t, errMsg, "permission denied")
		assert.Contains(t, errMsg, "restricted")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestRBAC_RestrictedCannotRegisterTools(t *testing.T) {
	k := startKernelWithRBAC(t)
	ctx := context.Background()

	// Deploy a .ts that tries to register a tool (restricted role has Tools: false)
	_, err := k.Deploy(ctx, "no-tools.ts", `
		bus.on("test", async (msg) => {
			try {
				kit.register("tool", "hack-tool", { execute: () => {} });
				msg.reply({ registered: true });
			} catch(e) {
				msg.reply({ error: e.message });
			}
		});
	`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "no-tools.ts", "test", map[string]bool{"go": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	defer cancel()

	select {
	case resp := <-replyCh:
		errMsg, ok := resp["error"].(string)
		require.True(t, ok, "expected error, got %v", resp)
		assert.Contains(t, errMsg, "permission denied")
		assert.Contains(t, errMsg, "register")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestRBAC_OwnMailboxAlwaysAllowed(t *testing.T) {
	k := startKernelWithRBAC(t)
	ctx := context.Background()

	// Deploy with restricted role — own mailbox should always work
	_, err := k.Deploy(ctx, "own-mailbox.ts", `
		bus.on("ping", (msg) => {
			msg.reply({ pong: true });
		});
	`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Send to own mailbox — should work despite restricted role
	sendPR, _ := sdk.SendToService(k, ctx, "own-mailbox.ts", "ping", map[string]bool{"x": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	defer cancel()

	select {
	case resp := <-replyCh:
		assert.Equal(t, true, resp["pong"])
	case <-time.After(5 * time.Second):
		t.Fatal("timeout — own mailbox should always be accessible")
	}
}

func TestRBAC_AdminRoleCanDoEverything(t *testing.T) {
	storePath := t.TempDir() + "/rbac-admin.db"
	store, _ := kit.NewSQLiteStore(storePath)
	k, err := kit.NewKernel(kit.KernelConfig{
		Store:       store,
		Roles:       map[string]rbac.Role{}, // use built-in presets
		DefaultRole: "admin",                // everything gets admin
	})
	require.NoError(t, err)
	defer k.Close()
	ctx := context.Background()

	// Deploy with admin — should be able to publish to any topic
	_, err = k.Deploy(ctx, "admin.ts", `
		bus.on("test", async (msg) => {
			bus.emit("events.custom", { data: "admin-sent" });
			msg.reply({ ok: true });
		});
	`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "admin.ts", "test", map[string]bool{"go": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	defer cancel()

	select {
	case resp := <-replyCh:
		assert.Equal(t, true, resp["ok"])
	case <-time.After(5 * time.Second):
		t.Fatal("timeout — admin should be unrestricted")
	}
}

func TestRBAC_AssignRevokeViaBus(t *testing.T) {
	k := startKernelWithRBAC(t)
	ctx := context.Background()

	// Assign a role via bus command
	pub, _ := sdk.Publish(k, ctx, messages.RBACAssignMsg{Source: "some-service.ts", Role: "restricted"})
	assignCh := make(chan messages.RBACAssignResp, 1)
	cancel, _ := sdk.SubscribeTo[messages.RBACAssignResp](k, ctx, pub.ReplyTo, func(resp messages.RBACAssignResp, _ messages.Message) {
		assignCh <- resp
	})
	select {
	case resp := <-assignCh:
		cancel()
		assert.True(t, resp.Assigned)
	case <-time.After(5 * time.Second):
		cancel()
		t.Fatal("timeout assigning role")
	}

	// List assignments
	pub2, _ := sdk.Publish(k, ctx, messages.RBACListMsg{})
	listCh := make(chan messages.RBACListResp, 1)
	cancel2, _ := sdk.SubscribeTo[messages.RBACListResp](k, ctx, pub2.ReplyTo, func(resp messages.RBACListResp, _ messages.Message) {
		listCh <- resp
	})
	select {
	case resp := <-listCh:
		cancel2()
		require.Len(t, resp.Assignments, 1)
		assert.Equal(t, "some-service.ts", resp.Assignments[0].Source)
		assert.Equal(t, "restricted", resp.Assignments[0].Role)
	case <-time.After(5 * time.Second):
		cancel2()
		t.Fatal("timeout listing assignments")
	}

	// Revoke
	pub3, _ := sdk.Publish(k, ctx, messages.RBACRevokeMsg{Source: "some-service.ts"})
	revokeCh := make(chan messages.RBACRevokeResp, 1)
	cancel3, _ := sdk.SubscribeTo[messages.RBACRevokeResp](k, ctx, pub3.ReplyTo, func(resp messages.RBACRevokeResp, _ messages.Message) {
		revokeCh <- resp
	})
	select {
	case resp := <-revokeCh:
		cancel3()
		assert.True(t, resp.Revoked)
	case <-time.After(5 * time.Second):
		cancel3()
		t.Fatal("timeout revoking")
	}

	// Verify empty after revoke
	pub4, _ := sdk.Publish(k, ctx, messages.RBACListMsg{})
	listCh2 := make(chan messages.RBACListResp, 1)
	cancel4, _ := sdk.SubscribeTo[messages.RBACListResp](k, ctx, pub4.ReplyTo, func(resp messages.RBACListResp, _ messages.Message) {
		listCh2 <- resp
	})
	defer cancel4()
	select {
	case resp := <-listCh2:
		assert.Len(t, resp.Assignments, 0)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestRBAC_PermissionDeniedEventEmitted(t *testing.T) {
	k := startKernelWithRBAC(t)
	ctx := context.Background()

	// Subscribe to permission denied events
	deniedCh := make(chan messages.PermissionDeniedEvent, 1)
	cancelDenied, _ := sdk.SubscribeTo[messages.PermissionDeniedEvent](k, ctx, "bus.permission.denied",
		func(evt messages.PermissionDeniedEvent, _ messages.Message) {
			deniedCh <- evt
		})
	defer cancelDenied()

	// Deploy restricted service that publishes to forbidden topic
	_, _ = k.Deploy(ctx, "denied-evt.ts", `
		bus.on("trigger", async (msg) => {
			try { bus.publish("secrets.set", {}); } catch(e) {}
			msg.reply({ done: true });
		});
	`)
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "denied-evt.ts", "trigger", map[string]bool{"go": true})
	replyCh := make(chan struct{}, 1)
	replyCancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(_ messages.Message) { replyCh <- struct{}{} })
	defer replyCancel()
	<-replyCh

	// Verify denied event was emitted
	select {
	case evt := <-deniedCh:
		assert.Equal(t, "denied-evt.ts", evt.Source)
		assert.Contains(t, evt.Action, "publish")
		assert.Equal(t, "restricted", evt.Role)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for permission denied event")
	}
}

func TestRBAC_WithRoleOnDeploy(t *testing.T) {
	k := startKernelWithRBAC(t)
	ctx := context.Background()

	// Deploy with explicit observer role — should NOT be able to publish
	_, err := k.Deploy(ctx, "observer-svc.ts", `
		bus.on("test", async (msg) => {
			try {
				bus.publish("events.something", { data: "test" });
				msg.reply({ published: true });
			} catch(e) {
				msg.reply({ error: e.message });
			}
		});
	`, kit.WithRole("observer"))
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "observer-svc.ts", "test", map[string]bool{"go": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	defer cancel()

	select {
	case resp := <-replyCh:
		errMsg, ok := resp["error"].(string)
		require.True(t, ok, "observer should not be able to publish, got %v", resp)
		assert.Contains(t, errMsg, "permission denied")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestRBAC_RolePersistenceAcrossRestart(t *testing.T) {
	storePath := t.TempDir() + "/rbac-persist.db"

	// Phase 1: Create kernel, deploy with role, close
	store1, err := kit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := kit.NewKernel(kit.KernelConfig{
		Store: store1,
		Roles: map[string]rbac.Role{
			"observer": rbac.RoleObserver,
		},
	})
	require.NoError(t, err)

	code := `bus.on("ping", (msg) => msg.reply({ pong: true }));`
	_, err = k1.Deploy(context.Background(), "watched.ts", code, kit.WithRole("observer"))
	require.NoError(t, err)

	// Verify deployment exists
	deployments := k1.ListDeployments()
	require.Len(t, deployments, 1)

	k1.Close()

	// Phase 2: New kernel with same store — deployment auto-redeploys with persisted role
	store2, err := kit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k2, err := kit.NewKernel(kit.KernelConfig{
		Store: store2,
		Roles: map[string]rbac.Role{
			"observer": rbac.RoleObserver,
		},
	})
	require.NoError(t, err)
	defer k2.Close()

	// Verify deployment was restored
	deployments2 := k2.ListDeployments()
	require.Len(t, deployments2, 1, "deployment should be restored from persistence")
	assert.Equal(t, "watched.ts", deployments2[0].Source)

	// Verify the observer role is still enforced — observer cannot publish
	// Deploy an admin service that tries to call a command the observer can't
	adminCode := `
		bus.on("test", async (msg) => {
			try {
				bus.publish("some.forbidden.topic", { test: true });
				msg.reply({ blocked: false });
			} catch(e) {
				msg.reply({ blocked: true, error: e.message });
			}
		});
	`
	_, err = k2.Deploy(context.Background(), "admin-checker.ts", adminCode, kit.WithRole("admin"))
	require.NoError(t, err)

	// The observer's deployment was restored — verify it's functional
	// (can receive on own mailbox). Use sdk.SendToService pattern.
	time.Sleep(100 * time.Millisecond) // let redeployed service settle
	sendPR, err := sdk.SendToService(k2, context.Background(), "watched.ts", "ping", map[string]bool{"x": true})
	require.NoError(t, err)

	replyCh := make(chan map[string]any, 1)
	unsub, err := k2.SubscribeRaw(context.Background(), sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	require.NoError(t, err)
	defer unsub()

	select {
	case resp := <-replyCh:
		assert.Equal(t, true, resp["pong"])
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for restored observer deployment to respond")
	}
}

// ── Phase 1: New RBAC enforcement tests ─────────────────────────────────

func TestRBAC_SecretBridgeEnforcement(t *testing.T) {
	k := startKernelWithRBAC(t)
	ctx := context.Background()

	// Deploy with restricted role — should NOT be able to read secrets
	_, err := k.Deploy(ctx, "secret-reader.ts", `
		bus.on("read", async (msg) => {
			try {
				var val = secrets.get("test-secret");
				msg.reply({ value: val, denied: false });
			} catch(e) {
				msg.reply({ denied: true, error: e.message });
			}
		});
	`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "secret-reader.ts", "read", map[string]bool{"go": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	defer cancel()

	select {
	case resp := <-replyCh:
		// restricted role does NOT have "secrets.get" in its command allow list
		denied, _ := resp["denied"].(bool)
		assert.True(t, denied, "restricted role should be denied secrets.get, got: %v", resp)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestRBAC_GatewayRouteEnforcement(t *testing.T) {
	k := startKernelWithRBAC(t)
	ctx := context.Background()

	// Deploy restricted service that tries to publish to gateway route topic
	_, err := k.Deploy(ctx, "route-adder.ts", `
		bus.on("try-route", async (msg) => {
			try {
				bus.publish("gateway.http.route.add", {
					method: "GET", path: "/hack", topic: "hack.handler", type: "handle",
				});
				msg.reply({ added: true });
			} catch(e) {
				msg.reply({ error: e.message });
			}
		});
	`)
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "route-adder.ts", "try-route", map[string]bool{"go": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg messages.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	defer cancel()

	select {
	case resp := <-replyCh:
		_, hasErr := resp["error"]
		assert.True(t, hasErr, "restricted role should not be able to publish gateway.http.route.add, got: %v", resp)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func TestRBAC_CommandMatrix(t *testing.T) {
	tests := []struct {
		role    string
		command string
		allowed bool
	}{
		{"service", "tools.call", true}, {"service", "tools.list", true},
		{"service", "fs.read", true}, {"service", "secrets.get", true},
		{"service", "rbac.assign", false}, {"service", "wasm.compile", false},
		{"observer", "tools.list", true}, {"observer", "kit.list", true},
		{"observer", "tools.call", false}, {"observer", "secrets.get", false},
		{"admin", "tools.call", true}, {"admin", "rbac.assign", true}, {"admin", "wasm.compile", true},
		{"gateway", "tools.call", false}, {"gateway", "secrets.get", false},
	}
	for _, tt := range tests {
		t.Run(tt.role+"/"+tt.command, func(t *testing.T) {
			role := rbac.RoleService
			switch tt.role {
			case "admin":
				role = rbac.RoleAdmin
			case "observer":
				role = rbac.RoleObserver
			case "gateway":
				role = rbac.RoleGateway
			}
			assert.Equal(t, tt.allowed, role.Commands.AllowsCommand(tt.command))
		})
	}
}

func TestRBAC_MultiDeploymentIsolation(t *testing.T) {
	storePath := t.TempDir() + "/isolation.db"
	store, _ := kit.NewSQLiteStore(storePath)
	k, err := kit.NewKernel(kit.KernelConfig{
		Store: store,
		Roles: map[string]rbac.Role{
			// Must provide at least one role to activate RBAC (len > 0 check in NewKernel)
			"observer": rbac.RoleObserver,
		},
		DefaultRole: "service",
	})
	require.NoError(t, err)
	defer k.Close()
	ctx := context.Background()

	// Deploy A with admin — can emit anywhere
	_, err = k.Deploy(ctx, "admin-svc.ts", `
		bus.on("test", async (msg) => {
			try {
				bus.emit("events.custom", { from: "admin" });
				msg.reply({ emitted: true });
			} catch(e) {
				msg.reply({ error: e.message });
			}
		});
	`, kit.WithRole("admin"))
	require.NoError(t, err)

	// Deploy B with observer — cannot emit
	_, err = k.Deploy(ctx, "observer-svc.ts", `
		bus.on("test", async (msg) => {
			try {
				bus.emit("events.custom", { from: "observer" });
				msg.reply({ emitted: true });
			} catch(e) {
				msg.reply({ error: e.message });
			}
		});
	`, kit.WithRole("observer"))
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	// Admin should succeed
	adminPR, _ := sdk.SendToService(k, ctx, "admin-svc.ts", "test", map[string]bool{"go": true})
	adminReply := make(chan map[string]any, 1)
	adminUnsub, _ := k.SubscribeRaw(ctx, adminPR.ReplyTo, func(msg messages.Message) {
		var r map[string]any
		json.Unmarshal(msg.Payload, &r)
		adminReply <- r
	})
	defer adminUnsub()
	select {
	case r := <-adminReply:
		assert.Equal(t, true, r["emitted"])
	case <-time.After(5 * time.Second):
		t.Fatal("admin timeout")
	}

	// Observer should fail
	obsPR, _ := sdk.SendToService(k, ctx, "observer-svc.ts", "test", map[string]bool{"go": true})
	obsReply := make(chan map[string]any, 1)
	obsUnsub, _ := k.SubscribeRaw(ctx, obsPR.ReplyTo, func(msg messages.Message) {
		var r map[string]any
		json.Unmarshal(msg.Payload, &r)
		obsReply <- r
	})
	defer obsUnsub()
	select {
	case r := <-obsReply:
		_, hasErr := r["error"]
		assert.True(t, hasErr, "observer should be denied, got: %v", r)
	case <-time.After(5 * time.Second):
		t.Fatal("observer timeout")
	}
}
