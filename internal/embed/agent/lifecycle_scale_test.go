package agentembed

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	quickjs "github.com/buke/quickjs-go"
)

func TestAgentGenerateConcurrentFakeModelCallsStayAsync(t *testing.T) {
	client := NewClient(ClientConfig{})
	sandbox, err := client.CreateSandbox(SandboxConfig{})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	defer sandbox.Close()

	result, err := sandbox.Eval(context.Background(), "agent-generate-concurrent.js", `
		(async function() {
			var embed = globalThis.__agent_embed;
			var active = 0;
			var maxActive = 0;
			function sleep(ms) { return new Promise((resolve) => setTimeout(resolve, ms)); }
			var fakeModel = {
				specificationVersion: "v3",
				provider: "brainkit.fake",
				modelId: "concurrent",
				supportedUrls: {},
				doGenerate: async function() {
					active++;
					if (active > maxActive) maxActive = active;
					try {
						await sleep(120);
						return {
							content: [{ type: "text", text: "ok" }],
							finishReason: { unified: "stop" },
							usage: {
								inputTokens: { total: 0, noCache: 0, cacheRead: 0, cacheWrite: 0 },
								outputTokens: { total: 0, text: 0, reasoning: 0 },
							},
							warnings: [],
							request: { body: {} },
							response: { id: "fake", timestamp: new Date(0), modelId: "concurrent" },
						};
					} finally {
						active--;
					}
				},
				doStream: async function() {
					throw new Error("doStream should not be called");
				},
			};
			var agent = new embed.Agent({
				id: "concurrent-agent",
				name: "concurrent-agent",
				model: fakeModel,
				instructions: "Return ok.",
			});
			var started = Date.now();
			var results = await Promise.all(Array.from({ length: 8 }, function(_, i) {
				return agent.generate("prompt-" + i);
			}));
			return JSON.stringify({
				elapsedMs: Date.now() - started,
				maxActive: maxActive,
				count: results.length,
				text: results.map(function(r) { return r.text || ""; }).join(",")
			});
		})()
	`)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	var got struct {
		ElapsedMs int    `json:"elapsedMs"`
		MaxActive int    `json:"maxActive"`
		Count     int    `json:"count"`
		Text      string `json:"text"`
	}
	if err := json.Unmarshal([]byte(result), &got); err != nil {
		t.Fatalf("parse result %q: %v", result, err)
	}
	if got.Count != 8 {
		t.Fatalf("count = %d, want 8; full result=%s", got.Count, result)
	}
	if got.MaxActive < 2 {
		t.Fatalf("maxActive = %d, want >= 2; full result=%s", got.MaxActive, result)
	}
	if got.ElapsedMs >= 1000 {
		t.Fatalf("elapsedMs = %d, want < 1000; full result=%s", got.ElapsedMs, result)
	}
}

func TestSandboxCloseContextCancelsPendingAgentGenerate(t *testing.T) {
	client := NewClient(ClientConfig{})
	sandbox, err := client.CreateSandbox(SandboxConfig{})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}

	entered := make(chan struct{})
	var enteredOnce sync.Once
	qctx := sandbox.bridge.Context()
	qctx.Globals().Set("__test_provider_entered", qctx.NewFunction(func(qctx *quickjs.Context, this *quickjs.Value, args []*quickjs.Value) *quickjs.Value {
		enteredOnce.Do(func() { close(entered) })
		return qctx.NewUndefined()
	}))

	done := make(chan error, 1)
	go func() {
		_, err := sandbox.Eval(context.Background(), "agent-generate-close-pending.js", `
			(async function() {
				var embed = globalThis.__agent_embed;
				var fakeModel = {
					specificationVersion: "v3",
					provider: "brainkit.fake",
					modelId: "pending-close",
					supportedUrls: {},
					doGenerate: async function() {
						globalThis.__test_provider_entered();
						await new Promise((resolve) => setTimeout(resolve, 5000));
						return {
							content: [{ type: "text", text: "late" }],
							finishReason: { unified: "stop" },
							usage: {
								inputTokens: { total: 0, noCache: 0, cacheRead: 0, cacheWrite: 0 },
								outputTokens: { total: 0, text: 0, reasoning: 0 },
							},
							warnings: [],
							request: { body: {} },
							response: { id: "fake", timestamp: new Date(0), modelId: "pending-close" },
						};
					},
					doStream: async function() {
						throw new Error("doStream should not be called");
					},
				};
				var agent = new embed.Agent({
					id: "pending-close-agent",
					name: "pending-close-agent",
					model: fakeModel,
					instructions: "Return late.",
				});
				await agent.generate("slow");
				return "done";
			})()
		`)
		done <- err
	}()

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("fake provider was not entered")
	}

	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := sandbox.CloseContext(closeCtx); err != nil {
		t.Fatalf("CloseContext: %v", err)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("pending generate returned nil error after sandbox close")
		}
	case <-time.After(time.Second):
		t.Fatal("pending generate did not return after sandbox close")
	}
	if !sandbox.closed || sandbox.closing || sandbox.bridge != nil {
		t.Fatalf("sandbox after close: closed=%v closing=%v bridge=%v", sandbox.closed, sandbox.closing, sandbox.bridge)
	}
}
