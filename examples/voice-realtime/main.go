// Command voice-realtime demonstrates a live bidirectional
// voice session between a browser and a brainkit Agent. Open
// the printed URL, click the mic button, and speak — the
// OpenAI Realtime API transcribes, the agent answers, and the
// reply streams back to the page as audio.
//
// Ships everything needed to run locally:
//
//   - Go process hosts a brainkit Kit + tiny HTTP gateway
//   - Gateway serves the web/ directory + a duplex audio WS
//   - Deployed .ts owns one `OpenAIRealtimeVoice` session per
//     browser connection
//
// Requires OPENAI_API_KEY plus access to the OpenAI Realtime
// API (gpt-4o-realtime-preview).
//
// Run from the repo root:
//
//	OPENAI_API_KEY=sk-... go run ./examples/voice-realtime
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/brainlet/brainkit"
	"github.com/brainlet/brainkit/audio/local"
	"github.com/brainlet/brainkit/modules/gateway"
)

//go:embed web
var webFS embed.FS

func main() {
	addr := flag.String("addr", "127.0.0.1:8787", "listen address for the browser page + /ws/voice endpoint")
	flag.Parse()
	if err := run(*addr); err != nil {
		log.Fatalf("voice-realtime: %v", err)
	}
}

func run(addr string) error {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		return fmt.Errorf("OPENAI_API_KEY is required")
	}

	// Strip the go:embed wrapper so the static route sees
	// `index.html` directly, not `web/index.html`.
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		return fmt.Errorf("embed: %w", err)
	}

	gw := gateway.New(gateway.Config{Listen: addr})
	// Serve the mic UI at `/`.
	gw.HandleStatic("/", sub)
	// Duplex audio WebSocket — browser frames land on
	// ts.voice-realtime.audio-in, replies published on
	// ts.voice-realtime.audio-out.<sessionId> flow back.
	gw.HandleWebSocketAudio("/ws/voice",
		"ts.voice-realtime.audio-in",
		"ts.voice-realtime.audio-out",
	)

	kit, err := brainkit.New(brainkit.Config{
		Namespace: "voice-realtime",
		Transport: brainkit.Memory(),
		Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
		// Desktop playback is wired so `.ts` calls to
		// `new Audio(buf).play()` route to the system device.
		// The realtime reply itself plays in the browser — the
		// local sink is here so future .ts additions (status
		// pings, alerts) have a zero-config path.
		Audio:   local.New(),
		Modules: []brainkit.Module{gw},
	})
	if err != nil {
		return fmt.Errorf("new kit: %w", err)
	}
	defer kit.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if _, err := kit.Deploy(ctx, brainkit.PackageInline("voice-realtime", "voice.ts", voiceSource)); err != nil {
		return fmt.Errorf("deploy: %w", err)
	}

	fmt.Printf("[voice-realtime] open http://%s in your browser\n", addr)
	fmt.Println("[voice-realtime] click the mic to start; Ctrl-C here to stop the server.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	fmt.Println("\n[voice-realtime] shutting down")
	return nil
}

// voiceSource bridges the browser WS with OpenAI Realtime.
// One OpenAIRealtimeVoice session per browser connection,
// keyed by sessionId the gateway stamps on every payload.
const voiceSource = `
import { Agent, model, OpenAIRealtimeVoice } from "agent";

const sessions = new Map();

async function openSession(sessionId) {
    const voice = new OpenAIRealtimeVoice({
        realtimeConfig: {
            model: "gpt-4o-realtime-preview",
            apiKey: process.env.OPENAI_API_KEY,
            options: {
                sessionConfig: {
                    turn_detection: {
                        type: "server_vad",
                        threshold: 0.5,
                        silence_duration_ms: 600,
                    },
                },
            },
        },
        speaker: "alloy",
    });

    const agent = new Agent({
        name: "voice-realtime-agent",
        model: model("openai", "gpt-4o-mini"),
        instructions: "You are a brief, warm voice assistant. Keep replies to one or two short sentences.",
        voice,
    });

    await voice.connect();

    const outTopic = "ts.voice-realtime.audio-out." + sessionId;

    // OpenAI's realtime speaker stream → browser as binary frames.
    voice.on("speaker", async (stream) => {
        try {
            for await (const chunk of stream) {
                const u8 = chunk instanceof Uint8Array ? chunk :
                           (typeof chunk === "string" ? new TextEncoder().encode(chunk) : new Uint8Array(chunk));
                // Base64 the PCM16 bytes so they survive the
                // JSON payload hop to the Go side.
                let bin = "";
                for (let i = 0; i < u8.length; i++) bin += String.fromCharCode(u8[i] & 0xFF);
                await bus.publish(outTopic, { binary: true, bytes_b64: btoa(bin) });
            }
        } catch (e) {
            console.error("speaker stream error:", e && e.message || e);
        }
    });

    // Transcripts + status → browser as text frames (JSON).
    voice.on("writing", async (ev) => {
        await bus.publish(outTopic, {
            binary: false,
            text: JSON.stringify({ type: "transcript", role: ev.role, text: ev.text }),
        });
    });
    voice.on("error", async (err) => {
        console.error("realtime voice error:", err && err.message || err);
    });

    sessions.set(sessionId, { voice, agent });
    console.log("voice-realtime: opened session", sessionId);
    return voice;
}

function closeSession(sessionId) {
    const s = sessions.get(sessionId);
    if (!s) return;
    try { s.voice.close(); } catch (_) {}
    sessions.delete(sessionId);
    console.log("voice-realtime: closed session", sessionId);
}

bus.on("audio-in", async (msg) => {
    const payload = msg.payload || {};
    const sessionId = payload.sessionId;
    if (!sessionId) return;

    try {
        if (payload.type === "connect") {
            await openSession(sessionId);
            return;
        }
        if (payload.type === "disconnect") {
            closeSession(sessionId);
            return;
        }
        if (payload.type !== "audio" || !payload.binary || !payload.bytes_b64) return;

        const session = sessions.get(sessionId);
        if (!session) return;

        // Browser sends PCM16 LE 24 kHz mono. Decode base64 →
        // Int16Array → voice.send (the realtime lib accepts
        // Int16Array directly).
        const bin = atob(payload.bytes_b64);
        const u8 = new Uint8Array(bin.length);
        for (let i = 0; i < bin.length; i++) u8[i] = bin.charCodeAt(i) & 0xFF;
        // Int16Array view needs even byte length.
        if ((u8.byteLength & 1) !== 0) return;
        const int16 = new Int16Array(u8.buffer, u8.byteOffset, u8.byteLength / 2);
        await session.voice.send(int16);
    } catch (e) {
        console.error("voice-realtime audio-in error:", e && e.stack || e);
    }
});
`
