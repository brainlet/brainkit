# agent-stream

Mastra's `agent.stream()` surface, exercised two ways:

1. **Plain text streaming** — `for await (const delta of stream.textStream)` → each token as a chunk on the bus → SSE gateway.
2. **Structured output streaming** — `agent.stream(prompt, { structuredOutput: { schema: z.array(...) } })` → filter `stream.fullStream` for `"object-result"` chunks → stream typed partials → `await stream.object` gives the final parsed value.

The example pairs each with a gateway SSE route so a browser /
curl sees the same streaming from outside the Kit.

## Run

```sh
OPENAI_API_KEY=sk-... go run ./examples/agent-stream
```

Expected bus round trip (from Go's `CallStream`):

```
[1/3] agent-stream deployed
[2/3] bus CallStream on ts.agent-stream.haiku:
Crimson whispers fall,
Dancing on the water edge,
Nature's soft embrace.

[3/3] bus CallStream on ts.agent-stream.plan (structured):
  • Define the core features… — Nails down the scope before you build
  • Create a project timeline… — Milestones keep every contributor aligned
  …

final plan (5 steps):
  1. Define the core features…
  2. Create a project timeline…
  3. Develop a prototype…
  4. Implement a marketing strategy…
  5. Gather user feedback…
```

After the bus round trip completes, the gateway is still up.
Open a second shell and hit the SSE endpoints live:

```sh
curl -N 'http://127.0.0.1:<port>/sse/haiku?prompt=spring+blossoms'
curl -N 'http://127.0.0.1:<port>/sse/plan?goal=launch+product'
```

Each returns a well-formed SSE stream with `data:` lines
carrying the JSON chunks the `.ts` side emitted via `msg.send`.

## Bus streaming vs `agent.stream()`

| | Bus `CallStream` | `agent.stream()` |
|---|---|---|
| Who emits | the `.ts` handler via `msg.send(chunk)` | Mastra's Agent via `stream.textStream` / `stream.fullStream` |
| Who consumes | Go caller, via `brainkit.CallStream[Req, Chunk, Resp]` | the deployed `.ts` (or a Go caller through a bus handler) |
| Typical shape | whatever the handler sends | model-driven token deltas + structured-output parts |

This example wires both: the `.ts` consumes `agent.stream()`
inside the compartment and re-emits the interesting parts as
bus chunks. That's the pattern for any UI that wants token-level
streaming backed by an LLM.

## Structured output — what you get

With `structuredOutput: { schema: zodSchema }` on `agent.stream()`:

| Property | What it gives you |
|---|---|
| `stream.textStream` | still usable — token deltas as strings |
| `stream.fullStream` | typed chunks: `"text-delta"`, `"object-result"`, `"tool-call"`, `"finish"`, `"error"` |
| `stream.objectStream` | async iterable of `Partial<OUTPUT>` as the object fills in |
| `stream.object` | `Promise<OUTPUT>` — final parsed value, resolves on `finish` |

This example filters `fullStream` for `"object-result"` chunks.
The runtime emits **snapshots of the whole partial array** each
chunk, so the `.ts` code sends only the tail (newest filled-in
step) as a bus chunk to keep the stream incremental. If you
want cumulative partials, forward `part.object` verbatim.

## Gateway SSE wiring

```go
gw := gateway.New(gateway.Config{Listen: listenAddr, Timeout: 60 * time.Second})
gw.HandleStream("GET", "/sse/haiku", "ts.agent-stream.haiku")
gw.HandleStream("GET", "/sse/plan",  "ts.agent-stream.plan")
```

`HandleStream` forwards query-string params into `msg.payload`
(so `?prompt=spring+blossoms` reaches the handler as
`msg.payload.prompt`), subscribes to `msg.send` chunks + the
terminal reply, and writes each as an SSE event. The post-1.0
fix that unwraps envelope-terminal replies lives in
`modules/gateway/stream.go:56-90` — it's what makes the SSE
`event: end` line carry the full JSON object on the final reply.

## The deployed `.ts`, annotated

```ts
const haikuAgent = new Agent({
    name: "haiku-streamer",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Write one short haiku about the topic. Three lines.",
});

bus.on("haiku", async (msg) => {
    const stream = await haikuAgent.stream(msg.payload.prompt);
    for await (const delta of stream.textStream) {
        msg.send({ delta });          // one bus chunk per token delta
    }
    msg.reply({ done: true });        // envelope-terminal marks end of SSE
});

const planSchema = z.array(z.object({ step: z.string(), why: z.string() }));

bus.on("plan", async (msg) => {
    const stream = await plannerAgent.stream(
        "Goal: " + msg.payload.goal,
        { structuredOutput: { schema: planSchema } },
    );
    for await (const part of stream.fullStream) {
        if (part && part.type === "object-result" && Array.isArray(part.object)) {
            const latest = part.object[part.object.length - 1];
            if (latest && latest.step) {
                msg.send({ step: latest.step, why: latest.why || "" });
            }
        }
    }
    msg.reply({ object: await stream.object });  // final value
});
```

## Extension ideas

- **Switch to `.objectStream`** — `for await (const partial of stream.objectStream)` iterates cumulative partials directly (same shape as filtering `fullStream` for `object-result`).
- **WS / Webhook parity** — swap `HandleStream` for `HandleWebSocket` / `HandleWebhook`; the `.ts` side is unchanged. See `examples/streaming/` for the WS pattern.
- **Reasoning + token usage** — both arrive as typed chunks inside `fullStream` (`"reasoning-delta"`, `"finish"` carries `usage`). Fan them out to a separate `msg.send({ type: "reasoning", delta })` for a richer UI.
- **Pipe into a UI** — a browser page reading `/sse/plan` shows plan steps filling in as the model generates them, no polling needed.

## See also

- `examples/streaming/` — raw bus `CallStream` + SSE + WS + Webhook (no LLM; deterministic counter).
- `examples/ai-chat/` — `generateText` baseline (no streaming).
- `docs/guides/ts-services.md` — streaming subsection.
