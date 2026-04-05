# Suite Tests

Pure test logic. No containers, no transport setup. Every domain runs on memory by default.

## Pattern

Each domain is a Go package with:
- `run.go` — exports `Run(t *testing.T, env *suite.TestEnv)`, registers all test functions
- `<topic>.go` — test functions: `func testXxx(t *testing.T, env *suite.TestEnv)`
- `<domain>_test.go` — standalone entry: creates env via `suite.Full(t, ...)` and calls `Run(t, env)`

## Adding a test to an existing domain

```go
// 1. Add function in the right .go file
func testMyNewBehavior(t *testing.T, env *suite.TestEnv) {
    k := env.Kernel  // shared kernel, or suite.Full(t) for fresh
    // ... test logic
}

// 2. Register in run.go inside Run()
t.Run("my_new_behavior", func(t *testing.T) { testMyNewBehavior(t, env) })
```

## Adding a new domain

1. `mkdir test/suite/newdomain/`
2. Create `run.go` with `func Run(t *testing.T, env *suite.TestEnv)`
3. Create `newdomain_test.go` with `func TestNewDomain(t *testing.T) { env := suite.Full(t); Run(t, env) }`
4. Add test functions in separate `.go` files by topic

## TestEnv options

```go
suite.Full(t)                                    // storage + vectors + AI + FS + tools
suite.Full(t, suite.WithRBAC(roles, "service"))  // + RBAC
suite.Full(t, suite.WithPersistence())           // + SQLite store
suite.Full(t, suite.WithTracing())               // + trace store
suite.Full(t, suite.WithSecretKey("key"))        // + encrypted secrets
suite.Minimal(t)                                 // bare kernel, nothing configured
```

## When to use a fresh kernel

- Tests asserting exact counts (ListDeployments, ListSchedules)
- Tests subscribing to global bus events (bus.handler.failed, bus.permission.denied)
- Tests that close/restart the kernel
- Tests with specific RBAC configs different from the shared env

## 20 domains

agents, bus, cli, cross, deploy, fs, gateway, health, mcp, packages, persistence, rbac, registry, scheduling, secrets, security, stress, tools, tracing, workflows
