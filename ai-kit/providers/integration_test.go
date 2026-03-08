package providers_test

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/brainlet/brainkit/ai-kit/provider/languagemodel"
	"github.com/brainlet/brainkit/ai-kit/providers/openai"
	"github.com/brainlet/brainkit/ai-kit/providers/openaicompatible"
)

// loadEnv reads a .env file and sets environment variables.
func loadEnv(t *testing.T) {
	t.Helper()

	// Walk up from the test file to find .env at the repo root.
	dir, _ := os.Getwd()
	for {
		envPath := filepath.Join(dir, ".env")
		if f, err := os.Open(envPath); err == nil {
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				if k, v, ok := strings.Cut(line, "="); ok {
					os.Setenv(strings.TrimSpace(k), strings.TrimSpace(v))
				}
			}
			f.Close()
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
}

func skipIfNoKey(t *testing.T, envVar string) string {
	t.Helper()
	key := os.Getenv(envVar)
	if key == "" {
		t.Skipf("skipping: %s not set", envVar)
	}
	return key
}

// --- OpenAI (native provider) ---

func TestOpenAI_ChatGenerate(t *testing.T) {
	loadEnv(t)
	skipIfNoKey(t, "OPENAI_API_KEY")

	provider := openai.CreateOpenAI(nil)
	model := provider.Chat("gpt-4o-mini")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := model.DoGenerate(languagemodel.CallOptions{
		Ctx: ctx,
		Prompt: languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "What is 2+2? Reply with just the number."},
				},
			},
		},
		MaxOutputTokens: intPtr(50),
	})
	if err != nil {
		t.Fatalf("DoGenerate failed: %v", err)
	}

	// Check we got text content back.
	var text string
	for _, c := range result.Content {
		if tc, ok := c.(languagemodel.Text); ok {
			text += tc.Text
		}
	}
	if text == "" {
		t.Fatal("expected non-empty text response")
	}
	if !strings.Contains(text, "4") {
		t.Errorf("expected response containing '4', got: %s", text)
	}

	t.Logf("OpenAI Chat response: %q", text)
	t.Logf("Usage: input=%s output=%s", fmtTokens(result.Usage.InputTokens.Total), fmtTokens(result.Usage.OutputTokens.Total))
	t.Logf("Finish reason: %s", result.FinishReason.Unified)
}

func TestOpenAI_ChatStream(t *testing.T) {
	loadEnv(t)
	skipIfNoKey(t, "OPENAI_API_KEY")

	provider := openai.CreateOpenAI(nil)
	model := provider.Chat("gpt-4o-mini")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	streamResult, err := model.DoStream(languagemodel.CallOptions{
		Ctx: ctx,
		Prompt: languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Count from 1 to 5, one number per line."},
				},
			},
		},
		MaxOutputTokens: intPtr(100),
	})
	if err != nil {
		t.Fatalf("DoStream failed: %v", err)
	}

	var fullText string
	var gotFinish bool
	for part := range streamResult.Stream {
		switch p := part.(type) {
		case languagemodel.StreamPartTextDelta:
			fullText += p.Delta
		case languagemodel.StreamPartFinish:
			gotFinish = true
			t.Logf("Stream finish: reason=%s, usage: input=%s output=%s",
				p.FinishReason.Unified, fmtTokens(p.Usage.InputTokens.Total), fmtTokens(p.Usage.OutputTokens.Total))
		}
	}

	if fullText == "" {
		t.Fatal("expected non-empty streamed text")
	}
	if !gotFinish {
		t.Error("expected finish part in stream")
	}
	t.Logf("OpenAI Stream response: %q", fullText)
}

// --- OpenAI-Compatible: MiniMax ---

func TestMiniMax_ChatGenerate(t *testing.T) {
	loadEnv(t)
	apiKey := skipIfNoKey(t, "MINIMAX_API_KEY")

	provider := openaicompatible.NewProvider(openaicompatible.ProviderSettings{
		BaseURL: "https://api.minimax.io/v1",
		Name:    "minimax",
		APIKey:  apiKey,
	})
	model := provider.ChatModel("MiniMax-M2.5-highspeed")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := model.DoGenerate(languagemodel.CallOptions{
		Ctx: ctx,
		Prompt: languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "What is 2+2? Reply with just the number."},
				},
			},
		},
		MaxOutputTokens: intPtr(50),
	})
	if err != nil {
		t.Fatalf("DoGenerate failed: %v", err)
	}

	var text string
	for _, c := range result.Content {
		if tc, ok := c.(languagemodel.Text); ok {
			text += tc.Text
		}
	}
	if text == "" {
		t.Fatal("expected non-empty text response")
	}

	t.Logf("MiniMax response: %q", text)
	t.Logf("Usage: input=%s output=%s", fmtTokens(result.Usage.InputTokens.Total), fmtTokens(result.Usage.OutputTokens.Total))
}

