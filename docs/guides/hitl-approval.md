# HITL Approval

Human-in-the-Loop (HITL) approval lets an agent suspend on a tool call and wait for external approval before executing it. The approval request is published to the bus — any surface (Go, .ts, plugin, gateway, Telegram bot) can approve or decline.

## The Flow

```
1. Agent calls tool with requireApproval: true
2. agent.generate() returns with finishReason: "suspended"
3. generateWithApproval publishes approval request to bus (Go bridge)
4. Go bridge subscribes to replyTo and waits with context.WithTimeout
5. Approver receives request, calls msg.reply({ approved: true/false })
6. Go bridge returns response to JS
7. JS calls agent.approveToolCallGenerate() or agent.declineToolCallGenerate()
8. Agent resumes execution (or declines and returns)
```

Steps 3-6 happen entirely in Go — no JS closures, no setTimeout, no GC risk during the wait. This was moved from JS to Go for reliability (see the codebase assessment).

## From .ts — generateWithApproval

```typescript
// fixtures/ts/agent/hitl-bus-approval/index.ts
const deleteTool = createTool({
    id: "delete-record",
    description: "Delete a record — requires human approval",
    inputSchema: z.object({ id: z.string() }),
    outputSchema: z.object({ deleted: z.boolean() }),
    requireApproval: true,
    execute: async ({ id }) => ({ deleted: true }),
});

const agent = new Agent({
    name: "hitl-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Always use delete-record when asked to delete. Don't ask for confirmation.",
    tools: { "delete-record": deleteTool },
    maxSteps: 3,
});

const result = await generateWithApproval(agent, "Delete record xyz-789", {
    approvalTopic: "approvals.pending",  // where to publish the request
    timeout: 10000,                      // ms before auto-decline
});
// result.text — agent's response after approval
```

### What generateWithApproval does

```typescript
// kit/runtime/kit_runtime.js — simplified
async function generateWithApproval(agent, prompt, options) {
    // Phase 1: agent.generate with requireToolApproval — may suspend
    var result = await agent.generate(prompt, { ...agentOptions, requireToolApproval: true });

    if (result.finishReason !== "suspended" || !result.runId) {
        return result; // Tool wasn't called or no approval needed
    }

    // Phase 2: Go bridge handles bus lifecycle
    var approvalPayload = JSON.stringify({
        runId: result.runId,
        toolCallId: result.suspendPayload?.toolCallId,
        toolName: result.suspendPayload?.toolName,
        args: result.suspendPayload?.args,
    });

    var responseJSON = await __go_brainkit_await_approval(
        options.approvalTopic,
        approvalPayload,
        options.timeout || 30000,
    );
    var response = JSON.parse(responseJSON);

    // Phase 3: resume agent
    if (response.approved !== false) {
        return await agent.approveToolCallGenerate({
            runId: result.runId,
            toolCallId: result.suspendPayload?.toolCallId,
        });
    } else {
        return await agent.declineToolCallGenerate({
            runId: result.runId,
            toolCallId: result.suspendPayload?.toolCallId,
        });
    }
}
```

## The Go Bridge

`__go_brainkit_await_approval(topic, payload, timeoutMs)` in `kit/bridges.go`:

1. Generates correlationID + replyTo
2. Subscribes to replyTo BEFORE publishing (no race)
3. Publishes approval request with replyTo metadata
4. Waits with `context.WithTimeout` + `select` on response channel
5. On response: resolves Promise with response JSON
6. On timeout: resolves with `{"approved":false,"reason":"timeout"}`
7. Cleanup via `defer unsub()` — guaranteed even on panic

The bus lifecycle is entirely in Go: `context.WithTimeout` (reliable, not GC-dependent), `select` on channels (no closure risk), `defer` cleanup (no race between timeout and response).

## Writing an Approver in Go

```go
// Pattern from test/fixtures/ts_test.go — auto-approver
cancel, err := sdk.SubscribeTo[json.RawMessage](rt, ctx, "approvals.pending",
    func(payload json.RawMessage, msg sdk.Message) {
        // payload: {"runId":"...", "toolCallId":"...", "toolName":"delete-record", "args":{"id":"xyz-789"}}

        var request struct {
            RunID      string `json:"runId"`
            ToolCallID string `json:"toolCallId"`
            ToolName   string `json:"toolName"`
            Args       any    `json:"args"`
        }
        json.Unmarshal(payload, &request)

        // Decision logic
        approved := request.ToolName != "drop-database" // approve everything except drops

        // Reply — this unblocks the Go bridge's select
        sdk.Reply(rt, ctx, msg, map[string]bool{"approved": approved})
    })
defer cancel()
```

