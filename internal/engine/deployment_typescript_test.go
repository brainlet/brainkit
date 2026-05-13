package engine

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/internal/embed/typescript"
	"github.com/brainlet/brainkit/internal/jsbridge"
	runtimejs "github.com/brainlet/brainkit/internal/jsruntime"
	"github.com/brainlet/brainkit/internal/types"
	"github.com/brainlet/brainkit/sdk/sdkerrors"
	_ "github.com/brainlet/brainkit/storagebridges/sqlite"

	quickjs "github.com/buke/quickjs-go"
)

func TestKernelRawTSDeployTranspilesBeforeEvaluation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	k, err := NewKernel(types.KernelConfig{
		Namespace:        "raw-ts-test",
		CallerID:         "raw-ts-test",
		DeferRouterStart: true,
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}
	t.Cleanup(func() { _ = k.Close() })

	if err := runtimejs.Enable(ctx, k, runtimejs.WithSourcePreparer(func(source, code string) (string, error) {
		return typescript.TranspileTS(code, source)
	})); err != nil {
		t.Fatalf("Enable JS runtime: %v", err)
	}

	_, err = k.Deploy(ctx, "raw-typescript.ts", `import { output } from "kit";

interface Config {
  value: string;
}

type Result = { value: string };

const cfg: Config = { value: "ok" };
const result: Result = { value: cfg.value };
output(result);`)
	if err != nil {
		t.Fatalf("Deploy raw TypeScript: %v", err)
	}

	got, err := k.EvalJS(ctx, "__read_raw_ts_result.js", `return globalThis.__module_result;`)
	if err != nil {
		t.Fatalf("EvalJS read result: %v", err)
	}
	if got != `{"value":"ok"}` {
		t.Fatalf("raw TypeScript deployment result = %q, want %q", got, `{"value":"ok"}`)
	}
}

func TestKernelDeployErrorIncludesJSRuntimeDiagnosticContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	k, err := NewKernel(types.KernelConfig{
		Namespace:        "deploy-diagnostic-test",
		CallerID:         "deploy-diagnostic-test",
		DeferRouterStart: true,
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}
	t.Cleanup(func() { _ = k.Close() })

	if err := runtimejs.Enable(ctx, k, runtimejs.WithSourcePreparer(func(source, code string) (string, error) {
		return typescript.TranspileTS(code, source)
	})); err != nil {
		t.Fatalf("Enable JS runtime: %v", err)
	}

	_, err = k.Deploy(ctx, "diagnostic-runtime.ts", `
function missingSurface() {
  const runtime = {};
  if (typeof runtime.getVersionOverrides !== "function") {
    throw new TypeError("missing runtime surface: runtime.getVersionOverrides is not a function");
  }
  return runtime.getVersionOverrides();
}
missingSurface();`)
	if err == nil {
		t.Fatal("Deploy error = nil")
	}
	var deployErr *sdkerrors.DeployError
	if !errors.As(err, &deployErr) {
		t.Fatalf("error does not expose DeployError: %T", err)
	}
	var diag *jsbridge.DiagnosticError
	if !errors.As(err, &diag) {
		t.Fatalf("error does not expose DiagnosticError: %T", err)
	}
	var jsErr *quickjs.Error
	if !errors.As(err, &jsErr) {
		t.Fatalf("error does not expose quickjs.Error: %T", err)
	}

	msg := err.Error()
	for _, want := range []string{
		"deploy diagnostic-runtime.ts: eval",
		"owner=jsruntime",
		"phase=eval",
		"source=diagnostic-runtime.ts",
		"getVersionOverrides",
		"not a function",
		"Bridge snapshot:",
		"JavaScript stack:",
		"missingSurface",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("deploy diagnostic missing %q:\n%s", want, msg)
		}
	}
}