func TestMiniMax_ChatStream(t *testing.T) {
	loadEnv(t)
	apiKey := skipIfNoKey(t, "MINIMAX_API_KEY")

	provider := openaicompatible.NewProvider(openaicompatible.ProviderSettings{
		BaseURL: "https://api.minimax.io/v1",
		Name:    "minimax",
		APIKey:  apiKey,
	})
	model := provider.ChatModel("MiniMax-M2.5-highspeed")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	streamResult, err := model.DoStream(languagemodel.CallOptions{
		Ctx: ctx,
		Prompt: languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Say hello in 3 different languages."},
				},
			},
		},
		MaxOutputTokens: intPtr(200),
	})
	if err != nil {
		t.Fatalf("DoStream failed: %v", err)
	}

	var fullText string
	for part := range streamResult.Stream {
		switch p := part.(type) {
		case languagemodel.StreamPartTextDelta:
			fullText += p.Delta
		case languagemodel.StreamPartFinish:
			t.Logf("Stream finish: reason=%s", p.FinishReason.Unified)
		}
	}

	if fullText == "" {
		t.Fatal("expected non-empty streamed text")
	}
	t.Logf("MiniMax Stream response: %q", fullText)
}

// --- OpenAI-Compatible: GLM (Zhipu AI) ---

func TestGLM_ChatGenerate(t *testing.T) {
	loadEnv(t)
	apiKey := skipIfNoKey(t, "GLM_API_KEY")

	provider := openaicompatible.NewProvider(openaicompatible.ProviderSettings{
		BaseURL: "https://api.z.ai/api/coding/paas/v4",
		Name:    "glm",
		APIKey:  apiKey,
	})
	model := provider.ChatModel("glm-5")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// GLM-5 is a reasoning model that uses tokens for chain-of-thought;
	// we need a larger budget so there's room for the actual answer.
	result, err := model.DoGenerate(languagemodel.CallOptions{
		Ctx: ctx,
		Prompt: languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "What is 2+2? Reply with just the number."},
				},
			},
		},
		MaxOutputTokens: intPtr(500),
	})
	if err != nil {
		t.Fatalf("DoGenerate failed: %v", err)
	}

	var text string
	var hasReasoning bool
	for _, c := range result.Content {
		if tc, ok := c.(languagemodel.Text); ok {
			text += tc.Text
		}
		if _, ok := c.(languagemodel.Reasoning); ok {
			hasReasoning = true
		}
	}
	if text == "" {
		t.Fatal("expected non-empty text response")
	}

	t.Logf("GLM response: %q (has_reasoning=%v)", text, hasReasoning)
}

func TestGLM_ChatStream(t *testing.T) {
	loadEnv(t)
	apiKey := skipIfNoKey(t, "GLM_API_KEY")

	provider := openaicompatible.NewProvider(openaicompatible.ProviderSettings{
		BaseURL: "https://api.z.ai/api/coding/paas/v4",
		Name:    "glm",
		APIKey:  apiKey,
	})
	model := provider.ChatModel("glm-5")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// GLM-5 is a reasoning model; needs more tokens for chain-of-thought + answer.
	streamResult, err := model.DoStream(languagemodel.CallOptions{
		Ctx: ctx,
		Prompt: languagemodel.Prompt{
			languagemodel.UserMessage{
				Content: []languagemodel.UserMessagePart{
					languagemodel.TextPart{Text: "Say hello in 3 different languages."},
				},
			},
		},
		MaxOutputTokens: intPtr(500),
	})
	if err != nil {
		t.Fatalf("DoStream failed: %v", err)
	}

	var fullText string
	for part := range streamResult.Stream {
		switch p := part.(type) {
		case languagemodel.StreamPartTextDelta:
			fullText += p.Delta
		case languagemodel.StreamPartFinish:
			t.Logf("Stream finish: reason=%s", p.FinishReason.Unified)
		}
	}

	if fullText == "" {
		t.Fatal("expected non-empty streamed text")
	}
	t.Logf("GLM Stream response: %q", fullText)
}

// --- Summary test ---

func TestAllProviders_Summary(t *testing.T) {
	loadEnv(t)

	providers := []struct {
		name   string
		envVar string
	}{
		{"OpenAI", "OPENAI_API_KEY"},
		{"MiniMax", "MINIMAX_API_KEY"},
		{"GLM", "GLM_API_KEY"},
	}

	fmt.Println("\n=== Provider API Key Status ===")
	for _, p := range providers {
		key := os.Getenv(p.envVar)
		status := "✓ available"
		if key == "" {
			status = "✗ not set"
		}
		fmt.Printf("  %s (%s): %s\n", p.name, p.envVar, status)
	}
}

func intPtr(v int) *int { return &v }

func fmtTokens(p *int) string {
	if p == nil {
		return "n/a"
	}
	return fmt.Sprintf("%d", *p)
}
