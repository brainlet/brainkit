# evals/ Fixtures

Tests the evaluation/scoring framework: `createScorer` builder pattern, function-based scorers, LLM judge scorers, preprocess+analyze pipelines, and batch evaluation runs.

## Fixtures

### batch/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| run-evals | no | none | Runs two independent scorers (length-based and keyword-based) against the same input/output pair; confirms both produce numeric scores and keyword scorer is positive |

### scorer/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| basic | no | none | `createScorer` with `.generateScore()` callback; verifies score is numeric, in [0,1] range, and runId is generated |
| with-llm-judge | yes | none | `createScorer` with `.generateScore({ judge: { model }, createPrompt })` -- uses GPT-4o-mini as LLM judge to score output quality |
| with-preprocess | no | none | Full scorer pipeline: `.preprocess()` extracts word sets, `.analyze()` computes overlap, `.generateScore()` derives final score from analysis |
| with-reason | no | none | `.generateScore()` + `.generateReason()` chain; verifies score is 1.0 for matching keywords and reason string explains the score |
