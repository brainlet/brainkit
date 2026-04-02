package adversarial_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/internal/registry"
	"github.com/brainlet/brainkit/internal/testutil"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ════════════════════════════════════════════════════════════════════════════
// DATA LEAKAGE
// Information leaks through side channels, shared state, error messages.
// ════════════════════════════════════════════════════════════════════════════

// Attack: error messages reveal internal structure (file paths, stack traces)
func TestDataLeakage_ErrorMessageContent(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	sensitivePatterns := []string{
		"/Users/", "/home/", "/var/", // absolute paths
		"goroutine", "runtime.go",     // Go stack traces
		"password", "secret",          // secrets in error messages
	}

	// Trigger various errors via bus and check what leaks in the message
	busErrors := []struct {
		name string
		msg  messages.BrainkitMessage
	}{
		{"tool-not-found", messages.ToolCallMsg{Name: "secret-internal-tool-name"}},
		{"agent-not-found", messages.AgentGetStatusMsg{Name: "internal-agent"}},
		{"deploy-bad", messages.KitDeployMsg{Source: "x.ts", Code: "throw new Error('DB_PASSWORD=secret123');"}},
	}

	for _, tc := range busErrors {
		t.Run(tc.name, func(t *testing.T) {
			payload, ok := sendAndReceive(t, tk, tc.msg, 5*time.Second)
			if !ok {
				return
			}
			errStr := string(payload)
			for _, pattern := range sensitivePatterns {
				if strings.Contains(strings.ToLower(errStr), pattern) {
					t.Logf("FINDING: error %s leaks '%s': %s", tc.name, pattern, errStr[:min(200, len(errStr))])
				}
			}
		})
	}

	// FS error via polyfill — reading a non-existent internal path should return an error, not internal info
	t.Run("fs-read-missing", func(t *testing.T) {
		result, err := tk.EvalTS(ctx, "__test.ts", `
			try { fs.readFileSync("/internal/config/secrets.json"); return "LEAKED"; }
			catch(e) { return e.code || "error"; }
		`)
		require.NoError(t, err)
		assert.NotEqual(t, "LEAKED", result)
		errStr := result
		for _, pattern := range sensitivePatterns {
			if strings.Contains(strings.ToLower(errStr), pattern) {
				t.Logf("FINDING: fs-read-missing leaks '%s': %s", pattern, errStr[:min(200, len(errStr))])
			}
		}
	})
}

