# custom-scorer

Two domain-specific scorers built with Mastra's `createScorer`
builder, run side by side against the same dataset so the
regex-vs-LLM tradeoff is visible.

| Scorer | Model | Cost | When it wins |
|---|---|---|---|
| `cites-sources-regex` | none (regex) | free | Strict match on `[doc:<id>]` markers |
| `cites-sources-llm`   | `gpt-4o-mini` judge | tokens per item | Catches implicit citations — "according to doc 7" scores 1.0; regex scores 0.0 |

Expected run output:

```
  input                                      regex     llm  reason
    What is brainkit?                           1.00    1.00  Citation detected.
    Where does it ship?                         0.00    0.00  No citation found.
  ≠ Quote from the docs?                        0.00    1.00  Citation detected.
    Name the modules.                           1.00    1.00  Citation detected.
    Is it fast?                                 0.00    0.00  No citation found.

average regex:0.40  llm:0.60  disagreements:1/5
```

The `≠` marker highlights items where the two scorers disagree —
the classic "LLM catches what regex misses" case.

## Run

```sh
OPENAI_API_KEY=sk-... go run ./examples/custom-scorer
```

## The `createScorer` builder

Mastra's scorer API is a chainable builder, not a single config
call:

```ts
createScorer({ id, name, description })     // returns builder
    .preprocess(fn)                         // optional
    .analyze(fn)                            // optional
    .generateScore(fn | {judge, createPrompt})  // required
    .generateReason(fn | {judge, createPrompt}) // optional
```

Each step's output is available to the next via
`results.preprocessStepResult`, `results.analyzeStepResult`,
`results.generateScoreStepResult`.

### Code-only score function

```ts
const scorer = createScorer({ id, name, description })
    .generateScore(({ run }) => {
        // run.input: Message[], run.output: { role, text }
        return /\[doc:\w+\]/.test(run.output.text) ? 1.0 : 0.0;
    });
```

### LLM-judge score

```ts
const scorer = createScorer({ id, name, description })
    .generateScore({
        description: "Score 0-1 based on citation presence.",
        judge: { model: model("openai", "gpt-4o-mini"),
                 instructions: "Reply with ONLY the number 0 or 1." },
        createPrompt: ({ run }) => {
            return `Question: ${run.input[0].content}\nAnswer: ${run.output.text}\nCites a source? Reply 0 or 1.`;
        },
    });
```

### Running

```ts
const result = await scorer.run({
    input: [{ role: "user", content: "What is brainkit?" }],
    output: { role: "assistant", text: "brainkit is..." },
});
// result.score      — number
// result.reason     — string (when .generateReason was set)
// result.runId      — per-run identifier
```

## When to reach for each flavor

| Scenario | Use |
|---|---|
| Exact string patterns, regex-matchable | Code-only. Free, deterministic, fast. |
| Soft judgments (helpful, polite, grounded, cites implicitly) | LLM judge. Slower, costs tokens, but generalizes. |
| Cross-checking LLM-generated content | Ship BOTH — pattern-match on the cheap path, fall back to the LLM judge only for items the cheap path flagged as ambiguous. |
| CI pass/fail gate | Code-only (deterministic). LLM judges drift between model revisions. |
| Dataset labeling pass | LLM judge (fast to iterate, humans verify a sample). |

## Wiring into `runEvals` (batch gate)

`createScorer`-produced scorers plug straight into `runEvals`:

```ts
await runEvals({
    target: agent,
    data: dataset,
    scorers: [regexScorer, llmScorer],
    concurrency: 4,
});
```

Session 05 (`examples/evals/`) wires this into a CI quality
gate. This session keeps the scope small — just demonstrates the
builder + the two tradeoffs on hand-rolled data.

## Extension ideas

- **Weighted ensemble**: run both scorers, return `0.7 * regex + 0.3 * llm` — deterministic baseline with soft LLM correction.
- **Preprocess sharing**: a `.preprocess(fn)` step runs once; both `analyze` and `generateScore` see the result, so you don't re-parse the output on every scorer method.
- **Trajectory scorer**: for agents, `createScorer({ type: "trajectory" })` scores the whole tool-call trajectory, not just the final output. See `fixtures/ts/evals/scorer/with-preprocess/` for the shape.
- **Reason audits**: always add `.generateReason` for LLM judges — it makes CI failures debuggable.
- **Inline per-generate scoring**: `agent.generate(prompt, { scorers: { quality: { scorer: regexScorer } } })` runs scoring alongside the call (the scorer picks up the agent's own output, not a supplied `output`).

## See also

- `examples/evals/` (session 05) — runEvals batch + CI gate.
- `fixtures/ts/evals/scorer/*` — every scorer shape (basic,
  with-preprocess, with-reason, with-llm-judge).
- `docs/llm/mastra.md` — scorer section.
