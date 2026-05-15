# modules/packages - stable

Package deployment command surface for `.ts` packages. This module owns package
deploy/teardown/list/info commands and emits deployment lifecycle events.
Caller-side helpers live in `modules/packages/client`, package source values in
`modules/packages/source`, and on-disk scaffolding in `modules/packages/scaffold`
so helper-only importers do not link the module/domain graph.

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

Owns package command handlers and delegates package preparation to an explicit
package builder. The standard builder is registered by
`modules/packages/bundlers/esbuild` and uses the package format helpers under
`modules/packages/internal/deploy`. Deployed JS artifacts/resources are created
through the JS runtime artifact-deployer capability and are marked as
normalized JS artifacts before runtime handoff.

## Resolver policy

The standard esbuild builder uses the `source-relative` resolver profile by
default. It is for package source deployment, not arbitrary npm application
bundling.

Allowed imports are:

- relative `.ts`, `.js`, `.mjs`, and `.json` files in the deployed package;
- package index files such as `./lib/index.ts`, `./lib/index.js`, and
  `./lib/index.json`;
- Brainkit-provided compartment endowments: `kit`, `ai`, `agent`, and
  `compiler`.

Any other bare import, for example `uuid`, is rejected with
`PACKAGE_RESOLVER_UNSUPPORTED_IMPORT`. The diagnostic records the requested
specifier, importer, source package, current profile (`source-relative`), the
allowed bare imports, and the future owner: an opt-in npm ecosystem resolver
profile. Do not treat esbuild's package resolver as that future profile.

An experimental `npm-preview` resolver profile exists for filesystem package
directories only. It is opt-in through `manifest.json`:

```json
{
  "name": "my-package",
  "entry": "index.ts",
  "resolver": "npm-preview"
}
```

or through caller-side source helpers:

```go
pkg, _ := packagesource.FromDir("my-package")
pkg = pkg.WithResolver(packagesource.ResolverNPMPreview)
```

`npm-preview` requires `package.json` and `pnpm-lock.yaml` in the package root,
runs `pnpm install --frozen-lockfile --ignore-scripts`, then bundles installed
bare npm imports into the normalized JS artifact. Inline packages and single
file deploys still use `source-relative` and reject `npm-preview`.

This profile is deliberately narrow. It can bundle npm dependencies that stay
inside Brainkit-owned runtime compatibility. Supported Node builtins are
resolved to jsbridge-backed stubs, for example `node:events`, `node:stream`,
`node:buffer`, `node:module`, `node:crypto`, and `node:fs` where Brainkit
already owns the behavior. Dynamic `import()` is lowered to the constrained
runtime `require` path so SES never sees package-manager-era dynamic import
syntax in normalized artifacts.

Unsupported Node APIs, native addons, workers, server listeners, and
browser/provider processes remain explicit Brainkit runtime/module boundaries.
A dependency that imports `node:vm`, `http2`, or a similar unsupported runtime
API must fail with structured resolver diagnostics until a real owner exists.
Arbitrary runtime `require()` also remains unsupported and is reported as a
typed dynamic-require boundary.

Triage package deploy failures this way:

- missing resolver profile: unsupported bare npm import from source deployment;
- unsupported Node runtime API: dependency imported or executed a Node API that
  jsbridge does not own yet;
- unsupported native/worker boundary: dependency needs native addons, workers,
  server listeners, or another explicit runtime boundary;
- unsupported dynamic require: dependency computes a package/module specifier at
  runtime instead of using a static import that the resolver can analyze;
- package patch required: a specific dependency version needs a declared,
  tested source/bundle patch.

## Hot unmount

Unmounting unregisters `package.*` commands and drops deployer capability
references. Existing deployments are owned by the JS runtime lifecycle and are
torn down when `jsruntime` unmounts or the Kit closes.
