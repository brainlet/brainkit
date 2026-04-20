# modules/workflow — stable

Declarative multi-step agent workflows. A workflow is a directed
graph of bus-driven steps; the module exposes `workflow.*` bus
commands to start / cancel / inspect runs and persists state
through the Kit's store.

## Usage

```go
import (
    "github.com/brainlet/brainkit"
    "github.com/brainlet/brainkit/modules/workflow"
)

brainkit.New(brainkit.Config{
    Store: store,
    Modules: []brainkit.Module{
        workflow.New(),
    },
})
```

## Bus commands

- `workflow.start` — kick off a workflow run (ID + inputs).
- `workflow.startAsync` — start a run and emit completion on `workflow.completed.<runId>`.
- `workflow.status` — inspect run state + step history.
- `workflow.resume` — resume a suspended run, optionally at a named step.
- `workflow.cancel` — abort an in-flight or suspended run.
- `workflow.list` — enumerate registered workflows.
- `workflow.runs` — list persisted runs for a workflow.
- `workflow.restart` — restart an active run from persisted state.

## Runtime surface

Deployed `.ts` packages declare workflows via `kit.register("workflow", ...)`.
The stable control surface is the `workflow.*` command set above; there is
no public `workflow.advance` command. See `examples/workflows`,
`examples/hitl-workflow`, and `test/suite/workflows/` for end-to-end shapes.
