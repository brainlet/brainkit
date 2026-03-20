package brainkit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/bus"
)

// setupMockMemory injects a mock memory instance into the JS runtime.
// This avoids needing a real Mastra memory provider for unit tests.
func setupMockMemory(t *testing.T, kit *Kit) {
	t.Helper()
	_, err := kit.EvalTS(context.Background(), "__mock_memory.ts", `
		var __mock_threads = {};
		var __mock_thread_counter = 0;
		var __mock_messages = {};

		globalThis.__kit_memory = {
			createThread: async function(opts) {
				var id = "thread-" + (++__mock_thread_counter);
				__mock_threads[id] = { id: id, title: (opts && opts.title) || "", metadata: (opts && opts.metadata) || {} };
				__mock_messages[id] = [];
				return __mock_threads[id];
			},
			getThread: async function(threadId) {
				return __mock_threads[threadId] || null;
			},
			listThreads: async function(filter) {
				var all = Object.values(__mock_threads);
				if (filter && filter.title) {
					all = all.filter(function(t) { return t.title && t.title.indexOf(filter.title) >= 0; });
				}
				if (filter && filter.limit) {
					all = all.slice(0, filter.limit);
				}
				return all;
			},
			save: async function(threadId, messages) {
				if (!__mock_messages[threadId]) __mock_messages[threadId] = [];
				for (var i = 0; i < messages.length; i++) {
					__mock_messages[threadId].push(messages[i]);
				}
				return { ok: true };
			},
			recall: async function(threadId, query) {
				return { messages: __mock_messages[threadId] || [] };
			},
			deleteThread: async function(threadId) {
				delete __mock_threads[threadId];
				delete __mock_messages[threadId];
				return { ok: true };
			},
		};
		return "ok";
	`)
	if err != nil {
		t.Fatalf("setup mock memory: %v", err)
	}
}

func TestMemoryHandler_CreateThread(t *testing.T) {
	kit := newTestKitNoKey(t)
	setupMockMemory(t, kit)

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "memory.createThread",
		Payload: json.RawMessage(`{"opts":{"title":"test thread"}}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var result struct {
		ThreadID string `json:"threadId"`
	}
	json.Unmarshal(resp.Payload, &result)
	if result.ThreadID == "" {
		t.Fatalf("expected threadId, got: %s", resp.Payload)
	}
}

func TestMemoryHandler_SaveAndRecall(t *testing.T) {
	kit := newTestKitNoKey(t)
	setupMockMemory(t, kit)

	// Create thread
	resp, _ := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "memory.createThread",
		Payload: json.RawMessage(`{}`),
	})
	var created struct{ ThreadID string `json:"threadId"` }
	json.Unmarshal(resp.Payload, &created)

	// Save messages
	savePayload, _ := json.Marshal(map[string]any{
		"threadId": created.ThreadID,
		"messages": []map[string]string{
			{"role": "user", "content": "hello"},
			{"role": "assistant", "content": "hi there"},
		},
	})
	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "memory.save",
		Payload: savePayload,
	})
	if err != nil {
		t.Fatal(err)
	}
	var saveResult struct{ OK bool `json:"ok"` }
	json.Unmarshal(resp.Payload, &saveResult)
	if !saveResult.OK {
		t.Fatalf("save: %s", resp.Payload)
	}

	// Recall
	recallPayload, _ := json.Marshal(map[string]string{
		"threadId": created.ThreadID,
		"query":    "hello",
	})
	resp, err = bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "memory.recall",
		Payload: recallPayload,
	})
	if err != nil {
		t.Fatal(err)
	}

	var recalled struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	json.Unmarshal(resp.Payload, &recalled)
	if len(recalled.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d: %s", len(recalled.Messages), resp.Payload)
	}
	if recalled.Messages[0].Content != "hello" {
		t.Errorf("first message = %q", recalled.Messages[0].Content)
	}
}

func TestMemoryHandler_GetThread(t *testing.T) {
	kit := newTestKitNoKey(t)
	setupMockMemory(t, kit)

	// Create
	resp, _ := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "memory.createThread",
		Payload: json.RawMessage(`{"opts":{"title":"lookup test"}}`),
	})
	var created struct{ ThreadID string `json:"threadId"` }
	json.Unmarshal(resp.Payload, &created)

	// Get
	getPayload, _ := json.Marshal(map[string]string{"threadId": created.ThreadID})
	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "memory.getThread",
		Payload: getPayload,
	})
	if err != nil {
		t.Fatal(err)
	}

	var thread struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	json.Unmarshal(resp.Payload, &thread)
	if thread.ID != created.ThreadID {
		t.Errorf("expected id=%s, got %s", created.ThreadID, thread.ID)
	}
}

func TestMemoryHandler_ListThreads(t *testing.T) {
	kit := newTestKitNoKey(t)
	setupMockMemory(t, kit)

	// Create 3 threads
	for i := 0; i < 3; i++ {
		bus.AskSync(kit.Bus, context.Background(), bus.Message{
			Topic:   "memory.createThread",
			Payload: json.RawMessage(`{}`),
		})
	}

	// List all
	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "memory.listThreads",
		Payload: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var threads []any
	json.Unmarshal(resp.Payload, &threads)
	if len(threads) != 3 {
		t.Fatalf("expected 3 threads, got %d: %s", len(threads), resp.Payload)
	}
}

func TestMemoryHandler_DeleteThread(t *testing.T) {
	kit := newTestKitNoKey(t)
	setupMockMemory(t, kit)

	// Create
	resp, _ := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "memory.createThread",
		Payload: json.RawMessage(`{}`),
	})
	var created struct{ ThreadID string `json:"threadId"` }
	json.Unmarshal(resp.Payload, &created)

	// Delete
	delPayload, _ := json.Marshal(map[string]string{"threadId": created.ThreadID})
	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "memory.deleteThread",
		Payload: delPayload,
	})
	if err != nil {
		t.Fatal(err)
	}
	var delResult struct{ OK bool `json:"ok"` }
	json.Unmarshal(resp.Payload, &delResult)
	if !delResult.OK {
		t.Fatalf("delete: %s", resp.Payload)
	}

	// Verify gone
	getPayload, _ := json.Marshal(map[string]string{"threadId": created.ThreadID})
	resp, _ = bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "memory.getThread",
		Payload: getPayload,
	})
	if string(resp.Payload) != "null" {
		t.Errorf("expected null after delete, got: %s", resp.Payload)
	}
}

func TestMemoryHandler_UnknownTopic(t *testing.T) {
	kit := newTestKitNoKey(t)

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "memory.bogus",
		Payload: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var errResult struct{ Error string `json:"error"` }
	json.Unmarshal(resp.Payload, &errResult)
	if errResult.Error == "" {
		t.Fatal("expected error for unknown memory topic")
	}
}
