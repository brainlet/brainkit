# voice-chat

Minimum canonical "agent talks back" pattern. Type a question
at the terminal; the agent generates a text answer; `OpenAIVoice`
synthesizes it; `new Audio(stream).play()` routes the bytes
through `brainkit/audio/local` to the desktop speakers.

No file I/O, no web page, no realtime WebSocket. This is the
baseline for voice-enabled agents — strip `examples/voice-agent`
down to just the "speak the answer" leg.

## Run

```sh
OPENAI_API_KEY=sk-... go run ./examples/voice-chat
```

Type a question, hit Enter. The agent speaks the reply out
loud + prints the text beside it. `exit` or Ctrl-D to quit.

```
voice-chat ready — type a question and press enter (exit to quit).

> What is the fastest land animal? One sentence.
  The fastest land animal is the cheetah, capable of reaching speeds up to 60 miles per hour.

> exit
voice-chat: bye
```

## What it shows

| Surface | Role |
|---|---|
| `Config.Audio = local.New()` | Wire desktop playback on the kit |
| `agent.generate(text)` | Model text answer |
| `agent.voice.speak(text, {responseFormat:"mp3"})` | TTS stream |
| `new Audio(stream).play()` | Web-standard playback — polyfill drains the stream through the wired sink |

Flow:

```
stdin ─► bus.ts.voice-chat.ask ─► agent.generate ─► agent.voice.speak ─► Audio.play
                                                                            │
                                                                            ▼
                                                                    audio/local
                                                                            │
                                                                            ▼
                                                                       speakers
```

## Compared to the other voice examples

| Example | Input | Output | Transport |
|---|---|---|---|
| [voice-chat](.) | Go stdin | Desktop speakers | In-process |
| [voice-agent](../voice-agent/) | Go var → TTS → MP3 file | MP3 file + speakers | In-process |
| [voice-broadcast](../voice-broadcast/) | Go var → TTS | Speakers + MP3 file + bus topic | In-process + bus fan-out |
| [voice-realtime](../voice-realtime/) | Browser mic (WebSocket) | Browser speakers | HTTP + WS gateway |

Start with this one when you're adding voice output to an
existing agent. Move to `voice-agent` when you also need to
transcribe user audio, `voice-broadcast` when multiple
consumers want the same stream, `voice-realtime` when you
want sub-second turn taking in a browser.

## See also

- `brainkit/audio` — `Sink`, `Null`, `Func`, `Composite`
- `brainkit/audio/local` — desktop sink (oto + go-mp3)
- `internal/jsbridge/audio.go` — the web-standard `Audio`
  polyfill that lets `new Audio(stream).play()` work identically
  in a browser console and in a brainkit deployment.
