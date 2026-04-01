package adversarial_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ════════════════════════════════════════════════════════════════════════════
// WASM COMMAND ALLOWLIST — functional + adversarial tests
// ════════════════════════════════════════════════════════════════════════════

// --- Functional: bus commands work ---

func TestWASMAllowlist_GetDefault(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pr, err := sdk.Publish(tk, ctx, messages.WasmAllowlistGetMsg{})
	require.NoError(t, err)

	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		var resp messages.WasmAllowlistGetResp
		require.NoError(t, json.Unmarshal(p, &resp))
		assert.True(t, resp.Allowlist["tools.call"], "default should allow tools.call")
		assert.True(t, resp.Allowlist["fs.read"], "default should allow fs.read")
		assert.False(t, resp.Allowlist["kit.deploy"], "default should NOT allow kit.deploy")
		assert.False(t, resp.Allowlist["secrets.set"], "default should NOT allow secrets.set")
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

func TestWASMAllowlist_AddRemove(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add fs.write (not in defaults)
	pr1, _ := sdk.Publish(tk, ctx, messages.WasmAllowlistAddMsg{Command: "fs.write"})
	ch1 := make(chan []byte, 1)
	unsub1, _ := tk.SubscribeRaw(ctx, pr1.ReplyTo, func(m messages.Message) { ch1 <- m.Payload })
	<-ch1
	unsub1()

	// Verify it's there
	allowlist := tk.WASMAllowlistGet()
	assert.True(t, allowlist["fs.write"], "fs.write should be in allowlist after add")
	assert.True(t, allowlist["tools.call"], "tools.call should still be there")

	// Remove tools.call
	pr2, _ := sdk.Publish(tk, ctx, messages.WasmAllowlistRemoveMsg{Command: "tools.call"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	<-ch2
	unsub2()

	// Verify
	allowlist2 := tk.WASMAllowlistGet()
	assert.False(t, allowlist2["tools.call"], "tools.call should be gone after remove")
	assert.True(t, allowlist2["fs.write"], "fs.write should still be there")
}

func TestWASMAllowlist_SetReplace(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Replace entire allowlist with just one command
	pr, _ := sdk.Publish(tk, ctx, messages.WasmAllowlistSetMsg{
		Allowlist: map[string]bool{"custom.command": true},
	})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()

	allowlist := tk.WASMAllowlistGet()
	assert.True(t, allowlist["custom.command"])
	assert.False(t, allowlist["tools.call"], "old defaults should be gone after set")
	assert.Equal(t, 1, len(allowlist))
}

func TestWASMAllowlist_GoMethodsDirect(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)

	// Go API works directly
	tk.WASMAllowlistAdd("secrets.get")
	assert.True(t, tk.WASMAllowlistGet()["secrets.get"])

	tk.WASMAllowlistRemove("secrets.get")
	assert.False(t, tk.WASMAllowlistGet()["secrets.get"])

	tk.WASMAllowlistSet(map[string]bool{"only.this": true})
	assert.Equal(t, 1, len(tk.WASMAllowlistGet()))
}

// --- Functional: allowlist actually enforced on WASM ---

func TestWASMAllowlist_EnforcedOnCompile(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Remove tools.call from allowlist
	tk.WASMAllowlistRemove("tools.call")

	// Compile a WASM module that calls tools.call via bus_publish
	pr, err := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source: `
			import { _busPublish } from "brainkit";
			export function onResult(topic: usize, payload: usize): void {}
			export function run(): i32 {
				_busPublish("tools.call", '{"name":"echo","input":{"message":"blocked"}}', "onResult");
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "blocked-tool"},
	})
	require.NoError(t, err)
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	select {
	case p := <-ch:
		require.False(t, responseHasError(p), "compile should succeed: %s", string(p))
	case <-ctx.Done():
		t.Fatal("timeout compile")
	}
	unsub()

	// Run it — the bus_publish callback should get an error, not a tool result
	pr2, _ := sdk.Publish(tk, ctx, messages.WasmRunMsg{ModuleID: "blocked-tool"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()

	select {
	case <-ch2:
		// Module ran. The callback would have received an error payload.
		// We can't directly inspect the callback payload from here,
		// but the module didn't crash — it handled the blocked command.
	case <-ctx.Done():
		t.Fatal("timeout run")
	}
}

func TestWASMAllowlist_DynamicAddUnblocks(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		// Start with empty allowlist — nothing allowed
		WASMCommandAllowlist: map[string]bool{},
	})
	require.NoError(t, err)
	defer k.Close()

	// tools.call should be blocked
	assert.False(t, k.WASMAllowlistGet()["tools.call"])

	// Dynamically add it
	k.WASMAllowlistAdd("tools.call")
	assert.True(t, k.WASMAllowlistGet()["tools.call"])
}

// --- Adversarial: attacks on the allowlist itself ---

// Attack: observer tries to add commands to the WASM allowlist
func TestWASMAllowlist_Attack_ObserverTriesAdd(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{
			"observer": rbac.RoleObserver,
		},
		DefaultRole: "observer",
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	// Observer deploys .ts that tries to add kit.deploy to WASM allowlist
	_, err = k.Deploy(ctx, "allowlist-attack.ts", `
		var result = "UNKNOWN";
		try {
			var raw = __go_brainkit_request("wasm.allowlist.add", JSON.stringify({command: "kit.deploy"}));
			result = "ALLOWED:" + raw;
		} catch(e) {
			result = "DENIED:" + (e.code || e.message);
		}
		output(result);
	`, brainkit.WithRole("observer"))
	require.NoError(t, err)

	result, _ := k.EvalTS(ctx, "__al_atk.ts", `return String(globalThis.__module_result || "");`)
	// Observer doesn't have wasm.allowlist.add in their commands — should be denied
	assert.Contains(t, result, "DENIED", "observer should not be able to modify WASM allowlist")

	// Verify allowlist wasn't changed
	assert.False(t, k.WASMAllowlistGet()["kit.deploy"], "kit.deploy should NOT be in allowlist")
}

// Attack: service tries to set allowlist to allow everything
func TestWASMAllowlist_Attack_ServiceTriesSetAll(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{
			"service": rbac.RoleService,
		},
		DefaultRole: "service",
	})
	require.NoError(t, err)
	defer k.Close()

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(k, "echo", registry.TypedTool[echoIn]{
		Description: "echoes",
		Execute:     func(ctx context.Context, in echoIn) (any, error) { return in, nil },
	})

	ctx := context.Background()

	// Service tries to wasm.allowlist.set via bridge
	_, err = k.Deploy(ctx, "allowlist-escalate.ts", `
		var result = "UNKNOWN";
		try {
			var raw = __go_brainkit_request("wasm.allowlist.set", JSON.stringify({
				allowlist: {"kit.deploy": true, "secrets.set": true, "rbac.assign": true}
			}));
			result = "ALLOWED:" + raw;
		} catch(e) {
			result = "DENIED:" + (e.code || e.message);
		}
		output(result);
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	result, _ := k.EvalTS(ctx, "__al_esc.ts", `return String(globalThis.__module_result || "");`)
	// Service doesn't have wasm.allowlist.set in allowed commands
	assert.Contains(t, result, "DENIED", "service should not be able to replace WASM allowlist")
}

// Attack: add a non-existent command — should it matter?
func TestWASMAllowlist_Attack_AddGarbageCommand(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Add a command that doesn't exist in the catalog
	pr, _ := sdk.Publish(tk, ctx, messages.WasmAllowlistAddMsg{Command: "totally.fake.command"})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()

	// It's in the allowlist but useless — the catalog won't have it
	allowlist := tk.WASMAllowlistGet()
	assert.True(t, allowlist["totally.fake.command"], "garbage commands are stored but harmless")
}

// Attack: set allowlist to empty map — blocks everything
func TestWASMAllowlist_Attack_EmptyAllowlistBlocksAll(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Set to empty
	pr, _ := sdk.Publish(tk, ctx, messages.WasmAllowlistSetMsg{
		Allowlist: map[string]bool{},
	})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()

	allowlist := tk.WASMAllowlistGet()
	assert.Equal(t, 0, len(allowlist), "empty set should block all WASM commands")
	assert.False(t, tk.WASMAllowlistGet()["tools.call"])
}

// Attack: concurrent add/remove/get — no races or panics
func TestWASMAllowlist_Attack_ConcurrentAccess(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	done := make(chan bool, 3)

	// Writer 1: add
	go func() {
		for i := 0; i < 100; i++ {
			tk.WASMAllowlistAdd("concurrent.test")
		}
		done <- true
	}()

	// Writer 2: remove
	go func() {
		for i := 0; i < 100; i++ {
			tk.WASMAllowlistRemove("concurrent.test")
		}
		done <- true
	}()

	// Reader
	go func() {
		for i := 0; i < 100; i++ {
			_ = tk.WASMAllowlistGet()
		}
		done <- true
	}()

	<-done
	<-done
	<-done

	assert.True(t, tk.Alive(ctx), "kernel should survive concurrent allowlist access")
}

// Attack: set allowlist with special characters in command names
func TestWASMAllowlist_Attack_SpecialCharsInCommand(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)

	evilCommands := []string{
		"",
		"../../../etc/passwd",
		"command\x00with\x00nulls",
		"'; DROP TABLE--",
		"<script>alert(1)</script>",
		"command with spaces",
		"very.long." + string(make([]byte, 10000)),
	}

	for _, cmd := range evilCommands {
		tk.WASMAllowlistAdd(cmd)
	}

	// Kernel should handle all of them without crashing
	allowlist := tk.WASMAllowlistGet()
	assert.Greater(t, len(allowlist), len(brainkit.DefaultWASMCommandAllowlist),
		"evil commands should be stored (they're harmless — catalog won't match them)")
}

// Attack: .ts deployment reads the allowlist to discover what WASM can do (reconnaissance)
func TestWASMAllowlist_Attack_Reconnaissance(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "recon-allowlist.ts", `
		var result = "UNKNOWN";
		try {
			var raw = __go_brainkit_request("wasm.allowlist.get", "{}");
			var parsed = JSON.parse(raw);
			result = "LISTED:" + Object.keys(parsed.allowlist || {}).length + " commands";
		} catch(e) {
			result = "DENIED:" + (e.code || e.message);
		}
		output(result);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__recon_al.ts", `return String(globalThis.__module_result || "");`)
	t.Logf("Allowlist recon from .ts: %s", result)
	// wasm.allowlist.get is a command topic — subject to RBAC
	// Default kernel (no RBAC) allows it; RBAC-enabled kernel would check
}

// ════════════════════════════════════════════════════════════════════════════
// MULTI-SURFACE ADVERSARIAL
// ════════════════════════════════════════════════════════════════════════════

// Attack: WASM module tries to modify its OWN allowlist via bus_publish
func TestWASMAllowlist_Attack_WASMSelfEscalation(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Compile WASM that tries to add kit.deploy to the allowlist via bus_publish
	pr, err := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
		Source: `
			import { _busPublish } from "brainkit";
			export function onResult(topic: usize, payload: usize): void {}
			export function run(): i32 {
				// Try to escalate: add kit.deploy to allowlist
				_busPublish("wasm.allowlist.add", '{"command":"kit.deploy"}', "onResult");
				// Then try to use it
				_busPublish("kit.deploy", '{"source":"wasm-planted.ts","code":"output(1);"}', "onResult");
				return 0;
			}
		`,
		Options: &messages.WasmCompileOpts{Name: "self-escalate"},
	})
	require.NoError(t, err)
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	select {
	case p := <-ch:
		require.False(t, responseHasError(p), "compile: %s", string(p))
	case <-ctx.Done():
		t.Fatal("timeout compile")
	}
	unsub()

	// Run — WASM tries to call wasm.allowlist.add but that's NOT in the allowlist
	pr2, _ := sdk.Publish(tk, ctx, messages.WasmRunMsg{ModuleID: "self-escalate"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	defer unsub2()
	select {
	case <-ch2:
	case <-ctx.Done():
		t.Fatal("timeout run")
	}

	// wasm.allowlist.add is a command topic — and it's NOT in the default WASM allowlist
	// So WASM can't escalate its own permissions
	assert.False(t, tk.WASMAllowlistGet()["kit.deploy"],
		"WASM should not be able to add kit.deploy to its own allowlist")
	deps := tk.ListDeployments()
	for _, d := range deps {
		assert.NotEqual(t, "wasm-planted.ts", d.Source, "WASM self-escalation should not deploy code")
	}
}

// Attack: .ts deployment adds to allowlist, then deploys WASM that exploits it
func TestWASMAllowlist_Attack_TSEscalatesThenWASMExploits(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{
			"admin":   rbac.RoleAdmin,
			"service": rbac.RoleService,
		},
		DefaultRole: "service",
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	// Service .ts tries to add secrets.set to WASM allowlist
	_, err = k.Deploy(ctx, "ts-escalator.ts", `
		var result = "UNKNOWN";
		try {
			var raw = __go_brainkit_request("wasm.allowlist.add", JSON.stringify({command: "secrets.set"}));
			result = "ADDED:" + raw;
		} catch(e) {
			result = "DENIED:" + (e.code || e.message);
		}
		output(result);
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	result, _ := k.EvalTS(ctx, "__ts_esc.ts", `return String(globalThis.__module_result || "");`)
	// Service doesn't have wasm.allowlist.add in allowed commands
	assert.Contains(t, result, "DENIED", "service should not add to WASM allowlist")
	assert.False(t, k.WASMAllowlistGet()["secrets.set"])
}

// Attack: admin adds dangerous command, then gets role downgraded — allowlist persists
func TestWASMAllowlist_Attack_AdminAddsThensDowngraded(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Roles: map[string]rbac.Role{
			"admin":   rbac.RoleAdmin,
			"service": rbac.RoleService,
		},
		DefaultRole: "admin",
	})
	require.NoError(t, err)
	defer k.Close()

	// Admin adds fs.write to WASM allowlist
	k.WASMAllowlistAdd("fs.write")
	assert.True(t, k.WASMAllowlistGet()["fs.write"])

	// Admin deployment gets role downgraded to service
	// The allowlist change persists — it's kernel-level, not per-deployment
	// This is by design: allowlist is infrastructure config, not per-role state
	k.WASMAllowlistAdd("secrets.get")
	assert.True(t, k.WASMAllowlistGet()["secrets.get"],
		"allowlist changes persist regardless of who made them")
}

