Experiments are source-only sandboxes for runtime probes, lifecycle investigations, and isolated QuickJS or WASM exploration work.

## Scope

- each subdirectory under `experiments/` is intentionally self-contained
- `quickjs-*` directories are standalone Go modules for focused runtime and bridge exploration
- `lifecycle/` holds tagged tests and notes for multicontext and sandbox behavior

## Rules

- experiments are not part of the public library import surface
- experiments do not need to follow the same package splitting discipline as the library packages
- local compiled executables from experiment modules are build artifacts and must stay untracked
- experiment coverage stays out of the default verification path unless a task is explicitly about experiments

## Graduation

- if an experiment becomes reusable library functionality, move it into a normal package with documented boundaries
- if an experiment remains exploratory, keep it flat and local to its sandbox instead of over-organizing it
