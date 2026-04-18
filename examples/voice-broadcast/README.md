# voice-broadcast

One TTS clip, three sinks, one `audio.Composite` — the Sink
fan-out pattern. The `.ts` side calls
`new Audio(stream).play()` exactly once; the Go side decides
what happens to the bytes:

1. **Desktop speakers** — `brainkit/audio/local` via oto
2. **Disk** — an MP3 file on `./voice-broadcast-out/`
3. **Bus** — published on `audio.broadcast`; a Go subscriber
   in the same process prints the chunk size

Demonstrates that new transports never touch the polyfill or
the `.ts` code. Adding a sink is one `audio.Func(fn)` + one
`audio.Composite(...)` entry.

## Run

```sh
OPENAI_API_KEY=sk-... go run ./examples/voice-broadcast
```

Expected output:

```
[voice-broadcast] synthesizing + broadcasting to speakers + disk + bus
        [bus]     audio.broadcast got 105600 bytes (mime=audio/mpeg)

[voice-broadcast] fan-out summary:
        speakers: 24000 Hz × 2 ch — played via audio/local
        disk:     105600 bytes → ./voice-broadcast-out/broadcast.mp3
        bus:      105600 bytes observed by subscriber on audio.broadcast
```

Every sink saw the same byte count — the bytes were handed
off once each.

## What the code looks like

```go
speakers := local.New()

fileSink := audio.Func(func(_ context.Context, buf []byte, mime string) error {
    return os.WriteFile("broadcast."+mimeExt(mime), buf, 0644)
})

busSink := audio.Func(func(ctx context.Context, buf []byte, mime string) error {
    payload, _ := json.Marshal(map[string]any{
        "mime":      mime,
        "bytes_b64": base64.StdEncoding.EncodeToString(buf),
    })
    _, err := kit.PublishRaw(ctx, "audio.broadcast", payload)
    return err
})

kit, _ := brainkit.New(brainkit.Config{
    Audio: audio.Composite(speakers, fileSink, busSink),
    ...
})
```

The .ts side is trivially short:

```ts
const stream = await agent.voice.speak(text, { responseFormat: "mp3" });
await new Audio(stream).play();
```

Nothing knows or cares that there are three sinks.

## What it shows

| Primitive | Used for |
|---|---|
| `audio.Sink` | Single contract; every transport implements it |
| `audio.Func(fn)` | Shim a plain Go closure into a Sink — no new package needed for one-offs |
| `audio.Composite(a, b, c)` | Fan-out with concurrent `Play` calls + joined errors |
| `brainkit/audio/local` | Desktop speakers; same Sink interface as `audio.Func` |
| `kit.PublishRaw` + `kit.SubscribeRaw` | Audio as a normal bus topic — any deployment or Go peer can consume it |

## When each sink makes sense

| Sink | When |
|---|---|
| `audio/local.New()` | Desktop app that should play locally |
| `audio.Func` writing to disk | Debug capture, audit persistence, replay |
| `audio.Func` publishing to a bus topic | Multi-consumer fan-out — browsers via gateway, other agents, plugins, CI runners |
| Future `audio/gateway.NewSink(topic)` | Shorthand for the bus-then-HTTP path; shipping in the polish pass (see [designs/13](../../internal/docs/superpowers/../../brainkit-maps/brainkit/designs/13-audio-deep-dive.md) if you have the map repo) |

## Compared to the other voice examples

| Example | Primary demo | Audio destination |
|---|---|---|
| [voice-chat](../voice-chat/) | Simplest "agent speaks answers" | Speakers only |
| [voice-agent](../voice-agent/) | Full TTS + STT + generate round trip | Speakers + MP3 files |
| [voice-broadcast](.) | Sink fan-out via `audio.Composite` | Speakers + file + bus |
| [voice-realtime](../voice-realtime/) | Browser mic + OpenAI Realtime | Browser speakers via WebSocket |

## See also

- `brainkit/audio/audio.go` — `Sink` + `Null` + `Func` + `Composite`
  implementations; all under 150 lines of Go.
- `brainkit/audio/local/` — the desktop implementation wrapped
  by speakers in this example.