## Writing an Approver in .ts

```typescript
bus.subscribe("approvals.pending", (msg) => {
    const request = msg.payload;
    console.log(`Approval requested: ${request.toolName}(${JSON.stringify(request.args)})`);

    // Auto-approve after inspection
    msg.reply({ approved: true });
});
```

## Timeout Behavior

If no approval arrives within the timeout:

1. The Go bridge's `context.WithTimeout` expires
2. Bridge resolves with `{"approved":false,"reason":"timeout"}`
3. JS calls `agent.declineToolCallGenerate()`
4. Agent returns with the decline result

The timeout is a `context.WithTimeout` in Go — not a JS `setTimeout`. It's not affected by QuickJS GC pressure or reentrant Await loop interactions.

## Approval Request Payload

Published to the `approvalTopic`:

```json
{
    "runId": "abc-123",
    "toolCallId": "call-456",
    "toolName": "delete-record",
    "args": {"id": "xyz-789"}
}
```

The approver sees exactly what tool is being called and with what arguments. It can make a decision based on the tool name, the arguments, external policies, or human input.

## Approval Response

Reply with:

```json
{"approved": true}   // approve — agent executes the tool
{"approved": false}  // decline — agent returns without executing
```

Any truthy value for `approved` (including omitting the field) is treated as approved. Only explicit `approved: false` declines.

## Multiple Pending Approvals

Each `generateWithApproval` call creates its own replyTo topic (UUID-based). Multiple agents can have pending approvals simultaneously without interference. The Go bridge waits on its specific replyTo — no global state, no race conditions.

## Testing

The HITL fixture (`fixtures/ts/agent/hitl-bus-approval/index.ts`) tests the full flow:

1. Deploys a .ts service with a `requireApproval: true` tool
2. Go test sets up an auto-approver on "test.approvals"
3. `generateWithApproval` suspends → Go bridge publishes → auto-approver replies → agent resumes
4. Verifies `hasText: true` and `approved: true` in output

```bash
go test ./test/fixtures/ -run 'TestTSFixturesE2E/agent/hitl-bus-approval' -v
```

## Workflow-Level HITL (Suspend/Resume)

Separate from agent HITL, Mastra workflows support suspend/resume for human-in-the-loop patterns. A workflow step calls `await suspend()` to pause execution, and an external caller sends `workflow.resume` via the bus to continue.

### From .ts — workflow with suspend

```typescript
const reviewStep = createStep({
    id: "review",
    inputSchema: z.object({ documentId: z.string() }),
    resumeSchema: z.object({ approved: z.boolean(), reviewer: z.string() }),
    suspendSchema: z.object({ reason: z.string(), documentId: z.string() }),
    outputSchema: z.object({ status: z.string(), reviewedBy: z.string() }),
    execute: async ({ inputData, resumeData, suspend }) => {
        if (!resumeData) {
            // Notify external callers that approval is needed
            bus.emit("approvals.needed", {
                workflowName: "doc-review",
                documentId: inputData.documentId,
            });
            return await suspend({
                reason: "Document needs review",
                documentId: inputData.documentId,
            });
        }
        return {
            status: resumeData.approved ? "approved" : "rejected",
            reviewedBy: resumeData.reviewer,
        };
    },
});

const wf = createWorkflow({
    id: "doc-review",
    inputSchema: z.object({ documentId: z.string() }),
    outputSchema: z.object({ status: z.string(), reviewedBy: z.string() }),
}).then(reviewStep).commit();
kit.register("workflow", "doc-review", wf);
```

### Resuming from Go

```go
sdk.Publish(k, ctx, sdk.WorkflowResumeMsg{
    Name:       "doc-review",
    RunID:      runId,
    Step:       "review",
    ResumeData: json.RawMessage(`{"approved": true, "reviewer": "alice@corp.com"}`),
})
```

### Resuming from .ts

```typescript
bus.publish("workflow.resume", {
    name: "doc-review",
    runId: runId,
    step: "review",
    resumeData: { approved: true, reviewer: "alice@corp.com" },
});
```

### Key Differences from Agent HITL

| | Agent HITL | Workflow HITL |
|---|---|---|
| Mechanism | `generateWithApproval` | `suspend()` + `workflow.resume` |
| What suspends | A single tool call | A workflow step |
| Resume data | `{approved: bool}` | Any shape (per `resumeSchema`) |
| Bus lifecycle | Go bridge handles publish/subscribe/timeout | Workflow author controls notification |
| Timeout | Built-in (Go context.WithTimeout) | No built-in timeout — stays suspended until resumed |
| Persistence | In-memory only | Snapshot persisted to storage — survives Kit restart |
