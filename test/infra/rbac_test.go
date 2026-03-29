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
