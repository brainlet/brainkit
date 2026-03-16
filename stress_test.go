package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestStress_ConcurrentEvalTS tests multiple EvalTS calls on the same Kit concurrently.
// This is the pattern the other session claimed deadlocks:
// "When Brainling (an agent) is streaming, the Bridge mutex is held.
//  Any tool callback that tries to call EvalTS on the same Bridge deadlocks."
func TestStress_ConcurrentEvalTS(t *testing.T) {
	kit := newTestKitNoKey(t)
	defer kit.Close()

	tmpDir := t.TempDir()

	// Run 5 concurrent EvalTS calls — each does file I/O + timers
	const concurrency = 5
	var wg sync.WaitGroup
	results := make([]string, concurrency)
	errors := make([]error, concurrency)

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			code := fmt.Sprintf(`
				import { output } from "brainlet";
				try {
					var start = Date.now();
					// File I/O
					await __go_fs_mkdir("%s/worker-%d", true);
					await __go_fs_writeFile("%s/worker-%d/data.txt", "Worker %d says hello at " + Date.now());
					var content = await __go_fs_readFile("%s/worker-%d/data.txt", "utf8");
					// Timer
					await new Promise(r => setTimeout(r, 50));
					// More file I/O
					var list = JSON.parse(await __go_fs_readdir("%s/worker-%d"));
					var elapsed = Date.now() - start;
					output({ worker: %d, content: content.substring(0, 30), files: list.length, elapsed: elapsed });
				} catch(e) {
					output({ worker: %d, error: e.message });
				}
			`, tmpDir, idx, tmpDir, idx, idx, tmpDir, idx, tmpDir, idx, idx, idx)

			result, err := kit.EvalModule(context.Background(), fmt.Sprintf("worker-%d.js", idx), code)
			results[idx] = result
			errors[idx] = err
		}(i)
	}

	wg.Wait()

	// Check results
	succeeded := 0
	for i := 0; i < concurrency; i++ {
		if errors[i] != nil {
			t.Logf("Worker %d ERROR: %v", i, errors[i])
			continue
		}

		var out map[string]interface{}
		json.Unmarshal([]byte(results[i]), &out)
		if errMsg, ok := out["error"]; ok && errMsg != nil {
			t.Logf("Worker %d JS ERROR: %v", i, errMsg)
			continue
		}

		t.Logf("Worker %d: content=%v files=%v elapsed=%vms", i, out["content"], out["files"], out["elapsed"])
		succeeded++
	}

	// In the single-Bridge model, concurrent EvalTS calls are serialized by the mutex.
	// They should all succeed — just sequentially, not in parallel.
	t.Logf("Succeeded: %d/%d", succeeded, concurrency)
	if succeeded != concurrency {
		t.Errorf("Expected all %d workers to succeed, got %d", concurrency, succeeded)
	}
}

// TestStress_EvalTSWithAgent tests EvalTS while an agent is running.
// This simulates the exact pattern: agent generating + another EvalTS on the same Kit.
func TestStress_EvalTSWithAgent(t *testing.T) {
	kit := newTestKit(t)
	defer kit.Close()

	// Start an agent generate in one goroutine
	var wg sync.WaitGroup
	var agentResult string
	var agentErr error
	var evalResult string
	var evalErr error

	wg.Add(2)

	// Goroutine 1: Agent generate
	go func() {
		defer wg.Done()
		code := `
			import { agent, output } from "brainlet";
			var a = agent({
				model: "openai/gpt-4o-mini",
				instructions: "Count from 1 to 5 slowly.",
			});
			var result = await a.generate("Count from 1 to 5");
			output({ text: result.text.substring(0, 50), done: true });
		`
		agentResult, agentErr = kit.EvalModule(context.Background(), "agent.js", code)
	}()

	// Goroutine 2: Simple EvalTS after a short delay
	go func() {
		defer wg.Done()
		time.Sleep(100 * time.Millisecond) // Let the agent start first

		code := `
			import { output } from "brainlet";
			output({ simple: true, timestamp: Date.now() });
		`
		evalResult, evalErr = kit.EvalModule(context.Background(), "simple.js", code)
	}()

	// Wait with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Both completed
	case <-time.After(30 * time.Second):
		t.Fatal("DEADLOCK: concurrent EvalTS + agent generate timed out after 30s")
	}

	// Check results
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

	// At least one should succeed. In the serialized model, both succeed sequentially.
	if agentErr != nil && evalErr != nil {
		t.Error("Both agent and eval failed — possible deadlock or crash")
	}
}
