# evals

Batch evaluation of an Agent using Mastra's `runEvals` + two
prebuilt scorers, shaped as a CI quality gate that mirrors the
bench regression gate (`bench-save` / `bench-check`).

Two scorers:

| Scorer | Kind | Cost |
|---|---|---|
| `createAnswerRelevancyScorer({ model })` | LLM-as-judge | tokens per item |
| `createCompletenessScorer()`             | code-only    | free |

## Run

```sh
OPENAI_API_KEY=sk-... go run ./examples/evals
OPENAI_API_KEY=sk-... go run ./examples/evals -check   # compare vs baseline.json
OPENAI_API_KEY=sk-... go run ./examples/evals -save    # write latest.json
```

Expected tail:

```
[2/3] running runEvals …
        6 items scored in 22s
        scorer averages:
          answer-relevancy-scorer 0.805
          completeness-scorer     0.414

[3/3] comparing to baseline.json:
  answer-relevancy-scorer  base=0.850  got=0.805  delta=-5.3%  ok
  completeness-scorer      base=0.400  got=0.414  delta=+3.6%  ok
all scorers within 25% tolerance
```

## Shape of the CI gate

```
dataset.json   (committed — 6 representative prompts)
     │
     ▼
runEvals({target: agent, data, scorers, concurrency: 2})
     │
     ▼
aggregate scores per scorer id:
 { "answer-relevancy-scorer": 0.805, "completeness-scorer": 0.414 }
     │
     ▼
baseline.json  (committed — expected averages + tolerance)
     │
     ▼
exit 0 if every scorer stays within tolerance_percent
exit 1 on regression → CI fails, PR needs either a fix or a
                        deliberate baseline refresh
```

## Shipped with the example

- **`dataset.json`** — 6 prompts, mix of easy factual + harder explanations. Add rows to extend coverage.
- **`baseline.json`** — expected averages + `_tolerance_percent` (default 25%, accommodates LLM-judge jitter).
- **`main.go`** — runs the eval, prints per-scorer averages, compares against the baseline on `-check`.

## Baseline workflow

```sh
# 1. Change the agent / prompt / dataset.
# 2. Regenerate baseline.json:
OPENAI_API_KEY=sk-... go run ./examples/evals -save
cp examples/evals/latest.json examples/evals/baseline.json
# Inspect the diff before committing.

# 3. Lock in — every PR's CI run loads the new baseline.
```

Never auto-overwrite `baseline.json` in CI. A regression is a
signal that either the agent got worse or the baseline is stale
— the decision needs a human.

## CI integration

The gate runs non-interactively given `OPENAI_API_KEY`. Add to
`.github/workflows/evals.yml`:

```yaml
- name: Evals regression gate
  env:
    OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
  run: go run ./examples/evals -check
```

Skip the job when the key isn't set (contributor PRs from forks
without the secret should succeed):

```yaml
if: ${{ env.OPENAI_API_KEY != '' }}
```

## Cost note

Each `answer-relevancy-scorer` hop is an LLM judge call. With
`concurrency: 2` and 6 items, the example does roughly:

- 6 agent generations (target)
- 6 judge calls (relevancy)
- 6 code scorings (completeness — zero tokens)

= **12 OpenAI round trips** on `gpt-4o-mini`, usually well under
a dollar per run. Scale linearly with dataset size and scorer
count.

## Prebuilt scorers now shipped as endowments

brainkit previously bundled the prebuilt scorer factories but
didn't expose them. This session added:

| Endowment | Via |
|---|---|
| `createAnswerRelevancyScorer` | `internal/engine/runtime/kit_runtime.js` + `agent_module.js` |
| `createCompletenessScorer`    | same |
| `createFaithfulnessScorer`, `createBiasScorer`, `createHallucinationScorer`, `createToxicityScorer`, `createContextPrecisionScorer`, `createAnswerSimilarityScorer` | same (LLM judges; pass a `model`) |
| `createKeywordCoverageScorer`, `createContentSimilarityScorer`, `createToneScorer`, `createTextualDifferenceScorer` | same (code-only; no model arg) |
| `createContextRelevanceScorerLLM`, `createNoiseSensitivityScorerLLM`, `createPromptAlignmentScorerLLM`, `createToolCallAccuracyScorerLLM` | same (LLM judges) |

Bundle + bytecode rebuilt per CLAUDE.md's 3-step protocol.

## Extension ideas

- **Trajectory scorers**: pass `scorers: { agent: [...], trajectory: [...] }`
  to evaluate the tool-call trajectory as well as the final
  output.
- **Custom scorers inline**: mix prebuilt with `createScorer`
  builders from `examples/custom-scorer/` — `runEvals` takes a
  flat array of `Scorer` values.
- **Dataset sourcing**: load production logs as the dataset
  (read-only copy, sample N items).
- **Score history**: pair with `modules/audit` to persist each
  run's averages, then chart over time.

## See also

- `examples/custom-scorer/` — writing your own `createScorer`
  builder.
- `fixtures/ts/evals/*` — reference fixtures for every scorer
  shape.
- `scripts/bench-compare.go` — the bench gate this evals gate
  mirrors.
- `docs/guides/hitl-approval.md` — mentions evals as part of
  the quality-engineering story.
