# Voice and audio

brainkit ships a voice + audio surface that stays web-standard
on the `.ts` side and composable on the Go side:

- **Speech synthesis / transcription** via Mastra
  (`OpenAIVoice`, `CompositeVoice`, `OpenAIRealtimeVoice`).
- **Audio playback** via the web-standard `new Audio(src).play()`
  polyfill + a pluggable `audio.Sink` on the Go side.
- **Live bidirectional voice** via OpenAI's realtime WebSocket
  API, backed by a client `WebSocket` polyfill that covers both
  the WHATWG surface and Node `ws` extensions (custom headers,
  `ws.on("message", …)`).

The `.ts` side never imports a brainkit audio type. Every
capability routes through a web spec so the same deployment
runs in a browser console and in a brainkit kit.

## Voice providers inside a deployment

```ts
import { Agent, model, OpenAIVoice, CompositeVoice } from "agent";

const voice = new OpenAIVoice();                    // uses OPENAI_API_KEY

const agent = new Agent({
    name: "voice-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Keep replies to one or two short sentences.",
    voice,
});
kit.register("agent", "voice-agent", agent);
```

Voice APIs on the Agent:

```ts
const stream = await agent.voice.speak("hello", { responseFormat: "mp3" });
const text   = await agent.voice.listen(stream, { filetype: "mp3" });
```

`speak()` returns a Node Readable of audio bytes; `listen()`
takes a Node Readable of audio bytes and returns the transcript.

Mix-and-match providers (different for TTS vs STT) with
`CompositeVoice({ speakProvider, listenProvider })`.

## Playing audio on the desktop

Wire a sink on the kit:

```go
import (
    "github.com/brainlet/brainkit"
    "github.com/brainlet/brainkit/audio/local"
)

kit, _ := brainkit.New(brainkit.Config{
    Namespace: "myapp",
    Transport: brainkit.Memory(),
    Providers: []brainkit.ProviderConfig{brainkit.OpenAI(key)},
    Audio:     local.New(),
})
```

`audio/local` wraps `oto` + `go-mp3` + a minimal WAV parser.
Opens the default output device on first `Play`. Without a
sink, `Config.Audio` is nil and `new Audio(…).play()` resolves
silently so portable agent code runs on headless kits too.

The `.ts` side is web-standard:

```ts
const stream = await agent.voice.speak("hello");
await new Audio(stream).play();
```

`new Audio(src)` accepts URLs, file paths, `Buffer`,
`Uint8Array`, `Blob`, Node Readable, or Web ReadableStream.
MIME type is sniffed from container magic (MP3 / WAV / OGG /
FLAC) with an extension fallback.

## Fan-out to multiple sinks

`audio.Composite` runs every wrapped sink concurrently; the
polyfill still calls `play()` once.

```go
import "github.com/brainlet/brainkit/audio"

kit, _ := brainkit.New(brainkit.Config{
    Audio: audio.Composite(
        local.New(),                                   // desktop speakers
        audio.Func(func(_ context.Context, buf []byte, mime string) error {
            return os.WriteFile("capture."+ext(mime), buf, 0644)
        }),
        audio.Func(func(ctx context.Context, buf []byte, mime string) error {
            payload, _ := json.Marshal(map[string]any{
                "mime":      mime,
                "bytes_b64": base64.StdEncoding.EncodeToString(buf),
            })
            _, err := kit.PublishRaw(ctx, "audio.broadcast", payload)
            return err
        }),
    ),
})
```

Common shapes:

- `local.New()` — desktop speakers.
- `audio.Func(fn)` — lift a Go closure into a `Sink`; use for
  one-off adapters (disk writer, bus publisher, HTTP
  responder).
- `audio.Null()` — silent; the default when `Config.Audio` is
  nil.
- `audio.Composite(a, b, c)` — fan out to any number of sinks
  concurrently with joined errors.

## Desktop audio self-test

`local.Sink` exposes a `Check(ctx)` that plays a known-good
1 second tone through the sink + probes system volume / mute /
output device. Useful in CI or remote sessions where no
human can confirm audibility:

