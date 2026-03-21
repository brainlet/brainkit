package kit

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/brainlet/brainkit/internal/bus"
)

func TestAiHandler_GenerateMock(t *testing.T) {
	kit := newTestKitNoKey(t)

	// Inject a mock ai.generate into the JS runtime
	_, err := kit.EvalTS(context.Background(), "__mock_ai.ts", `
		// Override ai.generate with a mock that returns a canned response
		globalThis.__kit.ai.generate = async function(req) {
			return {
				text: "mock response for: " + (req.prompt || "no prompt"),
				toolCalls: [],
				usage: { promptTokens: 10, completionTokens: 5, totalTokens: 15 },
			};
		};
		return "ok";
	`)
	if err != nil {
		t.Fatalf("mock setup: %v", err)
	}

	// Call ai.generate via bus
	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "ai.generate",
		Payload: json.RawMessage(`{"model":"test-model","prompt":"hello world"}`),
	})
	if err != nil {
		t.Fatalf("ai.generate: %v", err)
	}

	var result struct {
		Text  string `json:"text"`
		Usage struct {
			TotalTokens int `json:"totalTokens"`
		} `json:"usage"`
	}
	json.Unmarshal(resp.Payload, &result)

	if result.Text != "mock response for: hello world" {
		t.Errorf("text = %q", result.Text)
	}
	if result.Usage.TotalTokens != 15 {
		t.Errorf("totalTokens = %d", result.Usage.TotalTokens)
	}
}

func TestAiHandler_UnknownTopic(t *testing.T) {
	kit := newTestKitNoKey(t)

	resp, err := bus.AskSync(kit.Bus, context.Background(), bus.Message{
		Topic:   "ai.unknown",
		Payload: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatal(err)
	}

	var errResult struct{ Error string `json:"error"` }
	json.Unmarshal(resp.Payload, &errResult)
	if errResult.Error == "" {
		t.Fatal("expected error for unknown ai topic")
	}
}
