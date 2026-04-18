// Command workflows demonstrates declarative multi-step
// workflows. Wires modules/workflow, deploys a .ts that
// registers a 3-step pipeline (research → draft → review),
// runs it via brainkit.CallWorkflowStart, prints each step's
// output.
//
// Run from the repo root:
//
//	go run ./examples/workflows
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/brainlet/brainkit"
	workflowmod "github.com/brainlet/brainkit/modules/workflow"
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("workflows: %v", err)
	}
}

func run() error {
	kit, err := brainkit.New(brainkit.Config{
		Namespace: "workflows-demo",
		Transport: brainkit.Memory(),
		FSRoot:    ".",
		Modules: []brainkit.Module{
			workflowmod.New(),
		},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Deploy a .ts that registers three steps chained into one
	// workflow.
	tsCode := `
		const researchStep = createStep({
			id: "research",
			inputSchema: z.object({ topic: z.string() }),
			outputSchema: z.object({ notes: z.string() }),
			execute: async ({ inputData }) => ({
				notes: "researched: " + inputData.topic,
			}),
		});
		const draftStep = createStep({
			id: "draft",
			inputSchema: z.object({ notes: z.string() }),
			outputSchema: z.object({ draft: z.string() }),
			execute: async ({ inputData }) => ({
				draft: "drafted from " + inputData.notes,
			}),
		});
		const reviewStep = createStep({
			id: "review",
			inputSchema: z.object({ draft: z.string() }),
			outputSchema: z.object({ approved: z.boolean(), final: z.string() }),
			execute: async ({ inputData }) => ({
				approved: true,
				final: inputData.draft + " [reviewed]",
			}),
		});

		const wf = createWorkflow({
			id: "research-pipeline",
			inputSchema: z.object({ topic: z.string() }),
			outputSchema: z.object({ approved: z.boolean(), final: z.string() }),
		})
			.then(researchStep)
			.then(draftStep)
			.then(reviewStep)
			.commit();

		kit.register("workflow", "research-pipeline", wf);
	`
	if _, err := kit.Deploy(ctx, brainkit.PackageInline("workflows-demo", "wf.ts", tsCode)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	// Kick off a run.
	fmt.Println("starting research-pipeline for topic=\"brainkit\"…")
	resp, err := brainkit.CallWorkflowStart(kit, ctx, sdk.WorkflowStartMsg{
		Name:      "research-pipeline",
		InputData: json.RawMessage(`{"topic":"brainkit"}`),
	}, brainkit.WithCallTimeout(15*time.Second))
	if err != nil {
		return fmt.Errorf("workflow.start: %w", err)
	}

	fmt.Printf("run id:   %s\n", resp.RunID)
	fmt.Printf("status:   %s\n", resp.Status)
	if len(resp.Steps) > 0 {
		fmt.Println("steps:")
		var pretty any
		if err := json.Unmarshal(resp.Steps, &pretty); err == nil {
			enc := json.NewEncoder(log.Writer())
			_ = enc
			b, _ := json.MarshalIndent(pretty, "  ", "  ")
			fmt.Printf("  %s\n", string(b))
		} else {
			fmt.Println(string(resp.Steps))
		}
	}

	return nil
}
