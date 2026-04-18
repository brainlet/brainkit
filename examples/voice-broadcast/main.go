// Command voice-broadcast shows the audio.Composite fan-out:
// one TTS clip flows into three sinks concurrently — desktop
// speakers (audio/local), a WAV file on disk (audio.Func),
// and a bus topic another subscriber listens on (audio.Func +
// sdk.Publish). The agent only calls `new Audio(stream).play()`
// once; the Sink primitive handles the fan-out.
//
// Demonstrates:
//   - audio.Composite(a, b, c) — run every sink concurrently,
//     join errors
//   - audio.Func(fn) — lift a plain Go callback into a Sink
//     without a dedicated package
//   - brainkit/audio/local — desktop speakers
//
// Run from the repo root:
//
//	OPENAI_API_KEY=sk-... go run ./examples/voice-broadcast
package main

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/audio"
	"github.com/brainlet/brainkit/audio/local"
	"github.com/brainlet/brainkit/sdk"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("voice-broadcast: %v", err)
	}
}

func run() error {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	outDir, err := filepath.Abs("./voice-broadcast-out")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	// Sink 1 — desktop speakers via audio/local (oto + go-mp3).
	speakers := local.New()

	// Sink 2 — write the decoded MP3 bytes to disk so the user
	// can replay the broadcast later. audio.Func lifts a plain
	// callback into a Sink; this one just swallows the bytes
	// into a file named after the mime type so multiple
	// broadcasts don't collide.
	var fileBytes atomic.Int64
	fileSink := audio.Func(func(_ context.Context, buf []byte, mime string) error {
		name := "broadcast." + mimeExt(mime)
		path := filepath.Join(outDir, name)
		if werr := os.WriteFile(path, buf, 0o644); werr != nil {
			return werr
		}
		fileBytes.Store(int64(len(buf)))
		return nil
	})

	// Sink 3 — publish to a bus topic. Any other kit / Go
	// consumer / .ts deployment can subscribe and do whatever
	// it wants (stream to a browser, persist to audit, feed
	// another model).
	//
	// Construction order: Config.Audio is captured by
	// brainkit.New, so the Sink has to exist before the kit.
	// PublishRaw needs the kit. Resolve via a late-bound
	// pointer — Composite(speakers, fileSink, busSink) wraps
	// a closure that reads kitRef after New() returns.
	var kitRef atomic.Pointer[brainkit.Kit]
	busSink := audio.Func(func(ctx context.Context, buf []byte, mime string) error {
		k := kitRef.Load()
		if k == nil {
			return nil
		}
		payload, _ := json.Marshal(map[string]any{
			"mime":      mime,
			"bytes_b64": base64.StdEncoding.EncodeToString(buf),
		})
		_, perr := k.PublishRaw(ctx, "audio.broadcast", payload)
		return perr
	})

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "voice-broadcast-demo",
		Transport: brainkit.Memory(),
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
		Audio:     audio.Composite(speakers, fileSink, busSink),
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()
	kitRef.Store(kit)

	// Wire a bus subscriber so the "network" sink has
	// somebody to talk to.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var busChunks atomic.Int64
	unsub, err := kit.SubscribeRaw(ctx, "audio.broadcast", func(msg sdk.Message) {
		var env struct {
			Mime     string `json:"mime"`
			BytesB64 string `json:"bytes_b64"`
		}
		if jerr := json.Unmarshal(msg.Payload, &env); jerr == nil {
			raw, _ := base64.StdEncoding.DecodeString(env.BytesB64)
			busChunks.Add(int64(len(raw)))
			fmt.Printf("        [bus]     audio.broadcast got %d bytes (mime=%s)\n", len(raw), env.Mime)
		}
	})
	if err != nil {
		return fmt.Errorf("subscribe: %w", err)
	}
	defer unsub()

	deployCtx, deployCancel := context.WithTimeout(ctx, 20*time.Second)
	defer deployCancel()
	if _, err := kit.Deploy(deployCtx, brainkit.PackageInline("voice-broadcast", "broadcast.ts", broadcastSource)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	fmt.Println("[voice-broadcast] synthesizing + broadcasting to speakers + disk + bus")
	payload, _ := json.Marshal(map[string]string{
		"text": "Brainkit audio sinks fan out through one primitive. Same bytes, different destinations.",
	})
	callCtx, callCancel := context.WithTimeout(ctx, 45*time.Second)
	defer callCancel()
	if _, err := brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, callCtx, sdk.CustomMsg{
		Topic:   "ts.voice-broadcast.broadcast",
		Payload: payload,
	}, brainkit.WithCallTimeout(45*time.Second)); err != nil {
		return fmt.Errorf("broadcast: %w", err)
	}

	// Give the bus subscriber a breath to drain any
	// in-flight delivery before we print totals.
	time.Sleep(100 * time.Millisecond)

	fmt.Println()
	fmt.Println("[voice-broadcast] fan-out summary:")
	fmt.Printf("        speakers: %d Hz × %d ch — played via audio/local\n", 24000, 2)
	fmt.Printf("        disk:     %d bytes → %s\n", fileBytes.Load(), filepath.Join(outDir, "broadcast.mp3"))
	fmt.Printf("        bus:      %d bytes observed by subscriber on audio.broadcast\n", busChunks.Load())
	return nil
}

func mimeExt(mime string) string {
	switch mime {
	case "audio/mpeg", "audio/mp3":
		return "mp3"
	case "audio/wav", "audio/x-wav":
		return "wav"
	case "audio/ogg":
		return "ogg"
	default:
		return "bin"
	}
}

// wavHeader isn't used yet; left here so someone adding a raw
// PCM sink can wrap bytes in a WAV container without pulling a
// dep. Remove once a proper sink uses it.
func wavHeader(rate, channels int, payload []byte) []byte {
	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], uint32(36+len(payload)))
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1)
	binary.LittleEndian.PutUint16(header[22:24], uint16(channels))
	binary.LittleEndian.PutUint32(header[24:28], uint32(rate))
	binary.LittleEndian.PutUint32(header[28:32], uint32(rate*channels*2))
	binary.LittleEndian.PutUint16(header[32:34], uint16(channels*2))
	binary.LittleEndian.PutUint16(header[34:36], 16)
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], uint32(len(payload)))
	return append(header, payload...)
}

var _ = wavHeader // keep the helper reachable for future sinks

const broadcastSource = `
const voice = new OpenAIVoice();
const agent = new Agent({
    name: "voice-broadcast-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Ignore — this agent only voices the user-supplied message.",
    voice,
});
kit.register("agent", "voice-broadcast-agent", agent);

bus.on("broadcast", async (msg) => {
    const text = (msg.payload && msg.payload.text) || "";
    if (!text) { msg.reply({ ok: false }); return; }
    const stream = await agent.voice.speak(text, { responseFormat: "mp3" });
    await new Audio(stream).play();
    msg.reply({ ok: true });
});
`
