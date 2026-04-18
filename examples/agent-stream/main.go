// Command agent-stream demonstrates Mastra's `agent.stream()`
// surface from inside a deployed `.ts`, piping the chunks out via
// `msg.send` over the bus → SSE gateway → browser/curl.
//
// Two topics:
//
//   - ts.agent-stream.haiku — plain text streaming. Iterates
//     `stream.textStream` and emits one chunk per token delta.
//   - ts.agent-stream.plan  — structured-output streaming. Uses
//     `structuredOutput: { schema: z.array(z.object({...})) }`.
//     Iterates `stream.fullStream` filtering for `object-result`
//     chunks; the terminal reply carries the fully assembled
//     `stream.object`.
//
// Both topics are mapped to gateway routes so curl can hit them:
//
//   SSE plain: curl -N http://127.0.0.1:<port>/sse/haiku?prompt=…
//   SSE plan:  curl -N http://127.0.0.1:<port>/sse/plan?goal=…
//
// Requires OPENAI_API_KEY.
//
// Run from the repo root:
//
//	OPENAI_API_KEY=sk-... go run ./examples/agent-stream
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/modules/gateway"
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("agent-stream: %v", err)
	}
}

func run() error {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("probe listen: %w", err)
	}
	listenAddr := probe.Addr().String()
	_ = probe.Close()

	gw := gateway.New(gateway.Config{Listen: listenAddr, Timeout: 60 * time.Second})
	gw.HandleStream("GET", "/sse/haiku", "ts.agent-stream.haiku")
	gw.HandleStream("GET", "/sse/plan", "ts.agent-stream.plan")

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "agent-stream-demo",
		Transport: brainkit.Memory(),
		FSRoot:    ".",
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
		Modules:   []brainkit.Module{gw},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := kit.Deploy(ctx, brainkit.PackageInline("agent-stream", "stream.ts", streamSource)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}
	fmt.Println("[1/3] agent-stream deployed")

	// ── Bus round trip 1: plain text streaming ──
	fmt.Println("[2/3] bus CallStream on ts.agent-stream.haiku:")
	type chunk struct {
		Delta string `json:"delta,omitempty"`
	}
	type done struct {
		Done bool `json:"done"`
	}
	callCtx, cancelCall := context.WithTimeout(ctx, 60*time.Second)
	defer cancelCall()
	_, err = brainkit.CallStream[sdk.CustomMsg, chunk, done](
		kit, callCtx,
		sdk.CustomMsg{Topic: "ts.agent-stream.haiku", Payload: json.RawMessage(`{"prompt":"autumn leaves on a mountain stream"}`)},
		func(c chunk) error {
			if c.Delta != "" {
				fmt.Print(c.Delta)
			}
			return nil
		},
		brainkit.WithCallTimeout(45*time.Second),
	)
	fmt.Println()
	if err != nil {
		return fmt.Errorf("haiku CallStream: %w", err)
	}

	// ── Bus round trip 2: structured output streaming ──
	fmt.Println("\n[3/3] bus CallStream on ts.agent-stream.plan (structured):")
	type planChunk struct {
		Step string `json:"step,omitempty"`
		Why  string `json:"why,omitempty"`
	}
	type planFinal struct {
		Object []planChunk `json:"object"`
	}
	result, err := brainkit.CallStream[sdk.CustomMsg, planChunk, planFinal](
		kit, callCtx,
		sdk.CustomMsg{Topic: "ts.agent-stream.plan", Payload: json.RawMessage(`{"goal":"ship brainkit v1.0"}`)},
		func(p planChunk) error {
			if p.Step != "" {
				fmt.Printf("  • %s — %s\n", p.Step, p.Why)
			}
			return nil
		},
		brainkit.WithCallTimeout(60*time.Second),
	)
	if err != nil {
		return fmt.Errorf("plan CallStream: %w", err)
	}
	fmt.Printf("\nfinal plan (%d steps):\n", len(result.Object))
	for i, s := range result.Object {
		fmt.Printf("  %d. %s\n", i+1, s.Step)
	}

	fmt.Println()
	fmt.Printf("gateway SSE endpoints live — hit them from another shell:\n")
	fmt.Printf("  curl -N 'http://%s/sse/haiku?prompt=spring+blossoms'\n", listenAddr)
	fmt.Printf("  curl -N 'http://%s/sse/plan?goal=launch+product'\n", listenAddr)
	fmt.Println("press Ctrl+C to stop.")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	return nil
}

const streamSource = `
const haikuAgent = new Agent({
    name: "haiku-streamer",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You are a haiku poet. Write one short haiku about the topic. Three lines, nothing else.",
});

const plannerAgent = new Agent({
    name: "planner",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You are a pragmatic planner. Given a goal, break it into 3-5 concrete steps with a one-sentence rationale for each.",
});

kit.register("agent", "haiku-streamer", haikuAgent);
kit.register("agent", "planner", plannerAgent);

// Plain text streaming: every delta becomes a chunk on the bus.
bus.on("haiku", async (msg) => {
    const prompt = (msg.payload && msg.payload.prompt) || (msg.payload && msg.payload.query) || "nature in autumn";
    const stream = await haikuAgent.stream(prompt);
    for await (const delta of stream.textStream) {
        msg.send({ delta });
    }
    msg.reply({ done: true });
});

// Structured-output streaming: each plan step emits as a chunk;
// the terminal reply carries the full parsed object.
const planSchema = z.array(z.object({
    step: z.string(),
    why: z.string(),
}));

bus.on("plan", async (msg) => {
    const goal = (msg.payload && msg.payload.goal) || "win the game";
    const stream = await plannerAgent.stream("Goal: " + goal, {
        structuredOutput: { schema: planSchema },
    });
    // Emit partial step objects as the model fills them in. Mastra
    // surfaces these via stream.fullStream with type "object-result".
    for await (const part of stream.fullStream) {
        if (part && part.type === "object-result" && part.object) {
            // Mastra emits snapshots of the full array; send only
            // the latest-filled-in step so the chunk stream stays
            // incremental.
            const arr = Array.isArray(part.object) ? part.object : [];
            const latest = arr[arr.length - 1];
            if (latest && latest.step) {
                msg.send({ step: latest.step, why: latest.why || "" });
            }
        }
    }
    const object = await stream.object;
    msg.reply({ object });
});
`
