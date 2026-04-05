# Test Architecture

```
test/
  suite/        Tests logic. Each domain exports Run(t, env). Runs on memory (~30s).
  campaigns/    Tests infrastructure. Spins containers, calls suite.Run() on real backends.
  fixtures/     TS fixture runner. WalkDir discovery, path-based classification.
  bench/        Benchmarks. Same TestEnv pattern.
```

## Adding a new test

1. Find the right suite domain (`test/suite/<domain>/`)
2. Add your test function: `func testMyThing(t *testing.T, env *suite.TestEnv) { ... }`
3. Register it in that domain's `run.go` inside `Run()`
4. Run: `go test ./test/suite/<domain>/ -run TestX/domain/my_thing`

## Key rules

- Suite tests NEVER create containers. They use `env.Kernel` (memory transport).
- Tests needing specific configs create fresh kernels: `env := suite.Full(t, suite.WithRBAC(...))`
- Tests checking global state (ListSchedules, bus events) need fresh kernels to avoid pollution.
- Deploy source names must be unique — add domain suffix (`-rbac`, `-sec`, `-stress`).
- `Run(t, env)` is in a regular `.go` file (not `_test.go`) so campaigns can import it.

## Running tests

```bash
go test ./test/suite/...                              # all domains, memory (~30s)
go test ./test/suite/bus/                             # single domain
go test ./test/campaigns/transport/                   # all backends (needs Podman)
go test ./test/fixtures/ -run TestFixtures            # TS fixtures
go test -bench . ./test/bench/...                     # benchmarks
```