```go
sink := local.New()
res := sink.Check(context.Background())
if res.PeakSample < 1000 { /* sink isn't seeing real audio */ }
fmt.Print(res)   // prints a CI-friendly summary block
```

Example output:

```
platform:       darwin
output device:  MacBook Pro Speakers
system volume:  100% (muted=false)
oto context:    24000 Hz × 2 ch
tone duration:  1.16s
bytes written:  96044
peak sample:    9830 / 32767 (30.0%)
```

Non-Darwin platforms return `-1` volume + a warning; the tone
still plays so the rest of the pipeline is still covered.

## Live voice in a browser

The runtime ships a client WebSocket polyfill so
`@mastra/voice-openai-realtime` works inside the SES
compartment. The package imports `WebSocket` from Node's `ws`;
brainkit's bundle aliases that to `globalThis.WebSocket` →
`internal/jsbridge/websocket.go` (backed by
`github.com/coder/websocket`). Both WHATWG and Node shapes are
supported on the same object — `new WebSocket(url, protocols, {headers})`
for custom auth, `ws.on("message", fn)` for binary frames.

Gateway module also exposes:

- `HandleStatic(prefix, fs.FS)` — serve a mic UI straight out
  of the Go binary via `go:embed`.
- `HandleWebSocketAudio(path, inTopic, outTopic)` — duplex
  binary WebSocket that publishes every inbound frame to
  `inTopic` (with a session id) and writes every message
  received on `<outTopic>.<sessionId>` back as a binary
  frame.

See [`examples/voice-realtime`](../../examples/voice-realtime/)
for the full mic → OpenAI Realtime → browser playback flow.

## Example map

| Example | Demonstrates |
|---|---|
| [voice-chat](../../examples/voice-chat/) | Simplest "agent speaks answers". stdin → generate → speak → desktop speakers via `new Audio(stream).play()` |
| [voice-agent](../../examples/voice-agent/) | Full speak → listen → generate → speak file round trip. `-check-audio` runs the headless self-test |
| [voice-broadcast](../../examples/voice-broadcast/) | `audio.Composite` fan-out — one TTS clip to speakers + disk + bus topic in a single call |
| [voice-realtime](../../examples/voice-realtime/) | Live browser mic + OpenAI Realtime with PCM16 AudioWorklet and gapless playback |

## Architecture

```
.ts deployment
└─ new Audio(src).play()
      │ internal/jsbridge/audio.go
      ▼
   AudioSink (one method: Play(ctx, bytes, mime))
      │
      ▼
   audio.Sink impls (Go side)
      ├─ audio/local.Sink       → oto + go-mp3 + WAV → speakers
      ├─ audio.Func(fn)          → any closure (file, bus, HTTP)
      └─ audio.Composite(...)    → fan out to N sinks concurrently
```

The `Sink` interface is the single extension point. Any new
transport (WebRTC, ICE-server-backed, embedded speaker hardware)
lands as a `Sink` implementation; no polyfill or `.ts` change.

## Design notes + polish roadmap

The sink interface supports buffered `Play(bytes)` today. A
future `StreamingSink` interface will let sinks accept chunked
`AudioStream.Write(chunk)` for sub-buffer-duration latency —
relevant for realtime TTS. The roadmap + library choices live
in [`brainkit-maps/brainkit/designs/13-audio-deep-dive.md`](../../../brainkit-maps/brainkit/designs/13-audio-deep-dive.md)
(if you have the map repo checked out). The short version:

- Streaming `Audio` polyfill + `StreamingSink` — pending
- `audio/gateway` opt-in package (bus + chunked HTTP) — pending
- `audio/mic` (malgo-backed desktop capture) — pending
- Browser mic + realtime voice — shipped

Everything new lands as an opt-in subpackage under `audio/`.
Core stays cgo-free and pays zero cost on kits that don't use
audio.

## See also

- [ai-and-agents.md](ai-and-agents.md) — Agents, memory, tools.
- [ts-services.md](ts-services.md) — Deploying `.ts` packages.
- [`concepts/jsbridge-polyfills.md`](../concepts/jsbridge-polyfills.md)
  — the full polyfill catalog including Audio + WebSocket.
