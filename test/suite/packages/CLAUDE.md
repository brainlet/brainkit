# Packages Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests create real temp directories with manifest.json + .ts files, deploy via PackageDeployMsg bus command, and verify service wiring via SendToService. Each test creates a fresh kernel with `suite.Full(t, suite.WithPersistence(), suite.WithSecretKey(...))`.

## Adding a test

1. Add function to deploy.go
2. Register in run.go
3. Update TEST_MAP.md
