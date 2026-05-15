package packages

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/testutil"
	browsermod "github.com/brainlet/brainkit/modules/browser"
	"github.com/brainlet/brainkit/modules/packages/packagemsg"
	"github.com/brainlet/brainkit/modules/registry/registrymsg"
	"github.com/brainlet/brainkit/modules/secrets/secretmsg"
	"github.com/brainlet/brainkit/sdk"
	"github.com/brainlet/brainkit/test/suite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writePackageFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(path), 0755)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

func testMultiFileProject(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t, suite.WithPersistence(), suite.WithSecretKey("test-key"))
	ctx := context.Background()

	dir := t.TempDir()
	writePackageFile(t, dir, "manifest.json", `{
		"name": "test-pkg",
		"version": "1.0.0",
		"entry": "index.ts"
	}`)
	writePackageFile(t, dir, "config.ts", `export const PREFIX = "Hello";`)
	writePackageFile(t, dir, "index.ts", `
		import { PREFIX } from "./config";
		bus.on("greet", (msg) => {
			msg.reply({ text: PREFIX + " " + msg.payload.name });
		});
	`)

	deployResp, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{Path: dir}, sdk.WithCallTimeout(10*time.Second))
	require.NoError(t, err)
	require.True(t, deployResp.Deployed)
	assert.Equal(t, "test-pkg", deployResp.Name)
	assert.Equal(t, "test-pkg.ts", deployResp.Source)

	time.Sleep(200 * time.Millisecond)

	serviceResp := sendToServiceAndWait(t, env.Kit, "test-pkg", "greet", map[string]string{"name": "World"})
	assert.Equal(t, "Hello World", serviceResp["text"])
}

func testListAndTeardown(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t, suite.WithPersistence(), suite.WithSecretKey("test-key"))
	ctx := context.Background()

	dir := t.TempDir()
	writePackageFile(t, dir, "manifest.json", `{
		"name": "list-test",
		"version": "2.0.0",
		"entry": "index.ts"
	}`)
	writePackageFile(t, dir, "index.ts", `bus.on("ping", (msg) => { msg.reply({pong: true}); });`)

	deployResp, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{Path: dir}, sdk.WithCallTimeout(10*time.Second))
	require.NoError(t, err)
	require.True(t, deployResp.Deployed)

	listResp, err := packagemsg.CallPackageListDeployed(env.Kit, ctx, packagemsg.PackageListDeployedMsg{}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	require.Len(t, listResp.Packages, 1)
	assert.Equal(t, "list-test", listResp.Packages[0].Name)
	assert.Equal(t, "list-test.ts", listResp.Packages[0].Source)

	tearResp, err := packagemsg.CallPackageTeardown(env.Kit, ctx, packagemsg.PackageTeardownMsg{Name: "list-test"}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	assert.True(t, tearResp.Removed)

	listResp2, err := packagemsg.CallPackageListDeployed(env.Kit, ctx, packagemsg.PackageListDeployedMsg{}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	assert.Len(t, listResp2.Packages, 0)
}

func testSecretDependencyCheck(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t, suite.WithPersistence(), suite.WithSecretKey("test-key"))
	ctx := context.Background()

	dir := t.TempDir()
	writePackageFile(t, dir, "manifest.json", `{
		"name": "needs-secret",
		"version": "1.0.0",
		"entry": "index.ts",
		"requires": { "secrets": ["MY_REQUIRED_SECRET"] }
	}`)
	writePackageFile(t, dir, "index.ts", `bus.on("x", (msg) => { msg.reply({}); });`)

	_, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{Path: dir}, sdk.WithCallTimeout(5*time.Second))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MY_REQUIRED_SECRET")
	assert.Contains(t, err.Error(), "not set")

	setResp, err := secretmsg.CallSecretsSet(env.Kit, ctx, secretmsg.SecretsSetMsg{Name: "MY_REQUIRED_SECRET", Value: "secret-value"}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)
	assert.True(t, setResp.Stored)

	deployResp, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{Path: dir}, sdk.WithCallTimeout(10*time.Second))
	require.NoError(t, err)
	assert.True(t, deployResp.Deployed)
}

