# Secrets Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests use secretsEnv() helper that creates a fresh kernel with persistence + secret key. Tests that check encrypted persistence create their own kernels with explicit store paths. Deploy source names include `-sec-adv` suffix. All secret names must be unique per test to avoid interference.

## Adding a test

1. Add function to the right .go file (crud.go for CRUD, matrix.go for adversarial matrix, input_abuse.go for malformed inputs, integration.go for rotation, backend_advanced.go for transport)
2. Register in run.go
3. Update TEST_MAP.md
