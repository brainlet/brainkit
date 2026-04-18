// Command working-memory demonstrates a multi-turn agent that
// actually remembers what it was told in a previous call. Mastra
// Memory + the `memory: { thread, resource }` option on
// `agent.generate` make the agent see the conversation history
// on the next turn.
//
// The example runs three turns:
//
//  1. thread=t1, resource=user-alice — "Hi, my name is Alice."
//  2. thread=t1, resource=user-alice — "What's my name?"
//     (same thread → should recall Alice)
//  3. thread=t2, resource=user-alice — "What's my name?"
//     (different thread, same user → should NOT recall Alice)
//
// Turn 3 proves thread isolation: even though the resource id
// (user identity) is the same, a fresh thread starts with an
// empty conversation history.
//
// Storage is SQLite under the Kit's FSRoot so the DB persists
// between calls within the same process. Swap to Postgres /
// MongoDB / Upstash by changing one `brainkit.Config.Storages`
// entry — no .ts changes.
//
// Requires OPENAI_API_KEY.
//
// Run from the repo root:
//
//	OPENAI_API_KEY=sk-... go run ./examples/working-memory
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
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("working-memory: %v", err)
	}
}

func run() error {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	tmp, err := os.MkdirTemp("", "brainkit-working-memory-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "working-memory-demo",
		Transport: brainkit.Memory(),
		FSRoot:    tmp,
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
		Storages: map[string]brainkit.StorageConfig{
			"default": brainkit.SQLiteStorage(filepath.Join(tmp, "memory.db")),
		},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	if _, err := kit.Deploy(ctx, brainkit.PackageInline("working-memory", "memory.ts", memorySource)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	type reply struct {
		Text string `json:"text"`
	}
	ask := func(prompt, threadID, resourceID string) (string, error) {
		payload := json.RawMessage(fmt.Sprintf(`{"prompt":%q,"threadId":%q,"resourceId":%q}`, prompt, threadID, resourceID))
		r, err := brainkit.Call[sdk.CustomMsg, reply](kit, ctx, sdk.CustomMsg{
			Topic:   "ts.working-memory.ask",
			Payload: payload,
		}, brainkit.WithCallTimeout(45*time.Second))
		if err != nil {
			return "", err
		}
		return r.Text, nil
	}

	turns := []struct {
		label      string
		prompt     string
		threadID   string
		resourceID string
	}{
		{"[1/3] turn 1 — introduce the name",
			"Hi, my name is Alice. Please remember it.",
			"t1", "user-alice"},
		{"[2/3] turn 2 — same thread, ask for recall",
			"What did I tell you my name was? One word answer.",
			"t1", "user-alice"},
		{"[3/3] turn 3 — fresh thread, same resource — should NOT recall",
			"What did I tell you my name was? Answer in one short sentence.",
			"t2", "user-alice"},
	}

	for _, t := range turns {
		fmt.Printf("%s\n        thread=%s resource=%s\n", t.label, t.threadID, t.resourceID)
		fmt.Printf("        prompt: %q\n", t.prompt)
		reply, err := ask(t.prompt, t.threadID, t.resourceID)
		if err != nil {
			return fmt.Errorf("turn %s: %w", t.label, err)
		}
		fmt.Printf("        reply:  %s\n\n", reply)
	}

	fmt.Println("If the second turn's reply contains \"Alice\" and the third turn's doesn't, thread isolation works as designed.")
	return nil
}

const memorySource = `
const memory = new Memory({ storage: storage("default") });

const agent = new Agent({
    name: "memory-demo",
    model: model("openai", "gpt-4o-mini"),
    instructions:
        "You are a concise assistant. When the user introduces their name, remember it and use it in later turns. If asked for the name and you have no prior turn that set it, say you don't know.",
    memory,
});

kit.register("agent", "memory-demo", agent);

bus.on("ask", async (msg) => {
    const prompt = (msg.payload && msg.payload.prompt) || "";
    const threadId = (msg.payload && msg.payload.threadId) || "default-thread";
    const resourceId = (msg.payload && msg.payload.resourceId) || "default-user";
    const result = await agent.generate(prompt, {
        memory: { thread: { id: threadId }, resource: resourceId },
    });
    msg.reply({ text: result.text || "" });
});
`
