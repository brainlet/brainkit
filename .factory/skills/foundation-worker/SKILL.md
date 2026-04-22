---
name: foundation-worker
description: One-shot setup worker for M0. Refreshes mastra + ai clones to target versions, runs pnpm install in the bundle, adds a root package.json pinning tsc, wires the `make type-check` target, and captures baseline errors.
---

# foundation-worker

NOTE: Startup and cleanup are handled by `worker-base`. This skill defines the WORK PROCEDURE.

## When to Use This Skill

M0 only. This worker sets up the mission tooling so every downstream `types-worker` can run the type-check gate and consult canonical truth.

## Required Skills

None. Pure shell + file-editing work.

## Work Procedure

Every step must be verified with a concrete command. Do not skip any step.

### Step 1: Refresh mastra clone to @mastra/core@1.13.x

```
cd /Users/davidroman/Documents/code/clones/mastra
git fetch --tags --prune origin
# Find the target tag. The canonical target is @mastra/core@1.13.1; if it does not exist as a
# single tag, pick the latest 1.13.x tag that exists.
git tag --list '@mastra/core@1.13.*' | sort -V | tail -5
# Check out the tag (or the newest 1.13.x):
git checkout @mastra/core@1.13.1
# Verify version:
node -e "console.log(require('./packages/core/package.json').version)"  # expect 1.13.1
```

If the exact tag doesn't exist, use the newest available under `@mastra/core@1.13.*` and record the chosen tag in `.factory/library/architecture.md` (under a new "Pinned versions" subsection).

### Step 2: Refresh ai clone to ai@6.0.x

```
cd /Users/davidroman/Documents/code/clones/ai
git fetch --tags --prune origin
git tag --list 'ai@6.0.*' | sort -V | tail -10
git checkout ai@6.0.116   # or the newest 6.0.x
node -e "console.log(require('./packages/ai/package.json').version)"   # expect 6.0.x
```

Record the chosen tag in `.factory/library/architecture.md`.

### Step 3: Install bundle node_modules for non-cloned packages

```
cd /Users/davidroman/Documents/code/brainlet/brainkit/internal/embed/agent/bundle
pnpm install --prefer-offline
# Verify:
ls node_modules/@mastra/libsql node_modules/@mastra/pg node_modules/@mastra/mongodb \
   node_modules/@mastra/upstash node_modules/@mastra/chroma node_modules/@mastra/pinecone \
   node_modules/@mastra/qdrant node_modules/@mastra/observability \
   node_modules/@mastra/voice-openai node_modules/@mastra/voice-deepgram \
   node_modules/@mastra/voice-elevenlabs \
   node_modules/ai node_modules/@ai-sdk/openai
```

Every directory must exist. If any is missing, the pnpm install failed — do NOT proceed; return to orchestrator.

### Step 4: Create a root `package.json` pinning typescript

At `/Users/davidroman/Documents/code/brainlet/brainkit/package.json` write:

```json
{
  "name": "brainkit-type-gate",
  "private": true,
  "description": "Root package.json for the brainkit type-check gate. Do not add runtime deps here.",
  "devDependencies": {
    "typescript": "5.9.3"
  },
  "scripts": {
    "type-check": "tsc --noEmit -p fixtures/tsconfig.base.json"
  }
}
```

Then `npm install` at the repo root. Verify: `./node_modules/.bin/tsc --version` prints `5.9.3`.

### Step 5: Add `type-check` target to Makefile

Append (do NOT replace existing targets) a `type-check` target. Example:

```
.PHONY: type-check
type-check: ## Run tsc --noEmit on all fixtures
	tsc --noEmit -p fixtures/tsconfig.base.json
```

Verify: `make type-check` runs and produces output (likely with errors — that is expected at baseline).

### Step 6: Add v6 header comment to ai.d.ts

At the top of `/Users/davidroman/Documents/code/brainlet/brainkit/internal/engine/runtime/ai.d.ts`, add (do not remove any existing content):

```
// This module targets ai-sdk v6 (package `ai` ^6.0.x). The legacy v4 bundle at
// internal/embed/ai/bundle is out of scope and must not be referenced here.
```

### Step 7: Capture baseline type errors

```
make type-check > /tmp/baseline-type-check.log 2>&1 || true
wc -l /tmp/baseline-type-check.log
grep -cE '^[^ ].*error TS' /tmp/baseline-type-check.log || echo "0 errors"
```

