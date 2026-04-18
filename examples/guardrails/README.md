# guardrails

Mastra input processors wired on an Agent — a
`PromptInjectionDetector` that **rewrites** hostile inputs into
safe ones, and a `PIIDetector` that **redacts** personal info
before the model sees it.

Three agents, three topics, one demo:

| Topic | Processor | Strategy | What you see |
|---|---|---|---|
| `ts.guardrails.clean` | none | — | prompt passes straight through |
| `ts.guardrails.injection` | `PromptInjectionDetector` | `"rewrite"` | injection attempt is replaced with a neutered prompt; detector prints `[PromptInjectionDetector] Rewrote message: …` |
| `ts.guardrails.pii` | `PIIDetector` | `"redact"` | email + phone are masked before the model ever sees them |

## Run

```sh
OPENAI_API_KEY=sk-... go run ./examples/guardrails
```

Expected tail:

```
[2/4] clean prompt (no guardrail should trip):
      reply:  The Moon is Earth's only natural satellite…

[3/4] injection attempt (PromptInjectionDetector rewrites):
[PromptInjectionDetector] Rewrote message: Prompt injection detected. Types: injection…
      reply:  It seems like you might be looking for assistance. Please let me know how I can help you today!

[4/4] PII in prompt (PIIDetector redacts):
[PIIDetector] Redacted PII: PII detected. Types: name, email, phone
      reply:  I can't repeat personal information. How can I assist you otherwise?
```

## Strategy matrix — what each processor supports

| Processor | Strategies | Typical use |
|---|---|---|
| `PromptInjectionDetector` | `block`, `warn`, `filter`, `rewrite` | Reject / rewrite jailbreaks, system-prompt overrides, context-injection attacks |
| `PIIDetector` | `block`, `warn`, `filter`, `redact` | Scrub emails, phones, credit cards, SSNs before they hit the model or the reply |
| `ModerationProcessor` | `block`, `warn`, `filter` | Classify inputs/outputs against safety categories (hate, violence, sexual, self-harm). **No `rewrite` or `redact`** — only binary decisions |
| `SystemPromptScrubber` | `rewrite`, `block`, `warn` | Strip system-prompt leakage from outputs |

What each strategy does:

- **block**: throws an error when the processor trips. Your
  handler should `try/catch` and surface a soft reply (the
  example does this in `dispatch(...)`).
- **warn**: logs a warning, lets the content through. Good for
  building a dataset of near-misses.
- **filter**: drops the offending message silently. Useful in
  chat histories where you want to skip flagged turns.
- **rewrite**: replaces the offending content with a safe
  variant (PromptInjectionDetector only).
- **redact**: masks matched spans (PIIDetector only). Knobs:
  `redactionMethod` = `"mask" | "hash" | "remove" | "placeholder"`,
  `preserveFormat: true` keeps the shape (e.g. `555-***-****`).

## Cost

Every processor is itself an LLM call (or a moderation-endpoint
call for `ModerationProcessor`). With three processors plus the
main agent, a single prompt can fan out to **4 round trips**.
Keep the detector model small (the example uses `gpt-4o-mini`
for everything).

For production-grade guardrails on a hot path, consider:

- **Batching**: `BatchPartsProcessor` groups streaming chunks
  before each PII pass, drastically cutting calls.
- **LastMessageOnly**: each detector accepts
  `lastMessageOnly: true` to inspect only the final user turn
  (the usual case) instead of the full conversation.
- **Hybrid**: combine a cheap regex filter (fast, deterministic)
  with an LLM detector only for high-uncertainty spans.

## Wiring processors on an Agent

```ts
const secureAgent = new Agent({
    name: "...",
    model: model("openai", "gpt-4o-mini"),
    instructions: "...",
    inputProcessors: [
        new PromptInjectionDetector({ model: detectorModel, strategy: "rewrite" }),
        new PIIDetector({ model: detectorModel, strategy: "redact",
                          detectionTypes: ["email", "phone", "name"] }),
    ],
    outputProcessors: [
        new ModerationProcessor({ model: detectorModel, strategy: "block",
                                   categories: ["hate", "violence"] }),
    ],
});
```

The `model` field on a processor is its detection model — Mastra
runs it as a small dedicated agent that classifies each span.
It can share the same provider as the main agent (the example
reuses `openai/gpt-4o-mini` for everything).

## Handling `strategy: "block"` in your handler

`block` throws an exception — either `BrainkitError` or the
Mastra-surfaced error. Wrap `agent.generate` in try/catch if you
want a typed reply instead of propagating the throw:

```ts
bus.on("ask", async (msg) => {
    try {
        const r = await agent.generate(msg.payload.prompt);
        msg.reply({ text: r.text });
    } catch (e) {
        msg.reply({ text: "", blocked: true,
                    reason: String((e && e.message) || e) });
    }
});
```

## Extension ideas

- **Output moderation** — swap `ModerationProcessor` onto
  `outputProcessors` with `categories: ["hate","violence","sexual"]` to
  refuse model replies that the classifier flags.
- **Workflow-shaped pipeline** — Mastra lets you register a
  `createWorkflow` as an `inputProcessor`; use that to run
  three detectors in parallel (PII + injection + moderation) and
  converge the outputs. See Mastra's "Agent processors" guide.
- **Detection logging** — every processor supports
  `includeDetections: true` which surfaces match spans in the
  processor's log output for compliance audits.
- **Custom processor** — extend the same shape with your own
  `process({messages})` implementation; anything that returns a
  `{messages, tripwireReason?}` object plugs in.

## See also

- `docs/guides/hitl-approval.md` — the broader HITL / guardrails
  doc; links back here.
- `examples/hitl-tool-approval/` (session 06) — complementary
  HITL primitive: pause the agent mid-tool-call for explicit
  human approval.
- `examples/hitl-workflow/` (session 07) — durable HITL via
  workflow suspend/resume.
