package agentembed

import (
	"context"
	"encoding/json"
	"testing"
)

func TestAgentEmbedSESLoadOrderConformance(t *testing.T) {
	client := NewClient(ClientConfig{})
	sandbox, err := client.CreateSandbox(SandboxConfig{})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	defer sandbox.Close()

	result, err := sandbox.Eval(context.Background(), "ses-load-order-conformance.js", `
		var captures = globalThis.__brainkit_pre_lockdown || {};
		var compartmentValue = new Compartment({ value: 41 }).evaluate("value + 1");
		JSON.stringify({
			agent: typeof globalThis.__agent_embed.Agent,
			createTool: typeof globalThis.__agent_embed.createTool,
			compartment: typeof Compartment,
			harden: typeof harden,
			lockdown: typeof lockdown,
			arrayFrozen: Object.isFrozen(Array.prototype),
			objectHasOwn: typeof Object.hasOwn,
			preMathRandom: typeof captures.mathRandom,
			preDateNow: typeof captures.dateNow,
			preDate: typeof captures.Date,
			compartmentValue: compartmentValue
		});
	`)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(result), &got); err != nil {
		t.Fatalf("parse result %q: %v", result, err)
	}
	want := map[string]any{
		"agent":            "function",
		"createTool":       "function",
		"compartment":      "function",
		"harden":           "function",
		"lockdown":         "function",
		"arrayFrozen":      true,
		"objectHasOwn":     "function",
		"preMathRandom":    "function",
		"preDateNow":       "function",
		"preDate":          "function",
		"compartmentValue": float64(42),
	}
	for key, expected := range want {
		if got[key] != expected {
			t.Fatalf("%s = %#v, want %#v; full result=%s", key, got[key], expected, result)
		}
	}
}

func TestAgentGenerateStructuredOutputWithFakeV3Model(t *testing.T) {
	client := NewClient(ClientConfig{})
	sandbox, err := client.CreateSandbox(SandboxConfig{})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	defer sandbox.Close()

	result, err := sandbox.Eval(context.Background(), "fake-v3-structured-output.js", `
		(async function() {
			var embed = globalThis.__agent_embed;
			var fakeModel = {
				specificationVersion: "v3",
				provider: "brainkit.fake",
				modelId: "structured-output",
				supportedUrls: {},
				doGenerate: async function(options) {
					return {
						content: [{ type: "text", text: "{\"score\":1}" }],
						finishReason: { unified: "stop" },
						usage: {
							inputTokens: { total: 0, noCache: 0, cacheRead: 0, cacheWrite: 0 },
							outputTokens: { total: 0, text: 0, reasoning: 0 },
						},
						warnings: [],
						request: { body: options },
						response: { id: "fake", timestamp: new Date(0), modelId: "structured-output" },
					};
				},
				doStream: async function() {
					throw new Error("doStream should not be called");
				},
			};
			var agent = new embed.Agent({
				id: "fake-agent",
				name: "fake-agent",
				model: fakeModel,
				instructions: "Return JSON only.",
			});
			var out = await agent.generate("score this", {
				structuredOutput: { schema: embed.z.object({ score: embed.z.number() }) },
			});
			return JSON.stringify({ score: out.object && out.object.score, text: out.text });
		})()
	`)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}

	var got struct {
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(result), &got); err != nil {
		t.Fatalf("parse result %q: %v", result, err)
	}
	if got.Score != 1 {
		t.Fatalf("score = %v, want 1; full result=%s", got.Score, result)
	}
}

func TestAgentGenerateStructuredOutputWithFakeOpenAIV3Model(t *testing.T) {
	client := NewClient(ClientConfig{})
	sandbox, err := client.CreateSandbox(SandboxConfig{})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	defer sandbox.Close()

	result, err := sandbox.Eval(context.Background(), "fake-openai-v3-structured-output.js", `
		(async function() {
			var embed = globalThis.__agent_embed;
			var fakeModel = {
				specificationVersion: "v3",
				provider: "openai.responses",
				modelId: "gpt-4o-mini",
				supportedUrls: {},
				doGenerate: async function(options) {
					return {
						content: [{ type: "text", text: "{\"score\":1}" }],
						finishReason: { unified: "stop" },
						usage: {
							inputTokens: { total: 0, noCache: 0, cacheRead: 0, cacheWrite: 0 },
							outputTokens: { total: 0, text: 0, reasoning: 0 },
						},
						warnings: [],
						request: { body: options },
						response: { id: "fake", timestamp: new Date(0), modelId: "gpt-4o-mini" },
					};
				},
				doStream: async function() {
					throw new Error("doStream should not be called");
				},
			};
			var agent = new embed.Agent({
				id: "fake-openai-agent",
				name: "fake-openai-agent",
				model: fakeModel,
				instructions: "Return JSON only.",
			});
			var out = await agent.generate("score this", {
				structuredOutput: { schema: embed.z.object({ score: embed.z.number() }) },
			});
			return JSON.stringify({ score: out.object && out.object.score, text: out.text });
		})()
	`)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}

	var got struct {
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(result), &got); err != nil {
		t.Fatalf("parse result %q: %v", result, err)
	}
	if got.Score != 1 {
		t.Fatalf("score = %v, want 1; full result=%s", got.Score, result)
	}
}

