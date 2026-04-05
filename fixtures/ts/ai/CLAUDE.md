# ai/ Fixtures

Tests the Vercel AI SDK surface directly (no Mastra Agent wrapper): `generateText`, `streamText`, `generateObject`, `streamObject`, `embed`, `embedMany`, tool suspend/resume, and middleware.

All ai fixtures require AI (OPENAI_API_KEY) since the `ai` category is in `aiCategories`.

## Fixtures

### generate-text/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| basic | yes | no | Basic `generateText` call; asserts usage metadata present and finishReason="stop" |
| conversation | yes | no | Multi-turn `messages` array with prior context; model remembers name and city from earlier turns |
| max-tokens | yes | no | `maxTokens: 10` truncates output; asserts short response and text present |
| multi-step | yes | no | `generateText` with a weather tool and `maxSteps: 5`; asserts tool was called once and multi-step execution |
| stop-sequences | yes | no | `stopSequences: ["5"]` halts counting before reaching 6; asserts text does not contain "6" |
| temperature | yes | no | `temperature: 0` produces deterministic output; two identical calls return the same "4" |
| with-system | yes | no | `system` prompt sets persona (pirate); asserts text present, finishReason="stop", usage tokens > 0 |
| with-tools | yes | no | `generateText` with an `add` tool and `maxSteps: 3`; asserts tool calls appear in steps |

### stream-text/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| basic | yes | no | Basic `streamText` with `textStream` iteration; asserts real-time chunks received |
| full-stream | yes | no | `fullStream` iteration collecting part types; asserts "text-delta" type appears in the stream |
| on-chunk | yes | no | `experimental_onChunk` callback fires during streaming; asserts chunk events collected |
| on-finish | yes | no | `onFinish` callback fires after stream completes; asserts callback received text and usage data |
| with-tools | yes | no | `streamText` with a `multiply` tool and `maxSteps: 3`; asserts chunks, text, and usage present |

### generate-object/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| basic | yes | no | `generateObject` with Zod schema (name, age, hobbies); asserts all fields typed correctly, finishReason="stop" |
| array | yes | no | `generateObject` with `output: "array"`; asserts result is an array of objects, finishReason="stop" |
| enum | yes | no | `generateObject` with `output: "enum"` for sentiment classification; asserts valid enum value returned |

### stream-object/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| basic | yes | no | `streamObject` with Zod schema; iterates `partialObjectStream` and asserts final object has name, age, hobbies |

### embed/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| single | yes | no | `embed` + `embedMany` with text-embedding-3-small; asserts single embedding is 1536 dimensions, multi returns 3 vectors |
| many | yes | no | `embedMany` with 3 values; asserts count=3, all vectors non-empty, usage tokens > 0 |

### tool/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| with-suspend | yes | no | HITL via `requireApproval: true` tool; agent suspends, then `approveToolCallGenerate` resumes execution |

### middleware/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| wrap-model | yes | no | `generateText` with `maxOutputTokens`; basic model wrapping smoke test, asserts text and finishReason="stop" |
