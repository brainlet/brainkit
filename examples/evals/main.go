// Command evals runs Mastra's `runEvals` over a small dataset
// with two prebuilt scorers (answer-relevancy + completeness)
// and compares the aggregate scores to a committed baseline.
// Regressions beyond `tolerance_percent` exit non-zero — the
// shape of a CI quality gate.
//
// Dataset lives in dataset.json; baseline in baseline.json. Run
// `make evals-save` to capture a fresh baseline; inspect +
// commit when the new numbers are intentional.
//
// Requires OPENAI_API_KEY.
//
// Run from the repo root:
//
//	OPENAI_API_KEY=sk-... go run ./examples/evals
package main

import (
	_ "embed"
	"context"
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

//go:embed dataset.json
var datasetJSON []byte

type baselineDoc struct {
	TolerancePct float64            `json:"_tolerance_percent"`
	Scores       map[string]float64 `json:"scores"`
}

type evalResult struct {
	Scores  map[string]float64 `json:"scores"`
	Summary struct {
		TotalItems int `json:"totalItems"`
	} `json:"summary"`
}

func main() {
	checkOnly := flag.Bool("check", false, "compare run result to baseline.json and exit 1 on regression")
	save := flag.Bool("save", false, "write run result to latest.json for later inspection")
	flag.Parse()

	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		log.Fatalf("OPENAI_API_KEY is required")
	}
	if err := run(key, *save, *checkOnly); err != nil {
		log.Fatalf("evals: %v", err)
	}
}

func run(key string, save, check bool) error {
	kit, err := brainkit.New(brainkit.Config{
		Namespace: "evals-demo",
		Transport: brainkit.Memory(),
		FSRoot:    ".",
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if _, err := kit.Deploy(ctx, brainkit.PackageInline("evals", "evals.ts", evalsSource)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}
	fmt.Println("[1/3] evals deployed")

	fmt.Println("[2/3] running runEvals …")
	started := time.Now()
	reply, err := brainkit.Call[sdk.CustomMsg, evalResult](kit, ctx, sdk.CustomMsg{
		Topic:   "ts.evals.run",
		Payload: datasetJSON,
	}, brainkit.WithCallTimeout(9*time.Minute))
	if err != nil {
		return fmt.Errorf("runEvals: %w", err)
	}
	fmt.Printf("        %d items scored in %s\n", reply.Summary.TotalItems, time.Since(started).Round(time.Second))
	fmt.Println("        scorer averages:")
	for name, score := range reply.Scores {
		fmt.Printf("          %-20s %.3f\n", name, score)
	}

	if save {
		dir, _ := filepath.Abs("examples/evals")
		target := filepath.Join(dir, "latest.json")
		b, _ := json.MarshalIndent(reply, "", "  ")
		if err := os.WriteFile(target, b, 0o644); err != nil {
			return err
		}
		fmt.Printf("\nwrote %s\n", target)
	}

	if check {
		fmt.Println("\n[3/3] comparing to baseline.json:")
		base, err := os.ReadFile("examples/evals/baseline.json")
		if err != nil {
			return fmt.Errorf("read baseline: %w", err)
		}
		var doc baselineDoc
		if err := json.Unmarshal(base, &doc); err != nil {
			return fmt.Errorf("parse baseline: %w", err)
		}
		tol := doc.TolerancePct
		if tol <= 0 {
			tol = 25
		}
		failed := 0
		for name, basedScore := range doc.Scores {
			got := reply.Scores[name]
			delta := (basedScore - got) / basedScore * 100
			status := "ok"
			if delta > tol {
				status = "REGRESS"
				failed++
			}
			fmt.Printf("  %-20s  base=%.3f  got=%.3f  delta=%+.1f%%  %s\n",
				name, basedScore, got, -delta, status)
		}
		if failed > 0 {
			return fmt.Errorf("%d scorer(s) regressed beyond %.0f%%", failed, tol)
		}
		fmt.Printf("all scorers within %.0f%% tolerance\n", tol)
	}
	return nil
}

const evalsSource = `
const agent = new Agent({
    name: "evals-demo-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You answer concisely. One or two sentences maximum.",
});
kit.register("agent", "evals-demo-agent", agent);

const judge = model("openai", "gpt-4o-mini");
const relevancy = createAnswerRelevancyScorer({ model: judge });
const completeness = createCompletenessScorer();

bus.on("run", async (msg) => {
    const data = Array.isArray(msg.payload) ? msg.payload : [];
    const result = await runEvals({
        target: agent,
        data,
        scorers: [relevancy, completeness],
        concurrency: 2,
    });
    msg.reply({
        scores: result.scores || {},
        summary: result.summary || { totalItems: data.length },
    });
});
`
