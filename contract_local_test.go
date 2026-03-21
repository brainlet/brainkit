package brainkit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	agentembed "github.com/brainlet/brainkit/internal/embed/agent"
)

func TestContract_LocalAgentGenerate(t *testing.T) {
	kit := newTestKit(t)

	agent, err := kit.CreateAgent(agentembed.AgentConfig{
		Name:         "greeter",
		Model:        "openai/gpt-4o-mini",
		Instructions: "Reply with exactly: HELLO_CONTRACT",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Generate(context.Background(), agentembed.GenerateParams{
		Prompt: "Say the magic word",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(strings.ToUpper(result.Text), "HELLO_CONTRACT") {
		t.Errorf("unexpected: %q", result.Text)
	}
	t.Logf("Contract LOCAL agent.generate: %q", result.Text)
}

func TestContract_LocalAgentStream(t *testing.T) {
	kit := newTestKit(t)

	agent, err := kit.CreateAgent(agentembed.AgentConfig{
		Name:         "streamer",
		Model:        "openai/gpt-4o-mini",
		Instructions: "Count from 1 to 3, one per line.",
	})
	if err != nil {
		t.Fatal(err)
	}

	var tokens []string
	result, err := agent.Stream(context.Background(), agentembed.StreamParams{
		Prompt: "Count",
		OnToken: func(token string) {
			tokens = append(tokens, token)
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Text == "" {
		t.Error("expected non-empty text")
	}
	if len(tokens) == 0 {
		t.Error("expected real-time token callbacks (SSE streaming)")
	}
	t.Logf("Contract LOCAL agent.stream: %d tokens, text: %q", len(tokens), result.Text)
}

func TestContract_LocalAgentWithTools(t *testing.T) {
	kit := newTestKit(t)

	toolCalled := false
	agent, err := kit.CreateAgent(agentembed.AgentConfig{
		Name:         "tool-user",
		Model:        "openai/gpt-4o-mini",
		Instructions: "Always use the add tool when asked to compute.",
		Tools: map[string]agentembed.Tool{
			"add": {
				Description: "Adds two numbers",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"a":{"type":"number"},"b":{"type":"number"}},"required":["a","b"]}`),
				Execute: func(ctx agentembed.ToolContext, args json.RawMessage) (any, error) {
					toolCalled = true
					var input struct{ A, B float64 }
					json.Unmarshal(args, &input)
					return map[string]any{"result": input.A + input.B}, nil
				},
			},
		},
		MaxSteps: 3,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := agent.Generate(context.Background(), agentembed.GenerateParams{
		Prompt: "What is 7 + 5? Use the add tool.",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !toolCalled {
		t.Log("Warning: model didn't call the tool")
	}
	if !strings.Contains(result.Text, "12") {
		t.Errorf("expected 12 in response: %q", result.Text)
	}
	t.Logf("Contract LOCAL agent+tools: %q, toolCalled=%v", result.Text, toolCalled)
}

func TestContract_LocalMultipleAgents(t *testing.T) {
	kit := newTestKit(t)

	a1, err := kit.CreateAgent(agentembed.AgentConfig{
		Name:         "a1",
		Model:        "openai/gpt-4o-mini",
		Instructions: "Reply with exactly: ALPHA",
	})
	if err != nil {
		t.Fatal(err)
	}
	a2, err := kit.CreateAgent(agentembed.AgentConfig{
		Name:         "a2",
		Model:        "openai/gpt-4o-mini",
		Instructions: "Reply with exactly: BETA",
	})
	if err != nil {
		t.Fatal(err)
	}

	r1, err := a1.Generate(context.Background(), agentembed.GenerateParams{Prompt: "Go"})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := a2.Generate(context.Background(), agentembed.GenerateParams{Prompt: "Go"})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(strings.ToUpper(r1.Text), "ALPHA") {
		t.Errorf("a1: %q", r1.Text)
	}
	if !strings.Contains(strings.ToUpper(r2.Text), "BETA") {
		t.Errorf("a2: %q", r2.Text)
	}
	t.Logf("Contract LOCAL multi-agent: %q, %q", r1.Text, r2.Text)
}