func testInlineFilesRedeployPicksUpNewCode(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t, suite.WithPersistence())
	ctx := context.Background()

	manifest := `{
		"name": "evolve-pkg",
		"version": "1.0.0",
		"entry": "index.ts"
	}`

	v1Code := `bus.on("check", (msg) => { msg.reply({ version: "v1" }); });`

	resp, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{
		Manifest: json.RawMessage(manifest),
		Files:    map[string]string{"index.ts": v1Code},
	}, sdk.WithCallTimeout(10*time.Second))
	require.NoError(t, err)
	require.True(t, resp.Deployed)
	t.Logf("v1 deployed: %s", resp.Source)

	time.Sleep(200 * time.Millisecond)

	v1Resp := sendToServiceAndWait(t, env.Kit, "evolve-pkg", "check", nil)
	require.Equal(t, "v1", v1Resp["version"], "v1 should return version=v1")

	v2Code := `bus.on("check", (msg) => { msg.reply({ version: "v2" }); });`

	resp2, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{
		Manifest: json.RawMessage(manifest),
		Files:    map[string]string{"index.ts": v2Code},
	}, sdk.WithCallTimeout(10*time.Second))
	require.NoError(t, err)
	require.True(t, resp2.Deployed)

	time.Sleep(200 * time.Millisecond)

	v2Resp := sendToServiceAndWait(t, env.Kit, "evolve-pkg", "check", nil)
	require.Equal(t, "v2", v2Resp["version"], "REDEPLOY BUG: v2 should return version=v2 but got %v", v2Resp["version"])
}

func testTopicCollision(t *testing.T, _ *suite.TestEnv) {
	env := suite.Full(t, suite.WithPersistence())
	ctx := context.Background()

	dir := t.TempDir()
	writePackageFile(t, dir, "manifest.json", `{
		"name": "collision-test",
		"version": "1.0.0",
		"entry": "index.ts"
	}`)
	writePackageFile(t, dir, "index.ts", `
		bus.on("greet", (msg) => { msg.reply({ from: "first" }); });
		bus.on("greet", (msg) => { msg.reply({ from: "second" }); });
	`)

	_, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{Path: dir}, sdk.WithCallTimeout(10*time.Second))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already subscribed")
}

