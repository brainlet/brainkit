---
name: types-worker
description: Per-domain type-alignment worker (M1-M13). Reads canonical truth, rewrites the domain's slice of agent.d.ts or ai.d.ts, fixes or adds fixtures, and passes both the type gate and the runtime gate.
---

# types-worker

NOTE: Startup and cleanup are handled by `worker-base`. This skill defines the WORK PROCEDURE.

## When to Use This Skill

Every domain milestone M1-M13 (tools, agent, memory, workflow, rag, evals, voice, observability, mcp, processors, vector, ai-sdk, coverage). One worker per feature; each feature lists the exact symbols / fixtures in scope.

## Required Skills

None. Pure source-reading, file-editing, and shell/Go test execution.

## Work Procedure

### Step 0: Read context

Read, in order:
1. The feature description in the assigned feature (from features.json).
2. `.factory/library/architecture.md` (invariants, baseline errors, milestone list).
3. `.factory/library/user-testing.md` (how validation works).
4. The validation contract section for this domain: `grep -A 5 "VAL-<AREA>-" /Users/davidroman/.factory/missions/<missionId>/validation-contract.md`.

Understand what "done" means for this milestone BEFORE touching any file.

### Step 1: Inventory canonical truth

For each symbol in scope, locate canonical truth. Priority order (see architecture.md Canonical Truth Hierarchy):

1. mastra clone at `/Users/davidroman/Documents/code/clones/mastra/packages/<pkg>/src/**`
2. ai clone at `/Users/davidroman/Documents/code/clones/ai/packages/<pkg>/src/**`
3. `internal/embed/agent/bundle/node_modules/<pkg>/dist/**/*.d.ts`

For each symbol, capture:
- Full canonical signature (generics, parameter types, return type)
- Public surface (methods, properties, overloads)
- Inheritance chain (`extends` / `implements`)

Write this inventory to a scratch file at `/tmp/<domain>-canonical-inventory.md` before editing anything.

### Step 2: Inventory current brainkit declarations

For each symbol, grep the relevant section of `agent.d.ts` or `ai.d.ts`:

```
grep -n "class <Symbol>\|interface <Symbol>\|type <Symbol>\|function <Symbol>\|const <Symbol>" \
  /Users/davidroman/Documents/code/brainlet/brainkit/internal/engine/runtime/agent.d.ts
```

Read the declaration plus 40 lines of surrounding context. Compare against canonical and list every DIFFERENCE in `/tmp/<domain>-drift.md`:
- Missing methods / properties
- Extra methods / properties that are not in canonical (candidates for removal)
- Wrong generic constraints
- Wrong parameter types or optionality
- Wrong return types
- Wrong inheritance (`implements` vs `extends`)

### Step 3: Red-first fixture-driven TDD

For every type fix, use fixtures as the failing test. Workflow:

1. Pick an existing fixture under `fixtures/ts/<domain>/` that imports the symbol you're fixing — or create a new one if none exercises it (gap closure per VAL-COVERAGE-001).
2. Write fixture code that uses the canonical shape (e.g. `createTool<string, { a: number }, { sum: number }>(...)` with the 7-generic signature, or `new CompositeVoice({ input: openai, output: elevenlabs })` with canonical field names).
3. If you added a new fixture, add `expect.json` with at least one assertion on the runtime output (use `output({...})` pattern — see any existing fixture).
4. Run `make type-check 2>&1 | grep fixtures/ts/<domain>` — the fixture must FAIL with a type error today (red).
5. Fix the declaration in `agent.d.ts` or `ai.d.ts` to match canonical.
6. Re-run `make type-check` — the fixture now passes (green). Confirm: domain-filtered errors = 0.

If a fix cascades and breaks a different domain's fixtures, STOP. Note the cross-domain impact in the handoff's `discoveredIssues` and continue only if it is clearly within the current domain's scope; otherwise return to orchestrator.

### Step 4: Run both gates for the domain

```
make type-check > /tmp/m<N>-type-check.log 2>&1
grep -c "fixtures/ts/<domain>.*error TS" /tmp/m<N>-type-check.log || echo 0
go test ./test/fixtures/ -run "TestFixtures/<domain>" -count=1 -timeout 600s > /tmp/m<N>-runtime.log 2>&1
tail -30 /tmp/m<N>-runtime.log
```

