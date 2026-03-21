//go:build integration

package brainkit

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/brainlet/brainkit/bus"
	"github.com/brainlet/brainkit/registry"
	"github.com/stretchr/testify/require"
)

func TestShardFixture_ToolProvider(t *testing.T) {
	kit := newTestKitNoKey(t)
	desc := deployShard(t, kit, "testdata/as/shard/tool-provider.ts", "tools-shard")

	require.Equal(t, "stateless", desc.Mode)
	t.Log("tool-provider deployed successfully with mode:", desc.Mode)
}

func TestShardFixture_AiSummarizer(t *testing.T) {
	kit := newTestKit(t)

	completed := make(chan string, 5)
	kit.Bus.On("summarize.completed", func(msg bus.Message, _ bus.ReplyFunc) {
		completed <- string(msg.Payload)
	})

	deployShard(t, kit, "testdata/as/shard/ai-summarizer.ts", "summarizer")

	injectEvent(t, kit, "summarizer", "summarize.request", map[string]string{
		"text":  "The quick brown fox jumps over the lazy dog",
		"model": "openai/gpt-4o-mini",
	})

	select {
	case msg := <-completed:
		require.Contains(t, msg, "summary")
	case <-time.After(5 * time.Second):
		t.Fatal("summarize.completed not received")
	}

	result := injectEvent(t, kit, "summarizer", "summarize.last", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"requestCount":1`)
	require.Contains(t, result.ReplyPayload, `"lastSummary":`)

	injectEvent(t, kit, "summarizer", "summarize.request", map[string]string{
		"text":  "Second document to summarize",
		"model": "openai/gpt-4o-mini",
	})

	select {
	case <-completed:
	case <-time.After(5 * time.Second):
		t.Fatal("second summarize.completed not received")
	}

	result = injectEvent(t, kit, "summarizer", "summarize.last", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"requestCount":2`)
}

func TestShardFixture_ToolOrchestrator(t *testing.T) {
	kit := newTestKitNoKey(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/test@1.0.0/db_query", ShortName: "db_query",
		Owner: "brainlet", Package: "test", Version: "1.0.0",
		Description: "Test query tool",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(map[string]any{
					"tool":   "db_query",
					"result": "executed",
					"input":  json.RawMessage(input),
				})
			},
		},
	})

	deployShard(t, kit, "testdata/as/shard/tool-orchestrator.ts", "orchestrator")

	result, err := kit.InjectWASMEvent("orchestrator", "orchestrate.query", json.RawMessage(
		`{"tool":"db_query","input":{"sql":"SELECT 1"}}`,
	))
	require.NoError(t, err)
	require.Empty(t, result.Error)
}

func TestShardFixture_AgentDelegator(t *testing.T) {
	kit := newTestKit(t)

	_, err := kit.EvalTS(context.Background(), "__setup_delegator_agent.ts", `
		const coder = agent({
			name: "coder",
			model: "openai/gpt-4o-mini",
			instructions: "Reply with exactly: done with <prompt>. Keep it short.",
		});
		return "ok";
	`)
	if err != nil {
		t.Fatal(err)
	}

	completed := make(chan string, 5)
	kit.Bus.On("delegate.completed", func(msg bus.Message, _ bus.ReplyFunc) {
		completed <- string(msg.Payload)
	})

	deployShard(t, kit, "testdata/as/shard/agent-delegator.ts", "delegator")

	for i := 1; i <= 3; i++ {
		injectEvent(t, kit, "delegator", "delegate.task", map[string]interface{}{
			"agent":  "coder",
			"prompt": fmt.Sprintf("task %d", i),
		})

		select {
		case msg := <-completed:
			require.Contains(t, msg, fmt.Sprintf(`"taskNum":%d`, i))
		case <-time.After(5 * time.Second):
			t.Fatalf("delegate.completed not received for task %d", i)
		}
	}

	result := injectEvent(t, kit, "delegator", "delegate.results", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"completedTasks":3`)
	require.Contains(t, result.ReplyPayload, `"lastResult":`)
}

func TestShardFixture_MultiStepPipeline(t *testing.T) {
	kit := newTestKit(t)

	kit.Tools.Register(registry.RegisteredTool{
		Name: "brainlet/test@1.0.0/data_fetch", ShortName: "data_fetch",
		Owner: "brainlet", Package: "test", Version: "1.0.0",
		Description: "Fetch test data",
		Executor: &registry.GoFuncExecutor{
			Fn: func(ctx context.Context, callerID string, input json.RawMessage) (json.RawMessage, error) {
				return json.Marshal(map[string]any{
					"rows": []map[string]any{
						{"id": 1, "value": "data-row-1"},
						{"id": 2, "value": "data-row-2"},
					},
				})
			},
		},
	})

	completed := make(chan string, 5)
	kit.Bus.On("pipeline.completed", func(msg bus.Message, _ bus.ReplyFunc) {
		completed <- string(msg.Payload)
	})

	deployShard(t, kit, "testdata/as/shard/multi-step-pipeline.ts", "pipeline")

	injectEvent(t, kit, "pipeline", "pipeline.run", map[string]string{"source": "test-db"})

	select {
	case msg := <-completed:
		require.Contains(t, msg, `"run":1`)
	case <-time.After(10 * time.Second):
		t.Fatal("pipeline.completed not received")
	}

	result := injectEvent(t, kit, "pipeline", "pipeline.status", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"stage":"complete"`)
	require.Contains(t, result.ReplyPayload, `"runs":1`)
	require.NotContains(t, result.ReplyPayload, `"analysis":""`)

	injectEvent(t, kit, "pipeline", "pipeline.run", map[string]string{"source": "test-db-2"})

	select {
	case msg := <-completed:
		require.Contains(t, msg, `"run":2`)
	case <-time.After(10 * time.Second):
		t.Fatal("second pipeline.completed not received")
	}

	result = injectEvent(t, kit, "pipeline", "pipeline.status", map[string]string{})
	require.Contains(t, result.ReplyPayload, `"runs":2`)
}
