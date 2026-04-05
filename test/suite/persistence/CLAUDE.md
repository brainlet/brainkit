# Persistence Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests create their own kernels with SQLite stores (never use env.Kernel) because they need to close and reopen kernels to verify persistence. Each test uses t.TempDir() for isolated store paths. Deploy source names include `-persist` suffix to avoid collisions.

## Adding a test

1. Add function to the right .go file (store.go for deploy/metadata, schedule.go for schedules, backend_matrix.go for ported adversarial tests)
2. Register in run.go
3. Update TEST_MAP.md
