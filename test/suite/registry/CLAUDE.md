# Registry Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests use registryEnv() helper for kernels with pre-configured providers/storages/vectors. Packages client tests spin up httptest.Server instances. Deploy source names include `-reg-adv` suffix. Storage runtime tests use env.Kernel for shared state tests and fresh kernels for isolation tests.

## Adding a test

1. Add function to the right .go file (providers.go for Go/JS registry, packages_client.go for HTTP client, storage_runtime.go for storage lifecycle, input_abuse.go for malformed inputs)
2. Register in run.go
3. Update TEST_MAP.md
