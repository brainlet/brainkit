# Packages Test Map

**Purpose:** Verifies multi-file package deployment from manifest.json, including service wiring, listing, teardown, and secret dependency checking
**Tests:** 3 functions across 1 file
**Entry point:** `packages_test.go` → `Run(t, env)`
**Campaigns:** none (packages is standalone)

## Files

### deploy.go — Package deployment lifecycle

| Function | Purpose |
|----------|---------|
| testMultiFileProject | Creates a temp dir with manifest.json (name=test-pkg, one greeter service), config.ts exporting a constant, and greeter.ts importing it; deploys via PackageDeployMsg, then sends a message to the greeter service and verifies the response uses the imported prefix |
| testListAndTeardown | Deploys a package, lists deployed packages via PackageListDeployedMsg asserting 1 result, tears down via PackageTeardownMsg, lists again asserting 0 results |
| testSecretDependencyCheck | Deploys a package with `requires.secrets: ["MY_REQUIRED_SECRET"]` without the secret set, asserts the error mentions the missing secret; then sets the secret and retries, asserting deploy succeeds |

## Cross-references

- **Campaigns:** none
- **Related domains:** deploy (single-file deploy), secrets (secret dependency)
- **Fixtures:** none
