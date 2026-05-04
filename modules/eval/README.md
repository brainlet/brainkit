# modules/eval - beta

JS/TS evaluation command for a Kit with the embedded runtime mounted.

## Bus commands

- `kit.eval` - evaluate source in one of three modes:
  - `script`: deploy a temporary script and read `globalThis.__module_result`.
  - `ts`: evaluate TypeScript in the current runtime context.
  - `module`: evaluate an ES module with imports.

## Capabilities

Requires `jsruntime` and the `brainkit.core.eval_runtime` capability.

## Runtime resources

None. Evaluation runs through the mounted JS runtime's narrow eval capability;
this module owns only the `kit.eval` command handler and does not receive raw
runtime deploy/teardown access.

## Hot unmount

Unmounting unregisters `kit.eval`; unmounting `jsruntime` is refused while this
module remains mounted.
