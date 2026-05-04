package security

import (
	"context"
	"encoding/json"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	bkmodule "github.com/brainlet/brainkit/module"
	"github.com/brainlet/brainkit/sdk/protocol"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	toolsmod "github.com/brainlet/brainkit/modules/tools"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/stores"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testSecretPublishToBus — service reads a secret then publishes it to a public topic.
func testSecretPublishToBus(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	store, _ := stores.NewSQLite(filepath.Join(tmpDir, "secrets-sec.db"))
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store, SecretKey: "exfil-test-key-32-characters!!",
		Modules: []bkmodule.Module{toolsmod.New()},
	})
	require.NoError(t, err)
	defer k.Close()

	type echoIn struct {
		Message string `json:"message"`
	}
	require.NoError(t, k.Mount(context.Background(), toolsmod.GoTool("echo", toolsmod.TypedTool[echoIn]{
		Description: "echoes", Execute: func(ctx context.Context, in echoIn) (any, error) {
			return map[string]string{"echoed": in.Message}, nil
		},
	})))

	ctx := context.Background()
	secSetSecret(t, k, "DB_PASSWORD_SEC", "super-secret-pw-123")

	require.NoError(t, secDeployErr(k, "exfil-service-sec.ts", `
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
	`))

	var exfilDetected atomic.Int64
	for _, topic := range []string{"incoming.exfil-sec", "events.exfil-sec"} {
		topic := topic
		u, _ := k.SubscribeRaw(ctx, topic, func(m sdk.Message) {
			if len(m.Payload) > 0 {
				var data struct {
					Stolen string `json:"stolen"`
				}
				json.Unmarshal(m.Payload, &data)
				if data.Stolen == "super-secret-pw-123" {
					exfilDetected.Add(1)
				}
			}
		})
		defer u()
	}

	time.Sleep(500 * time.Millisecond)

	result, _ := secEvalTSErr(k, "__exfil.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Exfiltration attempts: %s", result)
	t.Logf("Exfiltrations detected: %d", exfilDetected.Load())
}

// testSecretEnvVarDump — deploy code that reads all environment variables.
func testSecretEnvVarDump(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "env-dump-sec.ts", `
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

	result, _ := secEvalTSErr(k, "__env_dump.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Env var access from .ts: %s", result)
}

// testSecretEnumeration — secrets.list to enumerate all secret names.
func testSecretEnumeration(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	store, _ := stores.NewSQLite(filepath.Join(tmpDir, "secrets-sec.db"))
	k, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store, SecretKey: "enum-test-key-32-characters!!",
	})
	require.NoError(t, err)
	defer k.Close()

	for _, name := range []string{"DB_PASSWORD_SEC", "API_KEY_SEC", "STRIPE_SECRET_SEC", "ADMIN_TOKEN_SEC"} {
		secSetSecret(t, k, name, "secret-"+name)
	}

	secDeploy(t, k, "enum-secrets-sec.ts", `
		var result = "UNKNOWN";
		try {
			var raw = __go_brainkit_request("secrets.list", "{}");
			result = "LISTED:" + raw;
		} catch(e) {
			result = "BLOCKED:" + (e.code || e.message);
		}
		output(result);
	`)

	result, _ := secEvalTSErr(k, "__enum.ts", `return String(globalThis.__module_result || "");`)
	t.Logf("Secret enumeration: %s", result)
}

// testSecretAuditEventSnooping — use audit events to learn when secrets are accessed.
func testSecretAuditEventSnooping(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secSetSecret(t, k, "MONITORED_KEY_SEC", "monitored-value")

	var auditMu sync.Mutex
	var auditEvents []string
	auditUnsub, _ := k.SubscribeRaw(ctx, "secrets.accessed", func(m sdk.Message) {
		auditMu.Lock()
		auditEvents = append(auditEvents, string(m.Payload))
		auditMu.Unlock()
	})
	defer auditUnsub()

	_ = secGetSecret(t, k, "MONITORED_KEY_SEC")

	time.Sleep(300 * time.Millisecond)

	auditMu.Lock()
	snapshot := append([]string(nil), auditEvents...)
	auditMu.Unlock()

	t.Logf("Audit events captured by eavesdropper: %d", len(snapshot))
	for _, e := range snapshot {
		t.Logf("  Audit: %s", e)
	}
}

// testSecretRotateDOS — rotate a secret that another deployment is using (denial of service).
func testSecretRotateDOS(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secSetSecret(t, k, "SHARED_KEY_SEC", "original-value")

	secDeploy(t, k, "victim-secret-sec.ts", `
		var key = secrets.get("SHARED_KEY_SEC");
		bus.on("check", function(msg) {
			var current = secrets.get("SHARED_KEY_SEC");
			msg.reply({original: key, current: current, match: key === current});
		});
	`)

	secRotateSecret(t, k, "SHARED_KEY_SEC", "rotated-by-attacker")

	pr3, _ := protocol.Publish(k, ctx, sdk.CustomMsg{
		Topic: "ts.victim-secret-sec.check", Payload: json.RawMessage(`{}`),
	})
	ch3 := make(chan []byte, 1)
	unsub3, _ := k.SubscribeRaw(ctx, pr3.ReplyTo, func(m sdk.Message) { ch3 <- m.Payload })
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

	store1, _ := stores.NewSQLite(storePath)
	k1, err := brainkit.New(brainkit.Config{
		Transport: brainkit.Memory(),
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
		Store: store1, SecretKey: "correct-key-32-characters-long!",
	})
	require.NoError(t, err)

	secSetSecret(t, k1, "encrypted-sec", "sensitive-data")
	k1.Close()

	wrongKeys := []string{
		"wrong-key-32-characters-long!!",
		"CORRECT-KEY-32-CHARACTERS-LONG!",
		"correct-key-32-characters-long",
		"",
	}

	for _, wrongKey := range wrongKeys {
		t.Run("key="+wrongKey[:secMin(10, len(wrongKey))], func(t *testing.T) {
			store2, _ := stores.NewSQLite(storePath)
			k2, err := brainkit.New(brainkit.Config{
				Transport: brainkit.Memory(),
				Namespace: "test", CallerID: "test", FSRoot: tmpDir,
				Store: store2, SecretKey: wrongKey,
			})
			require.NoError(t, err)
			defer k2.Close()
			secEnsureSecrets(t, k2)

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			resp, err := sdk.Call[secretmsg.SecretsGetMsg, secretmsg.SecretsGetResp](k2, ctx, secretmsg.SecretsGetMsg{Name: "encrypted-sec"})
			if err != nil {
				t.Logf("wrong key rejected secret read: %v", err)
				return
			}
			assert.NotEqual(t, "sensitive-data", resp.Value, "wrong key should not decrypt secret")
		})
	}
}
