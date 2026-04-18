# hitl-tool-approval

Synchronous human-in-the-loop — an Agent pauses mid-generation
when it's about to call a tool flagged `requireApproval: true`,
publishes the pending call to a bus topic, and resumes once the
Go side replies with an approve/decline decision.

brainkit ships a `generateWithApproval` helper that wires this
whole handshake through the bus, so the consumer only touches
three surfaces:

1. **Tool**: `createTool({ id, …, requireApproval: true, execute })`
2. **Agent call**: `generateWithApproval(agent, prompt, { approvalTopic, timeout })`
3. **Go side**: `kit.SubscribeRaw(ctx, "demo.approvals", fn)` — fn receives the pending call and replies with `{approved: true/false}`

Replaces the older `streamVNext` API (removed in Mastra 1.22) —
the approval flow is now driven by `generate` internally, so the
suspend+resume is one atomic call from the `.ts` handler's POV.

## Three demo turns

```
[2/4] approve path — agent asked to delete record xyz-789
        approval request: tool=delete-record args=map[id:xyz-789] → decision=approve
        finishReason: stop
        reply: …

[3/4] decline path — same prompt, Go-side rejects
        approval request: tool=delete-record args=map[id:abc-123] → decision=decline
        finishReason: stop
        reply: …

[4/4] no-op path — prompt doesn't trigger the tool
        finishReason: stop
        reply: Hello, Admin! How can I assist you today?
```

Turn 1 and 2 both fire the approval request with the arg the
model inferred (`{id: "xyz-789"}` / `{id: "abc-123"}`). Turn 3
has no deletion in the prompt, so no approval is needed and the
agent skips straight to the reply.

## Run

```sh
OPENAI_API_KEY=sk-... go run ./examples/hitl-tool-approval
```

## How it works

```
           ┌──────────────────────────┐
           │ Agent.generate           │
           │  via generateWithApproval│
           └──────────┬───────────────┘
                      │ model emits a tool-call for delete-record
                      ▼
       ┌────────────────────────────────────┐
       │ generateWithApproval intercepts:    │
       │  - publish pending to approvalTopic │
       │  - wait for the reply envelope      │
       └──────────┬───────────────────────  │
                  │                         │
                  ▼                         │
        ┌──────────────────────┐            │
        │ Go SubscribeRaw       │            │
        │  decides (CLI / UI /  │            │
        │  slack bot / tests)   │            │
        │  → Reply({approved})  │            │
        └──────────┬───────────┘            │
                   │                         │
                   ▼                         │
        ┌─────────────────────────┐          │
        │ approved: run execute   │─────────►┘
        │ declined: skip + mark   │
        └─────────────────────────┘
                   │
                   ▼
           agent's final text
```

Under the hood the helper lives in
`internal/engine/runtime/approval.js` + `generateWithApproval`
endowment at `internal/engine/runtime/kit_runtime.js`. It turns
the raw
`agent.generate(..., {requireToolApproval: true})` + follow-up
`approveToolCallGenerate({runId, toolCallId})` dance into a
single awaitable — the handler doesn't need to keep run state
between its own bus topic calls.

## Tool-level vs call-level approval

| Knob | Location | Semantics |
|---|---|---|
| `requireApproval: true` on the tool | `createTool({…, requireApproval: true})` | This tool ALWAYS requires approval, regardless of how the agent is called |
| `requireToolApproval: true` on the call | `agent.generate(prompt, { requireToolApproval: true })` | For this one call, every tool the agent invokes requires approval |

The example picks the tool-level knob (most common — a specific
destructive action needs approval; safe actions don't).

## When to use this vs workflow suspend (`examples/hitl-workflow/`)

| | Tool approval (this example) | Workflow suspend |
|---|---|---|
| Scope | one tool call mid-agent-generate | one step mid-workflow |
| Granularity | token-stream fidelity — approval only triggers when the model picks the tool | deterministic — the step runs on every pass |
| Durability | in-memory — dies with the agent process | persisted to `Storage` when configured |
| Resume API | `generateWithApproval` (handles internally) | `run.resume({step, resumeData})` via `CallWorkflowResume` |

Tool approval is the right fit when a human needs to sign off on
a specific destructive action the LLM might pick. Workflow
suspend is the right fit when you want a durable deploy pipeline
or approval queue that outlives a single agent run.

## Extension ideas

- **Interactive CLI**: replace the `atomic.Pointer[string]` mode
  toggle with `bufio.Scanner` on stdin — print the pending call,
  wait for operator keystroke, reply.
- **Slack / Teams webhook**: the `demo.approvals` subscriber can
  post the pending call to a channel, collect a reply, then
  publish the decision. The deployed agent doesn't change.
- **Multi-user quorum**: route `demo.approvals` to N subscribers,
  collect M-of-N approvals before responding.
- **Programmatic policies**: inspect `args.id` before the human
  gets a chance — auto-decline clearly-bad deletions (e.g.
  production database IDs) from the Go side.
- **Audit trail**: pair with `modules/audit` — every approval
  request + decision lands in the audit store.

## See also

- `examples/hitl-workflow/` — the out-of-band counterpart:
  workflow steps suspend with a reason, resume from a different
  process.
- `docs/guides/hitl-approval.md` — the broader HITL guide;
  distinguishes the two patterns.
- `fixtures/ts/agent/hitl/bus-approval/` — the test-suite
  reference fixture the example cribs from.
