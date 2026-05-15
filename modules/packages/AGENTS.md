# Packages Module Notes

This module owns package deployment commands and source package preparation.
It is intentionally not broad npm support.

## Current Resolver Profiles

The standard esbuild builder defaults to `source-relative`:

- relative `.ts`, `.js`, `.mjs`, and `.json` files in the deployed package are
  supported;
- package-local index files are supported;
- the only accepted bare imports are Brainkit endowments: `kit`, `ai`,
  `agent`, and `compiler`;
- any other bare import must fail with
  `PACKAGE_RESOLVER_UNSUPPORTED_IMPORT`.

The resolver diagnostic must include:

- requested specifier;
- importer and source package;
- profile: `source-relative`;
- allowed bare imports;
- suggested future owner: opt-in npm ecosystem resolver profile.

The experimental `npm-preview` profile is available for filesystem package
directories only:

- opt in with `"resolver": "npm-preview"` in `manifest.json`, or with
  `packagesource.FromDir(...).WithResolver(packagesource.ResolverNPMPreview)`;
- require `package.json` and `pnpm-lock.yaml` in the package root;
- run `pnpm install --frozen-lockfile --ignore-scripts`;
- bundle installed bare npm imports into the normalized JS artifact;
- reject inline packages and single-file deploys;
- resolve jsbridge-owned Node builtins to controlled Brainkit stubs;
- reject unsupported Node builtins with `BoundaryClass:
  unsupported-node-api`;
- lower dynamic `import()` syntax to the constrained runtime `require` path;
- reject arbitrary runtime `require()` as a typed dynamic-require boundary.

`npm-preview` is not full Mastra/browser provider support. It is the first
resolver slice needed before concrete provider packages such as
`@mastra/agent-browser` can be revisited with real lifecycle owners.

## Future Resolver Product

Do not silently add arbitrary npm resolution to `source-relative`. Future
resolver work still needs isolated install/cache policy, resolver traces,
package patch ownership, native/worker/server rejection, lifecycle cleanup, and
validation.

## Tests

Useful gates:

```sh
go test ./modules/packages ./modules/packages/bundlers/esbuild ./modules/packages/source -run 'Test.*Bundle|Test.*PackageDeploy|Test.*TypeScript|Test.*BareImport|Test.*Resolver|TestBuilder' -count=1 -timeout=600s
BRAINKIT_TEST_NPM_PREVIEW=1 go test ./modules/packages/bundlers/esbuild -run TestBuilderNPMPreviewLiveInstallsAndBundlesUUID -count=1 -timeout=600s -v
BRAINKIT_TEST_MASTRA_BROWSER_NPM_PREVIEW=1 go test ./modules/packages/bundlers/esbuild -run TestBuilderNPMPreviewMastraBrowserProviderBoundaries -count=1 -timeout=900s -v
go test ./test/fixtures -run 'TestFixtures/ecosystem' -count=1 -timeout=600s
```