func testPackageAIStreamTextLocalOpenAI(t *testing.T, _ *suite.TestEnv) {
	chunks := []string{"One", "\nTwo", "\nThree"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/responses" && r.URL.Path != "/v1/responses" &&
			r.URL.Path != "/chat/completions" && r.URL.Path != "/v1/chat/completions" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]any
		require.NoError(t, json.Unmarshal(body, &req))
		require.Equal(t, true, req["stream"])

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(http.StatusOK)
		if strings.HasSuffix(r.URL.Path, "/responses") {
			events := []map[string]any{
				{"type": "response.created", "response": map[string]any{
					"id": "resp-brainkit-stream", "created_at": 1234567890, "model": "gpt-4o-mini",
				}},
				{"type": "response.output_item.added", "output_index": 0, "item": map[string]any{
					"type": "message", "id": "msg_0",
				}},
			}
			for _, event := range events {
				raw, _ := json.Marshal(event)
				fmt.Fprintf(w, "data: %s\n\n", raw)
			}
			for _, chunk := range chunks {
				raw, _ := json.Marshal(map[string]any{
					"type":    "response.output_text.delta",
					"item_id": "msg_0",
					"delta":   chunk,
				})
				fmt.Fprintf(w, "data: %s\n\n", raw)
			}
			done, _ := json.Marshal(map[string]any{
				"type":         "response.output_item.done",
				"output_index": 0,
				"item":         map[string]any{"type": "message", "id": "msg_0"},
			})
			fmt.Fprintf(w, "data: %s\n\n", done)
			completed, _ := json.Marshal(map[string]any{
				"type": "response.completed",
				"response": map[string]any{
					"usage": map[string]any{
						"input_tokens":  4,
						"output_tokens": 3,
					},
				},
			})
			fmt.Fprintf(w, "data: %s\n\n", completed)
			fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}
		for i, chunk := range chunks {
			finishReason := any(nil)
			if i == len(chunks)-1 {
				finishReason = "stop"
			}
			data := map[string]any{
				"id":      "chatcmpl-brainkit-stream",
				"object":  "chat.completion.chunk",
				"created": 1234567890,
				"model":   "gpt-4o-mini",
				"choices": []map[string]any{{
					"index":         0,
					"delta":         map[string]any{"content": chunk},
					"finish_reason": finishReason,
				}},
			}
			raw, _ := json.Marshal(data)
			fmt.Fprintf(w, "data: %s\n\n", raw)
		}
		fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":4,\"completion_tokens\":3,\"total_tokens\":7}}\n\n")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer srv.Close()

	env := suite.Full(t, suite.WithPersistence(), suite.WithSecretKey("test-key"))
	ctx := context.Background()

	_, err := registrymsg.CallProviderAdd(env.Kit, ctx, registrymsg.ProviderAddMsg{
		Name:   "openai",
		Type:   "openai",
		Config: json.RawMessage(fmt.Sprintf(`{"apiKey":"test-key","baseURL":%q}`, srv.URL+"/v1")),
	}, sdk.WithCallTimeout(5*time.Second))
	require.NoError(t, err)

	manifest := `{"name":"local-ai-stream","version":"0.0.0","entry":"index.ts"}`
	source := `
		import { streamText } from "ai";
		import { model, output } from "kit";

		async function withTimeout(label, promise) {
			let timer;
			try {
				return await Promise.race([
					promise,
					new Promise((_, reject) => {
						timer = setTimeout(() => reject(new Error(label + " timed out")), 5000);
					}),
				]);
			} finally {
				clearTimeout(timer);
			}
		}

		const partialChunks = [];
		async function collectTextStream(result) {
			for await (const chunk of result.textStream) {
				partialChunks.push(chunk);
			}
			return partialChunks;
		}

		try {
			const result = streamText({
				model: model("openai", "gpt-4o-mini"),
				prompt: "Count from 1 to 3.",
			});
			const chunks = await withTimeout("textStream", collectTextStream(result));
			const text = await withTimeout("result.text", result.text);
			const usage = await withTimeout("result.usage", result.usage);
			output({
				text,
				chunks,
				usage,
				hasRealTimeTokens: chunks.length > 0,
			});
		} catch (err) {
			output({
				error: err && err.message ? err.message : String(err),
				partialChunks,
				stack: err && err.stack ? String(err.stack).split("\n").slice(0, 8) : [],
			});
		}
	`
	resp, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{
		Manifest: json.RawMessage(manifest),
		Files:    map[string]string{"index.ts": source},
	}, sdk.WithCallTimeout(20*time.Second))
	require.NoError(t, err)
	require.True(t, resp.Deployed)

	raw := testutil.EvalJS(t, env.Kit, "__read_local_ai_stream.ts",
		`return typeof globalThis.__module_result !== "undefined" ? globalThis.__module_result : ""`)
	var actual map[string]any
	require.NoError(t, json.Unmarshal([]byte(raw), &actual), raw)
	require.Empty(t, actual["error"], "stream output: %#v", actual)
	require.Equal(t, "One\nTwo\nThree", actual["text"])
	require.Equal(t, true, actual["hasRealTimeTokens"])
	require.Len(t, actual["chunks"], 3)
}

