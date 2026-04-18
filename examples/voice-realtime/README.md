# voice-realtime

Live bidirectional voice conversation between a browser and a
brainkit Agent. The mic stream goes up to `OpenAIRealtimeVoice`
over a WebSocket, the model replies as it's producing audio,
and the reply plays back in the page as it arrives.

Complements `examples/voice-agent` (file-based round trip —
synthesize, transcribe, generate, synthesize) with the live
version: sub-second turn-taking, the shape of a real voice
assistant.

## Run

```sh
OPENAI_API_KEY=sk-... go run ./examples/voice-realtime
# open http://127.0.0.1:8787 in a browser, click the mic button
```

Change the listen address with `-addr`. Ctrl-C stops the
server.

The **browser** gets the full reply audio streamed back through
the WebSocket; the page's Web Audio code queues each PCM16
chunk on an `AudioBufferSourceNode` for gapless playback.

`Config.Audio = local.New()` is wired on the Go side so any
`.ts` that calls `new Audio(...).play()` routes through the
desktop sink — ready for future additions like status pings.
Streaming the realtime reply through the desktop sink (not just
the browser) is on the roadmap in
`brainkit-maps/brainkit/designs/12-audio-and-microphone.md`;
today the Audio polyfill pre-drains each source, so rapid
small-chunk playback needs the streaming sink work tracked
there.

## What it shows

| Surface | Role |
|---|---|
| `OpenAIRealtimeVoice` | bidirectional WebSocket session with OpenAI Realtime |
| `gateway.HandleStatic("/", embed.FS)` | serves the `web/` page straight out of the Go binary |
| `gateway.HandleWebSocketAudio(path, inTopic, outTopic)` | duplex binary WS; every frame becomes a bus message and vice-versa |
| `jsbridge.WebSocket` polyfill | lets the SES compartment open a TLS WebSocket to OpenAI with the `Authorization` header |
| Browser `AudioWorklet` | downsamples mic to **PCM16 mono 24 kHz** (the realtime API's expected format) |
| `Config.Audio = local.New()` | desktop also hears the reply |

Shape of the pipeline:

```
browser mic ─► AudioWorklet (24 kHz PCM16) ─► WS binary frame ─┐
                                                               ▼
  ┌──────────────────────────── gateway /ws/voice ────────────────┐
  │  ts.voice-realtime.audio-in ──► .ts handler ──► voice.send()  │
  │                                                               │
  │  voice.on("speaker") ──► bus.publish(audio-out.<sessionId>)   │
  │  voice.on("writing") ──► bus.publish(audio-out.<sessionId>)   │
  └────────────────┬──────────────────────────────────┬───────────┘
                   ▼                                  ▼
      WS binary frame → browser            desktop Audio polyfill
      → AudioBufferSourceNode              → audio/local.Sink
      → speakers                           → speakers
```

## Format notes

- **Up** — mic PCM16 little-endian mono at 24 kHz. The
  `AudioWorklet` in `web/app.js` handles the downsample from the
  browser's native rate (usually 48 kHz).
- **Down** — the realtime API returns PCM16 24 kHz mono; the
  browser queues each chunk on a fresh `AudioBufferSourceNode`
  for gapless playback.

## Security / production

- **localhost is fine for dev** — browsers allow `getUserMedia`
  on `http://localhost`. Any other host needs HTTPS.
- **Origin restriction** — the gateway currently accepts
  `OriginPatterns: ["*"]`. Put it behind a reverse proxy or
  subclass the accept logic for production.
- **OpenAI key isolation** — the key never leaves the Go process;
  the browser talks only to your gateway.
- **Abuse** — rate-limit or token-gate `/ws/voice` behind a
  login; the realtime API is expensive and a single open
  session can burn minutes of model time fast.

## Extension ideas

- Auto-end-of-turn via energy-based VAD on the client.
- Persist the turn transcripts to `modules/audit`.
- Add tool-call support: the realtime API can issue function
  calls; surface them as brainkit `createTool`s.
- Swap `OpenAIRealtimeVoice` for another provider's realtime
  class (Mastra's interface is pluggable) once they're endowed.

## See also

- `examples/voice-agent/` — file-based voice round trip.
- `internal/jsbridge/websocket.go` — the client WebSocket
  polyfill. Covers both WHATWG and Node `ws` surfaces so any
  realtime client lib resolves.
- `brainkit-maps/brainkit/designs/12-audio-and-microphone.md`
  — the audio + mic roadmap; this example is the first slice
  of the "mic in, streaming audio out" matrix.