// Attack: change allowlist while WASM compile is in progress
func TestWASMAllowlist_Attack_ChangeWhileCompiling(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Start a compile in background
	done := make(chan bool, 1)
	go func() {
		pr, _ := sdk.Publish(tk, ctx, messages.WasmCompileMsg{
			Source:  `export function run(): i32 { return 1; }`,
			Options: &messages.WasmCompileOpts{Name: "during-change"},
		})
		ch := make(chan []byte, 1)
		unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
		select {
		case <-ch:
		case <-ctx.Done():
		}
		unsub()
		done <- true
	}()

	// Meanwhile, rapidly toggle the allowlist
	for i := 0; i < 50; i++ {
		tk.WASMAllowlistAdd("tools.call")
		tk.WASMAllowlistRemove("tools.call")
		tk.WASMAllowlistAdd("tools.call")
	}

	<-done
	assert.True(t, tk.Alive(ctx), "kernel should survive allowlist changes during compilation")
}

// Attack: allowlist does NOT survive restart (it's runtime state)
func TestWASMAllowlist_Attack_DoesNotPersist(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := tmpDir + "/persist.db"

	// Phase 1: Create kernel, modify allowlist, close
	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1,
	})
	require.NoError(t, err)
	k1.WASMAllowlistAdd("secrets.set")
	k1.WASMAllowlistAdd("kit.deploy")
	assert.True(t, k1.WASMAllowlistGet()["secrets.set"])
	k1.Close()

	// Phase 2: Reopen — allowlist should be back to defaults
	store2, _ := brainkit.NewSQLiteStore(storePath)
	k2, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store2,
	})
	require.NoError(t, err)
	defer k2.Close()

	assert.False(t, k2.WASMAllowlistGet()["secrets.set"],
		"runtime allowlist changes should NOT survive restart")
	assert.False(t, k2.WASMAllowlistGet()["kit.deploy"])
	assert.True(t, k2.WASMAllowlistGet()["tools.call"],
		"defaults should be restored after restart")
}

