# ai-chat

Library-mode AI surface: register a provider, deploy a `.ts`
handler that calls `generateText` through the bundled AI SDK,
print the model's reply.

## Run

```sh
export OPENAI_API_KEY=sk-...
go run ./examples/ai-chat
```

Expected output (OpenAI, gpt-4o-mini):

```
provider=openai  model=gpt-4o-mini
---
Hello, world! I hope you're having a great day.
---
tokens: prompt=14 completion=12 total=26
```

## Flags

- `--prompt "…"` — what to send to the model.
- `--provider <name>` — `openai` (default), `anthropic`, `google`,
  `groq`, `mistral`, `deepseek`, `xai`, `cohere`, `perplexity`,
  `togetherai`, `fireworks`, `cerebras`.
- `--model <id>` — model identifier for the provider
  (default `gpt-4o-mini`).
- `--api-key <key>` — override the environment variable.

Each provider reads its key from the standard env var
(`ANTHROPIC_API_KEY`, `GOOGLE_API_KEY`, etc.) unless `--api-key`
is passed.

## Cookbook

Swap to Anthropic:

```sh
export ANTHROPIC_API_KEY=sk-ant-...
go run ./examples/ai-chat \
    --provider anthropic \
    --model claude-3-5-sonnet-latest \
    --prompt "Write one sentence about brainkit."
```

## What it shows

- Provider registration via `brainkit.Config.Providers`
  (`brainkit.OpenAI(key)`, `brainkit.Anthropic(key)`, …).
- The `.ts` side calls `model(provider, modelID)` to resolve the
  configured provider, then feeds it into `generateText`.
- Go-side invocation through `brainkit.Call` — the `.ts` handler
  replies with `{text, usage, finishReason}`, we pretty-print it.

## Streaming

For token-by-token streaming, swap `generateText` for `streamText`
inside the `.ts`, and read chunks via `brainkit.CallStream` on the
Go side. See `examples/streaming/` once that example lands.
