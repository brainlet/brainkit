package security

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit"
	tools "github.com/brainlet/brainkit/internal/tools"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/sdk/messages"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLeakageErrorMessageContent — error messages reveal internal structure.
func testLeakageErrorMessageContent(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	sensitivePatterns := []string{
		"/Users/", "/home/", "/var/",
		"goroutine", "runtime.go",
		"password", "secret",
	}

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
			payload, ok := secSendAndReceive(t, k, tc.msg, 5*time.Second)
			if !ok {
				return
			}
			errStr := string(payload)
			for _, pattern := range sensitivePatterns {
				if strings.Contains(strings.ToLower(errStr), pattern) {
					t.Logf("FINDING: error %s leaks '%s': %s", tc.name, pattern, errStr[:secMin(200, len(errStr))])
				}
			}
		})
	}

	t.Run("fs-read-missing", func(t *testing.T) {
		result := secEvalTS(t, k, "__test.ts", `
			try { fs.readFileSync("/internal/config/secrets.json"); return "LEAKED"; }
			catch(e) { return e.code || "error"; }
		`)
		assert.NotEqual(t, "LEAKED", result)
		errStr := result
		for _, pattern := range sensitivePatterns {
			if strings.Contains(strings.ToLower(errStr), pattern) {
				t.Logf("FINDING: fs-read-missing leaks '%s': %s", pattern, errStr[:secMin(200, len(errStr))])
			}
		}
	})
}

// testLeakageSharedGlobalState — deployment A leaves data in globalThis that deployment B can find.
func testLeakageSharedGlobalState(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "leaker-a-sec.ts", `
		globalThis.leaked_secret = "password123";
		globalThis.__custom_data = {api_key: "sk-12345"};
		output("set");
	`)

	secDeploy(t, k, "reader-b-sec.ts", `
		var findings = {};
		findings.leaked_secret = typeof globalThis.leaked_secret !== "undefined" ? globalThis.leaked_secret : "NOT_FOUND";
		findings.custom_data = typeof globalThis.__custom_data !== "undefined" ? JSON.stringify(globalThis.__custom_data) : "NOT_FOUND";
		findings.kit_providers = typeof globalThis.__kit_providers !== "undefined" ? "VISIBLE" : "NOT_FOUND";
		findings.kit_compartments = typeof globalThis.__kit_compartments !== "undefined" ? "VISIBLE" : "NOT_FOUND";
		findings.bus_subs = typeof globalThis.__bus_subs !== "undefined" ? "VISIBLE" : "NOT_FOUND";
		findings.agent_embed = typeof globalThis.__agent_embed !== "undefined" ? "VISIBLE" : "NOT_FOUND";
		output(findings);
	`)

	result, _ := secEvalTSErr(k, "__leak.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Shared global state leakage: %s", result)
	assert.NotContains(t, result, "password123", "deployment B should not see A's globalThis writes")
	assert.NotContains(t, result, "sk-12345")
}

// testLeakageToolStateLeak — tool result contains data from a previous call.
func testLeakageToolStateLeak(t *testing.T, env *suite.TestEnv) {
	tmpDir := t.TempDir()
	k, err := brainkit.New(brainkit.Config{
		Namespace: "test", CallerID: "test", FSRoot: tmpDir,
	})
	require.NoError(t, err)
	defer k.Close()

	var lastInput string
	type leakyIn struct{ Data string `json:"data"` }
	brainkit.RegisterTool(k, "leaky-sec", tools.TypedTool[leakyIn]{
		Description: "returns previous caller's data",
		Execute: func(ctx context.Context, in leakyIn) (any, error) {
			prev := lastInput
			lastInput = in.Data
			return map[string]string{"previous": prev, "current": in.Data}, nil
		},
	})

	payload1, _ := secSendAndReceive(t, k, messages.ToolCallMsg{Name: "leaky-sec", Input: map[string]any{"data": "CALLER_A_SECRET"}}, 5*time.Second)
	_ = payload1

	payload2, ok := secSendAndReceive(t, k, messages.ToolCallMsg{Name: "leaky-sec", Input: map[string]any{"data": "CALLER_B"}}, 5*time.Second)
	require.True(t, ok)

	var resp struct{ Previous string `json:"previous"` }
	json.Unmarshal(payload2, &resp)
	if resp.Previous == "CALLER_A_SECRET" {
		t.Logf("FINDING: Go tool leaks previous caller's data (expected — Go state is shared)")
	}
}

// testLeakageMetadataLeak — bus message metadata reveals internal routing info.
func testLeakageMetadataLeak(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	secDeploy(t, k, "meta-leak-sec.ts", `
		bus.on("inspect", function(msg) {
			msg.reply({
				topic: msg.topic,
				replyTo: msg.replyTo,
				correlationId: msg.correlationId,
				callerId: msg.callerId,
				keys: Object.keys(msg),
			});
		});
	`)

	pr, _ := sdk.Publish(k, ctx, messages.CustomMsg{
		Topic: "ts.meta-leak-sec.inspect", Payload: json.RawMessage(`{}`),
	})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
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
	case <-ctx.Done():
		t.Fatal("timeout")
	}
}