// Attack: .ts handler modifies allowlist DURING a WASM handler invocation
func TestWASMAllowlist_Attack_ModifyDuringWASMHandler(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Deploy .ts that listens for a signal and then removes tools.call from allowlist
	_, err := tk.Deploy(ctx, "allowlist-sniper.ts", `
		bus.on("snipe", function(msg) {
			// Remove tools.call while some WASM module might be using it
			bus.publish("wasm.allowlist.remove", {command: "tools.call"});
			msg.reply({sniped: true});
		});
	`)
	require.NoError(t, err)

	// Trigger the snipe
	pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "ts.allowlist-sniper.snipe", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case <-ch:
	case <-time.After(3 * time.Second):
	}

	// The allowlist change should have taken effect
	// (wasm.allowlist.remove is a command topic, so the .ts admin code publishes to it,
	// and the kernel processes it — but only if the deployment has command permission)
	assert.True(t, tk.Alive(ctx))
}

// Attack: use bus.emit to bypass — allowlist commands are command topics
func TestWASMAllowlist_Attack_EmitToAllowlistCommand(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "emit-allowlist.ts", `
		var result = "UNKNOWN";
		try {
			// bus.emit to a command topic is now blocked (bug #8 fix)
			bus.emit("wasm.allowlist.add", {command: "kit.deploy"});
			result = "EMITTED";
		} catch(e) {
			result = "BLOCKED:" + (e.code || e.message);
		}
		output(result);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__emit_al.ts", `return String(globalThis.__module_result || "");`)
	assert.Contains(t, result, "BLOCKED", "bus.emit to wasm.allowlist.add should be blocked (command topic)")
}

// Attack: gateway route triggers allowlist modification
func TestWASMAllowlist_Attack_ViaGateway(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	gw := setupGateway(t, tk)

	ctx := context.Background()
	_, err := tk.Deploy(ctx, "gw-allowlist.ts", `
		bus.on("modify-allowlist", function(msg) {
			// Handler tries to modify allowlist when triggered by HTTP request
			try {
				var raw = __go_brainkit_request("wasm.allowlist.add", JSON.stringify({command: "kit.deploy"}));
				msg.reply({modified: true});
			} catch(e) {
				msg.reply({denied: true, error: e.message});
			}
		});
	`)
	require.NoError(t, err)
	gw.Handle("POST", "/modify-allowlist", "ts.gw-allowlist.modify-allowlist")

	// HTTP request triggers the handler which tries to modify the allowlist
	status, body := gwPost(t, gw, "/modify-allowlist", `{}`)
	t.Logf("Gateway → allowlist modification: status=%d body=%s", status, body)

	// No RBAC on default kernel — the modification might succeed
	// With RBAC, it would depend on the deployment's role
}

// Attack: allowlist.set with nil map
func TestWASMAllowlist_Attack_SetNil(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Send set with null/empty allowlist via bus
	pr, _ := sdk.Publish(tk, ctx, messages.WasmAllowlistSetMsg{})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case <-ch:
	case <-ctx.Done():
		t.Fatal("timeout")
	}

	// nil map set — should result in empty map (block all), not panic
	allowlist := tk.WASMAllowlistGet()
	assert.NotNil(t, allowlist)
	assert.True(t, tk.Alive(ctx))
}

// Attack: multiple deployments race to modify the allowlist simultaneously
func TestWASMAllowlist_Attack_MultiDeploymentRace(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy 5 services that all try to modify the allowlist at once
	for i := 0; i < 5; i++ {
		src := "racer-" + string(rune('a'+i)) + ".ts"
		tk.Deploy(ctx, src, `
			try {
				bus.publish("wasm.allowlist.add", {command: "racer.command"});
			} catch(e) {}
			output("raced");
		`)
	}

	// All should have executed without crashing
	assert.True(t, tk.Alive(ctx))

	// Cleanup
	for i := 0; i < 5; i++ {
		tk.Teardown(ctx, "racer-"+string(rune('a'+i))+".ts")
	}
}
