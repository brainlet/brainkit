# modules/jsruntime/artifact - beta

Artifact-only activation lease for the embedded JavaScript runtime. Mounting
this module enables the same runtime capabilities as `modules/jsruntime`, but
does not install a TypeScript source preparer.

The mounted module ID is still `jsruntime`, so modules that require
`jsruntime` work when this module is mounted explicitly. Raw `.ts` source
deploys are rejected with a configuration error; callers should deploy
normalized JavaScript through `brainkit.core.artifact_deployer`.

For the default runtime that accepts raw `.ts` source and transpiles it before
evaluation, use `modules/jsruntime`.

## Capabilities

- Requires: `brainkit.core.jsruntime_host`.
- Optional:
  - `brainkit.core.lifecycle_debug_registry`
- Provides:
  - `brainkit.core.enable_js_runtime`
  - `brainkit.core.has_js_runtime`
  - `brainkit.core.artifact_deployer`
  - `brainkit.core.eval_runtime`
  - `brainkit.core.test_runtime`
  - `brainkit.core.call_js`
  - `brainkit.core.harness_runtime`

## Runtime resources

- `runtime:jsruntime.heap` - the embedded JavaScript runtime activation lease.
