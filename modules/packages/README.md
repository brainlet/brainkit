# modules/packages - stable

Package deployment command surface for `.ts` packages. This module owns package
deploy/teardown/list/info commands and emits deployment lifecycle events.

## Bus commands

- `package.deploy` - deploy a package from a filesystem path or inline files.
- `package.teardown` - remove a deployed package.
- `package.list` - list deployed packages tracked by this module.
- `package.info` - inspect one package manifest/deployment.

## Events

- `kit.deployed`
- `kit.teardown.done`

## Capabilities

- Requires: `jsruntime`, `brainkit.core.artifact_deployer`.
- Uses when present: `brainkit.core.audit_recorder`,
  `brainkit.core.secret_store`, `brainkit.core.plugin_checker`, and
  `brainkit.core.runtime_id`.
- Provides: none.

## Runtime resources

Owns package command handlers and the package bundling pipeline under
`modules/packages/internal/deploy`. Deployed JS artifacts/resources are created
through the JS runtime artifact-deployer capability and are marked as
normalized JS artifacts before runtime handoff.

## Hot unmount

Unmounting unregisters `package.*` commands and drops deployer capability
references. Existing deployments are owned by the JS runtime lifecycle and are
torn down when `jsruntime` unmounts or the Kit closes.
