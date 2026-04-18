// Command voice-chat is the canonical "my agent talks back"
// pattern: you type a question at the terminal, the agent
// generates a text answer, the answer is synthesized with
// OpenAIVoice, and `new Audio(stream).play()` routes the
// bytes through brainkit/audio/local to the desktop speakers.
//
// No file I/O, no web page, no realtime WebSocket — just
// agent.generate → voice.speak → Audio.play. Stripped-down
// baseline you add voice to an existing agent with.
//
// Run from the repo root:
//
//	OPENAI_API_KEY=sk-... go run ./examples/voice-chat
//
// Type a question; hit Enter; the agent answers out loud.
// Type "exit" or Ctrl-D to quit.
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/audio/local"
	"github.com/brainlet/brainkit/sdk"
)

type ask struct {
	Text string `json:"text"`
}

type reply struct {
	Answer string `json:"answer"`
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("voice-chat: %v", err)
	}
}

func run() error {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "voice-chat-demo",
		Transport: brainkit.Memory(),
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
		Audio:     local.New(),
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := kit.Deploy(ctx, brainkit.PackageInline("voice-chat", "voice.ts", chatSource)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	fmt.Println("voice-chat ready — type a question and press enter (exit to quit).")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		if text == "exit" || text == "quit" {
			break
		}

		payload, _ := json.Marshal(ask{Text: text})
		turnCtx, turnCancel := context.WithTimeout(ctx, 60*time.Second)
		r, err := brainkit.Call[sdk.CustomMsg, reply](kit, turnCtx, sdk.CustomMsg{
			Topic:   "ts.voice-chat.ask",
			Payload: payload,
		}, brainkit.WithCallTimeout(60*time.Second))
		turnCancel()
		if err != nil {
			fmt.Printf("  error: %v\n\n", err)
			continue
		}
		// Agent reply prints alongside the audio playing out
		// the speakers — the desktop sink finishes before reply
		// returns because the .ts awaits Audio.play().
		fmt.Printf("  %s\n\n", r.Answer)
	}

	fmt.Println("voice-chat: bye")
	return nil
}

// chatSource is one Agent + one topic. Every turn:
//   1. agent.generate(question) → text answer
//   2. voice.speak(answer) → Node Readable of MP3 bytes
//   3. new Audio(stream).play() → bytes routed to audio/local
//      through Config.Audio; play() only resolves when the
//      desktop sink has drained so the next prompt doesn't
//      step on the current reply.
const chatSource = `
const voice = new OpenAIVoice();
const agent = new Agent({
    name: "voice-chat-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Answer in one or two short, conversational sentences. You are being spoken out loud, so avoid bullet lists and code blocks.",
    voice,
});
kit.register("agent", "voice-chat-agent", agent);

bus.on("ask", async (msg) => {
    const question = (msg.payload && msg.payload.text) || "";
    const gen = await agent.generate(question);
    const answer = gen.text || "";

    if (answer) {
        const stream = await agent.voice.speak(answer, { responseFormat: "mp3" });
        await new Audio(stream).play();
    }
    msg.reply({ answer });
});
`
