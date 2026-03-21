package kit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	agentembed "github.com/brainlet/brainkit/internal/embed/agent"
	"github.com/brainlet/brainkit/internal/registry"
)

// TestStress_ConcurrentEvalTS tests multiple EvalTS calls on the same Kit from different goroutines.
// Expected: serialized by mutex, all succeed.
func TestStress_ConcurrentEvalTS(t *testing.T) {
	kit := newTestKitNoKey(t)
	defer kit.Close()

	tmpDir := t.TempDir()

	const concurrency = 5
	var wg sync.WaitGroup
	results := make([]string, concurrency)
	errors := make([]error, concurrency)

	for i := range concurrency {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			code := fmt.Sprintf(`
				import { output } from "kit";
				try {
					await __go_fs_mkdir("%s/worker-%d", true);
					await __go_fs_writeFile("%s/worker-%d/data.txt", "Worker %d at " + Date.now());
					var content = await __go_fs_readFile("%s/worker-%d/data.txt", "utf8");
					await new Promise(r => setTimeout(r, 50));
					output({ worker: %d, content: content.substring(0, 30), ok: true });
				} catch(e) {
					output({ worker: %d, error: e.message });
				}
			`, tmpDir, idx, tmpDir, idx, idx, tmpDir, idx, idx, idx)

			result, err := kit.EvalModule(context.Background(), fmt.Sprintf("worker-%d.js", idx), code)
			results[idx] = result
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	succeeded := 0
	for i := range concurrency {
		if errors[i] != nil {
			t.Logf("Worker %d ERROR: %v", i, errors[i])
			continue
		}
		var out map[string]any
		json.Unmarshal([]byte(results[i]), &out)
		if errMsg, ok := out["error"]; ok && errMsg != nil {
			t.Logf("Worker %d JS ERROR: %v", i, errMsg)
			continue
		}
		t.Logf("Worker %d: %v", i, out)
		succeeded++
	}

	t.Logf("Succeeded: %d/%d", succeeded, concurrency)
	if succeeded != concurrency {
		t.Errorf("Expected all %d workers to succeed, got %d", concurrency, succeeded)
	}
}

// TestStress_EvalTSWithAgent tests EvalTS from a different goroutine while an agent is running.
// Expected: serialized by mutex, both succeed (agent finishes first, then eval runs).
func TestStress_EvalTSWithAgent(t *testing.T) {
	kit := newTestKit(t)
	defer kit.Close()

	var wg sync.WaitGroup
	var agentResult string
	var agentErr error
	var evalResult string
	var evalErr error

	wg.Add(2)

	go func() {
		defer wg.Done()
		code := `
			import { agent, output } from "kit";
			var a = agent({ model: "openai/gpt-4o-mini", instructions: "Count from 1 to 5." });
			var result = await a.generate("Count from 1 to 5");
			output({ text: result.text.substring(0, 50), done: true });
		`
		agentResult, agentErr = kit.EvalModule(context.Background(), "agent.js", code)
	}()

	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond)
		code := `
			import { output } from "kit";
			output({ simple: true, timestamp: Date.now() });
		`
		evalResult, evalErr = kit.EvalModule(context.Background(), "simple.js", code)
	}()

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("TIMEOUT: concurrent EvalTS + agent generate")
	}

	if agentErr != nil {
		t.Logf("Agent error: %v", agentErr)
	} else {
		t.Logf("Agent result: %s", agentResult)
	}
	if evalErr != nil {
		t.Logf("Eval error: %v", evalErr)
	} else {
		t.Logf("Eval result: %s", evalResult)
	}

	if agentErr != nil && evalErr != nil {
		t.Error("Both failed — possible deadlock")
	}
}

