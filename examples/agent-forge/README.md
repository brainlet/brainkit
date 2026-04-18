# agent-forge

**The flagship meta-programming example**: a Go process deploys a
multi-agent pipeline that designs, writes, and reviews a
brand-new brainkit agent from a freeform request. When the forge
returns approved source, Go scaffolds a real on-disk package
(`manifest.json` + `tsconfig.json` + `types/*.d.ts` + `index.ts`),
deploys it via `brainkit.PackageFromDir`, and calls the forged
agent through its public bus topic.

One `.ts` file wires, in a single compartment, every major
brainkit primitive the forge needs: Mastra `createWorkflow` +
`createStep`, a `dountil` review loop, three sub-reviewers
running in parallel, and the embedded reference corpus
(`reference.get("everything")`).

The Go side handles disk + deploy via the reusable
`brainkit.ScaffoldPackage` helper — the exact same layout the
`brainkit new package` CLI produces. After a forge run you can
`cd` into the scaffolded dir and open it in any IDE with full
TypeScript autocomplete, no `npm install` required.

## How it works

```
request
   │
   ▼
architect step (gpt-4o-mini)               ─┐
   │   JSON spec: { name, purpose,          │ inside forge.ts
   │   instructions, askShape, needsMemory }│ (SES compartment,
   ▼                                         │  Mastra workflow)
coder step (gpt-4o)                         │
   │   full brainkit reference corpus in   │
   │   system prompt; first-pass .ts       │
   ▼                                         │
dountil review loop (max 3 passes)          │
   │   ┌─ runReviewPanel (3 in parallel) ─┐ │
   │   │  safety-reviewer                  │ │
   │   │  style-reviewer                   │ │
   │   │  correctness-reviewer             │ │
   │   └───────────────────────────────────┘ │
   │        │ if any reviewer flags issues   │
   │        ▼                                 │
   │   patch-coder (gpt-4o) applies fixes    │
   │        │ unanimous approval              │
   ▼                                          │
shape-result step                            │
   │ { approved, name, code, iterations }   ─┘
   │
   ▼
Go process (main.go)
   │   ScaffoldPackage(./forged-agents/<name>/, name, "index.ts", code)
   │       writes manifest.json + tsconfig.json + types/*.d.ts + index.ts
   │   PackageFromDir(./forged-agents/<name>/) → kit.Deploy(…)
   │
   ▼
ts.<name>.ask
   │
   ▼
Go calls → forged agent replies
```

The workflow deliberately **stops before deploy**. Ownership is
clean: the forge produces reviewed source, the Go caller owns
the on-disk package directory and the deploy step. That means
every forged agent lives as an inspectable, editable,
version-controllable directory — not a blob of code wedged
into a JS string.

Iteration cap: 3 passes (original + 2 patches). When the cap is
hit without approval, the forge returns best-effort code + the
outstanding issue log so a human can finish the job. Nothing
ships without unanimous approval from all three reviewers.

## Run

```sh
OPENAI_API_KEY=sk-... go run ./examples/agent-forge
```

Flags:

| Flag | Default | Effect |
|------|---------|--------|
| `-request` | tweet-bot request | freeform description of the agent to forge |
| `-ask`     | demo prompt       | message sent to the forged agent after deploy |

Examples:

```sh
# Default run — forges a tweet-bot and makes it tweet
OPENAI_API_KEY=sk-... go run ./examples/agent-forge

# Explainer bot
OPENAI_API_KEY=sk-... go run ./examples/agent-forge \
  -request "Build me an agent that explains complex software concepts in plain English using a simple metaphor. Name it plain-explain." \
  -ask "What is eventual consistency in distributed databases?"

# Pirate translator
OPENAI_API_KEY=sk-... go run ./examples/agent-forge \
  -request "I need an agent that converts any English phrase into pirate speak. Name it pirate-translator." \
  -ask "hello there how are you doing today"
```

## Expected output

```
[1/3] forge pipeline deployed
        request: "..."
[2/3] running forge workflow (architect → coder → reviewer loop → deploy)…
        forge finished in 11s (iterations: 1)
        approved=true  deployed=true  name="tweet-bot"  topic="ts.tweet-bot.ask"
[3/3] calling ts.tweet-bot.ask with prompt="..."

--- forged agent reply ---
<witty tweet>
---
usage: prompt=71 completion=48 total=119
```

Typical wall time: 10–30 seconds, depending on model latency and
how many review passes the code needs.

## Models