// testLeakageSecretTimingSideChannel — timing side channel on secrets.get.
func testLeakageSecretTimingSideChannel(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit
	ctx := context.Background()

	pr, _ := sdk.Publish(k, ctx, messages.SecretsSetMsg{Name: "TIMING_KEY_SEC", Value: "secret-value"})
	ch := make(chan []byte, 1)
	unsub, _ := k.SubscribeRaw(ctx, pr.ReplyTo, func(m messages.Message) { ch <- m.Payload })
	<-ch
	unsub()

	var existsTimes []time.Duration
	for i := 0; i < 20; i++ {
		start := time.Now()
		result, err := secEvalTSErr(k, "__timing_exists.ts", `
			var val = secrets.get("TIMING_KEY_SEC");
			return val.length > 0 ? "found" : "empty";
		`)
		elapsed := time.Since(start)
		_ = result
		if err == nil {
			existsTimes = append(existsTimes, elapsed)
		}
	}

	var missingTimes []time.Duration
	for i := 0; i < 20; i++ {
		start := time.Now()
		result, err := secEvalTSErr(k, "__timing_missing.ts", `
			var val = secrets.get("NONEXISTENT_KEY_12345_SEC");
			return val.length > 0 ? "found" : "empty";
		`)
		elapsed := time.Since(start)
		_ = result
		if err == nil {
			missingTimes = append(missingTimes, elapsed)
		}
	}

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
	}
}

// testLeakageDeploymentReconnaissance — deployment list reveals what's running.
func testLeakageDeploymentReconnaissance(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	for _, name := range []string{"admin-panel-sec.ts", "stripe-webhook-sec.ts", "internal-api-sec.ts", "db-migration-sec.ts"} {
		secDeployErr(k, name, `output("running");`)
	}

	err := secDeployErr(k, "recon-sec.ts", `
		var raw = __go_brainkit_request("kit.list", "{}");
		var result = JSON.parse(raw);
		output(result);
	`)
	if err == nil {
		result, _ := secEvalTSErr(k, "__recon.ts", `
			var r = globalThis.__module_result;
			return JSON.stringify(r || {});
		`)
		if strings.Contains(result, "admin-panel") {
			t.Logf("FINDING: deployment can enumerate all other deployments via kit.list")
		}
	}

	err = secDeployErr(k, "tool-recon-sec.ts", `
		var raw = __go_brainkit_request("tools.list", "{}");
		output(JSON.parse(raw));
	`)
	if err == nil {
		result, _ := secEvalTSErr(k, "__tool_recon.ts", `
			var r = globalThis.__module_result;
			return JSON.stringify(r || {});
		`)
		t.Logf("Tool reconnaissance: %s", result[:secMin(200, len(result))])
	}
}

// testLeakageFilesystemReconnaissance — filesystem listing reveals workspace structure.
func testLeakageFilesystemReconnaissance(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secEvalTS(t, k, "__setup.ts", `
		fs.mkdirSync("config-sec", {recursive: true});
		fs.mkdirSync("secrets-sec", {recursive: true});
		fs.writeFileSync("config-sec/database.json", '{"host":"prod-db"}');
		fs.writeFileSync("secrets-sec/api-keys.txt", "sk-12345");
		fs.writeFileSync(".env-sec", "DB_PASSWORD=secret");
		return "ok";
	`)

	secDeploy(t, k, "fs-recon-sec.ts", `
		var results = {};
		try {
			var root = fs.readdirSync(".");
			results.root = root;
		} catch(e) { results.error = e.message; }

		try {
			var config = fs.readdirSync("config-sec");
			results.config = config;
		} catch(e) {}

		try {
			var secretsDir = fs.readdirSync("secrets-sec");
			results.secrets = secretsDir;
		} catch(e) {}

		try {
			results.envContent = fs.readFileSync(".env-sec", "utf8");
		} catch(e) {}

		try {
			results.apiKeysContent = fs.readFileSync("secrets-sec/api-keys.txt", "utf8");
		} catch(e) {}

		output(results);
	`)

	result, _ := secEvalTSErr(k, "__fs_recon.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("FS reconnaissance: %s", result)
	if strings.Contains(result, "DB_PASSWORD") {
		t.Logf("FINDING: .ts deployment can read .env and secrets from shared workspace")
	}
}

// testLeakageProviderReconnaissance — probe provider registry to learn what AI providers are configured.
func testLeakageProviderReconnaissance(t *testing.T, env *suite.TestEnv) {
	k := suite.Full(t).Kit

	secDeploy(t, k, "provider-recon-sec.ts", `
		var results = {};

		try {
			var providerList = registry.list("provider");
			results.providers = providerList;
		} catch(e) { results.providerError = e.message; }

		try {
			var storageList = registry.list("storage");
			results.storages = storageList;
		} catch(e) { results.storageError = e.message; }

		try {
			var vectorList = registry.list("vectorStore");
			results.vectors = vectorList;
		} catch(e) { results.vectorError = e.message; }

		try {
			var openai = registry.resolve("provider", "openai");
			results.openaiConfig = openai;
		} catch(e) { results.resolveError = e.message; }

		output(results);
	`)

	result, _ := secEvalTSErr(k, "__prov_recon.ts", `
		var r = globalThis.__module_result;
		return JSON.stringify(r || {});
	`)
	t.Logf("Provider recon: %s", result[:secMin(300, len(result))])
	if strings.Contains(result, "APIKey") || strings.Contains(result, "apiKey") {
		t.Logf("FINDING: registry.resolve exposes API keys to .ts deployments")
	}
}
