# HITL Approval

Human-in-the-Loop (HITL) ships on two stable surfaces:

1. **Agent tool approval** — `generateWithApproval` suspends an
   agent mid-run when it picks a tool with `requireApproval: true`,
   routes the approval request through the bus, and resumes once a
   reply arrives.
2. **Workflow suspend/resume** — a workflow step calls `await
   suspend(...)` to pause; any surface publishes
   `workflow.resume` (or `brainkit.CallWorkflowResume`) with the
   `resumeData` matching the step's `resumeSchema`, and the workflow
   continues from where it stopped.

A third surface — `modules/harness` — is **WIP**. It exposes a
frozen `Instance` interface and six frozen event types today;
everything else is subject to change. See the harness section at
the bottom for scope.

## Agent HITL — `generateWithApproval`

Any `.ts` tool declared with `requireApproval: true` flips the
agent into suspend-on-call mode:

```ts
import { Agent, createTool, z } from "agent";
import { model, generateWithApproval } from "kit";

const deleteTool = createTool({
    id:           "delete-record",
    description:  "Delete a record — requires human approval",
    inputSchema:  z.object({ id: z.string() }),
    outputSchema: z.object({ deleted: z.boolean() }),
    requireApproval: true,
    execute: async ({ id }) => ({ deleted: true }),
});

const agent = new Agent({
    name:         "hitl-agent",
    model:        model("openai", "gpt-4o-mini"),
    instructions: "Use delete-record when asked to delete. Don't confirm.",
    tools:        { "delete-record": deleteTool },
    maxSteps:     3,
});

const result = await generateWithApproval(agent, "Delete record xyz-789", {
    approvalTopic: "approvals.pending", // bus topic approvers watch
    timeout:       10000,               // ms before auto-decline
});
// result.text: the agent's answer after the tool approved and ran.
```

Working fixture:
[`fixtures/ts/agent/hitl/bus-approval/`](../../fixtures/ts/agent/hitl/bus-approval/).

### Flow

1. Agent calls a tool with `requireApproval: true`.
2. `agent.generate` returns `finishReason: "suspended"` with a
   `runId` + `suspendPayload`.
3. `generateWithApproval` publishes an approval request on
   `approvalTopic` with `replyTo` metadata.
4. The Go bridge (`__go_brainkit_await_approval` in
   `internal/engine/bridges_approval.go`) subscribes to `replyTo`
   *before* publishing and waits on a Go channel with
   `context.WithTimeout`.
5. The approver replies on the correlated subject with a JSON body.
6. The bridge resolves the JS Promise with that response.
7. `generateWithApproval` calls `agent.approveToolCallGenerate` or
   `agent.declineToolCallGenerate` and returns the final result.

Steps 3–6 run entirely in Go — no JS `setTimeout`, no closure
captured across the wait. `context.WithTimeout`, a `select` on
channels, and `defer unsub()` handle cancellation and cleanup.

### Approval request payload

```json
{
    "runId":      "abc-123",
    "toolCallId": "call-456",
    "toolName":   "delete-record",
    "args":       {"id": "xyz-789"}
}
```

Published to `approvalTopic` with a correlated `replyTo`.

### Approval response

Reply with either shape:

```json
{"approved": true}
{"approved": false, "reason": "policy: requires ticket"}
```

Only an explicit `approved: false` declines. Any other value — or
omitting the field — is treated as approve.

### Approver in Go

```go
unsub, err := sdk.SubscribeTo[json.RawMessage](kit, ctx, "approvals.pending",
    func(payload json.RawMessage, msg sdk.Message) {
        var req struct {
            RunID      string          `json:"runId"`
            ToolCallID string          `json:"toolCallId"`
            ToolName   string          `json:"toolName"`
            Args       json.RawMessage `json:"args"`
        }
        _ = json.Unmarshal(payload, &req)

        approved := req.ToolName != "drop-database"
        _ = sdk.Reply(kit, ctx, msg, map[string]bool{"approved": approved})
    })
defer unsub()
```

