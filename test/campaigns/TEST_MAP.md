# Campaigns Test Map

**Purpose:** Infrastructure composition -- runs suite tests on real backends with containers.
**Infra builder:** `infra.go` manages lazy container startup (Postgres, MongoDB, LibSQL) via `sync.Once`.
**Pattern:** `campaigns.NewInfra(t, options...) -> infra.Env(t) -> suite.Run(t, env)`

## transport/

All 5 backends run the same 14 suite domains. Each test creates an Infra with a single transport backend and calls every domain's `Run(t, env)`.

| File | Test Function | Backend | Needs Podman | Suite Domains |
|------|--------------|---------|--------------|---------------|
| nats_test.go | TestTransport_NATS | nats | yes | bus, deploy, tools, agents, scheduling, health, secrets, registry, mcp, workflows, tracing, fs, persistence, gateway |
| amqp_test.go | TestTransport_AMQP | amqp | yes | bus, deploy, tools, agents, scheduling, health, secrets, registry, mcp, workflows, tracing, fs, persistence, gateway |
| redis_test.go | TestTransport_Redis | redis | yes | bus, deploy, tools, agents, scheduling, health, secrets, registry, mcp, workflows, tracing, fs, persistence, gateway |
| postgres_test.go | TestTransport_Postgres | sql-postgres | yes | bus, deploy, tools, agents, scheduling, health, secrets, registry, mcp, workflows, tracing, fs, persistence, gateway |
| sqlite_test.go | TestTransport_SQLite | sql-sqlite | no | bus, deploy, tools, agents, scheduling, health, secrets, registry, mcp, workflows, tracing, fs, persistence, gateway |

## storage/

Storage campaigns run TS fixture subsets via `infra.RunFixtures()`. Each needs AI (for agent conversation tests).

| File | Test Function | Backend | Needs Podman | Fixture Patterns |
|------|--------------|---------|--------------|-----------------|
| postgres_test.go | TestStorage_Postgres | postgres | yes | `memory/storage/postgres*`, `agent/memory/postgres` |
| mongodb_test.go | TestStorage_MongoDB | mongodb | yes | `memory/storage/mongodb*`, `agent/memory/mongodb` |
| libsql_test.go | TestStorage_LibSQL | libsql | yes | `memory/storage/libsql*`, `agent/memory/libsql` |

## vector/

Vector campaigns run TS fixture subsets via `infra.RunFixtures()`. Each needs AI.

| File | Test Function | Backend | Needs Podman | Fixture Patterns |
|------|--------------|---------|--------------|-----------------|
| pgvector_test.go | TestVector_PgVector | pgvector | yes | `vector/*/pgvector` |
| mongodb_test.go | TestVector_MongoDB | mongodb | yes | `vector/*/mongodb` |
| libsql_test.go | TestVector_LibSQL | libsql | yes | `vector/*/libsql` |

## auth/

Auth campaigns test backend-specific authentication methods. Each creates a container with specific auth config, deploys minimal .ts code, and verifies CRUD through the JS driver.

| File | Test Function | Backend | Auth Method | Needs Podman |
|------|--------------|---------|-------------|--------------|
| postgres_test.go | TestPostgres_SCRAM_SHA256 | pgvector/pgvector:pg16 | SCRAM-SHA-256 | yes |
| postgres_test.go | TestPostgres_MD5 | postgres:16 | MD5 | yes |
| postgres_test.go | TestPostgres_Trust | postgres:16 | trust (no password) | yes |
| mongodb_test.go | TestMongoDB_SCRAM_SHA256 | mongo:7 | SCRAM-SHA-256 | yes |
| mongodb_test.go | TestMongoDB_SCRAM_SHA1 | mongo:7 | SCRAM-SHA-1 | yes |
| mongodb_test.go | TestMongoDB_NoAuth | mongo:7 | none | yes |
| libsql_test.go | TestLibSQL_EmbeddedNoAuth | embedded SQLite | none | no |
| libsql_test.go | TestLibSQL_ContainerNoAuth | libsql-server | none (HTTP) | yes |
| upstash_test.go | TestUpstash_TokenAuth | Upstash Redis | token auth | no (cloud) |
| upstash_test.go | TestInMemory_NoAuth | in-memory | none (baseline) | no |

## crosskit/

Cross-kit campaigns test multi-node topology (2 nodes sharing a transport). Each runs `cross.Run(t, env)`.

| File | Test Function | Backend | Needs Podman | Nodes |
|------|--------------|---------|--------------|-------|
| nats_test.go | TestCrossKit_NATS | nats | yes | 2 |
| postgres_test.go | TestCrossKit_Postgres | sql-postgres | yes | 2 |
| redis_test.go | TestCrossKit_Redis | redis | yes | 2 |

## plugins/

Plugin campaigns test plugin lifecycle on multi-node setups. Each builds a test plugin binary, creates 2-node infra, and runs `cross.Run(t, env)`.

| File | Test Function | Backend | Needs Podman | Nodes |
|------|--------------|---------|--------------|-------|
| nats_test.go | TestPlugins_NATS | nats | yes | 2 |
| postgres_test.go | TestPlugins_Postgres | sql-postgres | yes | 2 |
| redis_test.go | TestPlugins_Redis | redis | yes | 2 |

## fullstack/

Fullstack campaigns test production-realistic backend combinations. Each runs multiple suite domains.

| File | Test Function | Transport | Storage | Vector | Extras | Suite Domains |
|------|--------------|-----------|---------|--------|--------|---------------|
| nats_postgres_rbac_test.go | TestFullStack_NATS_Postgres_RBAC | nats | postgres | -- | persistence, RBAC, tracing | bus, deploy, tools, agents, health, secrets, registry, workflows, tracing, persistence, gateway, rbac |
| redis_mongodb_test.go | TestFullStack_Redis_MongoDB | redis | mongodb | -- | persistence, tracing | bus, deploy, tools, agents, health, secrets, registry, workflows, tracing, persistence, gateway |
| amqp_postgres_vector_test.go | TestFullStack_AMQP_Postgres_PgVector | amqp | postgres | pgvector | persistence, tracing | bus, deploy, tools, agents, health, secrets, registry, workflows, tracing, persistence, gateway |

## Cross-references

- Each transport campaign calls suite domains: bus, deploy, tools, agents, scheduling, health, secrets, registry, mcp, workflows, tracing, fs, persistence, gateway
- Storage/vector campaigns call `fixtures.RunMatching()` to run TS fixture subsets
- Auth campaigns use raw `EvalTS` with inline store code (no suite domains)
- Crosskit and plugins campaigns call `cross.Run()` for multi-node verification
- Fullstack campaigns combine transport + storage + optional vector/RBAC/persistence + suite domains