func TestKernelAgentStructuredOutputUsesRuntimeMastraShim(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	k, err := NewKernel(types.KernelConfig{
		Namespace: "agent-structured-runtime-test",
		CallerID:  "agent-structured-runtime-test",
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}
	t.Cleanup(func() { _ = k.Close() })

	if err := runtimejs.Enable(ctx, k, runtimejs.WithSourcePreparer(func(source, code string) (string, error) {
		return typescript.TranspileTS(code, source)
	})); err != nil {
		t.Fatalf("Enable JS runtime: %v", err)
	}

	_, err = k.Deploy(ctx, "agent-structured-runtime.ts", `
bus.on("probe", async (msg) => {
  const fakeModel = {
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
  const agent = new Agent({
    id: "fake-openai-runtime-agent",
    name: "fake-openai-runtime-agent",
    model: fakeModel,
    instructions: "Return JSON only.",
  });
  const out = await agent.generate("score this", {
    structuredOutput: { schema: z.object({ score: z.number() }) },
  });
  msg.reply({ score: out.object && out.object.score });
});`)
	if err != nil {
		t.Fatalf("Deploy agent structured runtime probe: %v", err)
	}

	got, err := k.EvalJS(ctx, "__agent_structured_probe.js", `
		return JSON.stringify(await globalThis.__kit_bus.call("ts.agent-structured-runtime.probe", {}, { timeoutMs: 10000 }));
	`)
	if err != nil {
		t.Fatalf("EvalJS structured runtime probe: %v", err)
	}
	if got != `{"score":1}` {
		t.Fatalf("structured runtime probe result = %q, want %q", got, `{"score":1}`)
	}
}

func TestKernelAgentMemoryUsesConfiguredStorageWithoutWorkflowSnapshotLock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	k, err := NewKernel(types.KernelConfig{
		Namespace: "agent-memory-storage-runtime-test",
		CallerID:  "agent-memory-storage-runtime-test",
		Storages: map[string]types.StorageConfig{
			"default": types.SQLiteStorage(filepath.Join(t.TempDir(), "memory.db")),
		},
	})
	if err != nil {
		t.Fatalf("NewKernel: %v", err)
	}
	t.Cleanup(func() { _ = k.Close() })

	if err := runtimejs.Enable(ctx, k, runtimejs.WithSourcePreparer(func(source, code string) (string, error) {
		return typescript.TranspileTS(code, source)
	})); err != nil {
		t.Fatalf("Enable JS runtime: %v", err)
	}

	_, err = k.Deploy(ctx, "agent-memory-storage-runtime.ts", `
const memory = new Memory({ storage: storage("default") });
const fakeModel = {
  specificationVersion: "v3",
  provider: "brainkit.fake",
  modelId: "memory",
  supportedUrls: {},
  doGenerate: async function(options) {
    return {
      content: [{ type: "text", text: "stored" }],
      finishReason: { unified: "stop" },
      usage: {
        inputTokens: { total: 0, noCache: 0, cacheRead: 0, cacheWrite: 0 },
        outputTokens: { total: 0, text: 0, reasoning: 0 },
      },
      warnings: [],
      request: { body: options },
      response: { id: "fake", timestamp: new Date(0), modelId: "memory" },
    };
  },
  doStream: async function() {
    throw new Error("doStream should not be called");
  },
};
const agent = new Agent({
  id: "fake-memory-agent",
  name: "fake-memory-agent",
  model: fakeModel,
  instructions: "Remember the user's message.",
  memory,
});

bus.on("ask", async (msg) => {
  const out = await agent.generate("remember this", {
    memory: { thread: { id: "t1" }, resource: "r1" },
  });
  msg.reply({ text: out.text || "" });
});`)
	if err != nil {
		t.Fatalf("Deploy agent memory runtime probe: %v", err)
	}

	got, err := k.EvalJS(ctx, "__agent_memory_probe.js", `
		return JSON.stringify(await globalThis.__kit_bus.call("ts.agent-memory-storage-runtime.ask", {}, { timeoutMs: 10000 }));
	`)
	if err != nil {
		t.Fatalf("EvalJS memory runtime probe: %v", err)
	}
	if got != `{"text":"stored"}` {
		t.Fatalf("memory runtime probe result = %q, want %q", got, `{"text":"stored"}`)
	}
}
