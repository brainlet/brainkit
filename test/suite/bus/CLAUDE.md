# Bus Tests

Read `TEST_MAP.md` before editing any test in this directory.

The largest domain (119 tests, 19 files). Tests are organized by topic: publish/subscribe mechanics, async patterns, failure/retry, error contracts, surface consistency, transport compliance, and cross-feature interactions. Many tests create fresh kernels via `suite.Full(t, ...)` with specific configs (retry policies, rate limits, persistence, log handlers) to avoid state pollution.

Key conventions:
- Tests asserting global bus events (bus.handler.failed, bus.handler.exhausted) MUST use fresh kernels
- Tests with retry policies create their own kernels since these are kernel-level config
- Deploy source names include domain suffix to avoid collisions across campaigns

## Adding a test

1. Add function to the right .go file by topic
2. Register in run.go
3. Update TEST_MAP.md