Both must pass:
- Type gate: domain-filtered error count = 0. (Whole-tree errors OUTSIDE the domain are acceptable at this milestone — they'll be cleaned up by their own milestones or by VAL-COVERAGE-004 at M13.)
- Runtime gate: all `TestFixtures/<domain>/*` subtests PASS or skip cleanly (skips only acceptable when an AI key or container is intentionally unavailable).

If the runtime gate fails for a reason UNRELATED to type changes (e.g. an existing flaky test), document it under the handoff's `discoveredIssues` with `severity: "low"` or `"medium"` and a suggested fix — DO NOT silently merge.

### Step 5: Canonical cross-reference audit

For each contract assertion in scope, prove it with an evidence line. Append to `/tmp/m<N>-evidence.md`:

```
VAL-<AREA>-001: <title>
  Canonical: path:line at clones/mastra/packages/<pkg>/src/<file>.ts
  Brainkit: path:line at internal/engine/runtime/agent.d.ts
  Diff: <summary of drift now fixed>
  Proof: <make type-check / go test command, result>
```

### Step 6: Update shared state

If you discovered a pattern worth recording for other workers, append to `.factory/library/architecture.md` under a domain-specific subsection. If you found a new invariant worth enforcing, add it to the "Invariants" list.

### Step 7: Commit

Commit in logical groups. At minimum:
1. "m<N>/<domain>: fixture updates for <symbol-family>"
2. "m<N>/<domain>: align <symbol-family> with canonical"
3. "m<N>/<domain>: add missing fixture coverage for <symbol-family>" (if applicable)

### Step 8: Handoff

Structure the handoff precisely. See Example Handoff below. Every contract assertion in scope must have a concrete evidence entry in `verification.commandsRun`.

## Example Handoff

```json
{
  "salientSummary": "M1/tools complete. Aligned createTool (now 7-generic), Tool, ToolAction, ToolExecutionContext, ToolsInput with @mastra/core/tools canonical. All 9 tools fixtures typecheck clean and run green under go test. Added generic-inference demonstration in tools/create-with-schema/index.ts; fixture no longer needs explicit arg type annotation.",
  "whatWasImplemented": "Rewrote createTool, Tool<...>, ToolAction<...>, ToolExecutionContext<...>, and ToolsInput in /Users/davidroman/Documents/code/brainlet/brainkit/internal/engine/runtime/agent.d.ts lines 540-670 to match /Users/davidroman/Documents/code/clones/mastra/packages/core/src/tools/tool.ts and types.ts 1:1 (7 generics on createTool, heterogeneous ToolsInput accepting VercelTool | VercelToolV5 | ProviderDefinedTool | ToolAction). Updated fixtures/ts/tools/create-with-schema/index.ts to demonstrate schema-driven arg inference (removed explicit type annotation on execute's first parameter; compiles via inference). Added per-symbol fixture coverage audit: 12 tools-related exports, 12 fixtures importing at least one (confirmed via rg).",
  "whatWasLeftUndone": "",
  "verification": {
    "commandsRun": [
      { "command": "make type-check 2>&1 | grep -c 'fixtures/ts/tools.*error TS'", "exitCode": 0, "observation": "0 errors" },
      { "command": "go test ./test/fixtures/ -run 'TestFixtures/tools' -count=1 -timeout 600s 2>&1 | tail -5", "exitCode": 0, "observation": "ok github.com/brainlet/brainkit/test/fixtures 4.2s; 9 subtests PASS 0 FAIL" },
      { "command": "diff <(rg 'export (class|function|interface|type) (createTool|Tool|ToolAction|ToolExecutionContext|ToolsInput)' /Users/davidroman/Documents/code/brainlet/brainkit/internal/engine/runtime/agent.d.ts | sort) <(rg 'export (class|function|interface|type) (createTool|Tool|ToolAction|ToolExecutionContext|ToolsInput)' /Users/davidroman/Documents/code/clones/mastra/packages/core/src/tools/*.ts | sort)", "exitCode": 0, "observation": "structural match (name/arity)" },
      { "command": "rg -l 'import \\{.*(createTool|Tool|ToolsInput).*\\} from \"agent\"' fixtures/ts/tools | wc -l", "exitCode": 0, "observation": "9 fixtures importing tools symbols" }
    ],
    "interactiveChecks": []
  },
  "tests": {
    "added": [
      { "file": "fixtures/ts/tools/create-with-schema/index.ts", "cases": [{ "name": "generic schema inference", "verifies": "adder.execute's first parameter is typed as { a: number; b: number } via inference, no explicit annotation; sum return type is number" }] }
    ]
  },
  "discoveredIssues": []
}
```

## When to Return to Orchestrator

- If canonical truth for a symbol cannot be located (absent from clones and from bundle node_modules). Document which symbol and which sources you checked.
- If a type fix in-scope would require edits to an off-limits file (`kit.d.ts`, `globals.d.ts`, `brainkit.d.ts`, `vendor_*`, `internal/embed/ai/bundle/`).
- If a type fix requires dropping a fixture that the user hasn't authorized dropping. (User policy: fixtures are fixed to match canonical; deletion needs explicit orchestrator approval.)
- If the runtime gate fails for reasons unrelated to your changes and no test pattern can isolate it.
- If a cross-domain impact is discovered that would require edits outside your feature's scope.
- If the bundle `node_modules` is missing for a canonical reference package (M0 should have installed it — escalate if it's absent when you look for it).