func TestAgentGenerateStructuredOutputFromCompartmentWithFakeOpenAIV3Model(t *testing.T) {
	client := NewClient(ClientConfig{})
	sandbox, err := client.CreateSandbox(SandboxConfig{})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	defer sandbox.Close()

	result, err := sandbox.Eval(context.Background(), "fake-openai-v3-compartment-structured-output.js", `
		(async function() {
			var embed = globalThis.__agent_embed;
			var endowments = {
				Agent: embed.Agent,
				z: embed.z,
				JSON: JSON,
				Promise: Promise,
				Date: Date,
				console: console,
				setTimeout: setTimeout,
				clearTimeout: clearTimeout,
				ReadableStream: ReadableStream,
				TransformStream: TransformStream,
			};
			var compartment = new Compartment({ __options__: true, globals: endowments });
			return await compartment.evaluate('\
				(async function() {\
					var fakeModel = {\
						specificationVersion: "v3",\
						provider: "openai.responses",\
						modelId: "gpt-4o-mini",\
						supportedUrls: {},\
						doGenerate: async function(options) {\
							return {\
								content: [{ type: "text", text: "{\\"score\\":1}" }],\
								finishReason: { unified: "stop" },\
								usage: {\
									inputTokens: { total: 0, noCache: 0, cacheRead: 0, cacheWrite: 0 },\
									outputTokens: { total: 0, text: 0, reasoning: 0 },\
								},\
								warnings: [],\
								request: { body: options },\
								response: { id: "fake", timestamp: new Date(0), modelId: "gpt-4o-mini" },\
							};\
						},\
						doStream: async function() {\
							throw new Error("doStream should not be called");\
						},\
					};\
					var agent = new Agent({\
						id: "fake-openai-compartment-agent",\
						name: "fake-openai-compartment-agent",\
						model: fakeModel,\
						instructions: "Return JSON only.",\
					});\
					var out = await agent.generate("score this", {\
						structuredOutput: { schema: z.object({ score: z.number() }) },\
					});\
					return JSON.stringify({ score: out.object && out.object.score, text: out.text });\
				})()\
			');
		})()
	`)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}

	var got struct {
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(result), &got); err != nil {
		t.Fatalf("parse result %q: %v", result, err)
	}
	if got.Score != 1 {
		t.Fatalf("score = %v, want 1; full result=%s", got.Score, result)
	}
}

func TestCreateScorerLLMJudgeWithFakeOpenAIV3Model(t *testing.T) {
	client := NewClient(ClientConfig{})
	sandbox, err := client.CreateSandbox(SandboxConfig{})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	defer sandbox.Close()

	result, err := sandbox.Eval(context.Background(), "fake-openai-v3-scorer-judge.js", `
		(async function() {
			var embed = globalThis.__agent_embed;
			var fakeModel = {
				specificationVersion: "v3",
				provider: "openai.responses",
				modelId: "gpt-4o-mini",
				supportedUrls: {},
				doGenerate: async function(options) {
					return {
						content: [{ type: "text", text: "{\"score\":1}" }],
						finishReason: { unified: "stop" },
						usage: {
							inputTokens: { total: 0, noCache: 0, cacheRead: 0, cacheWrite: 0 },
							outputTokens: { total: 0, text: 0, reasoning: 0 },
						},
						warnings: [],
						request: { body: options },
						response: { id: "fake", timestamp: new Date(0), modelId: "gpt-4o-mini" },
					};
				},
				doStream: async function() {
					throw new Error("doStream should not be called");
				},
			};
			var scorer = embed.createScorer({
				id: "fake-openai-judge",
				name: "fake-openai-judge",
				description: "fake OpenAI judge",
			}).generateScore({
				description: "always one",
				judge: { model: fakeModel, instructions: "Return score." },
				createPrompt: function() { return "score"; },
			});
			var out = await scorer.run({ output: { role: "assistant", text: "x" } });
			return JSON.stringify({ score: out.score });
		})()
	`)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}

	var got struct {
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(result), &got); err != nil {
		t.Fatalf("parse result %q: %v", result, err)
	}
	if got.Score != 1 {
		t.Fatalf("score = %v, want 1; full result=%s", got.Score, result)
	}
}

