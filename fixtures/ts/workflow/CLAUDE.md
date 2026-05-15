# workflow/ Fixtures

Tests the Mastra workflow engine: step chaining, branching, parallel execution, foreach iteration, loops, nested workflows, sleep, shared state, suspend/resume (HITL), error handling, hooks, and agent integration.

## Fixtures

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| basic-then | no | none | `createWorkflow` with two `.then()` steps: format (uppercase) then emphasize (append "!!!"); verifies "HELLO WORLD!!!" output |
| branch | no | none | `.branch()` with two conditions routes to "HIGH" or "LOW" step based on input value; collect step reads whichever branch ran via `getStepResult` |
| foreach | no | none | `.foreach()` iterates `processStep` over array produced by `produceStep`; doubles each element |
| loop-dountil | no | none | `.dountil()` increments counter via shared state until counter reaches 5; verifies loop ran exactly 5 times |
| nested | no | none | Outer workflow calls inner workflow within a step's execute; 11 + 10 = 21, 21 * 2 = 42 |
| parallel | no | none | `.parallel([stepA, stepB])` runs doubler and tripler concurrently; collect step sums results (5*2 + 5*3 = 25) |
| sleep | no | none | `.sleep(100)` pauses workflow between two steps; measures elapsed time to confirm pause occurred |

### errors/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| error-handling | no | none | Step throws intentional error; workflow reports "failed" status. Second run with no throw reports "completed". |

### hooks/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| after | no | none | Two-step workflow (add 1, multiply 2); verifies step-b output is 22 (from input 10) through step output inspection |

### integration/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| with-agent-step | yes | none | Workflow step creates Agent, calls `agent.generate()` to answer a question, formats result; combines workflows + AI |

### run/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| stream-writer | no | none | `run.stream()` delivers custom events emitted by step `writer.custom()`; records whether `writer.write()` also bubbles |

### scheduled/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| basic | no | none | Declarative `createWorkflow({ schedule })` auto-promotes to evented workflow, registers a Mastra schedule row, fires through Brainkit's workflow-event listener after a forced due tick, records trigger history, and tears down scheduler/listener state |
| multi | no | none | Array-form scheduled workflow registers one Mastra row per stable schedule id, preserves per-entry input/context/metadata, fires both due rows through Brainkit's workflow-event listener, records trigger history, and tears down scheduler/listener state |
| pause-resume | no | none | Paused schedule rows are skipped while due, preserve `nextFireAt`, then fire through Brainkit's workflow-event listener after the row is resumed to `active` |
| redeploy-diff | no | none | Declarative redeploy diffing updates cron/target/metadata without unpausing, recomputes `nextFireAt`, migrates single-form rows to array-form rows, deletes stale `wf_` rows, and preserves user-created rows |
| timezone | no | none | Timezone-aware scheduled workflow with `timezone: "America/New_York"`; verifies Croner computes initial and recomputed `nextFireAt` at 09:00 in the configured zone and fires through Brainkit's workflow-event listener |

### state/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| get-step-result | no | none | `getStepResult("compute")` in second step accesses first step's output (doubled = 42); verifies cross-step data flow |
| shared | no | none | Three steps share state via `setState`/`state`: init sets items array, accumulate appends, read outputs final state with 3 items and count 3 |

### suspend-resume/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| basic | no | none | Step calls `suspend()` with payload, workflow status becomes "suspended", then `run.resume()` with approval data completes the workflow |
| with-data | no | none | Suspend with structured payload (`needsApproval`, `item`), resume with `{ approved: true, approver: "david" }`; verifies round-trip data integrity |
