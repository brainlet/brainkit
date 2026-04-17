# Suite Tests

Read `../CLAUDE.md` for the full conventions. This file covers suite-specific rules.

Pure test logic. No containers, no transport setup. Every domain runs on memory by default. Campaigns re-run these same tests on real backends.

## Conventions — READ BEFORE EDITING

### File structure (mandatory for every domain)
```
suite/<domain>/
  run.go            ← REGULAR .go file. Exports Run(t, env). NOT _test.go.
  <topic>.go        ← test functions: func testXxx(t, env) — UNEXPORTED
  <domain>_test.go  ← ONLY creates env + calls Run(). Nothing else.
  CLAUDE.md         ← domain instructions, references TEST_MAP.md
  TEST_MAP.md       ← every test function documented
```

### Why run.go must NOT be _test.go
Campaign files (`test/campaigns/transport/nats_test.go`) import domain packages to call `Run()`. Go cannot import `_test.go` files across packages. If `Run()` is in a `_test.go`, campaigns break.

### Test function rules
- **Always unexported**: `func testXxx` not `func TestXxx`
- **Always takes env**: `func testXxx(t *testing.T, env *suite.TestEnv)`
- **Always registered in run.go**: unregistered = dead code that never runs
- **One concern per function**: don't test publish AND subscribe AND error handling in one function

### Deploy source name uniqueness
Deploy source names collide when shared kernel is used. Convention:
```
<descriptive-name>-<domain-suffix>.ts
```
Examples: `greeter-pers.ts`, `thrower-sec.ts`, `slow-handler-stress.ts`

### When to use env.Kernel (shared) vs suite.Full(t) (fresh)

**Shared kernel (env.Kernel)** — fast, reuses the env created by `_test.go`:
- Tests that deploy, call, teardown within the test
- Tests that don't assert global state

**Fresh kernel (suite.Full(t))** — slower, but isolated:
- Asserting exact ListDeployments/ListSchedules counts
- Subscribing to global events (bus.handler.failed, bus.permission.denied)
- Closing/restarting kernel (persistence tests)
- Needing specific tracing configs
- Tests where another test's deploy could interfere

### TestEnv options
```go
suite.Full(t)                                    // storage + vectors + AI + FS + tools
suite.Full(t, suite.WithPersistence())           // + SQLite store
suite.Full(t, suite.WithTracing())               // + trace store
suite.Full(t, suite.WithSecretKey("key"))        // + encrypted secrets
suite.Full(t, suite.WithTransport("nats"))       // specific transport (campaigns use this)
suite.Minimal(t)                                 // bare kernel, nothing
```

### After editing
1. `go build ./test/suite/<domain>/`
2. `go test ./test/suite/<domain>/ -count=1 -short`
3. Update TEST_MAP.md

## Adding a test
1. Add function to the right `.go` file (by topic)
2. Register in `run.go`
3. Update `TEST_MAP.md`

## Adding a domain
1. `mkdir test/suite/newdomain/`
2. Create `run.go`, `newdomain_test.go`, test files
3. Create `CLAUDE.md` + `TEST_MAP.md`
4. If transport-sensitive, add to all `test/campaigns/transport/*_test.go` files

## 20 domains
agents, bus, cli, cross, deploy, fs, gateway, health, mcp, packages, persistence, registry, scheduling, secrets, security, stress, tools, tracing, workflows