| Role | Model | Why |
|------|-------|-----|
| Architect | `gpt-4o-mini` | Short structured JSON output, cheap |
| Coder + patch-coder | `gpt-4o` | Writes the .ts source — quality matters most |
| Safety / Style / Correctness reviewers | `gpt-4o` | `gpt-4o-mini` hallucinated issues; reliable grounding needs the bigger model |

Swap `CODER_MODEL` / `REVIEWER_MODEL` / `ARCHITECT_MODEL` in
`forge.ts` for your provider's strongest tier when it's available.

## Brainkit primitives on display

| Primitive | Where |
|-----------|-------|
| Embedded reference corpus | `await reference.get("everything")` — the entire self-description feeds the coder; `tool-author` pack feeds reviewers |
| Mastra workflows | `createWorkflow` + `.then(architectStep).then(coderStep).dountil(reviewStep, …).then(deployStep).commit()` |
| `dountil` loop | Early exit when reviewers approve OR iterationCount ≥ 3 — both conditions enforced in-step |
| Sub-agents | Three specialist reviewers running in parallel via `Promise.all`, aggregated deterministically |
| Agent composition | `createStep(agent)` + LLM-driven branching + Go-side orchestration |
| In-JS deployment | `bus.call("package.deploy", { manifest, files }, { timeoutMs })` — the same command the CLI uses |
| Typed Go caller | `brainkit.Call[sdk.CustomMsg, json.RawMessage](kit, ctx, sdk.CustomMsg{Topic, Payload})` |
| Defensive token mapping | The generated code maps `inputTokens` / `outputTokens` (AI SDK v5) alongside `promptTokens` / `completionTokens` (Mastra v4) |
| Error envelope | `BrainkitError("…", "VALIDATION_ERROR", {...})` inside steps → typed envelope on the Go side |

## Design notes

### Why text-parsed JSON instead of `output: ZodSchema`

The `agent.generate({ output: ZodSchema })` path does not
reliably populate `result.object` in this Mastra / AI-SDK fork —
the model receives the schema as a soft prompt but emits
markdown-fenced JSON in `result.text`, and the parsed object
never materializes. The forge instructs every agent to emit
JSON-only output with explicit shape, then parses `.text`
manually via `parseJSON()`. This works every time.

### Why three parallel reviewers instead of one supervisor

Mastra's `agents: { safety, style, correctness }` supervisor
pattern works when the coordinator's final reply is free-form,
but stitching three structured verdicts into one aggregated
verdict through a supervisor adds a fourth LLM call and a
failure mode (supervisor returns its own unstructured summary).
Calling the three reviewers directly and folding their verdicts
in JS is simpler, cheaper, and deterministic.

### Why feed the coder the entire corpus

Earlier attempts injected just the `agent-author` pack (~130kb).
The coder kept inventing symbols or using shapes that exist in
Mastra proper but not in brainkit's fork (e.g. ES `import`
statements, legacy `LibSQLVector({ connectionUrl })`). Feeding
the full `everything` pack (~250kb) costs a few more tokens but
eliminates that entire class of bug. Modern models easily fit
this inside their context window.

### Why the reviewers must QUOTE the code they flag

Reviewer hallucination was the biggest bug class during
development — reviewers would list every problem type from
their instructions regardless of whether it appeared in the
code. The REVIEWER_VERDICT_INSTRUCTIONS demand a quoted excerpt
with every issue. Combined with the reviewer reference corpus,
this grounds the review in the actual source.

## Extend this

- **Add a typecheck reviewer** that spawns a throwaway
  compartment with the generated code, catches the deploy
  failure, and feeds the error text to the patch coder.
- **Swap the in-memory workflow for a persistent one** —
  `createWorkflow` already supports `suspend`/`resume`, so you
  could pause the loop for human approval between passes.
- **Cache the reference corpus** — `await reference.get("everything")`
  at deploy time gives a stable snapshot; invalidate on
  brainkit upgrade.
- **Multi-tenant forge** — one forge per namespace, each with
  its own reviewer panel, gated by secrets or scopes.
- **Teardown** — call `brainkit.CallPackageTeardown` for the
  forged agent when you're done so bus subscriptions are
  cleaned up.

## Under the hood

- `forge.ts` is embedded into the Go binary via `//go:embed
  forge.ts` and deployed as the `agent-forge` package on Kit
  boot.
- `ts.agent-forge.create` is the public entry point; internally
  it creates a new Mastra workflow run (`forgeWorkflow.createRun()`)
  and awaits `run.start({ inputData: { request } })`.
- Each forged agent ends up as its own package (`ts.<name>.ask`)
  and is independent from the forge — delete the forge and the
  spawned agent keeps working.
