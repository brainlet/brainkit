# modules/control - stable

Runtime control plane for a live Kit. This is the bus-facing module that makes
module inspection and hot mount/unmount available to operators and remote tools.

## Bus commands

- `kit.set-draining` - toggle runtime drain mode.
- `kit.modules` - return mounted module manifests plus current preflight
  readiness for each mounted module.
- `kit.lifecycle` - return lifecycle/debug counters for the runtime and
  mounted components, including provider `closing` / `closed` state, active
  probe and provider runtime-operation counts, and transport ownership plus
  router/caller/transport `closing` / `closed` state.
- `kit.module.describe` - inspect a registered or mounted module and report
  whether its required modules and capabilities are currently satisfiable.
- `kit.module.mount` - hot-mount a registered module.
- `kit.module.unmount` - unmount a mounted module.
- `cluster.peers` - list visible runtime peers.

## Capabilities

- Requires: `brainkit.core.lifecycle_debug_snapshot`,
  `brainkit.core.module_lifecycle`, `brainkit.core.mounted_modules`,
  `brainkit.core.runtime_control`.
- Provides: none.

## Runtime resources

None. The module owns command handlers only; lifecycle state remains in Kit.

## Hot unmount

Unmounting unregisters the control commands. The module refuses to unmount
itself through its own command; direct `Kit.Unmount` still enforces dependency
checks.
