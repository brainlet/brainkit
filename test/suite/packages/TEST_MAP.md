# Packages Test Map

**Purpose:** Verifies multi-file package deployment from manifest.json, including service wiring, listing, teardown, secret dependency checking, and gated npm-preview package-provider canaries
**Tests:** 7 functions across 1 file
**Entry point:** `packages_test.go` → `Run(t, env)`
**Campaigns:** none (packages is standalone)

## Files

### deploy.go — Package deployment lifecycle

| Function | Purpose |
|----------|---------|
| testMultiFileProject | Creates a temp dir with manifest.json (name=test-pkg, one greeter service), config.ts exporting a constant, and greeter.ts importing it; deploys via PackageDeployMsg, then sends a message to the greeter service and verifies the response uses the imported prefix |
| testListAndTeardown | Deploys a package, lists deployed packages via PackageListDeployedMsg asserting 1 result, tears down via PackageTeardownMsg, lists again asserting 0 results |
| testSecretDependencyCheck | Deploys a package with `requires.secrets: ["MY_REQUIRED_SECRET"]` without the secret set, asserts the error mentions the missing secret; then sets the secret and retries, asserting deploy succeeds |
| testInlineFilesRedeployPicksUpNewCode | Deploys an inline package twice under the same name and asserts calls hit the redeployed code |
| testTopicCollision | Deploys a package with duplicate topic handlers and asserts deployment fails cleanly |
| testNPMPreviewStagehandPackageImport | Gated by `BRAINKIT_TEST_MASTRA_BROWSER_NPM_PREVIEW=1`; creates a real filesystem package with `@mastra/stagehand`, generates a real `pnpm-lock.yaml`, deploys through `resolver: "npm-preview"`, and asserts `StagehandBrowser` imports/evals in Brainkit runtime |
| testNPMPreviewStagehandLocalBrowserLifecycle | Gated by `BRAINKIT_TEST_MASTRA_BROWSER_NPM_PREVIEW=1` and `BRAINKIT_TEST_LIVE_AI=1`; mounts `modules/browser`, launches a real local Chrome/Chromium CDP session, deploys a real `@mastra/stagehand` npm-preview package, connects `StagehandBrowser` to the Brainkit-owned CDP endpoint, exercises navigate/tabs/screenshot/observe/extract/act/close with live OpenAI credentials, and asserts no browser sessions leak |

## Cross-references

- **Campaigns:** none
- **Related domains:** deploy (single-file deploy), secrets (secret dependency)
- **Fixtures:** none