Append the baseline summary (error count, error list grouped by fixture path prefix) to `.factory/library/architecture.md` under the "Baseline error snapshot" section.

### Step 8: Verify off-limits files untouched

```
cd /Users/davidroman/Documents/code/brainlet/brainkit
git diff --name-only HEAD | tee /tmp/m0-changes.txt
```

Confirm NONE of these are in the diff:
- `internal/engine/runtime/kit.d.ts`
- `internal/engine/runtime/globals.d.ts`
- `internal/engine/runtime/brainkit.d.ts`
- `internal/engine/runtime/assemblyscript.d.ts`
- `internal/embed/ai/bundle/**`
- `vendor_quickjs/**`
- `vendor_typescript/**`

If any off-limits file is in the diff, REVERT that change.

### Step 9: Commit

Split into three commits for clarity:
1. "m0: refresh clones and pnpm-install bundle dependencies"
2. "m0: add root package.json + Makefile type-check target"  
3. "m0: capture baseline type errors and ai.d.ts v6 header"

## Example Handoff

```json
{
  "salientSummary": "M0 foundation complete. Mastra clone pinned to @mastra/core@1.13.1; ai clone pinned to ai@6.0.116. pnpm install succeeded (594MB node_modules) with all 9 @mastra/* and 12 @ai-sdk/* subpackages present. Root package.json with typescript@5.9.3, Makefile `type-check` target wired. Baseline captured: 147 TS errors across 52 fixtures, 8 error classes. ai.d.ts v6 header added.",
  "whatWasImplemented": "Refreshed /Users/davidroman/Documents/code/clones/mastra to tag @mastra/core@1.13.1 and /Users/davidroman/Documents/code/clones/ai to tag ai@6.0.116; ran pnpm install in /Users/davidroman/Documents/code/brainlet/brainkit/internal/embed/agent/bundle populating node_modules with all required @mastra/* and @ai-sdk/* packages; created /Users/davidroman/Documents/code/brainlet/brainkit/package.json pinning typescript@5.9.3 and a `type-check` npm script; ran `npm install` at repo root; appended a `.PHONY: type-check` target to the Makefile that runs `tsc --noEmit -p fixtures/tsconfig.base.json`; added the v6 target comment to ai.d.ts header; ran `make type-check`, captured baseline output to /tmp/baseline-type-check.log, summarized to .factory/library/architecture.md under 'Baseline error snapshot'.",
  "whatWasLeftUndone": "",
  "verification": {
    "commandsRun": [
      { "command": "cd /Users/davidroman/Documents/code/clones/mastra && git describe --tags --exact-match", "exitCode": 0, "observation": "@mastra/core@1.13.1" },
      { "command": "cd /Users/davidroman/Documents/code/clones/ai && git describe --tags --exact-match", "exitCode": 0, "observation": "ai@6.0.116" },
      { "command": "ls /Users/davidroman/Documents/code/brainlet/brainkit/internal/embed/agent/bundle/node_modules/@mastra | wc -l", "exitCode": 0, "observation": "14 packages present" },
      { "command": "tsc --version", "exitCode": 0, "observation": "Version 5.9.3" },
      { "command": "make type-check 2>&1 | tail -3", "exitCode": 2, "observation": "147 errors found — matches expected baseline drift; captured to /tmp/baseline-type-check.log" },
      { "command": "git diff --name-only HEAD -- internal/engine/runtime/kit.d.ts internal/engine/runtime/globals.d.ts internal/engine/runtime/brainkit.d.ts vendor_quickjs vendor_typescript internal/embed/ai/bundle", "exitCode": 0, "observation": "empty (no off-limits files touched)" }
    ],
    "interactiveChecks": []
  },
  "tests": {
    "added": []
  },
  "discoveredIssues": []
}
```

## When to Return to Orchestrator

- If a required tag (`@mastra/core@1.13.1` or `ai@6.0.116`) cannot be fetched.
- If `pnpm install` fails with dependency resolution errors (likely indicates a need to adjust the bundle's `package.json`, which is out of scope).
- If any off-limits file change is introduced by pnpm install or another tool and cannot be reverted.
- If `make type-check` fails to run at all (tsc not found, config missing) rather than just producing type errors.