Because the bridge owns the `replyTo` subject, a call to
`sdk.Reply(kit, ctx, msg, ...)` routes straight back to the waiting
Go select — no extra plumbing.

### Approver in `.ts`

```ts
bus.subscribe("approvals.pending", (msg) => {
    const req = msg.payload;
    console.log(`approval: ${req.toolName}(${JSON.stringify(req.args)})`);
    msg.reply({ approved: true });
});
```

### Timeout

If no reply arrives within `timeout` ms:

1. Go's `context.WithTimeout` expires.
2. The bridge resolves with `{"approved":false,"reason":"timeout"}`.
3. JS calls `agent.declineToolCallGenerate`.
4. The agent returns a decline result — no tool execution.

### Multiple pending approvals

Every call generates its own UUID-derived `replyTo`, so concurrent
`generateWithApproval` invocations don't interfere. The bus bridge
holds one subscription per call; `defer unsub()` tears it down
whether the call approves, declines, or times out.

## Workflow HITL — `suspend()` / `workflow.resume`

Workflows pause on any step that calls `await suspend(...)`. A
resume message picks them back up. Because workflows persist their
state through `storage("workflows")`, suspended runs survive a Kit
restart.

### Suspending step

```ts
import { createStep, createWorkflow, z } from "agent";
import { kit } from "kit";

const reviewStep = createStep({
    id:            "review",
    inputSchema:   z.object({ documentId: z.string() }),
    suspendSchema: z.object({ reason: z.string(), documentId: z.string() }),
    resumeSchema:  z.object({ approved: z.boolean(), reviewer: z.string() }),
    outputSchema:  z.object({ status: z.string(), reviewedBy: z.string() }),
    execute: async ({ inputData, resumeData, suspend }) => {
        if (!resumeData) {
            // Notify listeners that a document needs review.
            bus.emit("approvals.needed", {
                workflow:   "doc-review",
                documentId: inputData.documentId,
            });
            return await suspend({
                reason:     "Document needs review",
                documentId: inputData.documentId,
            });
        }
        return {
            status:     resumeData.approved ? "approved" : "rejected",
            reviewedBy: resumeData.reviewer,
        };
    },
});

const wf = createWorkflow({
    id:           "doc-review",
    inputSchema:  z.object({ documentId: z.string() }),
    outputSchema: z.object({ status: z.string(), reviewedBy: z.string() }),
}).then(reviewStep).commit();

kit.register("workflow", "doc-review", wf);
```

### Resuming from Go

```go
resp, err := brainkit.CallWorkflowResume(kit, ctx, sdk.WorkflowResumeMsg{
    Name:       "doc-review",
    RunID:      runID,
    Step:       "review",
    ResumeData: json.RawMessage(`{"approved":true,"reviewer":"alice@corp.com"}`),
}, brainkit.WithCallTimeout(10*time.Second))
// resp.Status: "success" | "failed" — full step tree in resp.Steps.
```

`WorkflowResumeMsg` (in `sdk/workflow_messages.go`):

```go
type WorkflowResumeMsg struct {
    Name       string          `json:"name"`
    RunID      string          `json:"runId"`
    Step       string          `json:"step,omitempty"`
    ResumeData json.RawMessage `json:"resumeData,omitempty"`
}
// BusTopic() == "workflow.resume"
```

### Resuming from `.ts`

```ts
await bus.call("workflow.resume", {
    name:       "doc-review",
    runId:      runId,
    step:       "review",
    resumeData: { approved: true, reviewer: "alice@corp.com" },
});
```

Working example: [`examples/workflows/`](../../examples/workflows/).

## Agent HITL vs Workflow HITL

