# Stress Tests

Read `TEST_MAP.md` before editing any test in this directory.

Most tests check testing.Short() and skip in short mode. Tests that close kernels or need specific config create their own via brainkit.NewKernel. Tests use env.Kernel for shared-state stress. Deploy source names include `-stress` suffix. The sendAndReceive helper publishes a typed message and waits for raw response. ConcurrentDo from testutil runs N goroutines.

## Adding a test

1. Add function to the right .go file (gc.go for cleanup, scaling.go for pool, concurrent.go for parallel ops, concurrency.go for race conditions, concurrency_stress.go for heavy load, exhaustion.go for resource attacks, e2e_stress.go for E2E)
2. Register in run.go
3. Update TEST_MAP.md
