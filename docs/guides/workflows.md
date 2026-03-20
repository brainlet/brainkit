# Workflows

Workflows define multi-step processes with branching, suspension, and resumption. The bus handlers route through EvalTS to Mastra's Workflow API. Workflows are defined in `.ts` code using `createWorkflow` + `createStep` + `commit`.

---

## Bus Topics

| Topic | Payload | Response |
|-------|---------|----------|
| `workflows.run` | `{"name":"my-workflow","input":{...}}` | Run result (step outputs, status) |
| `workflows.resume` | `{"runId":"...","stepId":"...","data":{...}}` | Resumed run result |
| `workflows.cancel` | `{"runId":"..."}` | `{"ok":true}` |
| `workflows.status` | `{"runId":"..."}` | `{"status":"running","step":"step-2"}` |

---

## Defining a Workflow

Workflows are created in `.ts` code deployed to the Kit:

```typescript
import { createWorkflow, createStep, z } from "kit";

const fetchData = createStep({
    id: "fetch-data",
    inputSchema: z.object({ url: z.string() }),
    outputSchema: z.object({ body: z.string() }),
    execute: async ({ context }) => {
        const res = await fetch(context.inputData.url);
        return { body: await res.text() };
    },
});

const summarize = createStep({
    id: "summarize",
    inputSchema: z.object({ body: z.string() }),
    outputSchema: z.object({ summary: z.string() }),
    execute: async ({ context }) => {
        const result = await ai.generate("openai/gpt-4o-mini",
            `Summarize: ${context.inputData.body}`);
        return { summary: result.text };
    },
});

const workflow = createWorkflow({
    name: "fetch-and-summarize",
    triggerSchema: z.object({ url: z.string() }),
});

workflow.step(fetchData).then(summarize).commit();
```

---

## Running a Workflow

### From Go (via bus.AskSync)

```go
resp, err := bus.AskSync(kit.Bus, ctx, bus.Message{
    Topic: "workflows.run",
    Payload: json.RawMessage(`{
        "name": "fetch-and-summarize",
        "input": {"url": "https://example.com"}
    }`),
})
```

### From .ts

```typescript
import { bus } from "kit";

const result = await bus.ask("workflows.run", {
    name: "fetch-and-summarize",
    input: { url: "https://example.com" },
});
```

### From Go API directly

```go
result, err := kit.EvalTS(ctx, "run-workflow.ts", `
    var wf = globalThis.__kit_workflows["fetch-and-summarize"];
    var run = await createWorkflowRun(wf);
    var result = await run.start({ triggerData: { url: "https://example.com" } });
    return JSON.stringify(result);
`)
```

---

## Suspending and Resuming

Steps can suspend execution to wait for external input (human approval, webhook, etc.):

```typescript
const approvalStep = createStep({
    id: "approval",
    inputSchema: z.object({ request: z.string() }),
    outputSchema: z.object({ approved: z.boolean() }),
    execute: async ({ context, suspend }) => {
        await suspend({ request: context.inputData.request });
        // Execution resumes here after resume() is called
        return { approved: true };
    },
});
```

### Resume via bus

```go
resp, err := bus.AskSync(kit.Bus, ctx, bus.Message{
    Topic: "workflows.resume",
    Payload: json.RawMessage(`{
        "runId": "run-abc-123",
        "stepId": "approval",
        "data": {"approved": true}
    }`),
})
```

### Resume via Go API

```go
result, err := kit.ResumeWorkflow(ctx, "run-abc-123", "approval", `{"approved":true}`)
```

---

## Checking Status

```go
resp, err := bus.AskSync(kit.Bus, ctx, bus.Message{
    Topic:   "workflows.status",
    Payload: json.RawMessage(`{"runId":"run-abc-123"}`),
})
// resp.Payload: {"status":"running","step":"summarize"}
```

---

## Cancelling a Run

```go
bus.AskSync(kit.Bus, ctx, bus.Message{
    Topic:   "workflows.cancel",
    Payload: json.RawMessage(`{"runId":"run-abc-123"}`),
})
```

---

## From Plugins (via SDK)

```go
// Run
sdk.Ask[any](client, ctx, messages.WorkflowRunMsg{
    Name:  "fetch-and-summarize",
    Input: map[string]string{"url": "https://example.com"},
}, func(result any, err error) {
    fmt.Println("workflow result:", result)
})

// Resume
client.Ask(ctx, messages.WorkflowResumeMsg{
    RunID:  "run-abc-123",
    StepID: "approval",
    Data:   map[string]bool{"approved": true},
}, func(msg messages.Message) {})

// Check status
sdk.Ask[messages.WorkflowStatusResp](client, ctx,
    messages.WorkflowStatusMsg{RunID: "run-abc-123"},
    func(resp messages.WorkflowStatusResp, err error) {
        fmt.Printf("status: %s, step: %s\n", resp.Status, resp.Step)
    },
)
```

---

## Lifecycle

```
createWorkflow({name, triggerSchema})
  .step(step1)
  .then(step2)
  .commit()

workflows.run --> start({triggerData}) --> step1.execute --> step2.execute --> result
                                            |
                                            suspend() --> workflows.resume --> continue
```

Workflow definitions are stored in `globalThis.__kit_workflows` by name. Pending runs are tracked in `globalThis.__kit_pending_runs` by run ID.
