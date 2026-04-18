# voice-agent

Full speak → listen → generate → speak round trip with
`OpenAIVoice` on a brainkit Agent. Exercises every leg of the
voice surface without needing a pre-recorded sample.

## Run

```sh
OPENAI_API_KEY=sk-... go run ./examples/voice-agent
```

Optional flags:

- `-out ./voice-agent-out` — directory for the two MP3 files.
  Persisted on disk so you can play them back; default sits next
  to your CWD.
- `-question "..."` — the question synthesized to audio.
  Defaults to a short factual prompt.

Expected tail:

```
[1/3] voice-agent deployed
[2/3] driving the round trip (speak → listen → generate → speak)
        question:   "What is the capital of France? One short sentence."
        transcript: "What is the capital of France? One short sentence."
        answer:     "The capital of France is Paris."
[3/3] audio files on disk (open these in any media player):
        ✓ ./voice-agent-out/question.mp3 (~60 KB)
        ✓ ./voice-agent-out/answer.mp3   (~38 KB)
```

## What it shows

| Surface | Used for |
|---|---|
| `OpenAIVoice` | TTS (`speak`) + STT (`listen`) on the same client |
| `agent.voice` | wiring a voice provider on an Agent |
| `agent.generate` | the model in the middle of the round trip |
| `fs.writeFile` / `fs.createReadStream` | persisting MP3s + replaying them as upload bodies |
| `Buffer.concat` | drain a Node Readable into a single buffer |

The MP3s round-trip cleanly because the jsbridge boundary is
binary-safe — fetch responses, FormData uploads, and
`createReadStream` chunks all preserve raw bytes (see
`internal/jsbridge/{fetch,fs,encoding}.go`).

## Wiring shape

```
              brainkit.Config{Providers: [OpenAI(key)]}
                              │
                              ▼
         kit.Deploy("voice-agent", voice.ts)
                              │
                              ▼
   bus.on("ask", async msg => {
     speak(question)  ──► question.mp3
     listen(stream)   ──► transcript
     generate(text)   ──► answer
     speak(answer)    ──► answer.mp3
   })
```

## See also

- `internal/engine/runtime/kit_runtime.js` — endows
  `OpenAIVoice` and `CompositeVoice` for `.ts` deployments.
- `internal/jsbridge/fetch.go` — multipart/form-data + binary
  body serialization used by `voice.listen`.
