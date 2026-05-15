# voice/ Fixtures

Tests Mastra voice provider availability through the `agent` bundle. Most
fixtures are constructor/API-shape checks; OpenAI speak/listen fixtures perform
live provider calls when `OPENAI_API_KEY` and live AI testing are enabled.

## Fixtures

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| agent-integration | yes | none | `Agent` accepts a voice provider and exposes `agent.voice` |
| basic | yes | none | `OpenAIVoice` construction with `speak` and `listen` methods |
| composite/basic | yes | none | `CompositeVoice` routes separate input/output voice providers |
| azure/construct | yes | none | `AzureVoice` constructor shape with dummy credentials |
| cloudflare/construct | yes | none | `CloudflareVoice` constructor shape with dummy credentials |
| deepgram/construct | yes | none | `DeepgramVoice` constructor shape with dummy credentials |
| elevenlabs/construct | yes | none | `ElevenLabsVoice` constructor shape with dummy credentials |
| murf/construct | yes | none | `MurfVoice` constructor shape with dummy credentials |
| playai/construct | yes | none | `PlayAIVoice` constructor shape and preset voice list |
| sarvam/construct | yes | none | `SarvamVoice` constructor shape with dummy credentials |
| speechify/construct | yes | none | `SpeechifyVoice` constructor shape with dummy credentials |
| openai/speak | yes | none | Live `OpenAIVoice.speak()` returns non-empty audio stream chunks |
| openai/listen | yes | none | Live `OpenAIVoice.speak()` output can be transcribed by `listen()` |
| openai-realtime/connect | yes | none | `OpenAIRealtimeVoice` connect/disconnect/send/on surface exists |
| openai-realtime/speaker-event | yes | none | Realtime voice event registration/removal shape |

## Notes

- `@mastra/node-audio` microphone/playback helpers are not part of Brainkit's
  QuickJS runtime promise.
- Google Gemini Live is intentionally not asserted as working here; it belongs
  behind an explicit runtime support decision because it depends on live
  realtime voice transport behavior.
- Provider-specific live calls should stay opt-in and credential-gated.