func testNPMPreviewStagehandPackageImport(t *testing.T, _ *suite.TestEnv) {
	if os.Getenv("BRAINKIT_TEST_MASTRA_BROWSER_NPM_PREVIEW") != "1" {
		t.Skip("set BRAINKIT_TEST_MASTRA_BROWSER_NPM_PREVIEW=1 to run real Mastra browser-provider npm-preview deploy canary")
	}
	if _, err := exec.LookPath("pnpm"); err != nil {
		t.Skipf("pnpm not available: %v", err)
	}

	env := suite.Full(t, suite.WithPersistence(), suite.WithSecretKey("test-key"))
	ctx := context.Background()

	dir := t.TempDir()
	writePackageFile(t, dir, "package.json", `{
		"name": "brainkit-stagehand-package-import",
		"private": true,
		"type": "module",
		"dependencies": {
			"@mastra/stagehand": "0.2.2",
			"@mastra/core": "1.33.0",
			"zod": "3.25.76"
		}
	}`)
	writePackageFile(t, dir, "manifest.json", `{
		"name": "stagehand-package-import",
		"version": "0.0.0",
		"entry": "index.ts",
		"resolver": "npm-preview"
	}`)
	writePackageFile(t, dir, "index.ts", `
		import { StagehandBrowser } from "@mastra/stagehand";

		bus.on("inspect", (msg) => {
			msg.reply({
				importedType: typeof StagehandBrowser,
				exportName: StagehandBrowser && StagehandBrowser.name,
			});
		});
	`)

	lockCmd := exec.Command("pnpm", "install", "--lockfile-only", "--ignore-scripts")
	lockCmd.Dir = dir
	lockCmd.Env = append(os.Environ(), "CI=1")
	if out, err := lockCmd.CombinedOutput(); err != nil {
		t.Fatalf("pnpm lockfile: %v\n%s", err, strings.TrimSpace(string(out)))
	}

	deployResp, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{Path: dir}, sdk.WithCallTimeout(6*time.Minute))
	require.NoError(t, err)
	require.True(t, deployResp.Deployed)

	time.Sleep(200 * time.Millisecond)

	serviceResp := sendToServiceAndWait(t, env.Kit, "stagehand-package-import", "inspect", nil)
	assert.Equal(t, "function", serviceResp["importedType"])
	assert.Equal(t, "StagehandBrowser", serviceResp["exportName"])
}

