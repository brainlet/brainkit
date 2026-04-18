// Command ai-chat demonstrates the library-mode AI surface:
// register a provider, deploy a .ts package that calls
// generateText through the bundled AI SDK, print the result.
//
// Requires OPENAI_API_KEY (or an override via --provider/--api-key
// flags). See README for Anthropic / other provider variants.
//
// Run from the repo root:
//
//	OPENAI_API_KEY=sk-... go run ./examples/ai-chat
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	provider := flag.String("provider", "openai", "provider name (openai, anthropic, google, …)")
	modelID := flag.String("model", "gpt-4o-mini", "model identifier passed to model(provider, modelID)")
	prompt := flag.String("prompt", "Say hello to world. One sentence only.", "prompt to send to the model")
	apiKey := flag.String("api-key", "", "provider API key (defaults to $<PROVIDER>_API_KEY)")
	flag.Parse()

	key := resolveAPIKey(*provider, *apiKey)
	if key == "" {
		log.Fatalf("no API key for %q — set %s or --api-key",
			*provider, envKeyFor(*provider))
	}

	providerCfg, err := buildProvider(*provider, key)
	if err != nil {
		log.Fatalf("%v", err)
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "ai-chat-demo",
		Transport: brainkit.Memory(),
		FSRoot:    ".",
		Providers: []brainkit.ProviderConfig{providerCfg},
	})
	if err != nil {
		log.Fatalf("new kit: %v", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Deploy a .ts handler that calls generateText through the
	// bundled AI SDK. The handler replies with { text, usage }.
	code := fmt.Sprintf(`
		bus.on("chat", async (msg) => {
			const result = await generateText({
				model: model(%q, %q),
				prompt: msg.payload.prompt,
				maxTokens: 200,
			});
			msg.reply({
				text: result.text,
				usage: result.usage,
				finishReason: result.finishReason,
			});
		});
	`, *provider, *modelID)

	if _, err := kit.Deploy(ctx, brainkit.PackageInline("chatter", "chatter.ts", code)); err != nil {
		log.Fatalf("deploy: %v", err)
	}

	// Call the deployed handler via the bus. CustomMsg carries
	// dynamic topics so it doesn't get a CallCustomMsg wrapper —
	// drop to the generic Call which accepts any Resp type.
	payload := json.RawMessage(fmt.Sprintf(`{"prompt":%q}`, *prompt))
	reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{
		Topic:   "ts.chatter.chat",
		Payload: payload,
	}, brainkit.WithCallTimeout(45*time.Second))
	if err != nil {
		log.Fatalf("call: %v", err)
	}

	// Pretty-print the result.
	fmt.Printf("provider=%s  model=%s\n---\n", *provider, *modelID)
	var out map[string]any
	if err := json.Unmarshal(reply, &out); err != nil {
		fmt.Println(string(reply))
		return
	}
	if text, ok := out["text"].(string); ok {
		fmt.Println(text)
	}
	if u, ok := out["usage"].(map[string]any); ok {
		fmt.Printf("---\ntokens: prompt=%v completion=%v total=%v\n",
			u["promptTokens"], u["completionTokens"], u["totalTokens"])
	}
}

// resolveAPIKey picks an API key from (in priority) --api-key
// flag, the provider's standard env var, or empty.
func resolveAPIKey(provider, override string) string {
	if override != "" {
		return override
	}
	return os.Getenv(envKeyFor(provider))
}

func envKeyFor(provider string) string {
	switch provider {
	case "openai":
		return "OPENAI_API_KEY"
	case "anthropic":
		return "ANTHROPIC_API_KEY"
	case "google":
		return "GOOGLE_API_KEY"
	case "groq":
		return "GROQ_API_KEY"
	case "mistral":
		return "MISTRAL_API_KEY"
	case "deepseek":
		return "DEEPSEEK_API_KEY"
	case "xai":
		return "XAI_API_KEY"
	case "cohere":
		return "COHERE_API_KEY"
	case "perplexity":
		return "PERPLEXITY_API_KEY"
	case "togetherai":
		return "TOGETHERAI_API_KEY"
	case "fireworks":
		return "FIREWORKS_API_KEY"
	case "cerebras":
		return "CEREBRAS_API_KEY"
	default:
		return ""
	}
}

func buildProvider(provider, key string) (brainkit.ProviderConfig, error) {
	switch provider {
	case "openai":
		return brainkit.OpenAI(key), nil
	case "anthropic":
		return brainkit.Anthropic(key), nil
	case "google":
		return brainkit.Google(key), nil
	case "mistral":
		return brainkit.Mistral(key), nil
	case "groq":
		return brainkit.Groq(key), nil
	case "deepseek":
		return brainkit.DeepSeek(key), nil
	case "xai":
		return brainkit.XAI(key), nil
	case "cohere":
		return brainkit.Cohere(key), nil
	case "perplexity":
		return brainkit.Perplexity(key), nil
	case "togetherai":
		return brainkit.TogetherAI(key), nil
	case "fireworks":
		return brainkit.Fireworks(key), nil
	case "cerebras":
		return brainkit.Cerebras(key), nil
	default:
		return brainkit.ProviderConfig{}, fmt.Errorf("unsupported provider %q", provider)
	}
}
