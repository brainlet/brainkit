# Security Tests

Read `TEST_MAP.md` before editing any test in this directory.


## Adding a test

1. Add function to the right .go file (sandbox.go for escape, data_leakage.go for leaks, bus_forgery.go for bus attacks, cross_deploy.go for cross-deployment, internal_exploit.go for runtime exploits, reply_token.go for token security, timing.go for races, secrets.go for exfiltration, gateway.go for HTTP, state.go for corruption, persistence.go for store attacks, libsql_validation.go for LibSQL)
2. Register in run.go
3. Update TEST_MAP.md