// TestStress_DirectToolEvalTS tests the EXACT brainlet pattern: agentembed.Tool.Execute
// callback calls Kit.EvalTS synchronously on the JS thread (same goroutine as Await).
func TestStress_DirectToolEvalTS(t *testing.T) {
	key := requireKey(t)

	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{"openai": {APIKey: key}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	// Create agent with a Go tool that calls EvalTS — same pattern as brainling
	agent, err := kit.CreateAgent(agentembed.AgentConfig{
		Name:         "test-agent",
		Model:        "openai/gpt-4o-mini",
		Instructions: "Use the eval_code tool when asked. Be concise.",
		Tools: map[string]agentembed.Tool{
			"eval_code": {
				Description: "Evaluate JavaScript code and return the result",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"code":{"type":"string","description":"JS code"}},"required":["code"]}`),
				Execute: func(ctx agentembed.ToolContext, args json.RawMessage) (any, error) {
					var input struct {
						Code string `json:"code"`
					}
					if err := json.Unmarshal(args, &input); err != nil {
						return nil, err
					}
					// THIS IS THE BRAINLET PATTERN: direct tool callback → EvalTS
					result, err := kit.EvalTS(context.Background(), "tool-eval.ts", input.Code)
					if err != nil {
						return map[string]any{"error": err.Error()}, nil
					}
					return map[string]any{"result": result}, nil
				},
			},
		},
		MaxSteps: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stream (the pattern brainlet uses via HandleMessage)
	_, err = agent.Stream(ctx, agentembed.StreamParams{
		Prompt: `Use eval_code to evaluate: return 40 + 2`,
		OnToken: func(token string) {
			// Just consume tokens
		},
	})
	if err != nil {
		t.Fatalf("DEADLOCK or error: %v", err)
	}
	t.Log("Direct tool EvalTS during stream: PASSED (no deadlock)")
}

// TestStress_ReentrantEvalTS tests the EXACT pattern that was claimed to deadlock:
// A Go-registered tool calls Kit.EvalTS from within an agent's generate/stream execution.
// This requires the Bridge mutex to be reentrant.
func TestStress_ReentrantEvalTS(t *testing.T) {
	key := requireKey(t)
	tmpDir := t.TempDir()

	kit, err := New(Config{
		Namespace: "test",
		Providers: map[string]ProviderConfig{"openai": {APIKey: key}},
		EnvVars:   map[string]string{"TEST_TMPDIR": tmpDir},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer kit.Close()

	// Register a Go tool that calls EvalTS — the pattern that deadlocked before
	kit.Tools.Register(registry.RegisteredTool{
		Name:        "brainlet/test@1.0.0/eval_code",
		ShortName:   "eval_code",
		Owner:       "brainlet",
		Package:     "test",
		Version:     "1.0.0",
		Description: "Evaluate JavaScript code and return the result",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"code":{"type":"string"}},"required":["code"]}`),
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				var args struct {
					Code string `json:"code"`
				}
				if err := json.Unmarshal(input, &args); err != nil {
					return nil, err
				}
				// THIS IS THE DEADLOCK PATTERN: calling EvalTS from within a tool callback
				result, err := kit.EvalTS(ctx, "tool-eval.ts", args.Code)
				if err != nil {
					return json.Marshal(map[string]string{"error": err.Error()})
				}
				return json.Marshal(map[string]string{"result": result})
			},
		},
	})

	// Agent uses the Go tool — during generate, the tool calls EvalTS on the same Bridge
	code := fmt.Sprintf(`
		import { agent, tool, output } from "kit";
		var a = agent({
			model: "openai/gpt-4o-mini",
			instructions: "Use the eval_code tool when asked to evaluate code. Be concise.",
			tools: { eval: tool("eval_code") },
			maxSteps: 3,
		});
		var result = await a.generate('Use eval_code to evaluate this code: return 40 + 2');
		output({ text: result.text, toolCalls: result.toolCalls.length, has42: result.text.includes("42") });
	`)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := kit.EvalModule(ctx, "reentrant-test.js", code)
	if err != nil {
		t.Fatalf("DEADLOCK or error: %v", err)
	}

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	if errMsg, ok := out["error"]; ok && errMsg != nil {
		t.Fatalf("JS error: %v", errMsg)
	}
	t.Logf("Reentrant EvalTS: text=%v toolCalls=%v has42=%v", out["text"], out["toolCalls"], out["has42"])
	if out["has42"] != true {
		t.Errorf("Expected 42 in response, got: %v", out["text"])
	}
}

// TestStress_ToolCallbackWithGoIO tests that Go tool callbacks doing async I/O
// work correctly during agent execution (NOT calling EvalTS — that's the deadlock case).
// This is the pattern that works: tool callbacks do Go-side I/O via bridges.
func TestStress_ToolCallbackWithGoIO(t *testing.T) {
	kit := newTestKit(t)
	defer kit.Close()

	tmpDir := t.TempDir()

	code := fmt.Sprintf(`
		import { agent, createTool, z, output } from "kit";

		var heavy = createTool({
			id: "heavy-io",
			description: "Does heavy concurrent file I/O and exec",
			inputSchema: z.object({ n: z.number() }),
			execute: async (input) => {
				var promises = [];
				for (var i = 0; i < input.n; i++) {
					promises.push(
						__go_fs_writeFile("%s/tool-" + i + ".txt", "data-" + i)
							.then(() => __go_fs_readFile("%s/tool-" + i + ".txt", "utf8"))
					);
				}
				var results = await Promise.all(promises);
				var exec = await globalThis.child_process.exec("echo done-" + input.n);
				return { files: results.length, exec: exec.stdout.trim() };
			},
		});

		var a = agent({
			model: "openai/gpt-4o-mini",
			instructions: "Use heavy-io tool when asked. Be concise.",
			tools: { "heavy-io": heavy },
			maxSteps: 3,
		});

		// Generate (tool does heavy I/O during generate)
		var r1 = await a.generate("Call heavy-io with n=10");

		// Stream (tool does heavy I/O during streaming)
		var r2 = await a.stream("Call heavy-io with n=5");
		var text = "";
		for await (var chunk of r2.textStream) text += chunk;

		output({
			generate: { text: r1.text.substring(0, 80), tools: r1.toolCalls.length },
			stream: { text: text.substring(0, 80), hasContent: text.length > 0 },
			success: true,
		});
	`, tmpDir, tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := kit.EvalModule(ctx, "stress-tool-io.js", code)
	if err != nil {
		t.Fatalf("EvalModule: %v", err)
	}

	var out map[string]any
	json.Unmarshal([]byte(result), &out)
	if errMsg, ok := out["error"]; ok && errMsg != nil {
		t.Fatalf("fixture error: %v", errMsg)
	}
	if out["success"] != true {
		t.Errorf("expected success, got: %v", out)
	}
	t.Logf("generate: %v", out["generate"])
	t.Logf("stream: %v", out["stream"])
}
