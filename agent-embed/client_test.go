package agentembed

import (
	"encoding/json"
	"fmt"
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

func TestBundleLoads(t *testing.T) {
	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	val, err := c.bridge.Eval("test.js", `typeof globalThis.__agent_embed`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	if val.String() != "object" {
		t.Errorf("__agent_embed type = %q, want 'object'", val.String())
	}
}

func TestExportsExist(t *testing.T) {
	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	val, err := c.bridge.Eval("test.js", `
		JSON.stringify({
			Agent: typeof globalThis.__agent_embed.Agent,
			createTool: typeof globalThis.__agent_embed.createTool,
			Mastra: typeof globalThis.__agent_embed.Mastra,
			createOpenAI: typeof globalThis.__agent_embed.createOpenAI,
			z: typeof globalThis.__agent_embed.z,
		});
	`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	var types map[string]string
	json.Unmarshal([]byte(val.String()), &types)
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

func TestCreateTool(t *testing.T) {
	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	val, err := c.bridge.Eval("test.js", `
		const embed = globalThis.__agent_embed;
		const tool = embed.createTool({
			id: "greet",
			description: "Greets someone",
			inputSchema: embed.z.object({
				name: embed.z.string().describe("Name to greet"),
			}),
			execute: async (input) => ({ message: "hello " + input.name }),
		});
		JSON.stringify({ id: tool.id, desc: tool.description });
	`)
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	defer val.Free()

	var result struct {
		ID   string `json:"id"`
		Desc string `json:"desc"`
	}
	json.Unmarshal([]byte(val.String()), &result)
	if result.ID != "greet" {
		t.Errorf("tool id = %q, want 'greet'", result.ID)
	}
	t.Logf("Tool: %+v", result)
}

func TestGenerateRealOpenAI(t *testing.T) {
	loadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.Generate(GenerateParams{
		Provider:     "openai",
		APIKey:       key,
		Model:        "openai/gpt-4o-mini",
		Instructions: "You are a concise assistant. Reply with exactly the text requested.",
		Prompt:       "Reply with exactly: HELLO_FROM_MASTRA_AGENT",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	t.Logf("OpenAI response: %q", result.Text)
	if !strings.Contains(strings.ToUpper(result.Text), "HELLO_FROM_MASTRA_AGENT") {
		t.Errorf("unexpected response: %q", result.Text)
	}
}

func TestGenerateRealOpenAIWithTool(t *testing.T) {
	loadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	toolCalled := false
	result, err := c.Generate(GenerateParams{
		Provider:     "openai",
		APIKey:       key,
		Model:        "openai/gpt-4o-mini",
		Instructions: "You are a helpful assistant. Always use the available tools when appropriate.",
		Prompt:       "What is 2 + 2? Use the calculator tool to compute it.",
		Tools: []ToolDef{
			{
				ID:          "calculator",
				Description: "Performs arithmetic calculations. Takes an expression string and returns the numeric result.",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"expression": map[string]interface{}{
							"type":        "string",
							"description": "The arithmetic expression to evaluate, e.g. '2+2'",
						},
					},
				},
				Execute: func(args map[string]interface{}) (interface{}, error) {
					toolCalled = true
					expr, _ := args["expression"].(string)
					t.Logf("Calculator called with: %q", expr)
					return map[string]interface{}{
						"result": 4,
					}, nil
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	t.Logf("OpenAI response: %q (tool called: %v)", result.Text, toolCalled)
	if !toolCalled {
		t.Log("Warning: model didn't call the tool (it may have answered directly)")
	}
	if result.Text == "" {
		t.Error("expected non-empty response")
	}
	fmt.Printf("\n=== REAL OPENAI + TOOL RESULT ===\n%s\n", result.Text)
}

func TestGenerateRealOpenAIWithBaseURL(t *testing.T) {
	loadEnv(t)
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}

	c, err := NewClient(ClientConfig{})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer c.Close()

	result, err := c.Generate(GenerateParams{
		Provider:     "openai",
		APIKey:       key,
		BaseURL:      "https://api.openai.com/v1",
		Model:        "openai/gpt-4o-mini",
		Instructions: "You are a concise assistant.",
		Prompt:       "Reply with exactly: CUSTOM_BASE_URL_WORKS",
	})
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	t.Logf("OpenAI response (custom base URL): %q", result.Text)
	if !strings.Contains(strings.ToUpper(result.Text), "CUSTOM_BASE_URL_WORKS") {
		t.Errorf("unexpected response: %q", result.Text)
	}
}
