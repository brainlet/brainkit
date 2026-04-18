# hitl-workflow

Out-of-band human-in-the-loop via Mastra workflows: a step calls
`suspend({reason})`, the workflow snapshot persists to SQLite,
and a separate Go-side decision resumes the run with
`CallWorkflowResume`.

This is the counterpart to tool approval (session 06): tool
approval pauses mid-generation and must resolve inside the same
process; workflow suspend/resume is designed to survive a
process restart — the snapshot lives in storage, a human (or a
separate service) can approve hours or days later, and the
workflow picks up where it left off.

Runs **without an API key** — the steps are deterministic.

## What the example proves

Three-step workflow, `deploy-pipeline`:

1. **build** — stamps a version (deterministic, just a timestamp).
2. **approve** — calls `suspend({reason, artifact})` when there's
   no `resumeData` yet; the step returns, the workflow status
   becomes `"suspended"`.
3. **publish** — runs on resume with `{approved, approver}`;
   short-circuits with `{published: false, aborted: true}` if
   approval was declined.

```
[1/4] deploy-pipeline workflow registered
[2/4] starting run — build step fires, approve step will suspend
        runId=…  status=suspended
        suspended: "manual approval needed to publish brainkit@v1776524107 to staging"
[3/4] decision — auto-approving
[4/4] resuming with the decision
        status=success
        publish output: {
          "published": true,
          "releaseUrl": "https://releases.example.com/brainkit/v1776524107"
        }
```

## Run

```sh
go run ./examples/hitl-workflow
```

## Durability — why the Storage config matters

Mastra's default storage is in-memory. A suspended run lives in
RAM; kill the process and the snapshot is gone.

The example wires SQLite under the Kit's `FSRoot`:

```go
Storages: map[string]brainkit.StorageConfig{
    "default": brainkit.SQLiteStorage(filepath.Join(tmp, "workflow.db")),
},
```

brainkit's `internal/engine/runtime/patches.js` upgrades the
workflow's internal `InMemoryStore` to whatever lands in the
`"default"` slot, so `createWorkflow` + `kit.register("workflow",
...)` automatically persist their state.

Swap for Postgres to survive real infra:

```go
"default": brainkit.PostgresStorage(os.Getenv("DATABASE_URL"))
```

`examples/storage-vectors/docker-compose.yml` starts a ready-to-use
Postgres on port 5433.

## Shape of `suspendSchema` + `resumeSchema`

Each step declares both schemas so Mastra + TypeScript can
validate the handshake:

```ts
createStep({
    id: "approve",
    inputSchema: z.object({ version: z.string(), ... }),
    outputSchema: z.object({ approved: z.boolean(), ... }),
    resumeSchema: z.object({ approved: z.boolean(), approver: z.string().optional() }),
    suspendSchema: z.object({ reason: z.string(), artifact: z.string() }),
    execute: async ({ inputData, resumeData, suspend }) => {
        if (!resumeData) {
            return await suspend({ reason: "...", artifact: "..." });
        }
        return { ...inputData, approved: resumeData.approved };
    },
});
```

The first call into the step has `resumeData === undefined` — it
calls `suspend()` and returns. On `CallWorkflowResume`, the
same step executes again with `resumeData` populated from the
Go call's `ResumeData` payload.

## Go-side round trip

```go
// Start.
start, _ := brainkit.CallWorkflowStart(kit, ctx, sdk.WorkflowStartMsg{
    Name:      "deploy-pipeline",
    InputData: json.RawMessage(`{"component":"brainkit","env":"staging"}`),
})
// start.Status == "suspended"
// start.Steps contains { steps: { approve: { suspendedPayload: {...} } } }

// Resume (later — could be a different process).
resume, _ := brainkit.CallWorkflowResume(kit, ctx, sdk.WorkflowResumeMsg{
    Name:       "deploy-pipeline",
    RunID:      start.RunID,
    Step:       "approve",
    ResumeData: json.RawMessage(`{"approved":true}`),
})
// resume.Status == "success"
```

`CallWorkflowStatus` with the same `{Name, RunID}` polls the
current state without advancing it. Useful for dashboards +
watchers.

## Extension ideas

- **Multi-approver** — chain two `approve` steps with different
  `suspendSchema.reason` prefixes; require both to resume.
- **TTL on approval** — combine with `modules/schedules` to
  auto-decline suspended runs after N minutes.
- **External notification** — publish a `bus.emit("approval.needed", {runId, reason})`
  from the suspend step so a Slack bot / webhook can pick it up.
- **Real-world HITL UI** — the `runId` is stable across processes
  when storage is durable, so a separate operator UI can
  `CallWorkflowStatus` + display the `suspendedPayload.reason` +
  collect a decision + `CallWorkflowResume`.

## Tool HITL vs workflow HITL — one-line diff

| | Tool approval (session 06) | Workflow suspend (this session) |
|---|---|---|
| Granularity | one tool call inside a generate | one step inside a workflow |
| Scope | mid-generation | between steps |
| Durability | in-memory — dies with the agent | persisted to `Storage` — survives a restart |
| Resume API | `agent.approveToolCallGenerate({runId, toolCallId})` | `run.resume({step, resumeData})` / `CallWorkflowResume` |

## Under the hood

- `suspend()` + `resume()` on the workflow run are Mastra core
  APIs. brainkit's workflow module forwards through
  `__brainkit.workflow.resume` (`internal/engine/runtime/dispatch.js:63-73`).
- The `Storage` hook that persists snapshots is wired via
  `internal/engine/runtime/patches.js:14-55` — it swaps every
  workflow's default `InMemoryStore` for whatever `Storages["default"]`
  resolves to.
- `restartAllActiveWorkflowRuns` (same `dispatch.js`) rehydrates
  active runs on Kit boot; use it when you want durable
  suspend-and-wait-for-days semantics.
