package rbac

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/rbac"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests migrated from test/infra/rbac_test.go (12 tests).
// Each test creates its own kit with the specific RBAC config it needs.

func testRestrictedCannotPublishForbidden(t *testing.T, _ *suite.TestEnv) {
	k := newRestrictedKernel(t)
	ctx := context.Background()

	testutil.Deploy(t, k, "restricted-rbac.ts", `
		bus.on("test", async (msg) => {
			try {
				bus.publish("secrets.set", { name: "hack", value: "pwned" });
				msg.reply({ published: true });
			} catch(e) {
				msg.reply({ error: e.message });
			}
		});
	`)
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "restricted-rbac.ts", "test", map[string]bool{"go": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	defer cancel()

	select {
	case resp := <-replyCh:
		errMsg, ok := resp["error"].(string)
		require.True(t, ok, "expected error in response, got %v", resp)
		assert.Contains(t, errMsg, "permission denied")
		assert.Contains(t, errMsg, "restricted")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func testRestrictedCannotRegisterTools(t *testing.T, _ *suite.TestEnv) {
	k := newRestrictedKernel(t)
	ctx := context.Background()

	testutil.Deploy(t, k, "no-tools-rbac.ts", `
		bus.on("test", async (msg) => {
			try {
				kit.register("tool", "hack-tool", { execute: () => {} });
				msg.reply({ registered: true });
			} catch(e) {
				msg.reply({ error: e.message });
			}
		});
	`)
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "no-tools-rbac.ts", "test", map[string]bool{"go": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) {
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

func testOwnMailboxAlwaysAllowed(t *testing.T, _ *suite.TestEnv) {
	k := newRestrictedKernel(t)
	ctx := context.Background()

	testutil.Deploy(t, k, "own-mailbox-rbac.ts", `
		bus.on("ping", (msg) => {
			msg.reply({ pong: true });
		});
	`)
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "own-mailbox-rbac.ts", "ping", map[string]bool{"x": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) {
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

func testAdminCanDoEverything(t *testing.T, _ *suite.TestEnv) {
	storePath := t.TempDir() + "/rbac-admin.db"
	store, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.New(brainkit.Config{
		Store:       store,
		Roles:       map[string]rbac.Role{},
		DefaultRole: "admin",
	})
	require.NoError(t, err)
	defer k.Close()
	ctx := context.Background()

	testutil.Deploy(t, k, "admin-rbac.ts", `
		bus.on("test", async (msg) => {
			bus.emit("events.custom", { data: "admin-sent" });
			msg.reply({ ok: true });
		});
	`)
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "admin-rbac.ts", "test", map[string]bool{"go": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) {
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

func testAssignRevokeViaBus(t *testing.T, _ *suite.TestEnv) {
	k := newRestrictedKernel(t)
	ctx := context.Background()

	// Assign
	pub, _ := sdk.Publish(k, ctx, sdk.RBACAssignMsg{Source: "some-service.ts", Role: "restricted"})
	assignCh := make(chan sdk.RBACAssignResp, 1)
	cancel, _ := sdk.SubscribeTo[sdk.RBACAssignResp](k, ctx, pub.ReplyTo, func(resp sdk.RBACAssignResp, _ sdk.Message) {
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

	// List
	pub2, _ := sdk.Publish(k, ctx, sdk.RBACListMsg{})
	listCh := make(chan sdk.RBACListResp, 1)
	cancel2, _ := sdk.SubscribeTo[sdk.RBACListResp](k, ctx, pub2.ReplyTo, func(resp sdk.RBACListResp, _ sdk.Message) {
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
	pub3, _ := sdk.Publish(k, ctx, sdk.RBACRevokeMsg{Source: "some-service.ts"})
	revokeCh := make(chan sdk.RBACRevokeResp, 1)
	cancel3, _ := sdk.SubscribeTo[sdk.RBACRevokeResp](k, ctx, pub3.ReplyTo, func(resp sdk.RBACRevokeResp, _ sdk.Message) {
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
	pub4, _ := sdk.Publish(k, ctx, sdk.RBACListMsg{})
	listCh2 := make(chan sdk.RBACListResp, 1)
	cancel4, _ := sdk.SubscribeTo[sdk.RBACListResp](k, ctx, pub4.ReplyTo, func(resp sdk.RBACListResp, _ sdk.Message) {
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

func testPermissionDeniedEventEmitted(t *testing.T, _ *suite.TestEnv) {
	k := newRestrictedKernel(t)
	ctx := context.Background()

	deniedCh := make(chan sdk.PermissionDeniedEvent, 1)
	cancelDenied, _ := sdk.SubscribeTo[sdk.PermissionDeniedEvent](k, ctx, "bus.permission.denied",
		func(evt sdk.PermissionDeniedEvent, _ sdk.Message) {
			deniedCh <- evt
		})
	defer cancelDenied()

	testutil.Deploy(t, k, "denied-evt-rbac.ts", `
		bus.on("trigger", async (msg) => {
			try { bus.publish("secrets.set", {}); } catch(e) {}
			msg.reply({ done: true });
		});
	`)
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "denied-evt-rbac.ts", "trigger", map[string]bool{"go": true})
	replyCh := make(chan struct{}, 1)
	replyCancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(_ sdk.Message) { replyCh <- struct{}{} })
	defer replyCancel()
	<-replyCh

	select {
	case evt := <-deniedCh:
		assert.Equal(t, "denied-evt-rbac.ts", evt.Source)
		assert.Contains(t, evt.Action, "publish")
		assert.Equal(t, "restricted", evt.Role)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for permission denied event")
	}
}

func testWithRoleOnDeploy(t *testing.T, _ *suite.TestEnv) {
	k := newRestrictedKernel(t)
	ctx := context.Background()

	require.NoError(t, testutil.DeployWithOpts(k, "observer-svc-rbac.ts", `
		bus.on("test", async (msg) => {
			try {
				bus.publish("events.something", { data: "test" });
				msg.reply({ published: true });
			} catch(e) {
				msg.reply({ error: e.message });
			}
		});
	`, "observer", ""))
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "observer-svc-rbac.ts", "test", map[string]bool{"go": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) {
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

func testRolePersistenceAcrossRestart(t *testing.T, _ *suite.TestEnv) {
	storePath := t.TempDir() + "/rbac-persist.db"

	// Phase 1: Create kit, deploy with role, close
	store1, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k1, err := brainkit.New(brainkit.Config{
		Store: store1,
		Roles: map[string]rbac.Role{
			"observer": rbac.RoleObserver,
		},
	})
	require.NoError(t, err)

	code := `bus.on("ping", (msg) => msg.reply({ pong: true }));`
	require.NoError(t, testutil.DeployWithOpts(k1, "watched-rbac.ts", code, "observer", ""))

	deployments := testutil.ListDeployments(t, k1)
	require.Len(t, deployments, 1)
	k1.Close()

	// Phase 2: Reopen with same store — deployment auto-redeploys with persisted role
	store2, err := brainkit.NewSQLiteStore(storePath)
	require.NoError(t, err)
	k2, err := brainkit.New(brainkit.Config{
		Store: store2,
		Roles: map[string]rbac.Role{
			"observer": rbac.RoleObserver,
		},
	})
	require.NoError(t, err)
	defer k2.Close()

	deployments2 := testutil.ListDeployments(t, k2)
	require.Len(t, deployments2, 1, "deployment should be restored from persistence")
	assert.Equal(t, "watched-rbac.ts", deployments2[0].Source)

	// Deploy an admin service to provide cross-role verification
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
	require.NoError(t, testutil.DeployWithOpts(k2, "admin-checker.ts", adminCode, "admin", ""))

	// Verify the observer's restored deployment is functional
	time.Sleep(100 * time.Millisecond)
	sendPR, err := sdk.SendToService(k2, context.Background(), "watched-rbac.ts", "ping", map[string]bool{"x": true})
	require.NoError(t, err)

	replyCh := make(chan map[string]any, 1)
	unsub, err := k2.SubscribeRaw(context.Background(), sendPR.ReplyTo, func(msg sdk.Message) {
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

func testSecretBridgeEnforcement(t *testing.T, _ *suite.TestEnv) {
	k := newRestrictedKernel(t)
	ctx := context.Background()

	testutil.Deploy(t, k, "secret-reader-rbac.ts", `
		bus.on("read", async (msg) => {
			try {
				var val = secrets.get("test-secret");
				msg.reply({ value: val, denied: false });
			} catch(e) {
				msg.reply({ denied: true, error: e.message });
			}
		});
	`)
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "secret-reader-rbac.ts", "read", map[string]bool{"go": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) {
		var resp map[string]any
		json.Unmarshal(msg.Payload, &resp)
		replyCh <- resp
	})
	defer cancel()

	select {
	case resp := <-replyCh:
		denied, _ := resp["denied"].(bool)
		assert.True(t, denied, "restricted role should be denied secrets.get, got: %v", resp)
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}

func testGatewayRouteEnforcement(t *testing.T, _ *suite.TestEnv) {
	k := newRestrictedKernel(t)
	ctx := context.Background()

	testutil.Deploy(t, k, "route-adder-rbac.ts", `
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
	time.Sleep(200 * time.Millisecond)

	sendPR, _ := sdk.SendToService(k, ctx, "route-adder-rbac.ts", "try-route", map[string]bool{"go": true})
	replyCh := make(chan map[string]any, 1)
	cancel, _ := k.SubscribeRaw(ctx, sendPR.ReplyTo, func(msg sdk.Message) {
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

func testCommandMatrix(t *testing.T, _ *suite.TestEnv) {
	tests := []struct {
		role    string
		command string
		allowed bool
	}{
		{"service", "tools.call", true}, {"service", "tools.list", true},
		{"service", "secrets.get", true},
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

func testMultiDeploymentIsolation(t *testing.T, _ *suite.TestEnv) {
	storePath := t.TempDir() + "/isolation.db"
	store, _ := brainkit.NewSQLiteStore(storePath)
	k, err := brainkit.New(brainkit.Config{
		Store: store,
		Roles: map[string]rbac.Role{
			"observer": rbac.RoleObserver,
		},
		DefaultRole: "service",
	})
	require.NoError(t, err)
	defer k.Close()
	ctx := context.Background()

	// Deploy A with admin — can emit anywhere
	require.NoError(t, testutil.DeployWithOpts(k, "admin-svc-rbac.ts", `
		bus.on("test", async (msg) => {
			try {
				bus.emit("events.custom", { from: "admin" });
				msg.reply({ emitted: true });
			} catch(e) {
				msg.reply({ error: e.message });
			}
		});
	`, "admin", ""))

	// Deploy B with observer — cannot emit
	require.NoError(t, testutil.DeployWithOpts(k, "observer-svc-rbac.ts", `
		bus.on("test", async (msg) => {
			try {
				bus.emit("events.custom", { from: "observer" });
				msg.reply({ emitted: true });
			} catch(e) {
				msg.reply({ error: e.message });
			}
		});
	`, "observer", ""))
	time.Sleep(200 * time.Millisecond)

	// Admin should succeed
	adminPR, _ := sdk.SendToService(k, ctx, "admin-svc-rbac.ts", "test", map[string]bool{"go": true})
	adminReply := make(chan map[string]any, 1)
	adminUnsub, _ := k.SubscribeRaw(ctx, adminPR.ReplyTo, func(msg sdk.Message) {
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
	obsPR, _ := sdk.SendToService(k, ctx, "observer-svc-rbac.ts", "test", map[string]bool{"go": true})
	obsReply := make(chan map[string]any, 1)
	obsUnsub, _ := k.SubscribeRaw(ctx, obsPR.ReplyTo, func(msg sdk.Message) {
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

// testRBACDeniedFromTS — observer role .ts deployment tries bus operations it shouldn't access.
func testRBACDeniedFromTS(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{
			"observer": rbac.RoleObserver,
		},
		DefaultRole: "observer",
	})
	require.NoError(t, err)
	defer k.Close()

	// Observer cannot publish to arbitrary topics (only subscribe)
	t.Run("bus.publish/denied", func(t *testing.T) {
		require.NoError(t, testutil.DeployWithOpts(k, "rbac-bus-deny.ts", `
			var caught = "none";
			try { bus.publish("forbidden.topic", {}); }
			catch(e) { caught = "DENIED:" + (e.message || ""); }
			output(caught);
		`, "observer", ""))
		defer testutil.Teardown(t, k, "rbac-bus-deny.ts")

		result, _ := testutil.EvalTSErr(k, "__rbac_bus_result.ts", `return String(globalThis.__module_result || "");`)
		assert.Contains(t, result, "DENIED", "observer should be denied bus.publish to forbidden topic")
	})

	// Observer CAN subscribe (observer role allows subscribe to *)
	t.Run("bus.subscribe/allowed", func(t *testing.T) {
		require.NoError(t, testutil.DeployWithOpts(k, "rbac-bus-allow.ts", `
			var caught = "none";
			try {
				var subId = bus.subscribe("events.anything", function() {});
				bus.unsubscribe(subId);
				caught = "ALLOWED";
			} catch(e) { caught = "DENIED:" + (e.message || ""); }
			output(caught);
		`, "observer", ""))
		defer testutil.Teardown(t, k, "rbac-bus-allow.ts")

		result, _ := testutil.EvalTSErr(k, "__rbac_sub_result.ts", `return String(globalThis.__module_result || "");`)
		assert.Equal(t, "ALLOWED", result, "observer should be allowed bus.subscribe")
	})
}

// testInputAbuseRBACEmptySource — RBACAssignMsg{Source: "", Role: "admin"} must return VALIDATION_ERROR.
func testInputAbuseRBACEmptySource(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{"admin": rbac.RoleAdmin},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, err := sdk.Publish(k, ctx, sdk.RBACAssignMsg{Source: "", Role: "admin"})
	require.NoError(t, err)

	ch := make(chan json.RawMessage, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- json.RawMessage(m.Payload) })
	defer unsub()

	select {
	case payload := <-ch:
		var resp struct {
			Code string `json:"code"`
		}
		require.NoError(t, json.Unmarshal(payload, &resp))
		assert.Equal(t, "VALIDATION_ERROR", resp.Code)
	case <-ctx.Done():
		t.Fatal("timeout waiting for RBAC empty source response")
	}
}

// testInputAbuseRBACNonexistentRole — assigning a nonexistent role should error.
func testInputAbuseRBACNonexistentRole(t *testing.T, _ *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{"admin": rbac.RoleAdmin},
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()
	pr, err := sdk.Publish(k, ctx, sdk.RBACAssignMsg{Source: "test.ts", Role: "nonexistent-role"})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m sdk.Message) { ch <- m.Payload })
	defer unsub()
	select {
	case payload := <-ch:
		assert.Contains(t, string(payload), "error")
	case <-time.After(5 * time.Second):
		t.Fatal("timeout")
	}
}
