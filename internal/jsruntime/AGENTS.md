# JS Runtime Notes

This subtree owns deployed JS/TS execution in SES Compartments and the JS-side
Brainkit bridge. It does not own npm package resolution.

## Deployment Profiles

- Default runtime profiles accept raw `.ts` source through the source preparer.
- Artifact-only profiles reject raw `.ts` and accept normalized JavaScript.
- `vendor_typescript` support is intentional and must not be removed.
- Package deployment can hand normalized JS artifacts to the runtime, but the
  package builder owns source graph bundling.

## Runtime Boundaries

- Keep Compartment endowments explicit.
- Preserve async behavior for `bus.call`, `bus.callStream`, subscriptions, and
  message handlers without blocking the shared caller path.
- Bridge errors crossing into JS should remain typed `BrainkitError` values
  where possible.
- Runtime-owned subscriptions, handlers, goroutines, timers, and resources need
  mount/unmount proof.

## Tests

Useful gates:

```sh
go test ./internal/jsruntime -count=1 -timeout=600s
go test ./internal/jsruntime -run 'TestPrepareDeployCode.*TypeScript|TestPrepareDeployCodeLeavesNormalizedJSAlone|TestPrepareDeployCodeRejectsRawTSWithoutSourcePreparer' -count=1 -timeout=600s
make jsbridge-lifecycle-check
```
