# Tracing Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests use tracingEnv() helper that creates a fresh kernel with MemoryTraceStore and returns both. This allows direct inspection of the store after operations. The suite's WithTracing() is used for the standalone test entry point but tracingEnv() provides the store reference needed for assertions. An "echo" tool is pre-registered.

## Adding a test

1. Add function to spans.go
2. Register in run.go
3. Update TEST_MAP.md
