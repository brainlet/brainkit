package agentembed

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func loadEnv(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile("../.env")
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			os.Setenv(k, v)
		}
	}
}

func requireKey(t *testing.T) string {
	t.Helper()
	loadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	return key
}

func testClient(t *testing.T) *Client {
	t.Helper()
	key := requireKey(t)
	return NewClient(ClientConfig{
		Providers: map[string]ProviderConfig{
			"openai": {APIKey: key},
		},
	})
}

func TestSandboxCreation(t *testing.T) {
	client := NewClient(ClientConfig{})
	sandbox, err := client.CreateSandbox(SandboxConfig{})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	defer sandbox.Close()

	if sandbox.ID() == "" {
		t.Error("expected non-empty sandbox ID")
	}
	t.Logf("Sandbox ID: %s", sandbox.ID())
}

func TestBundleExports(t *testing.T) {
	client := NewClient(ClientConfig{})
	sandbox, err := client.CreateSandbox(SandboxConfig{})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	defer sandbox.Close()

	result, err := sandbox.Eval(context.Background(), "test.js", `
		JSON.stringify({
			Agent: typeof globalThis.__agent_embed.Agent,
			createTool: typeof globalThis.__agent_embed.createTool,
			Mastra: typeof globalThis.__agent_embed.Mastra,
			createOpenAI: typeof globalThis.__agent_embed.createOpenAI,
			z: typeof globalThis.__agent_embed.z,
		});
	`)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}

	var types map[string]string
	json.Unmarshal([]byte(result), &types)
	for name, typ := range types {
		if name == "z" {
			if typ != "object" {
				t.Errorf("__agent_embed.%s = %q, want 'object'", name, typ)
			}
		} else if typ != "function" {
			t.Errorf("__agent_embed.%s = %q, want 'function'", name, typ)
		}
	}
	t.Logf("Exports: %v", types)
}

func TestStatelessGenerate(t *testing.T) {
	client := testClient(t)

	result, err := client.Generate(context.Background(), QuickGenerateParams{
		Model:        "openai/gpt-4o-mini",
		Instructions: "Reply with exactly the text requested.",
		Prompt:       "Reply with exactly: HELLO_FROM_AGENT_EMBED",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	t.Logf("Response: %q, usage: %+v", result.Text, result.Usage)
	if !strings.Contains(strings.ToUpper(result.Text), "HELLO_FROM_AGENT_EMBED") {
		t.Errorf("unexpected response: %q", result.Text)
	}
}

func TestPersistentAgent(t *testing.T) {
	client := testClient(t)

	agent, err := client.CreateAgent(AgentConfig{
		Name:         "persistent-test",
		Model:        "openai/gpt-4o-mini",
		Instructions: "Always respond in exactly one word.",
	})
	if err != nil {
		t.Fatalf("CreateAgent: %v", err)
	}
	defer agent.Sandbox().Close()

	r1, err := agent.Generate(context.Background(), GenerateParams{Prompt: "What color is the sky?"})
	if err != nil {
		t.Fatalf("Generate 1: %v", err)
	}

	r2, err := agent.Generate(context.Background(), GenerateParams{Prompt: "What color is grass?"})
	if err != nil {
		t.Fatalf("Generate 2: %v", err)
	}

	t.Logf("Persistent agent: %q then %q", r1.Text, r2.Text)
	if r1.Text == "" || r2.Text == "" {
		t.Error("expected non-empty responses")
	}
}

func TestAgentWithTools(t *testing.T) {
	client := testClient(t)

	toolCalled := false
	result, err := client.Generate(context.Background(), QuickGenerateParams{
		Model:        "openai/gpt-4o-mini",
		Instructions: "Always use the calculator tool when asked to compute.",
		Prompt:       "What is 42 multiplied by 17? Use the calculator tool.",
		MaxSteps:     3,
		Tools: map[string]Tool{
			"calculator": {
				Description: "Multiplies two numbers together",
				Parameters:  json.RawMessage(`{"type":"object","properties":{"a":{"type":"number"},"b":{"type":"number"}},"required":["a","b"]}`),
				Execute: func(ctx ToolContext, args json.RawMessage) (any, error) {
					toolCalled = true
					var input struct{ A, B float64 }
					json.Unmarshal(args, &input)
					t.Logf("Calculator called: %v * %v", input.A, input.B)
					return map[string]any{"result": input.A * input.B}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	t.Logf("Tool result: %q (tool called: %v)", result.Text, toolCalled)
	if !toolCalled {
		t.Log("Warning: model didn't call the tool")
	}
	if !strings.Contains(result.Text, "714") {
		t.Errorf("expected '714' in response, got: %q", result.Text)
	}
}

func TestAgentStream(t *testing.T) {
	client := testClient(t)

	var tokens []string
	result, err := client.Stream(context.Background(), QuickStreamParams{
		Model:        "openai/gpt-4o-mini",
		Instructions: "Be concise.",
		Prompt:       "Count from 1 to 5, one number per word, nothing else",
		OnToken: func(token string) {
			tokens = append(tokens, token)
		},
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	t.Logf("Stream: %d tokens, text: %q, usage: %+v", len(tokens), result.Text, result.Usage)
	if result.Text == "" {
		t.Error("expected non-empty text")
	}
	if len(tokens) == 0 {
		t.Error("expected token callbacks to fire (real-time SSE streaming)")
	} else {
		t.Logf("Token streaming working: %d tokens received", len(tokens))
	}
}

func TestMultipleAgentsInSandbox(t *testing.T) {
	client := testClient(t)

	sandbox, err := client.CreateSandbox(SandboxConfig{})
	if err != nil {
		t.Fatalf("CreateSandbox: %v", err)
	}
	defer sandbox.Close()

	agent1, err := sandbox.CreateAgent(AgentConfig{
		Name:         "agent-1",
		Model:        "openai/gpt-4o-mini",
		Instructions: "Always reply with exactly: AGENT_ONE",
	})
	if err != nil {
		t.Fatalf("CreateAgent 1: %v", err)
	}

	agent2, err := sandbox.CreateAgent(AgentConfig{
		Name:         "agent-2",
		Model:        "openai/gpt-4o-mini",
		Instructions: "Always reply with exactly: AGENT_TWO",
	})
	if err != nil {
		t.Fatalf("CreateAgent 2: %v", err)
	}

	r1, err := agent1.Generate(context.Background(), GenerateParams{Prompt: "Hi"})
	if err != nil {
		t.Fatalf("Agent 1 Generate: %v", err)
	}

	r2, err := agent2.Generate(context.Background(), GenerateParams{Prompt: "Hi"})
	if err != nil {
		t.Fatalf("Agent 2 Generate: %v", err)
	}

	t.Logf("Agent 1: %q, Agent 2: %q", r1.Text, r2.Text)
	if !strings.Contains(strings.ToUpper(r1.Text), "AGENT_ONE") {
		t.Errorf("Agent 1 unexpected: %q", r1.Text)
	}
	if !strings.Contains(strings.ToUpper(r2.Text), "AGENT_TWO") {
		t.Errorf("Agent 2 unexpected: %q", r2.Text)
	}
}
