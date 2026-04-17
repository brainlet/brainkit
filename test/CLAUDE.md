# Test Architecture

```
test/
  suite/        Test logic. Each domain exports Run(t, env). Runs on memory.
  campaigns/    Infrastructure. Spins containers, calls suite.Run() on real backends.
  fixtures/     TS fixture runner. WalkDir discovery, path-based classification.
  bench/        Benchmarks.
```

## Conventions — READ BEFORE EDITING

### No mocks, no fakes
Real tests only. Real kernels, real QuickJS, real bus. No mock objects, no fake data, no stub implementations. Tests that need AI use `env.RequireAI(t)` to skip when no key.

### Approach B pattern (mandatory)
Every domain exports `Run(t *testing.T, env *suite.TestEnv)` in a **regular .go file** (not `_test.go`). This is so campaigns can import it. The standalone `_test.go` only creates the env and calls `Run()`.

```
suite/<domain>/
  run.go            ← exports Run(t, env), registers all tests
  <topic>.go        ← test functions: func testXxx(t, env) — UNEXPORTED
  <domain>_test.go  ← entry: func TestDomain(t) { env := suite.Full(t); Run(t, env) }
```

### Test function signature
Always `func testXxx(t *testing.T, env *suite.TestEnv)`. Unexported. Takes env even if unused (write `_ *suite.TestEnv` if you create your own kernel).

### Registration is mandatory
Every test function MUST be registered in `run.go` inside `Run()`:
```go
t.Run("my_test_name", func(t *testing.T) { testMyTestName(t, env) })
```
Unregistered functions are dead code that never runs.

### Fresh kernels vs shared kernel
- **Use `env.Kernel`** (shared) for tests that don't pollute global state
- **Use `suite.Full(t)` (fresh)** when:
  - Asserting exact counts (ListDeployments, ListSchedules)
  - Subscribing to global events (bus.handler.failed, bus.permission.denied)
  - Closing/restarting the kernel
  - Needing specific tracing configs

### Deploy source name uniqueness
Deploy source names must be unique across all tests sharing a kernel. Add a domain suffix:
`-sec` for security, `-stress` for stress, `-adv` for adversarial
- Two tests deploying `"greeter.ts"` on the same kernel = collision = flaky failure

### File organization
- One file per concern (publish.go, failure.go, error_contract.go)
- Adversarial/edge-case tests go in separate files with clear names (input_abuse.go, state_corruption.go)
- Keep files under 300 lines. Split if growing.

### After editing any test
1. Verify: `go build ./test/suite/<domain>/`
2. Run: `go test ./test/suite/<domain>/ -count=1 -short`
3. Update `TEST_MAP.md` in the same directory

## Adding a new test

1. Find the right domain in `test/suite/<domain>/`
2. Add `func testMyThing(t *testing.T, env *suite.TestEnv)` in the right `.go` file
3. Register in `run.go` inside `Run()`
4. Update `TEST_MAP.md`
5. Run: `go test ./test/suite/<domain>/ -run 'TestDomain/domain/my_thing'`

## Adding a new domain

1. `mkdir test/suite/newdomain/`
2. Create `run.go` with `func Run(t *testing.T, env *suite.TestEnv)`
3. Create `newdomain_test.go`: `func TestNewDomain(t *testing.T) { env := suite.Full(t); Run(t, env) }`
4. Add test functions in `.go` files by topic
5. Create `CLAUDE.md` (reference this file) + `TEST_MAP.md`
6. Add the domain to transport campaigns if transport-sensitive

## Running tests

```bash
go test ./test/suite/...                              # all domains (~30s)
go test ./test/suite/bus/                             # single domain
go test ./test/campaigns/transport/                   # all backends (needs Podman)
go test ./test/fixtures/ -run TestFixtures            # TS fixtures
go test -bench . ./test/bench/...                     # benchmarks
```
