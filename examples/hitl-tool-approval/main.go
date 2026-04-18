// Command hitl-tool-approval demonstrates synchronous human-in-
// the-loop: a tool marked `requireApproval: true` pauses the
// agent mid-generation, the Go side reviews the pending call,
// and either approves or declines before the agent resumes.
//
// brainkit ships a `generateWithApproval` helper that routes
// the suspend + approve/decline through a bus topic, so the Go
// caller's job is just to subscribe to that topic and publish a
// decision.
//
// Three demo turns:
//
//  1. approve  — Go side auto-approves, agent completes
//  2. decline  — Go side rejects, agent falls back
//  3. no-op    — Prompt doesn't trigger the tool; approval never
//                fires
//
// Requires OPENAI_API_KEY.
//
// Run from the repo root:
//
//	OPENAI_API_KEY=sk-... go run ./examples/hitl-tool-approval
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync/atomic"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("hitl-tool-approval: %v", err)
	}
}

func run() error {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "hitl-tool-approval-demo",
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

	// Mode flag: "approve" / "decline" — the approval listener
	// reads it from an atomic.Pointer so we can switch between
	// turns without redeploying.
	var mode atomic.Pointer[string]
	approveMode := "approve"
	mode.Store(&approveMode)

	// Auto-approver: subscribe to the approval topic the .ts
	// publishes on. Reply with {approved: true/false} per mode.
	unsub, err := kit.SubscribeRaw(ctx, "demo.approvals", func(msg sdk.Message) {
		var pending struct {
			ToolCallID string         `json:"toolCallId"`
			ToolName   string         `json:"toolName"`
			Args       map[string]any `json:"args"`
		}
		_ = json.Unmarshal(msg.Payload, &pending)
		current := *mode.Load()
		fmt.Printf("        approval request: tool=%s args=%v → decision=%s\n",
			pending.ToolName, pending.Args, current)
		decision := map[string]bool{"approved": current == "approve"}
		_ = sdk.Reply(kit, ctx, msg, decision)
	})
	if err != nil {
		return fmt.Errorf("subscribe approvals: %w", err)
	}
	defer unsub()

	if _, err := kit.Deploy(ctx, brainkit.PackageInline("hitl-tool-approval", "hitl.ts", hitlSource)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}
	fmt.Println("[1/4] hitl-tool-approval deployed")

	type reply struct {
		Text         string `json:"text"`
		FinishReason string `json:"finishReason"`
		Error        string `json:"error,omitempty"`
	}
	ask := func(prompt string) (reply, error) {
		payload := json.RawMessage(fmt.Sprintf(`{"prompt":%q}`, prompt))
		return brainkit.Call[sdk.CustomMsg, reply](kit, ctx, sdk.CustomMsg{
			Topic:   "ts.hitl-tool-approval.delete",
			Payload: payload,
		}, brainkit.WithCallTimeout(60*time.Second))
	}

	// ── Turn 1: approve ───────────────────────────────────────
	fmt.Println("\n[2/4] approve path — agent asked to delete record xyz-789")
	r1, err := ask("Delete record xyz-789.")
	if err != nil {
		return fmt.Errorf("approve: %w", err)
	}
	if r1.Error != "" {
		fmt.Printf("        error: %s\n", r1.Error)
	} else {
		fmt.Printf("        finishReason: %s\n        reply: %s\n", r1.FinishReason, r1.Text)
	}

	// ── Turn 2: decline ───────────────────────────────────────
	fmt.Println("\n[3/4] decline path — same prompt, Go-side rejects")
	declineMode := "decline"
	mode.Store(&declineMode)
	r2, err := ask("Delete record abc-123.")
	if err != nil {
		return fmt.Errorf("decline: %w", err)
	}
	if r2.Error != "" {
		fmt.Printf("        decline surfaced: %s\n", truncate(r2.Error, 300))
	} else {
		fmt.Printf("        finishReason: %s\n        reply: %s\n", r2.FinishReason, r2.Text)
	}

	// ── Turn 3: no approval needed ────────────────────────────
	fmt.Println("\n[4/4] no-op path — prompt doesn't trigger the tool")
	mode.Store(&approveMode)
	r3, err := ask("Just say hello to the admin, no deletion needed.")
	if err != nil {
		return fmt.Errorf("no-op: %w", err)
	}
	fmt.Printf("        finishReason: %s\n        reply: %s\n", r3.FinishReason, r3.Text)
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

const hitlSource = `
const deleteTool = createTool({
    id: "delete-record",
    description: "Delete a record by ID. Requires human approval.",
    inputSchema: z.object({ id: z.string() }),
    outputSchema: z.object({ deleted: z.boolean() }),
    requireApproval: true,
    execute: async (args) => {
        const input = (args && args.context) || args || {};
        return { deleted: true };
    },
});

const agent = new Agent({
    name: "hitl-demo-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions:
        "When asked to delete a record, call the delete-record tool exactly once with the id mentioned. " +
        "If the tool call is declined, explain briefly that deletion was not performed. " +
        "If no deletion is mentioned, just answer normally.",
    tools: { "delete-record": deleteTool },
});
kit.register("agent", "hitl-demo-agent", agent);

bus.on("delete", async (msg) => {
    const prompt = (msg.payload && msg.payload.prompt) || "";
    try {
        const result = await generateWithApproval(agent, prompt, {
            approvalTopic: "demo.approvals",
            timeout: 30000,
        });
        msg.reply({
            text: result.text || "",
            finishReason: result.finishReason || "",
        });
    } catch (e) {
        msg.reply({
            text: "",
            finishReason: "error",
            error: String((e && e.message) || e),
        });
    }
});
`