// Attack: deployment A leaves data in globalThis that deployment B can find
func TestDataLeakage_SharedGlobalStateLeakage(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy A sets data on globalThis
	_, err := tk.Deploy(ctx, "leaker-a.ts", `
		// Try to set something on globalThis (inside Compartment)
		globalThis.leaked_secret = "password123";
		globalThis.__custom_data = {api_key: "sk-12345"};
		output("set");
	`)
	require.NoError(t, err)

	// Deploy B tries to read it
	_, err = tk.Deploy(ctx, "reader-b.ts", `
		var findings = {};
		findings.leaked_secret = typeof globalThis.leaked_secret !== "undefined" ? globalThis.leaked_secret : "NOT_FOUND";
		findings.custom_data = typeof globalThis.__custom_data !== "undefined" ? JSON.stringify(globalThis.__custom_data) : "NOT_FOUND";
		// Also check for data from the infrastructure
		findings.kit_providers = typeof globalThis.__kit_providers !== "undefined" ? "VISIBLE" : "NOT_FOUND";
		findings.kit_compartments = typeof globalThis.__kit_compartments !== "undefined" ? "VISIBLE" : "NOT_FOUND";
		findings.bus_subs = typeof globalThis.__bus_subs !== "undefined" ? "VISIBLE" : "NOT_FOUND";
		findings.agent_embed = typeof globalThis.__agent_embed !== "undefined" ? "VISIBLE" : "NOT_FOUND";
		output(findings);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__leak.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Shared global state leakage: %s", result)
	// Compartments should NOT see each other's globalThis writes
	assert.NotContains(t, result, "password123", "deployment B should not see A's globalThis writes")
	assert.NotContains(t, result, "sk-12345")
}

// Attack: tool result contains data from a previous call (stale state)
func TestDataLeakage_ToolStateLeak(t *testing.T) {
	tmpDir := t.TempDir()
	k, err := brainkit.NewKernel(brainkit.KernelConfig{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)
	defer k.Close()

	// Register a tool that captures state between calls
	var lastInput string
	type leakyIn struct{ Data string `json:"data"` }
	brainkit.RegisterTool(k, "leaky", registry.TypedTool[leakyIn]{
		Description: "returns previous caller's data",
		Execute: func(ctx context.Context, in leakyIn) (any, error) {
			prev := lastInput
			lastInput = in.Data
			return map[string]string{"previous": prev, "current": in.Data}, nil
		},
	})

	// Caller A sends sensitive data
	payload1, _ := sendAndReceive(t, k, messages.ToolCallMsg{Name: "leaky", Input: map[string]any{"data": "CALLER_A_SECRET"}}, 5*time.Second)
	_ = payload1

	// Caller B calls the same tool — does it see A's data?
	payload2, ok := sendAndReceive(t, k, messages.ToolCallMsg{Name: "leaky", Input: map[string]any{"data": "CALLER_B"}}, 5*time.Second)
	require.True(t, ok)

	var resp struct{ Previous string `json:"previous"` }
	json.Unmarshal(payload2, &resp)
	if resp.Previous == "CALLER_A_SECRET" {
		t.Logf("FINDING: Go tool leaks previous caller's data (expected — Go state is shared)")
		// This is a Go-developer responsibility, not a brainkit bug.
		// But worth documenting: Go tools share state between callers.
	}
}

// Attack: bus message metadata reveals internal routing info
func TestDataLeakage_MetadataLeak(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Deploy handler that returns ALL metadata it receives
	_, err := tk.Deploy(ctx, "meta-leak.ts", `
		bus.on("inspect", function(msg) {
			msg.reply({
				topic: msg.topic,
				replyTo: msg.replyTo,
				correlationId: msg.correlationId,
				callerId: msg.callerId,
				// Dump all msg properties
				keys: Object.keys(msg),
			});
		});
	`)
	require.NoError(t, err)

	pr, _ := sdk.Publish(tk, ctx, messages.CustomMsg{
		Topic: "ts.meta-leak.inspect", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	defer unsub()

	select {
	case p := <-ch:
		var resp struct {
			Topic         string   `json:"topic"`
			ReplyTo       string   `json:"replyTo"`
			CorrelationId string   `json:"correlationId"`
			CallerId      string   `json:"callerId"`
			Keys          []string `json:"keys"`
		}
		json.Unmarshal(p, &resp)
		t.Logf("Message metadata visible to handler: keys=%v, callerId=%s", resp.Keys, resp.CallerId)
		// callerID reveals who sent the message — is this a problem?
		// replyTo reveals the topic pattern — useful for eavesdropping
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// Attack: timing side channel on secrets.get (exists vs doesn't exist)
func TestDataLeakage_SecretTimingSideChannel(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Set a secret
	pr, _ := sdk.Publish(tk, ctx, messages.SecretsSetMsg{Name: "TIMING_KEY", Value: "secret-value"})
	ch := make(chan []byte, 1)
	unsub, _ := tk.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()

	// Time 100 requests for existing key
	var existsTimes []time.Duration
	for i := 0; i < 20; i++ {
		start := time.Now()
		result, err := tk.EvalTS(ctx, "__timing_exists.ts", `
			var val = secrets.get("TIMING_KEY");
			return val.length > 0 ? "found" : "empty";
		`)
		elapsed := time.Since(start)
		_ = result
		if err == nil {
			existsTimes = append(existsTimes, elapsed)
		}
	}

	// Time 100 requests for non-existing key
	var missingTimes []time.Duration
	for i := 0; i < 20; i++ {
		start := time.Now()
		result, err := tk.EvalTS(ctx, "__timing_missing.ts", `
			var val = secrets.get("NONEXISTENT_KEY_12345");
			return val.length > 0 ? "found" : "empty";
		`)
		elapsed := time.Since(start)
		_ = result
		if err == nil {
			missingTimes = append(missingTimes, elapsed)
		}
	}

	// Compare averages — significant difference enables enumeration
	if len(existsTimes) > 0 && len(missingTimes) > 0 {
		var avgExists, avgMissing time.Duration
		for _, d := range existsTimes {
			avgExists += d
		}
		avgExists /= time.Duration(len(existsTimes))
		for _, d := range missingTimes {
			avgMissing += d
		}
		avgMissing /= time.Duration(len(missingTimes))

		t.Logf("Secret timing: exists=%v, missing=%v, diff=%v", avgExists, avgMissing, avgExists-avgMissing)
		// If there's a >2x timing difference, it's exploitable
	}
}

// Attack: deployment list reveals what's running (reconnaissance)
func TestDataLeakage_DeploymentReconnaissance(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Deploy several services with sensitive names
	for _, name := range []string{"admin-panel.ts", "stripe-webhook.ts", "internal-api.ts", "db-migration.ts"} {
		tk.Deploy(ctx, name, `output("running");`)
	}

	// Attacker enumerates via kit.list
	_, err := tk.Deploy(ctx, "recon.ts", `
		var raw = __go_brainkit_request("kit.list", "{}");
		var result = JSON.parse(raw);
		output(result);
	`)
	// If bridge is available in compartment, attacker can list all deployments
	if err == nil {
		result, _ := tk.EvalTS(ctx, "__recon.ts", `
			var r = globalThis.__module_result;
			return JSON.stringify(r || {});
		`)
		if strings.Contains(result, "admin-panel") {
			t.Logf("FINDING: deployment can enumerate all other deployments via kit.list")
		}
	}

	// Also check tools.list
	_, err = tk.Deploy(ctx, "tool-recon.ts", `
		var raw = __go_brainkit_request("tools.list", "{}");
		output(JSON.parse(raw));
	`)
	if err == nil {
		result, _ := tk.EvalTS(ctx, "__tool_recon.ts", `
			var r = globalThis.__module_result;
			return JSON.stringify(r || {});
		`)
		t.Logf("Tool reconnaissance: %s", result[:min(200, len(result))])
	}
}

// Attack: filesystem listing reveals workspace structure
func TestDataLeakage_FilesystemReconnaissance(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	// Create some files via polyfill
	_, err := tk.EvalTS(ctx, "__setup.ts", `
		fs.mkdirSync("config", {recursive: true});
		fs.mkdirSync("secrets", {recursive: true});
		fs.writeFileSync("config/database.json", '{"host":"prod-db"}');
		fs.writeFileSync("secrets/api-keys.txt", "sk-12345");
		fs.writeFileSync(".env", "DB_PASSWORD=secret");
		return "ok";
	`)
	require.NoError(t, err)

	// Deployment lists everything
	_, err = tk.Deploy(ctx, "fs-recon.ts", `
		var results = {};
		try {
			var root = fs.readdirSync(".");
			results.root = root;
		} catch(e) { results.error = e.message; }

		try {
			var config = fs.readdirSync("config");
			results.config = config;
		} catch(e) {}

		try {
			var secretsDir = fs.readdirSync("secrets");
			results.secrets = secretsDir;
		} catch(e) {}

		// Try to read sensitive files
		try {
			results.envContent = fs.readFileSync(".env", "utf8");
		} catch(e) {}

		try {
			results.apiKeysContent = fs.readFileSync("secrets/api-keys.txt", "utf8");
		} catch(e) {}

		output(results);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__fs_recon.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("FS reconnaissance: %s", result)
	// Every deployment CAN list and read ALL files in the workspace
	// There's no per-deployment filesystem isolation
	if strings.Contains(result, "DB_PASSWORD") {
		t.Logf("FINDING: .ts deployment can read .env and secrets from shared workspace")
	}
}

// Attack: probe provider registry to learn what AI providers are configured
func TestDataLeakage_ProviderReconnaissance(t *testing.T) {
	tk := testutil.NewTestKernelFull(t)
	ctx := context.Background()

	_, err := tk.Deploy(ctx, "provider-recon.ts", `
		var results = {};

		// List all providers
		try {
			var providerList = registry.list("provider");
			results.providers = providerList;
		} catch(e) { results.providerError = e.message; }

		// List all storages
		try {
			var storageList = registry.list("storage");
			results.storages = storageList;
		} catch(e) { results.storageError = e.message; }

		// List all vector stores
		try {
			var vectorList = registry.list("vectorStore");
			results.vectors = vectorList;
		} catch(e) { results.vectorError = e.message; }

		// Try to resolve a provider to get its config (API key?)
		try {
			var openai = registry.resolve("provider", "openai");
			results.openaiConfig = openai;
		} catch(e) { results.resolveError = e.message; }

		output(results);
	`)
	require.NoError(t, err)

	result, _ := tk.EvalTS(ctx, "__prov_recon.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Provider recon: %s", result[:min(300, len(result))])
	// Check if API keys leak via registry.resolve
	if strings.Contains(result, "APIKey") || strings.Contains(result, "apiKey") {
		t.Logf("FINDING: registry.resolve exposes API keys to .ts deployments")
	}
}
