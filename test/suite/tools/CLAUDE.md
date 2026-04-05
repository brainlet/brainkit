# Tools Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests use env.Kernel which has pre-registered "echo" and "add" tools from suite.Full. The E2E pipeline test deploys .ts that registers a tool at runtime. Deploy source names include `-adv` suffix. Tests use SDK typed message publishing (sdk.Publish + sdk.SubscribeTo).

## Adding a test

1. Add function to the right .go file (registry.go for list/resolve/call, input_abuse.go for malformed inputs, e2e.go for pipeline, backend_advanced.go for transport)
2. Register in run.go
3. Update TEST_MAP.md