func testNPMPreviewStagehandLocalBrowserLifecycle(t *testing.T, env *suite.TestEnv) {
	testutil.LoadEnv(t)
	if os.Getenv("BRAINKIT_TEST_MASTRA_BROWSER_NPM_PREVIEW") != "1" {
		t.Skip("set BRAINKIT_TEST_MASTRA_BROWSER_NPM_PREVIEW=1 to run real Mastra browser-provider npm-preview deploy canary")
	}
	if !testutil.LiveAIEnabled() {
		t.Skip("set BRAINKIT_TEST_LIVE_AI=1 to run real Stagehand provider behavior with live OpenAI credentials")
	}
	if _, ok := testutil.OpenAIKey(); !ok {
		t.Skip("OPENAI_API_KEY is required for real Stagehand provider behavior")
	}
	if _, err := exec.LookPath("pnpm"); err != nil {
		t.Skipf("pnpm not available: %v", err)
	}

	ctx := context.Background()
	if _, mounted := env.Kit.Module("browser"); !mounted {
		require.NoError(t, env.Kit.Mount(ctx, browsermod.New(browsermod.Config{Headless: true})))
		t.Cleanup(func() {
			unmountCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			require.NoError(t, env.Kit.Unmount(unmountCtx, "browser"))
		})
	}

	dir := t.TempDir()
	writePackageFile(t, dir, "package.json", `{
		"name": "brainkit-stagehand-local-browser",
		"private": true,
		"type": "module",
		"dependencies": {
			"@mastra/stagehand": "0.2.2",
			"@mastra/core": "1.33.0",
			"zod": "3.25.76"
		}
	}`)
	writePackageFile(t, dir, "manifest.json", `{
		"name": "stagehand-local-browser",
		"version": "0.0.0",
		"entry": "index.ts",
		"resolver": "npm-preview"
	}`)
	writePackageFile(t, dir, "index.ts", `
		import { StagehandBrowser } from "@mastra/stagehand";
		import { browser } from "kit";

		function pageURL() {
			const html = [
				"<!doctype html>",
				"<html>",
				"<head><title>Brainkit Stagehand Fixture</title></head>",
				"<body>",
				"<h1 id=\"headline\">Brainkit Stagehand Fixture</h1>",
				"<button id=\"ready\" onclick=\"document.body.dataset.clicked='yes';document.getElementById('headline').textContent='Clicked by Stagehand';\">Mark ready</button>",
				"<p id=\"summary\">The fixture color is blue and the count is 42.</p>",
				"</body>",
				"</html>",
			].join("");
			return "data:text/html;charset=utf-8," + encodeURIComponent(html);
		}

		function cleanError(err) {
			return {
				name: err && err.name ? String(err.name) : "Error",
				message: err && err.message ? String(err.message) : String(err),
				stack: err && err.stack ? String(err.stack).split("\n").slice(0, 8) : [],
			};
		}

		function isErrorResult(value) {
			return !!(value && typeof value === "object" && "error" in value);
		}

		bus.on("exercise", async (msg) => {
			let session;
			let stagehand;
			const result = { ok: false };
			try {
				session = await browser.launch({
					provider: "stagehand",
					scope: "shared",
					headless: true,
				});
				stagehand = new StagehandBrowser({
					cdpUrl: session.cdpUrl,
					scope: "shared",
					headless: true,
					model: "openai/gpt-4o-mini",
					verbose: 2,
					selfHeal: false,
					domSettleTimeout: 1000,
				});
				await stagehand.ensureReady();

				const nav = await stagehand.navigate({
					url: pageURL(),
					waitUntil: "domcontentloaded",
				});
				const tabs = await stagehand.tabs({ action: "list" });
				const screenshot = await stagehand.screenshot({ fullPage: false });
				const observe = await stagehand.observe({ instruction: "Find the Mark ready button" });
				const extract = await stagehand.extract({
					instruction: "Extract the headline text and the count from the page.",
				});
				const instructionAct = await stagehand.act({ instruction: "Click the Mark ready button" });
				let act = instructionAct;
				let nativeAct = null;
				if (!act || act.success !== true) {
					try {
						const nativeStagehand = stagehand.requireStagehand();
						const observedAction = observe && observe.actions && observe.actions[0];
						if (!observedAction) {
							throw new Error("observe did not return an action to execute");
						}
						nativeAct = await nativeStagehand.act(observedAction, {
							page: stagehand.getPage(),
							timeout: 30000,
						});
						act = {
							success: nativeAct && nativeAct.success === true,
							message: nativeAct && nativeAct.message,
							action: (nativeAct && (nativeAct.actionDescription || nativeAct.action)) || observedAction.description,
							url: stagehand.getPage().url(),
							hint: "Executed observed action through Stagehand v3 act(action).",
							fallback: "observed-action",
						};
					} catch (err) {
						nativeAct = { error: cleanError(err) };
					}
				}
				const after = await stagehand.extract({
					instruction: "Extract the current headline text from the page.",
				});

				result.ok = true;
				result.session = {
					id: session.id,
					cdpUrl: session.cdpUrl,
					webSocketDebuggerUrl: session.webSocketDebuggerUrl,
				};
				result.nav = nav;
				result.tabs = tabs;
				result.screenshot = {
					base64Length: screenshot && screenshot.base64 ? screenshot.base64.length : 0,
					url: screenshot && screenshot.url,
					title: screenshot && screenshot.title,
					error: isErrorResult(screenshot) ? screenshot.error : undefined,
				};
				result.observe = observe;
				result.extract = extract;
				result.nativeAct = nativeAct;
				result.instructionAct = instructionAct;
				result.act = act;
				result.after = after;
			} catch (err) {
				result.error = cleanError(err);
			} finally {
				if (stagehand) {
					try {
						await stagehand.close();
					} catch (err) {
						result.stagehandCloseError = cleanError(err);
					}
				}
				if (session) {
					try {
						await browser.close(session.id);
					} catch (err) {
						result.browserCloseError = cleanError(err);
					}
				}
				try {
					result.remainingSessions = (await browser.list()).length;
				} catch (err) {
					result.browserListError = cleanError(err);
				}
			}
			msg.reply(result);
		});
	`)

	lockCmd := exec.Command("pnpm", "install", "--lockfile-only", "--ignore-scripts")
	lockCmd.Dir = dir
	lockCmd.Env = append(os.Environ(), "CI=1")
	if out, err := lockCmd.CombinedOutput(); err != nil {
		t.Fatalf("pnpm lockfile: %v\n%s", err, strings.TrimSpace(string(out)))
	}

	deployResp, err := packagemsg.CallPackageDeploy(env.Kit, ctx, packagemsg.PackageDeployMsg{Path: dir}, sdk.WithCallTimeout(6*time.Minute))
	require.NoError(t, err)
	require.True(t, deployResp.Deployed)

	time.Sleep(200 * time.Millisecond)

	serviceResp := sendToServiceAndWaitTimeout(t, env.Kit, "stagehand-local-browser", "exercise", nil, 3*time.Minute)
	require.Equal(t, true, serviceResp["ok"], "Stagehand exercise failed: %#v", serviceResp)
	require.Empty(t, serviceResp["error"], "Stagehand exercise error: %#v", serviceResp["error"])
	require.Equal(t, float64(0), serviceResp["remainingSessions"], "browser session leaked: %#v", serviceResp)

	session, ok := serviceResp["session"].(map[string]any)
	require.True(t, ok, "session shape: %#v", serviceResp["session"])
	require.NotEmpty(t, session["id"])
	require.Contains(t, session["cdpUrl"], "http://127.0.0.1:")

	nav, ok := serviceResp["nav"].(map[string]any)
	require.True(t, ok, "nav shape: %#v", serviceResp["nav"])
	require.Equal(t, true, nav["success"], "navigate result: %#v", nav)
	require.Equal(t, "Brainkit Stagehand Fixture", nav["title"])

	screenshot, ok := serviceResp["screenshot"].(map[string]any)
	require.True(t, ok, "screenshot shape: %#v", serviceResp["screenshot"])
	require.Greater(t, screenshot["base64Length"], float64(100), "screenshot result: %#v", screenshot)
	require.Empty(t, screenshot["error"], "screenshot error: %#v", screenshot)

	act, ok := serviceResp["act"].(map[string]any)
	require.True(t, ok, "act shape: %#v", serviceResp["act"])
	require.Equal(t, true, act["success"], "act result: %#v nativeAct: %#v", act, serviceResp["nativeAct"])

	assertStagehandTextContains(t, serviceResp["extract"], []string{"Brainkit Stagehand Fixture", "42"})
	assertStagehandTextContains(t, serviceResp["after"], []string{"Clicked by Stagehand"})
}

