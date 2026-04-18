// Command agent-forge is the flagship meta-programming example:
// a Go process that boots a Kit, deploys a multi-agent forge
// pipeline, then asks the forge to design, write, review, and
// deploy a brand-new brainkit agent from a freeform request.
// After the forge completes, Go calls the newly forged agent
// through its public bus topic.
//
// The forge itself is a Mastra workflow inside a SES Compartment.
// It wires — in a single .ts file — every major brainkit
// primitive: Agents, createTool, createWorkflow / createStep,
// dountil review loop, sub-agents via `agents: {}`, structured
// output via Zod, the bundled reference corpus, and in-JS
// package deployment via `bus.call("package.deploy", …)`.
//
// Requires OPENAI_API_KEY. Every forge run makes several live
// OpenAI calls (architect + coder + three reviewers + possible
// patches + the forged agent itself). Expect ~15-60s per run
// depending on model latency and review iterations.
//
// Run from the repo root:
//
//	OPENAI_API_KEY=sk-... go run ./examples/agent-forge
//
// Flags:
//
//	-request  free-form description of the agent to forge
//	-ask      prompt sent to the forged agent after deploy
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
)

//go:embed forge.ts
var forgeSource string

type ForgeResult struct {
	Deployed   bool         `json:"deployed"`
	Name       string       `json:"name"`
	Topic      string       `json:"topic"`
	Iterations int          `json:"iterations"`
	Approved   bool         `json:"approved"`
	Issues     []ForgeIssue `json:"issues"`
	Code       string       `json:"code"`
	Error      string       `json:"error,omitempty"`
}

type ForgeIssue struct {
	Category string `json:"category"`
	Message  string `json:"message"`
}

func main() {
	request := flag.String("request",
		"An agent that turns a plain-English sentence into a single concise and witty tweet under 240 characters. Name it tweet-bot.",
		"freeform description of the agent to forge")
	askPrompt := flag.String("ask",
		"brainkit just shipped v1.0-rc.1 with a self-describing reference corpus so agents can design other agents",
		"prompt sent to the forged agent's ts.<name>.ask topic")
	flag.Parse()

	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		log.Fatalf("OPENAI_API_KEY is required — agent-forge runs multiple live OpenAI calls")
	}
	if err := run(key, *request, *askPrompt); err != nil {
		log.Fatalf("agent-forge: %v", err)
	}
}

func run(apiKey, request, askPrompt string) error {
	tmp, err := os.MkdirTemp("", "brainkit-agent-forge-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "agent-forge-demo",
		Transport: brainkit.Memory(),
		FSRoot:    tmp,
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(apiKey)},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// ── Step 1: deploy the forge pipeline ───────────────────────
	if _, err := kit.Deploy(ctx, brainkit.PackageInline("agent-forge", "forge.ts", forgeSource)); err != nil {
		return fmt.Errorf("deploy forge: %w", err)
	}
	fmt.Println("[1/3] forge pipeline deployed")
	fmt.Printf("        request: %q\n", request)

	// ── Step 2: drive the forge ─────────────────────────────────
	fmt.Println("[2/3] running forge workflow (architect → coder → reviewer loop → deploy)…")
	started := time.Now()
	payload := json.RawMessage(fmt.Sprintf(`{"request":%q}`, request))
	reply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{
		Topic:   "ts.agent-forge.create",
		Payload: payload,
	}, brainkit.WithCallTimeout(4*time.Minute))
	if err != nil {
		return fmt.Errorf("forge create: %w", err)
	}
	elapsed := time.Since(started)

	var result ForgeResult
	if err := json.Unmarshal(reply, &result); err != nil {
		return fmt.Errorf("decode forge result: %w\nraw: %s", err, string(reply))
	}

	if result.Error != "" {
		return fmt.Errorf("forge reported error: %s", result.Error)
	}
	fmt.Printf("        forge finished in %s (iterations: %d)\n", elapsed.Round(time.Second), result.Iterations)
	fmt.Printf("        approved=%v  deployed=%v  name=%q  topic=%q\n",
		result.Approved, result.Deployed, result.Name, result.Topic)
	if len(result.Issues) > 0 {
		fmt.Println("        unresolved issues:")
		for _, iss := range result.Issues {
			fmt.Printf("          [%s] %s\n", iss.Category, iss.Message)
		}
	}

	if !result.Deployed {
		fmt.Println()
		fmt.Println("forge did not reach approval within the iteration cap. Returning best-effort code so a human can finish the job:")
		fmt.Println()
		fmt.Println(result.Code)
		return nil
	}

	// ── Step 3: call the forged agent directly ──────────────────
	fmt.Printf("[3/3] calling %s with prompt=%q\n", result.Topic, askPrompt)
	askPayload := json.RawMessage(fmt.Sprintf(`{"prompt":%q}`, askPrompt))
	spawnedReply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{
		Topic:   result.Topic,
		Payload: askPayload,
	}, brainkit.WithCallTimeout(60*time.Second))
	if err != nil {
		return fmt.Errorf("call forged agent: %w", err)
	}
	var ans struct {
		Text  string         `json:"text"`
		Usage map[string]any `json:"usage"`
	}
	if err := json.Unmarshal(spawnedReply, &ans); err != nil {
		fmt.Printf("        raw reply: %s\n", string(spawnedReply))
		return nil
	}
	fmt.Println()
	fmt.Println("--- forged agent reply ---")
	fmt.Println(ans.Text)
	fmt.Println("---")
	if ans.Usage != nil {
		fmt.Printf("usage: prompt=%v completion=%v total=%v\n",
			ans.Usage["promptTokens"], ans.Usage["completionTokens"], ans.Usage["totalTokens"])
	}
	return nil
}
