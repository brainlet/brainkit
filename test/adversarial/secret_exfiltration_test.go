package adversarial_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync/atomic"
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
// SECRET EXFILTRATION
// Deployed code tries to steal secrets through side channels.
// ════════════════════════════════════════════════════════════════════════════

// Attack: service reads a secret then publishes it to a public topic
func TestSecretExfiltration_PublishSecretToBus(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "secrets.db"))
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store, SecretKey: "exfil-test-key-32-characters!!",
		Roles: map[string]rbac.Role{
			"service": rbac.RoleService,
		},
		DefaultRole: "service",
	})
	require.NoError(t, err)
	defer k.Close()

	type echoIn struct{ Message string `json:"message"` }
	brainkit.RegisterTool(k, "echo", registry.TypedTool[echoIn]{
		Description: "echoes", Execute: func(ctx context.Context, in echoIn) (any, error) {
			return map[string]string{"echoed": in.Message}, nil
		},
	})

	ctx := context.Background()

	// Set a secret
	pr, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "DB_PASSWORD", Value: "super-secret-pw-123"})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()

	// Service reads the secret (allowed: service has secrets.get)
	// then tries to exfiltrate via bus.publish
	_, err = k.Deploy(ctx, "exfil-service.ts", `
		var password = secrets.get("DB_PASSWORD");
		var results = {};

		// Exfiltration attempt 1: publish to incoming topic
		try {
			bus.publish("incoming.exfil", {stolen: password});
			results.publishExfil = "SENT";
		} catch(e) { results.publishExfil = "BLOCKED:" + (e.code || ""); }

		// Exfiltration attempt 2: emit to events topic
		try {
			bus.emit("events.exfil", {stolen: password});
			results.emitExfil = "SENT";
		} catch(e) { results.emitExfil = "BLOCKED:" + (e.code || ""); }

		// Exfiltration attempt 3: put secret in tool result
		try {
			var t = createTool({
				id: "leak-tool",
				description: "returns stolen secret",
				execute: async () => ({leaked: password}),
			});
			kit.register("tool", "leak-tool", t);
			results.toolExfil = "REGISTERED";
		} catch(e) { results.toolExfil = "BLOCKED:" + (e.code || ""); }

		output(results);
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	// Listen for exfiltrated data
	var exfilDetected atomic.Int64
	for _, topic := range []string{"incoming.exfil", "events.exfil"} {
		topic := topic
		u, _ := k.SubscribeRaw(ctx, topic, func(m messages.Message) {
			if len(m.Payload) > 0 {
				var data struct{ Stolen string `json:"stolen"` }
				json.Unmarshal(m.Payload, &data)
				if data.Stolen == "super-secret-pw-123" {
					exfilDetected.Add(1)
				}
			}
		})
		defer u()
	}

	time.Sleep(500 * time.Millisecond)

	result, _ := k.EvalTS(ctx, "__exfil.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Exfiltration attempts: %s", result)
	t.Logf("Exfiltrations detected: %d", exfilDetected.Load())
	// Service role CAN publish to incoming.* and emit to events.* — this is by design
	// But the audit trail (secrets.accessed event) should have fired
}

// Attack: read secret via bridge in a Compartment that shouldn't have access
func TestSecretExfiltration_ObserverReadsSecret(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "secrets.db"))
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store, SecretKey: "observer-test-key-32chars!!!!",
		Roles: map[string]rbac.Role{
			"observer": rbac.RoleObserver,
		},
		DefaultRole: "observer",
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	// Set secret as Go code (admin level)
	sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "API_TOKEN", Value: "tok-secret-999"})
	time.Sleep(100 * time.Millisecond)

	// Observer tries to read it
	_, err = k.Deploy(ctx, "observer-secret.ts", `
		var result = "UNKNOWN";
		try {
			var val = secrets.get("API_TOKEN");
			if (val && val.length > 0) {
				result = "LEAKED:" + val;
			} else {
				result = "EMPTY";
			}
		} catch(e) {
			result = "DENIED:" + (e.code || e.message);
		}
		output(result);
	`, brainkit.WithRole("observer"))
	require.NoError(t, err)

	result, _ := k.EvalTS(ctx, "__obs_sec.ts", `return String(globalThis.__module_result || "");`)
	// Observer shouldn't have secrets.get command permission
	assert.NotContains(t, result, "tok-secret", "observer should not be able to read secrets")
}

// Attack: deploy code that reads all environment variables
func TestSecretExfiltration_EnvVarDump(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "env-dump.ts", `
		var envVars = {};
		try {
			// process.env is available as an endowment
			if (typeof process !== "undefined" && process.env) {
				// Try to read specific sensitive env vars
				var sensitive = [
					"OPENAI_API_KEY", "ANTHROPIC_API_KEY",
					"AWS_SECRET_ACCESS_KEY", "DATABASE_URL",
					"BRAINKIT_SECRET_KEY", "HOME", "PATH",
				];
				for (var i = 0; i < sensitive.length; i++) {
					var val = process.env[sensitive[i]];
					if (val && val.length > 0) {
						envVars[sensitive[i]] = val.substring(0, 10) + "...";
					}
				}
			}
		} catch(e) { envVars.error = e.message; }
		output(envVars);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__env_dump.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Env var access from .ts: %s", result)
	// process.env IS accessible (by design — needed for provider auto-detection)
	// But this means any .ts deployment can read ALL env vars including API keys
}

