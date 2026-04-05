# Deploy Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests cover the full .ts deployment lifecycle including edge cases, input abuse, state corruption recovery, and TS surface capabilities. Many tests use `env.Kernel.Deploy()` directly (not bus messages) for convenience. State corruption tests create SQLite stores, seed them with corrupted data, then verify kernel recovery on restart.

Key conventions:
- Deploy source names include "-edge", "-adv", "-deploy-adv" suffixes
- Tests checking exact deployment counts need fresh kernels (`suite.Full(t)`)
- State corruption tests create their own stores and kernels

## Adding a test

1. Add function to the right .go file by topic
2. Register in run.go
3. Update TEST_MAP.md
