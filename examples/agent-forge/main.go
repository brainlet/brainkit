// Command agent-forge is the flagship meta-programming example.
// A Go process boots a Kit, deploys a multi-agent forge
// pipeline, then asks the forge to design, write, and review a
// brand-new brainkit agent from a freeform request. When the
// forge returns approved source, Go scaffolds a proper on-disk
// package (manifest.json + tsconfig.json + types/* + index.ts)
// via brainkit.ScaffoldPackage, deploys it from that directory
// via brainkit.PackageFromDir, and calls the forged agent
// through its public bus topic.
//
// The scaffolded directory survives the process so you can open
// the forged agent in an IDE — the tsconfig already paths-maps
// the shipped .d.ts types, no npm install required.
//
// The forge itself is a Mastra workflow inside a SES Compartment
// wiring every major brainkit primitive: Agents, createTool,
// createWorkflow / createStep, dountil review loop, sub-agents
// running in parallel, structured output via Zod, and the
// bundled reference corpus (reference.get("everything")).
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
//	-out      scaffold destination (default ./forged-agents)
//	-keep     keep the scaffolded dir on exit (default true)
package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
)

//go:embed forge.ts
var forgeSource string

// ForgeResult mirrors the workflow terminal output. Deploy is
// done by this Go process now, not by the workflow — the forge
// just returns approved source.
type ForgeResult struct {
	Approved   bool         `json:"approved"`
	Name       string       `json:"name"`
	Code       string       `json:"code"`
	Iterations int          `json:"iterations"`
	Issues     []ForgeIssue `json:"issues"`
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
	outRaw := flag.String("out", "./forged-agents",
		"scaffold destination — each forged agent becomes ./<out>/<name>/")
	keep := flag.Bool("keep", true,
		"keep the scaffolded forged agent on disk after exit (default true)")
	flag.Parse()

	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		log.Fatalf("OPENAI_API_KEY is required — agent-forge runs multiple live OpenAI calls")
	}

	out, err := filepath.Abs(*outRaw)
	if err != nil {
		log.Fatalf("resolve out: %v", err)
	}
	if err := run(key, *request, *askPrompt, out, *keep); err != nil {
		log.Fatalf("agent-forge: %v", err)
	}
}

func run(apiKey, request, askPrompt, outRoot string, keep bool) error {
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
	fmt.Println("[1/4] forge pipeline deployed")
	fmt.Printf("        request: %q\n", request)

	// ── Step 2: drive the forge ─────────────────────────────────
	fmt.Println("[2/4] running forge workflow (architect → coder → reviewer loop)…")
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
	fmt.Printf("        approved=%v  name=%q\n", result.Approved, result.Name)
	if len(result.Issues) > 0 {
		fmt.Println("        unresolved issues:")
		for _, iss := range result.Issues {
			fmt.Printf("          [%s] %s\n", iss.Category, iss.Message)
		}
	}

	if !result.Approved {
		fmt.Println()
		fmt.Println("forge did not reach approval within the iteration cap. Returning best-effort code so a human can finish the job:")
		fmt.Println()
		fmt.Println(result.Code)
		return nil
	}

	// ── Step 3: scaffold the forged agent on disk + deploy ──────
	packageDir := filepath.Join(outRoot, result.Name)
	if _, statErr := os.Stat(packageDir); statErr == nil {
		// Previous run of the same forge output — wipe so the
		// scaffold goes in clean.
		if err := os.RemoveAll(packageDir); err != nil {
			return fmt.Errorf("clear stale scaffold at %s: %w", packageDir, err)
		}
	}
	if !keep {
		defer os.RemoveAll(packageDir)
	}

	fmt.Printf("[3/4] scaffolding forged agent at %s\n", packageDir)
	if err := brainkit.ScaffoldPackage(packageDir, result.Name, "index.ts", result.Code); err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}
	listScaffold(packageDir)

	pkg, err := brainkit.PackageFromDir(packageDir)
	if err != nil {
		return fmt.Errorf("PackageFromDir: %w", err)
	}
	if _, err := kit.Deploy(ctx, pkg); err != nil {
		return fmt.Errorf("deploy forged agent: %w", err)
	}
	topic := "ts." + result.Name + ".ask"
	fmt.Printf("        deployed — callable on %s\n", topic)

	// ── Step 4: call the forged agent directly ──────────────────
	fmt.Printf("[4/4] calling %s with prompt=%q\n", topic, askPrompt)
	askPayload := json.RawMessage(fmt.Sprintf(`{"prompt":%q}`, askPrompt))
	spawnedReply, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{
		Topic:   topic,
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
	if keep {
		fmt.Println()
		fmt.Printf("Forged agent kept on disk at: %s\n", packageDir)
		fmt.Println("  Open it in an IDE — tsconfig.json + types/ give full autocomplete.")
		fmt.Println("  Edit index.ts and redeploy via `brainkit deploy` or a fresh run of this example.")
	}
	return nil
}

func listScaffold(dir string) {
	_ = filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		fmt.Printf("          %s  (%d bytes)\n", rel, info.Size())
		return nil
	})
}
