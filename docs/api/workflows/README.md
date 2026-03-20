# Workflows API Reference

Run, resume, cancel, and check status of named workflows.

---

## Bus Topics

All workflow operations use Ask (request/response) over the bus.

### workflows.run

Start a named workflow with input data.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"name": string, "input": any}` |
| **Response** | Run result (workflow-defined) |

### workflows.resume

Resume a suspended workflow run, optionally at a specific step.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"runId": string, "stepId": string, "data": any}` |
| **Response** | Resume result (workflow-defined) |

`stepId` is optional. When omitted, resumes at the current suspended step.

### workflows.cancel

Cancel a running workflow.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"runId": string}` |
| **Response** | *(ack)* |

### workflows.status

Get the current status of a workflow run.

| Direction | Shape |
|-----------|-------|
| **Request** | `{"runId": string}` |
| **Response** | `{"status": string, "step": string}` |

---

## SDK Messages

Typed messages for bus interactions. All implement `BusMessage`.

```go
import "github.com/brainlet/brainkit/sdk/messages"
```

### Request Messages

| Message | Fields | BusTopic() |
|---------|--------|------------|
| `WorkflowRunMsg` | `Name string`, `Input any` | `"workflows.run"` |
| `WorkflowResumeMsg` | `RunID string`, `StepID string`, `Data any` | `"workflows.resume"` |
| `WorkflowCancelMsg` | `RunID string` | `"workflows.cancel"` |
| `WorkflowStatusMsg` | `RunID string` | `"workflows.status"` |

### Response Messages

| Message | Fields |
|---------|--------|
| `WorkflowStatusResp` | `Status string`, `Step string` |