| | Agent HITL | Workflow HITL |
|---|---|---|
| Trigger | Tool with `requireApproval: true` | Step calls `await suspend(...)` |
| Host call | `generateWithApproval(agent, prompt, opts)` | `agent.generate` / `workflow.start` finishes suspended |
| Resume call | Handled internally by the Go bridge | `brainkit.CallWorkflowResume` / `workflow.resume` |
| Bus lifecycle | Bridge subscribes + times out in Go | Author emits notifications; resume is pulled, not pushed |
| Resume payload | `{ approved: bool, reason?: string }` | Any shape matching `resumeSchema` |
| Timeout | `timeout` option; `context.WithTimeout` in Go | None — stays suspended until resumed or cancelled |
| Persistence | In-memory — expires with the process | Snapshot persisted to `storage("workflows")` |
| Per-run isolation | One UUID `replyTo` per call | Run ID + step ID uniquely identify the suspend |

Pick agent HITL for per-tool policy gates, pick workflow HITL for
multi-step business processes with long review cycles.

## Harness Module — WIP

`modules/harness` wraps an Agent + Memory + Modes config into a
session-style surface for building IDE-, chat-, or agent-plane
clients. Status: **WIP** — only the pieces below are frozen. See
[`examples/harness-lite/`](../../examples/harness-lite/) for a
minimal driver that exercises the frozen surface.

### Frozen — safe to depend on

`modules/harness.Instance`:

```go
type Instance interface {
    SendMessage(content string, opts ...SendOption) error
    Abort() error
    Steer(content string, opts ...SendOption) error
    FollowUp(content string, opts ...SendOption) error
    Subscribe(fn func(Event)) func()
    CurrentThread() string
    CurrentMode() string
    Close() error
}
```

Six event types (`modules/harness/instance.go`):

```go
const (
    EvAgentStart   EventType = "agent_start"
    EvAgentEnd     EventType = "agent_end"
    EvMessageDelta EventType = "message_update"
    EvToolStart    EventType = "tool_start"
    EvToolEnd      EventType = "tool_end"
    EvError        EventType = "error"
)
```

Every other event type a harness may emit is internal — treat the
raw `EventType` string as opaque and ignore events outside this
set.

### Not frozen — may move without deprecation

- `HarnessConfig` field layout
- `DisplayState` and display-related events
- Subagent wiring
- Observational memory hooks
- Module-level knobs beyond `harness.Config{ Harness: ... }`

### Wiring the frozen surface

```go
import "github.com/brainlet/brainkit/modules/harness"

mod := harness.NewModule(harness.Config{
    Harness: harness.HarnessConfig{
        ID: "my-harness",
        Modes: []harness.ModeConfig{{
            ID:        "build",
            Name:      "Build",
            Default:   true,
            AgentName: "my-agent",
        }},
        Permissions: harness.DefaultPermissions(),
    },
})

kit, _ := brainkit.New(brainkit.Config{
    Namespace: "harness-demo",
    Transport: brainkit.Memory(),
    FSRoot:    "/tmp/harness",
    Modules:   []brainkit.Module{mod},
})

inst := mod.Instance()
unsub := inst.Subscribe(func(ev harness.Event) {
    switch ev.Type {
    case harness.EvAgentStart, harness.EvAgentEnd,
         harness.EvMessageDelta,
         harness.EvToolStart, harness.EvToolEnd,
         harness.EvError:
        // Frozen — safe to match on.
    default:
        // Non-frozen internal event; ignore or log.
    }
})
defer unsub()

_ = inst.SendMessage("hello world")
```

`Instance()` returns nil when the Kit has no JS runtime, or when
the JS-side harness shim isn't wired in the current build. The
`harness-lite` example shows the expected fallback — detect the
boot error, print the frozen contract, and exit cleanly.

## Testing

The shipped HITL path is exercised end-to-end by
`fixtures/ts/agent/hitl/bus-approval/`:

```bash
go test ./test/fixtures/ -run 'TestTSFixturesE2E/agent/hitl/bus-approval' -v
```

Workflow suspend/resume is covered by the workflow storage and
commands suites under `test/suite/workflows/`.

## Summary

| Need | Surface | Stability |
|---|---|---|
| Approve / decline a single agent tool call | `generateWithApproval` + bus approver | Stable |
| Long-running multi-step process with human step | Workflow `suspend()` + `CallWorkflowResume` | Stable |
| Session wrapper for agent + modes + memory | `modules/harness` `Instance` | Frozen surface only — module is WIP |
