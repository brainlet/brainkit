// Command voice-agent demonstrates Mastra's OpenAIVoice on a
// brainkit Agent. The example does the full round trip without
// needing a pre-committed sample audio file:
//
//  1. speak: synthesize an audio stream from a text question.
//  2. listen: transcribe the audio back to text.
//  3. generate: ask the agent for an answer.
//  4. speak: synthesize the answer to a second audio file.
//
// After the run, inspect the two files to hear the round trip.
// (They land in ./examples/voice-agent/out/ by default.)
//
// Requires OPENAI_API_KEY. brainkit ships `OpenAIVoice` and
// `CompositeVoice` as Compartment endowments — wired up in the
// same PR as this example.
//
// Run from the repo root:
//
//	OPENAI_API_KEY=sk-... go run ./examples/voice-agent
package main

import (
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

type reply struct {
	Question    string `json:"question"`
	Transcript  string `json:"transcript"`
	AnswerText  string `json:"answerText"`
	QuestionMP3 string `json:"questionMp3"`
	AnswerMP3   string `json:"answerMp3"`
}

func main() {
	outDir := flag.String("out", "./voice-agent-out", "directory for generated audio files (survives the run so you can play back)")
	question := flag.String("question", "What is the capital of France? One short sentence.", "the question synthesized to audio then transcribed + answered")
	flag.Parse()

	if err := run(*outDir, *question); err != nil {
		log.Fatalf("voice-agent: %v", err)
	}
}

func run(outRaw, question string) error {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	outDir, err := filepath.Abs(outRaw)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	// Audio files land under FSRoot/out/ on the brainkit side —
	// match the Go path so both ends resolve to the same files.
	wsRoot := filepath.Dir(outDir)
	outSubdir := filepath.Base(outDir)

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "voice-agent-demo",
		Transport: brainkit.Memory(),
		FSRoot:    wsRoot,
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	source := fmt.Sprintf(voiceSource, outSubdir)
	if _, err := kit.Deploy(ctx, brainkit.PackageInline("voice-agent", "voice.ts", source)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}
	fmt.Println("[1/3] voice-agent deployed")

	fmt.Printf("[2/3] driving the round trip (speak → listen → generate → speak)\n")
	fmt.Printf("        question: %q\n", question)
	payload, _ := json.Marshal(map[string]string{"question": question})
	r, err := brainkit.Call[sdk.CustomMsg, reply](kit, ctx, sdk.CustomMsg{
		Topic:   "ts.voice-agent.ask",
		Payload: payload,
	}, brainkit.WithCallTimeout(120*time.Second))
	if err != nil {
		return fmt.Errorf("ask: %w", err)
	}

	fmt.Printf("        transcript: %q\n", r.Transcript)
	fmt.Printf("        answer:     %q\n", r.AnswerText)
	fmt.Println()
	fmt.Printf("[3/3] audio files on disk (open these in any media player):\n")
	for _, name := range []string{"question.mp3", "answer.mp3"} {
		p := filepath.Join(outDir, name)
		if info, err := os.Stat(p); err == nil && info.Size() > 0 {
			fmt.Printf("        ✓ %s (%d bytes)\n", p, info.Size())
		} else {
			fmt.Printf("        ✗ %s (missing or empty)\n", p)
		}
	}
	return nil
}

const voiceSource = `
const voice = new OpenAIVoice();

const agent = new Agent({
    name: "voice-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You answer concisely. One or two sentences maximum.",
    voice,
});
kit.register("agent", "voice-agent", agent);

async function streamToBuffer(stream) {
    const chunks = [];
    for await (const chunk of stream) {
        chunks.push(chunk);
    }
    // Node streams emit Buffer chunks; concatenate into one.
    return Buffer.concat(chunks.map(c => Buffer.isBuffer(c) ? c : Buffer.from(c)));
}

bus.on("ask", async (msg) => {
    const question = (msg.payload && msg.payload.question) || "";
    const outDir = %q;

    // 1. speak(question) → MP3 stream → disk.
    const qStream = await agent.voice.speak(question, { responseFormat: "mp3" });
    const qBuf = await streamToBuffer(qStream);
    const questionMp3 = outDir + "/question.mp3";
    await fs.writeFile(questionMp3, qBuf);

    // 2. listen(audio) → transcript.
    const transcript = await agent.voice.listen(fs.createReadStream(questionMp3), {
        filetype: "mp3",
    });

    // 3. generate(transcript).
    const gen = await agent.generate(String(transcript));

    // 4. speak(answer) → MP3 stream → disk.
    const aStream = await agent.voice.speak(gen.text || "", { responseFormat: "mp3" });
    const aBuf = await streamToBuffer(aStream);
    const answerMp3 = outDir + "/answer.mp3";
    await fs.writeFile(answerMp3, aBuf);

    msg.reply({
        question,
        transcript: String(transcript),
        answerText: gen.text || "",
        questionMp3,
        answerMp3,
    });
});
`
