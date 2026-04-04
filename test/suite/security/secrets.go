package security

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/rbac"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSecretPublishToBus — service reads a secret then publishes it to a public topic.
func testSecretPublishToBus(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	store, _ := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "secrets-sec.db"))
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

	pr, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "DB_PASSWORD_SEC", Value: "super-secret-pw-123"})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()

	_, err = k.Deploy(ctx, "exfil-service-sec.ts", `
		var password = secrets.get("DB_PASSWORD_SEC");
		var results = {};

		try {
			bus.publish("incoming.exfil-sec", {stolen: password});
			results.publishExfil = "SENT";
		} catch(e) { results.publishExfil = "BLOCKED:" + (e.code || ""); }

		try {
			bus.emit("events.exfil-sec", {stolen: password});
			results.emitExfil = "SENT";
		} catch(e) { results.emitExfil = "BLOCKED:" + (e.code || ""); }

		try {
			var t = createTool({
				id: "leak-tool-sec",
				description: "returns stolen secret",
				execute: async () => ({leaked: password}),
			});
			kit.register("tool", "leak-tool-sec", t);
			results.toolExfil = "REGISTERED";
		} catch(e) { results.toolExfil = "BLOCKED:" + (e.code || ""); }

		output(results);
	`, brainkit.WithRole("service"))
	require.NoError(t, err)

	var exfilDetected atomic.Int64
	for _, topic := range []string{"incoming.exfil-sec", "events.exfil-sec"} {
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
}

// testSecretObserverReadsSecret — read secret via bridge in a Compartment that shouldn't have access.
func testSecretObserverReadsSecret(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	store, _ := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "secrets-sec.db"))
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

	sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "API_TOKEN_SEC", Value: "tok-secret-999"})
	time.Sleep(100 * time.Millisecond)

	_, err = k.Deploy(ctx, "observer-secret-sec.ts", `
		var result = "UNKNOWN";
		try {
			var val = secrets.get("API_TOKEN_SEC");
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
	assert.NotContains(t, result, "tok-secret", "observer should not be able to read secrets")
}

// testSecretEnvVarDump — deploy code that reads all environment variables.
func testSecretEnvVarDump(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	ctx := context.Background()

	_, err := k.Deploy(ctx, "env-dump-sec.ts", `
		var envVars = {};
		try {
			if (typeof process !== "undefined" && process.env) {
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

	result, _ := k.EvalTS(ctx, "__env_dump.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Env var access from .ts: %s", result)
}

// testSecretEnumeration — secrets.list to enumerate all secret names.
func testSecretEnumeration(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	store, _ := brainkit.NewSQLiteStore(filepath.Join(tmpDir, "secrets-sec.db"))
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store, SecretKey: "enum-test-key-32-characters!!",
	})
	require.NoError(t, err)
	defer k.Close()

	ctx := context.Background()

	for _, name := range []string{"DB_PASSWORD_SEC", "API_KEY_SEC", "STRIPE_SECRET_SEC", "ADMIN_TOKEN_SEC"} {
		pr, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: name, Value: "secret-" + name})
		ch := make(chan []byte, 1)
		unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
		<-ch
		unsub()
	}

	_, err = k.Deploy(ctx, "enum-secrets-sec.ts", `
		var result = "UNKNOWN";
		try {
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
}

// testSecretAuditEventSnooping — use audit events to learn when secrets are accessed.
func testSecretAuditEventSnooping(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "MONITORED_KEY_SEC", Value: "monitored-value"})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()

	var auditEvents []string
	auditUnsub, _ := k.SubscribeRaw(ctx, "secrets.accessed", func(m messages.Message) {
		auditEvents = append(auditEvents, string(m.Payload))
	})
	defer auditUnsub()

	pr2, _ := sdk.Publish(k, ctx, messages.SecretsGetMsg{Name: "MONITORED_KEY_SEC"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := k.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	<-ch2
	unsub2()

	time.Sleep(300 * time.Millisecond)

	t.Logf("Audit events captured by eavesdropper: %d", len(auditEvents))
	for _, e := range auditEvents {
		t.Logf("  Audit: %s", e)
	}
}

// testSecretRotateDOS — rotate a secret that another deployment is using (denial of service).
func testSecretRotateDOS(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kernel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pr, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "SHARED_KEY_SEC", Value: "original-value"})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()

	_, err := k.Deploy(ctx, "victim-secret-sec.ts", `
		var key = secrets.get("SHARED_KEY_SEC");
		bus.on("check", function(msg) {
			var current = secrets.get("SHARED_KEY_SEC");
			msg.reply({original: key, current: current, match: key === current});
		});
	`)
	require.NoError(t, err)

	pr2, _ := sdk.Publish(k, ctx, messages.SecretsRotateMsg{Name: "SHARED_KEY_SEC", NewValue: "rotated-by-attacker"})
	ch2 := make(chan []byte, 1)
	unsub2, _ := k.SubscribeRaw(ctx, pr2.ReplyTo, func(m messages.Message) { ch2 <- m.Payload })
	<-ch2
	unsub2()

	pr3, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.victim-secret-sec.check", Payload: json.RawMessage(`{}`),
	})
	ch3 := make(chan []byte, 1)
	unsub3, _ := k.SubscribeRaw(ctx, pr3.ReplyTo, func(m messages.Message) { ch3 <- m.Payload })
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

// testSecretDecryptionOracle — encrypted secret store with wrong key.
func testSecretDecryptionOracle(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	storePath := filepath.Join(tmpDir, "oracle-sec.db")

	store1, _ := brainkit.NewSQLiteStore(storePath)
	k1, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1, SecretKey: "correct-key-32-characters-long!",
	})
	require.NoError(t, err)

	ctx := context.Background()
	pr, _ := sdk.Publish(k1, ctx, messages.SecretsSetMsg{Name: "encrypted-sec", Value: "sensitive-data"})
	ch := make(chan []byte, 1)
	unsub, _ := k1.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()
	k1.Close()

	wrongKeys := []string{
		"wrong-key-32-characters-long!!",
		"CORRECT-KEY-32-CHARACTERS-LONG!",
		"correct-key-32-characters-long",
		"",
	}

	for _, wrongKey := range wrongKeys {
		t.Run("key="+wrongKey[:secMin(10, len(wrongKey))], func(t *testing.T) {
			store2, _ := brainkit.NewSQLiteStore(storePath)
			k2, err := brainkit.NewKernel(brainkit.KernelConfig{
				Namespace: "test", CallerID: "test", FSRoot: tmpDir,
				Store: store2, SecretKey: wrongKey,
			})
			require.NoError(t, err)
			defer k2.Close()

			pr2, _ := sdk.Publish(k2, ctx, messages.SecretsGetMsg{Name: "encrypted-sec"})
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