// Attack: secrets.list to enumerate all secret names
func TestSecretExfiltration_SecretEnumeration(t *testing.T) {
	tmpDir := t.TempDir()
	store, _ := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "secrets.db"))
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store, SecretKey: "enum-test-key-32-characters!!",
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	// Set several secrets
	for _, name := range []string{"DB_PASSWORD", "API_KEY", "STRIPE_SECRET", "ADMIN_TOKEN"} {
		pr, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: name, Value: "secret-" + name})
		ch := make(chan []byte, 1)
		unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
		<-ch
		unsub()
	}

	// Can a deployment enumerate secret names?
	_, err = k.Deploy(ctx, "enum-secrets.ts", `
		var result = "UNKNOWN";
		try {
			// secrets.list is not available as an endowment — only secrets.get
			// But what about via bridge?
			var raw = __go_brainkit_request("secrets.list", "{}");
			result = "LISTED:" + raw;
		} catch(e) {
			result = "BLOCKED:" + (e.code || e.message);
		}
		output(result);
	`)
	require.NoError(t, err)

	result, _ := k.EvalTS(ctx, "__enum.ts", `return String(globalThis.__module_result || "");`)
	t.Logf("Secret enumeration: %s", result)
	// If the bridge is accessible, .ts code can list all secret names
}

// Attack: use audit events to learn when secrets are accessed
func TestSecretExfiltration_AuditEventSnooping(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set a secret
	pr, _ := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: "MONITORED_KEY", Value: "monitored-value"})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()

	// Attacker subscribes to audit events to learn WHEN secrets are accessed
	var auditEvents []string
	auditUnsub, _ := tk.SubscribeRaw(ctx, "secrets.accessed", func(m messages.Message) {
		auditEvents = append(auditEvents, string(m.Payload))
	})
	defer auditUnsub()

	// Legitimate access
	pr2, _ := sdk.Publish(tk, ctx, messages.SecretsGetMsg{Name: "MONITORED_KEY"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	<-ch2
	unsub2()

	time.Sleep(300 * time.Millisecond)

	// Audit events reveal access patterns (which secrets, when, by whom)
	t.Logf("Audit events captured by eavesdropper: %d", len(auditEvents))
	for _, e := range auditEvents {
		t.Logf("  Audit: %s", e)
	}
	// This is by design (audit is public on the bus) but worth documenting
	// An observer-role deployment can subscribe to secrets.accessed and learn
	// which secrets are being read and by whom
}

// Attack: rotate a secret that another deployment is using (denial of service)
func TestSecretExfiltration_RotateDOS(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Set a secret
	pr, _ := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: "SHARED_KEY", Value: "original-value"})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()

	// Victim reads the secret
	_, err := tk.Deploy(ctx, "victim-secret.ts", `
		var key = secrets.get("SHARED_KEY");
		bus.on("check", function(msg) {
			var current = secrets.get("SHARED_KEY");
			msg.reply({original: key, current: current, match: key === current});
		});
	`)
	require.NoError(t, err)

	// Attacker rotates the secret to break the victim
	pr2, _ := sdk.Publish(tk, ctx, messages.SecretsRotateMsg{Name: "SHARED_KEY", NewValue: "rotated-by-attacker"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := tk.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	<-ch2
	unsub2()

	// Check if victim's cached key is now wrong
	pr3, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "ts.victim-secret.check", Payload: json.RawMessage(`{}`),
	})
	ch3 := make(chan []byte, 1)
	unsub3, _ := tk.SubscribeRaw(ctx, pr3.ReplyTo, func(m messages.Message) { ch3 <- m.Payload })
	defer unsub3()

	select {
	case p := <-ch3:
		var resp struct {
			Original string `json:"original"`
			Current  string `json:"current"`
			Match    bool   `json:"match"`
		}
		json.Unmarshal(p, &resp)
		if !resp.Match {
			t.Logf("FINDING: secret rotation broke victim (original=%s, current=%s)", resp.Original, resp.Current)
		}
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// Attack: encrypted secret store with wrong key — does decryption fail safely?
func TestSecretExfiltration_DecryptionOracle(t *testing.T) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "oracle.db")

	// Phase 1: Store with correct key
	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1, SecretKey: "correct-key-32-characters-long!",
	})
	require.NoError(t, err)

	ctx := context.Background()
	pr, _ := sdk.Publish(k1, ctx, messages.SecretsSetMsg{Name: "encrypted", Value: "sensitive-data"})
	ch := make(chan []byte, 1)
	unsub, _ := k1.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()
	k1.Close()

	// Phase 2: Try multiple wrong keys — should all fail
	wrongKeys := []string{
		"wrong-key-32-characters-long!!",
		"CORRECT-KEY-32-CHARACTERS-LONG!", // case difference
		"correct-key-32-characters-long",  // one char short
		"",                                // empty key (dev mode)
	}

	for _, wrongKey := range wrongKeys {
		t.Run("key="+wrongKey[:min(10, len(wrongKey))], func(t *testing.T) {
			store2, _ := brainkit.NewSQLiteStore(storePath)
			k2, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test", CallerID: "test", FSRoot: tmpDir,
				Store: store2, SecretKey: wrongKey,
			})
			require.NoError(t, err)
			defer k2.Close()

			pr2, _ := sdk.Publish(k2, ctx, messages.SecretsGetMsg{Name: "encrypted"})
			ch2 := make(chan []byte, 1)
			unsub2, _ := k2.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
			defer unsub2()

			select {
			case p := <-ch2:
				var resp struct {
					Value string `json:"value"`
					Error string `json:"error"`
				}
				json.Unmarshal(p, &resp)
				assert.NotEqual(t, "sensitive-data", resp.Value, "wrong key should not decrypt secret")
			case <-time.After(5 * time.Second):
				t.Fatal("timeout")
			}
		})
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
