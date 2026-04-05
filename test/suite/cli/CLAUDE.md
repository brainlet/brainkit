# CLI Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests use `cmd.NewRootCmd()` to invoke CLI commands programmatically via Cobra. Full E2E tests (testFullWorkflow, testSendWithAsyncHandler) start a real brainkit instance and require `!testing.Short()`. Bus command tests use the standard suite env pattern with `publishAndWait` generic helper.

## Adding a test

1. Add function to cobra.go (CLI commands) or commands.go (bus-level commands)
2. Register in run.go
3. Update TEST_MAP.md
