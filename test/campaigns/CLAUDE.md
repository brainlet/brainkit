# Campaigns

Read TEST_MAP.md before editing. Read `../CLAUDE.md` for overall conventions.

Infrastructure composition layer. Each campaign spins up containers, creates a TestEnv with real backends, and calls suite `Run()` functions.

## Conventions — READ BEFORE EDITING

### Every campaign must start with RequirePodman
```go
campaigns.RequirePodman(t)  // skip entire campaign when no Podman
```
Without this guard, CI without Podman will hang or fail instead of skipping.

### Campaigns don't contain test logic
Campaigns call `suite.Run()` — they never have their own assertions. The test logic lives in `test/suite/`. If you need a new test, add it to the suite domain, not the campaign.

### Container sharing
The Infra builder starts each container type once. `Storage("postgres")` + `Vector("pgvector")` reuse the same Postgres container. Don't start containers manually.

### Transport-sensitive domains
These 14 domains should appear in every transport campaign: bus, deploy, tools, agents, scheduling, health, secrets, registry, mcp, workflows, tracing, fs, persistence, gateway.

### After editing
1. `go vet ./test/campaigns/<category>/`
2. With Podman: `go test ./test/campaigns/<category>/ -count=1 -timeout 300s`
3. Update TEST_MAP.md

## How campaigns work

```go
func TestTransport_NATS(t *testing.T) {
    campaigns.RequirePodman(t)
    infra := campaigns.NewInfra(t, campaigns.Transport("nats"))
    env := infra.Env(t)
    bus.Run(t, env)       // same suite tests, different backend
    deploy.Run(t, env)
    // ...
}
```

The Infra builder manages containers (lazy startup, shared across tests) and converts backend names to brainkit configs.

## Adding a new transport backend

1. Add container startup logic in `infra.go`
2. Create `test/campaigns/transport/<backend>_test.go`
3. Call all transport-sensitive suite domains

## Adding a new storage/vector backend

1. Add container logic in `infra.go` (or reuse existing)
2. Create `test/campaigns/storage/<backend>_test.go`
3. Call `infra.RunFixtures(t, "memory/storage/<backend>*", ...)`

## Categories

- `transport/` — 5 backends (nats, amqp, redis, sqlite, postgres). Runs 14 suite domains on each.
- `storage/` — 3 backends (postgres, mongodb, libsql). Runs TS fixtures.
- `vector/` — 3 backends (pgvector, mongodb, libsql). Runs TS fixtures.
- `auth/` — Auth method matrix (postgres SCRAM/MD5/trust, mongodb SCRAM, libsql, upstash).
- `crosskit/` — Cross-kit on 3 transports. Uses `Nodes(2)`.
- `plugins/` — Plugin tests on 3 transports.
- `fullstack/` — Production combos (Redis+MongoDB, AMQP+Postgres+PgVector).

## Infra options

```go
campaigns.Transport("nats")         // start NATS container
campaigns.Storage("postgres")       // start/reuse Postgres
campaigns.Vector("pgvector")        // reuse Postgres for pgvector
campaigns.Persistence()             // enable SQLite store
campaigns.Tracing()                 // enable trace store
campaigns.AI()                      // load .env, configure AI providers
campaigns.Nodes(2)                  // multi-node topology
campaigns.Plugins(cfg)              // plugin config
```
