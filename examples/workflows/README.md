# workflows

Declarative multi-step workflow. Wire `modules/workflow`, deploy
a `.ts` that registers a 3-step pipeline (research → draft →
review), run it, print each step's output.

## Run

```sh
go run ./examples/workflows
```

Expected output (abridged):

```
starting research-pipeline for topic="brainkit"…
run id:   <uuid>
status:   success
steps:
  {
    "research": { "output": { "notes": "researched: brainkit" } },
    "draft":    { "output": { "draft": "drafted from researched: brainkit" } },
    "review":   { "output": { "approved": true, "final": "… [reviewed]" } },
    …
  }
```

## Step graph

```
research (topic → notes)
   ↓
draft    (notes → draft)
   ↓
review   (draft → { approved, final })
```

Each step has `inputSchema` / `outputSchema` (Zod) that gate the
data shape at runtime. The workflow runner verifies the output
of one step matches the input of the next.

## What it shows

- `modules/workflow.New()` registers `workflow.start`,
  `workflow.status`, `workflow.cancel`, `workflow.list`,
  `workflow.resume`, `workflow.restart`, `workflow.runs`,
  `workflow.startAsync` bus commands.
- `brainkit.CallWorkflowStart(kit, ctx, {Name, InputData})`
  runs the workflow synchronously. Use `CallWorkflowStartAsync`
  for fire-and-forget runs + poll `workflow.status` by run ID.
- `createStep` / `createWorkflow` / `kit.register("workflow", ...)`
  on the `.ts` side build the graph and expose it by name.

## Persistence

Without a `KitStore`, workflow state is in-memory only. Pair with
`brainkit.NewSQLiteStore(path)` to survive restarts —
`workflow.restart` + `workflow.runs` rehydrate in-flight runs
from storage.

## Branching

`.then(step)` chains sequentially; `.branch([...])` forks in
parallel; `.map(fn)` transforms between steps. See the Mastra
workflow docs for the full combinator set.
