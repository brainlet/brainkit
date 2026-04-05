# Security Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests use secRBACKernel() (4 standard roles, observer default) and secReplyTokenKernel() (3 roles, service default) helpers. Many tests create fresh kernels with suite.Full(t) for isolation. Deploy source names include `-sec` suffix. Helper secSendAndReceive publishes a typed message and waits for raw response. Many tests are probe-style: they log findings rather than hard-fail, documenting known attack surface behaviors.

## Adding a test

1. Add function to the right .go file (sandbox.go for escape, data_leakage.go for leaks, bus_forgery.go for bus attacks, cross_deploy.go for cross-deployment, internal_exploit.go for runtime exploits, rbac_escape.go for privilege escalation, reply_token.go for token security, timing.go for races, secrets.go for exfiltration, gateway.go for HTTP, state.go for corruption, persistence.go for store attacks, libsql_validation.go for LibSQL)
2. Register in run.go
3. Update TEST_MAP.md