func TestRuntimeGlobalsDynamicRequireConformance(t *testing.T) {
	client := NewClient(ClientConfig{})
	sandbox, err := client.CreateSandbox(SandboxConfig{})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	defer sandbox.Close()

	result, err := sandbox.Eval(context.Background(), "dynamic-require-conformance.js", `
		(function() {
			var req = globalThis.require;
			var createRequire = globalThis.node_module && globalThis.node_module.createRequire;
			var zod = req("zod");
			var zodV4 = req("zod/v4");
			var otel = req("@opentelemetry/api");
			var tracer = otel.trace.getTracer("brainkit-test");
			var span = tracer.startSpan("noop");
			span.setAttribute("k", "v").addEvent("event").end();
			var execa = req("execa");
			var vscode = req("vscode-jsonrpc/node");
			var lsp = req("vscode-languageserver-protocol");
			var moduleReq = createRequire("agent-embed-test");
			return JSON.stringify({
				require: typeof req,
				createRequire: typeof createRequire,
				zodObject: typeof zod.z.object,
				zodSingleton: zod === zodV4,
				zodViaModule: moduleReq("zod") === zod,
				toJSONSchema: typeof zod.toJSONSchema,
				otelTracer: typeof tracer.startSpan,
				otelSpanContext: typeof span.spanContext,
				execa: typeof execa.execa,
				vscode: typeof vscode,
				lsp: typeof lsp
			});
		})()
	`)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}

	var got struct {
		Require       string `json:"require"`
		CreateRequire string `json:"createRequire"`
		ZodObject     string `json:"zodObject"`
		ZodSingleton  bool   `json:"zodSingleton"`
		ZodViaModule  bool   `json:"zodViaModule"`
		ToJSONSchema  string `json:"toJSONSchema"`
		OTelTracer    string `json:"otelTracer"`
		OTelSpanCtx   string `json:"otelSpanContext"`
		Execa         string `json:"execa"`
		VSCode        string `json:"vscode"`
		LSP           string `json:"lsp"`
	}
	if err := json.Unmarshal([]byte(result), &got); err != nil {
		t.Fatalf("parse result %q: %v", result, err)
	}
	if got.Require != "function" || got.CreateRequire != "function" {
		t.Fatalf("dynamic require shape = %+v", got)
	}
	if got.ZodObject != "function" || !got.ZodSingleton || !got.ZodViaModule || got.ToJSONSchema != "function" {
		t.Fatalf("zod dynamic require singleton failed: %+v", got)
	}
	if got.OTelTracer != "function" || got.OTelSpanCtx != "function" {
		t.Fatalf("otel dynamic require shape failed: %+v", got)
	}
	if got.Execa != "function" || got.VSCode != "object" || got.LSP != "object" {
		t.Fatalf("optional dynamic require shape failed: %+v", got)
	}
}

func TestCreateScorerLLMJudgeWithFakeV3Model(t *testing.T) {
	client := NewClient(ClientConfig{})
	sandbox, err := client.CreateSandbox(SandboxConfig{})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	defer sandbox.Close()

	result, err := sandbox.Eval(context.Background(), "fake-v3-scorer-judge.js", `
		(async function() {
			var embed = globalThis.__agent_embed;
			var fakeModel = {
				specificationVersion: "v3",
				provider: "brainkit.fake",
				modelId: "scorer-judge",
				supportedUrls: {},
				doGenerate: async function(options) {
					return {
						content: [{ type: "text", text: "{\"score\":1}" }],
						finishReason: { unified: "stop" },
						usage: {
							inputTokens: { total: 0, noCache: 0, cacheRead: 0, cacheWrite: 0 },
							outputTokens: { total: 0, text: 0, reasoning: 0 },
						},
						warnings: [],
						request: { body: options },
						response: { id: "fake", timestamp: new Date(0), modelId: "scorer-judge" },
					};
				},
				doStream: async function() {
					throw new Error("doStream should not be called");
				},
			};
			var scorer = embed.createScorer({
				id: "fake-judge",
				name: "fake-judge",
				description: "fake judge",
			}).generateScore({
				description: "always one",
				judge: { model: fakeModel, instructions: "Return score." },
				createPrompt: function() { return "score"; },
			});
			var out = await scorer.run({ output: { role: "assistant", text: "x" } });
			return JSON.stringify({ score: out.score });
		})()
	`)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}

	var got struct {
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(result), &got); err != nil {
		t.Fatalf("parse result %q: %v", result, err)
	}
	if got.Score != 1 {
		t.Fatalf("score = %v, want 1; full result=%s", got.Score, result)
	}
}
