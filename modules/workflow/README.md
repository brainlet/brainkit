# modules/workflow — beta

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
        workflow.New(workflow.Config{}),
    },
})
```

## Bus commands

- `workflow.start` — kick off a workflow run (ID + inputs).
- `workflow.cancel` — abort an in-flight run.
- `workflow.status` — inspect run state + step history.
- `workflow.list` — enumerate active runs.

## Runtime surface

Deployed `.ts` packages declare workflows via `kit.register("workflow", ...)`
and transition steps through `workflow.advance` messages. See the
fixtures under `test/fixtures/workflow/` for end-to-end shapes.
