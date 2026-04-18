// Command hitl-workflow demonstrates out-of-band human-in-the-loop
// via Mastra workflows: a step calls `suspend({reason})`, the
// workflow state persists to SQLite, and a separate Go-side
// decision calls `run.resume({step, resumeData})` to continue.
//
// No LLM is involved — the steps are deterministic — so the
// example runs without an API key. It's the counterpart to
// session 06's tool-approval flow which pauses mid-generation;
// this one pauses between steps and survives process restart
// when you configure a real storage backend.
//
// Run from the repo root:
//
//	go run ./examples/hitl-workflow
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/brainlet/brainkit"
	workflowmod "github.com/brainlet/brainkit/modules/workflow"
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("hitl-workflow: %v", err)
	}
}

func run() error {
	tmp, err := os.MkdirTemp("", "brainkit-hitl-workflow-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	// SQLite storage on the "default" slot. patches.js picks this
	// up + upgrades workflow's internal InMemoryStore. That's what
	// makes suspended runs durable across a process restart.
	kit, err := brainkit.New(brainkit.Config{
		Namespace: "hitl-workflow-demo",
		Transport: brainkit.Memory(),
		FSRoot:    tmp,
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmp, "workflow.db")),
		},
		Modules: []brainkit.Module{workflowmod.New()},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := kit.Deploy(ctx, brainkit.PackageInline("hitl-workflow-demo", "wf.ts", workflowSource)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}
	fmt.Println("[1/4] deploy-pipeline workflow registered")

	// Start the run.
	fmt.Println("[2/4] starting run — build step fires, approve step will suspend")
	start, err := brainkit.CallWorkflowStart(kit, ctx, sdk.WorkflowStartMsg{
		Name:      "deploy-pipeline",
		InputData: json.RawMessage(`{"component":"brainkit","env":"staging"}`),
	}, brainkit.WithCallTimeout(15*time.Second))
	if err != nil {
		return fmt.Errorf("workflow.start: %w", err)
	}
	runID := start.RunID
	fmt.Printf("        runId=%s  status=%s\n", runID, start.Status)
	if start.Status != "suspended" {
		return fmt.Errorf("expected suspended status after build step, got %q", start.Status)
	}
	printSuspendReason(start.Steps)

	// Out-of-band decision. In a real system this is where a UI /
	// Slack bot / pager waits for the human.
	fmt.Println("[3/4] decision — auto-approving")
	approved := true

	// Resume.
	fmt.Println("[4/4] resuming with the decision")
	resumeData, _ := json.Marshal(map[string]any{"approved": approved, "approver": "alice@example.com"})
	resume, err := brainkit.CallWorkflowResume(kit, ctx, sdk.WorkflowResumeMsg{
		Name:       "deploy-pipeline",
		RunID:      runID,
		Step:       "approve",
		ResumeData: resumeData,
	}, brainkit.WithCallTimeout(15*time.Second))
	if err != nil {
		return fmt.Errorf("workflow.resume: %w", err)
	}
	fmt.Printf("        status=%s\n", resume.Status)
	printFinalOutput(resume.Steps)
	return nil
}

func printSuspendReason(steps json.RawMessage) {
	if len(steps) == 0 {
		return
	}
	var tree map[string]any
	if err := json.Unmarshal(steps, &tree); err != nil {
		return
	}
	// Mastra nests step results under {"steps": {"step-id": {...}}}.
	if approve, ok := digStep(tree, "approve"); ok {
		if payload, ok := approve["suspendedPayload"].(map[string]any); ok {
			if reason, ok := payload["reason"].(string); ok {
				fmt.Printf("        suspended: %q\n", reason)
				return
			}
		}
		if reason, ok := approve["reason"].(string); ok {
			fmt.Printf("        suspended: %q\n", reason)
		}
	}
}

func printFinalOutput(steps json.RawMessage) {
	if len(steps) == 0 {
		return
	}
	var tree map[string]any
	if err := json.Unmarshal(steps, &tree); err != nil {
		return
	}
	if publish, ok := digStep(tree, "publish"); ok {
		if output, ok := publish["output"].(map[string]any); ok {
			b, _ := json.MarshalIndent(output, "        ", "  ")
			fmt.Printf("        publish output: %s\n", string(b))
		}
	}
}

// digStep unwraps either {steps: {id: {...}}} or a raw {id: {...}} tree.
func digStep(tree map[string]any, id string) (map[string]any, bool) {
	if inner, ok := tree["steps"].(map[string]any); ok {
		if step, ok := inner[id].(map[string]any); ok {
			return step, true
		}
	}
	if step, ok := tree[id].(map[string]any); ok {
		return step, true
	}
	return nil, false
}

const workflowSource = `
const build = createStep({
    id: "build",
    inputSchema: z.object({ component: z.string(), env: z.string() }),
    outputSchema: z.object({ version: z.string(), component: z.string(), env: z.string() }),
    execute: async ({ inputData }) => {
        const version = "v" + Math.floor(Date.now() / 1000);
        return { version, component: inputData.component, env: inputData.env };
    },
});

const approve = createStep({
    id: "approve",
    inputSchema: z.object({ version: z.string(), component: z.string(), env: z.string() }),
    outputSchema: z.object({ version: z.string(), component: z.string(), env: z.string(), approved: z.boolean(), approver: z.string().optional() }),
    resumeSchema: z.object({ approved: z.boolean(), approver: z.string().optional() }),
    suspendSchema: z.object({ reason: z.string(), artifact: z.string() }),
    execute: async ({ inputData, resumeData, suspend }) => {
        if (!resumeData) {
            return await suspend({
                reason: "manual approval needed to publish " + inputData.component + "@" + inputData.version + " to " + inputData.env,
                artifact: inputData.component + "@" + inputData.version,
            });
        }
        return {
            version: inputData.version,
            component: inputData.component,
            env: inputData.env,
            approved: !!resumeData.approved,
            approver: resumeData.approver,
        };
    },
});

const publish = createStep({
    id: "publish",
    inputSchema: z.object({ version: z.string(), component: z.string(), env: z.string(), approved: z.boolean(), approver: z.string().optional() }),
    outputSchema: z.object({ published: z.boolean(), releaseUrl: z.string().optional(), aborted: z.boolean().optional() }),
    execute: async ({ inputData }) => {
        if (!inputData.approved) {
            return { published: false, aborted: true };
        }
        return {
            published: true,
            releaseUrl: "https://releases.example.com/" + inputData.component + "/" + inputData.version,
        };
    },
});

const wf = createWorkflow({
    id: "deploy-pipeline",
    inputSchema: z.object({ component: z.string(), env: z.string() }),
    outputSchema: z.object({ published: z.boolean(), releaseUrl: z.string().optional(), aborted: z.boolean().optional() }),
})
    .then(build)
    .then(approve)
    .then(publish)
    .commit();

kit.register("workflow", "deploy-pipeline", wf);
`
