# modules/jsruntime - beta

Activation lease for the embedded JavaScript runtime. Mounting this module enables
the QuickJS/Ses runtime and provides the runtime capabilities consumed by eval,
packages, testing, workflow, harness, and JS-backed storage/vector paths.

## Runtime surface

Enables JavaScript deployment/evaluation, JS request dispatch, harness
execution, and JS runtime presence checks through explicit core capabilities.
Raw `.ts` source deploys are transpiled to JavaScript by this module before
runtime evaluation. Package/file-graph normalization is still owned by
package/test tooling before artifact handoff.

For a runtime profile that does not link the TypeScript source preparer, use
`modules/jsruntime/artifact` or `presets/standard/artifactruntime`; that
variant accepts normalized JavaScript artifacts and rejects raw `.ts` deploys.

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

## Hot unmount

Unmounting disables the attached JS runtime through the shared kernel path,
tears down JS deployments/resources, detaches the scoped JS tool evaluator
lease, and clears runtime-owned bridges.

When lifecycle debug is available, the module registers a scoped `jsruntime`
component reporting deployment, resource, bridge subscription, and lifecycle
phase counters.
