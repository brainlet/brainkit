# Ecosystem Fixture Notes

This category proves the boundary between current source package deployment and
future broad npm support.

## Purpose

- Keep representative Node ecosystem package behavior visible through checked
  canaries.
- Keep default/source package deployment honest: arbitrary bare npm imports are
  rejected unless a package explicitly opts into a named resolver profile.
- Keep `npm-preview` separate from source-relative fixture deploys. It belongs
  to `modules/packages` package-deploy tests because it requires a filesystem
  package root, `package.json`, and `pnpm-lock.yaml`.

## Negative Fixtures

`bare-npm-rejected` intentionally imports `uuid` as a bare npm package. It must
fail during deploy with `PACKAGE_RESOLVER_UNSUPPORTED_IMPORT`.

Because that import is intentionally unsupported by the current source-relative
profile, the fixture is excluded from the global TypeScript fixture type-check.
Do not "fix" the type-check by adding a global `uuid` path. That would hide the
resolver boundary.

## npm-preview Coverage

`npm-preview` is currently covered in
`modules/packages/bundlers/esbuild/*_test.go` and
`test/suite/packages`, including gated live npm registry checks:

```sh
BRAINKIT_TEST_NPM_PREVIEW=1 go test ./modules/packages/bundlers/esbuild -run TestBuilderNPMPreviewLiveInstallsAndBundlesUUID -count=1 -timeout=600s -v
BRAINKIT_TEST_MASTRA_BROWSER_NPM_PREVIEW=1 go test ./modules/packages/bundlers/esbuild -run TestBuilderNPMPreviewMastraBrowserProviderBoundaries -count=1 -timeout=900s -v
BRAINKIT_TEST_MASTRA_BROWSER_NPM_PREVIEW=1 go test ./test/suite/packages -run 'TestPackages/packages/npm_preview_stagehand_package_import' -count=1 -timeout=900s -v
```

The browser-provider canary installs real `@mastra/agent-browser`,
`@mastra/stagehand`, and `@mastra/browser-viewer` packages. The current state:

- `@mastra/agent-browser` still hits an explicit `node:vm`
  unsupported-node-api boundary;
- `@mastra/stagehand` bundles and import-evals through `package.deploy`;
- `@mastra/browser-viewer` still hits an explicit `http2`
  unsupported-node-api boundary.

Do not mark those providers supported from this alone. The Stagehand smoke only
proves package import/eval compatibility. Real provider support still requires
concrete package fixtures plus lifecycle owners for browser/CDP/subprocess
resources and live model/browser behavior.

## Validation

```sh
go test ./test/fixtures -run 'TestFixtures/ecosystem' -count=1 -timeout=600s
go test ./internal/embed/agent -run 'Test.*EcosystemCanary|Test.*Canary' -count=1 -timeout=600s
BRAINKIT_TEST_NPM_PREVIEW=1 go test ./modules/packages/bundlers/esbuild -run TestBuilderNPMPreviewLiveInstallsAndBundlesUUID -count=1 -timeout=600s -v
BRAINKIT_TEST_MASTRA_BROWSER_NPM_PREVIEW=1 go test ./modules/packages/bundlers/esbuild -run TestBuilderNPMPreviewMastraBrowserProviderBoundaries -count=1 -timeout=900s -v
BRAINKIT_TEST_MASTRA_BROWSER_NPM_PREVIEW=1 go test ./test/suite/packages -run 'TestPackages/packages/npm_preview_stagehand_package_import' -count=1 -timeout=900s -v
```
