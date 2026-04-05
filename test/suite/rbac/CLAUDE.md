# RBAC Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests use helper functions newRestrictedKernel (custom restricted role) and newRBACKernel (all 4 standard roles) to create fresh kernels. Bridge tests use bridgeDeployAndCheck which deploys TS code with a role and reads the output() result via EvalTS. Deploy source names include `-rbac` suffix.

## Adding a test

1. Add function to the right .go file (enforcement.go for kernel-level, bridge.go for JS Compartment, matrix.go for permission matcher unit tests)
2. Register in run.go
3. Update TEST_MAP.md
