# modules/testing - beta

`.test.ts` runner command surface for a Kit with the embedded runtime mounted.
It deploys/evaluates test files through the same runtime capabilities used by
packages.

## Bus commands

- `test.run` - run a directory/pattern of TypeScript tests and return the JSON
  test result payload.

## Capabilities

Requires `jsruntime` and `brainkit.core.test_runtime`.

## Runtime resources

No long-lived resources. Test deployments/evaluations are run through the
explicit JS test runtime capability, and the per-command test runner/bundler
implementation lives under `modules/testing/internal/braintest`. Bundled test
files are handed to the runtime as normalized JS artifacts; raw inline/fixture
source remains limited to this dev/test capability.

## Hot unmount

Unmounting unregisters `test.run` and drops runtime capability references.
