# Workflows Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests use wfDeploy() helper to deploy .ts that registers workflows via createWorkflow/createStep/kit.register. The wfPublishAndWait generic function publishes a typed bus message and waits for the typed response. Storage tests create their own kernels with explicit SQLite storage configs. Workflows use z (Zod) schemas for input/output validation.

## Adding a test

1. Add function to the right .go file (commands.go for happy/error paths, storage.go for persistence, concurrent.go for concurrency, developer.go for real-world patterns)
2. Register in run.go
3. Update TEST_MAP.md