func sendToServiceAndWait(t *testing.T, k interface {
	sdk.Runtime
	sdk.CallerRuntime
}, service, topic string, payload any) map[string]any {
	t.Helper()
	return sendToServiceAndWaitTimeout(t, k, service, topic, payload, 10*time.Second)
}

func sendToServiceAndWaitTimeout(t *testing.T, k interface {
	sdk.Runtime
	sdk.CallerRuntime
}, service, topic string, payload any, timeout time.Duration) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	data, err := json.Marshal(payload)
	require.NoError(t, err)
	resp, err := sdk.Call[sdk.CustomMsg, map[string]any](k, ctx, sdk.CustomMsg{
		Topic:   packageServiceTopic(service, topic),
		Payload: data,
	}, sdk.WithCallTimeout(timeout))
	require.NoError(t, err)
	return resp
}

func assertStagehandTextContains(t *testing.T, value any, needles []string) {
	t.Helper()
	require.NotNil(t, value, "stagehand result is nil")
	encoded, err := json.Marshal(value)
	require.NoError(t, err)
	text := string(encoded)
	for _, needle := range needles {
		require.Contains(t, text, needle, "stagehand result: %s", text)
	}
	require.NotContains(t, text, `"error"`, "stagehand result contained error: %s", text)
}

func packageServiceTopic(service, topic string) string {
	name := strings.TrimSuffix(service, ".ts")
	name = strings.ReplaceAll(name, "/", ".")
	return "ts." + name + "." + topic
}
