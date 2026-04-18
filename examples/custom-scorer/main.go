// Command custom-scorer demonstrates the Mastra `createScorer`
// builder for domain-specific quality metrics. Two scorers run
// side by side against the same dataset:
//
//  1. "cites-sources-regex" — code-only, deterministic: regex-
//     counts `[doc:<id>]` markers. Cheap, zero LLM calls.
//  2. "cites-sources-llm"   — LLM-as-judge via `.generateScore`
//     with `judge + createPrompt`. Slower, costs tokens, but
//     picks up cases the regex misses ("according to doc 7" →
//     the regex scores 0, the judge scores 1).
//
// Go side prints per-item scores from each side by side so the
// disagreement is visible.
//
// Requires OPENAI_API_KEY.
//
// Run from the repo root:
//
//	OPENAI_API_KEY=sk-... go run ./examples/custom-scorer
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("custom-scorer: %v", err)
	}
}

type evalItem struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type scoreRow struct {
	Input      string  `json:"input"`
	RegexScore float64 `json:"regexScore"`
	LLMScore   float64 `json:"llmScore"`
	LLMReason  string  `json:"llmReason,omitempty"`
}

type evalReply struct {
	Rows []scoreRow `json:"rows"`
}

func run() error {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "custom-scorer-demo",
		Transport: brainkit.Memory(),
		FSRoot:    ".",
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	if _, err := kit.Deploy(ctx, brainkit.PackageInline("custom-scorer", "scorer.ts", scorerSource)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}
	fmt.Println("[1/3] custom-scorer deployed")

	// Dataset — mix of explicit markers, implicit references, and
	// naked outputs. Lets the two scorers disagree.
	dataset := []evalItem{
		{"What is brainkit?", "brainkit is a Go runtime for AI agents [doc:1]."},
		{"Where does it ship?", "It ships as a Go module."},
		{"Quote from the docs?", "According to doc 7, the Kit owns the bus."},
		{"Name the modules.", "The module set includes gateway, audit, tracing, plugins [doc:2] [doc:3]."},
		{"Is it fast?", "Yes, it's fast."},
	}
	datasetJSON, _ := json.Marshal(dataset)
	fmt.Printf("[2/3] running both scorers on %d items…\n", len(dataset))
	reply, err := brainkit.Call[sdk.CustomMsg, evalReply](kit, ctx, sdk.CustomMsg{
		Topic:   "ts.custom-scorer.score",
		Payload: datasetJSON,
	}, brainkit.WithCallTimeout(90*time.Second))
	if err != nil {
		return fmt.Errorf("score: %w", err)
	}

	fmt.Println("\n[3/3] results:")
	fmt.Printf("  %-40s  %6s  %6s  %s\n", "input", "regex", "llm", "reason")
	disagreed := 0
	var regexSum, llmSum float64
	for _, row := range reply.Rows {
		regexSum += row.RegexScore
		llmSum += row.LLMScore
		mark := "  "
		if (row.RegexScore >= 0.5) != (row.LLMScore >= 0.5) {
			mark = "≠ "
			disagreed++
		}
		fmt.Printf("  %s%-40s  %6.2f  %6.2f  %s\n",
			mark, truncate(row.Input, 40), row.RegexScore, row.LLMScore, truncate(row.LLMReason, 60))
	}
	n := float64(len(reply.Rows))
	if n > 0 {
		fmt.Printf("\naverage regex:%.2f  llm:%.2f  disagreements:%d/%d\n",
			regexSum/n, llmSum/n, disagreed, len(reply.Rows))
	}
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

const scorerSource = `
// Code-only scorer: regex counts the [doc:<id>] marker. Zero
// LLM calls. Deterministic.
const regexScorer = createScorer({
    id: "cites-sources-regex",
    name: "Cites-sources (regex)",
    description: "Returns 1.0 iff the output contains at least one [doc:<id>] marker.",
}).generateScore(({ run }) => {
    const text = (run && run.output && run.output.text) || "";
    return /\[doc:\w+\]/.test(text) ? 1.0 : 0.0;
});

// LLM-judge scorer: a small LLM decides whether the response
// cites sources, including implicit references like "according
// to doc 7".
const llmScorer = createScorer({
    id: "cites-sources-llm",
    name: "Cites-sources (LLM judge)",
    description: "Returns 1.0 when the response cites a source explicitly or implicitly, else 0.0.",
}).generateScore({
    description: "1 when the output cites any source (explicit [doc:id] or implicit like 'according to doc 7'), else 0.",
    judge: {
        model: model("openai", "gpt-4o-mini"),
        instructions:
            "You grade whether an assistant reply cites a source. Citations can be explicit markers like '[doc:1]' OR implicit phrases like 'according to doc 7'. Reply with ONLY the number 0 or 1.",
    },
    createPrompt: ({ run }) => {
        const q = (run && run.input && run.input[0] && run.input[0].content) || "";
        const a = (run && run.output && run.output.text) || "";
        return "Question: " + q + "\\nAnswer: " + a + "\\n\\nDoes the answer cite any source (explicit or implicit)? Reply 0 or 1.";
    },
}).generateReason(({ results }) => {
    const s = results && results.generateScoreStepResult;
    if (typeof s === "number") {
        return s >= 0.5 ? "Citation detected." : "No citation found.";
    }
    return "Unable to score.";
});

bus.on("score", async (msg) => {
    const items = Array.isArray(msg.payload) ? msg.payload : [];
    const rows = [];
    for (const item of items) {
        const run = {
            input: [{ role: "user", content: item.input }],
            output: { role: "assistant", text: item.output },
        };
        const [rx, jd] = await Promise.all([
            regexScorer.run(run),
            llmScorer.run(run),
        ]);
        rows.push({
            input: item.input,
            regexScore: Number(rx.score) || 0,
            llmScore: Number(jd.score) || 0,
            llmReason: String(jd.reason || ""),
        });
    }
    msg.reply({ rows });
});
`
