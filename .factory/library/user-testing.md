# User Testing

Everything validators need in order to test this mission. Validators MUST follow this file — do not re-derive.

## Validation Surface

This mission has TWO validation surfaces, both offline / CI-style (no browser, no TUI):

1. **Type surface** (primary): `tsc --noEmit -p fixtures/tsconfig.base.json` invoked via `make type-check` from the repo root. Single whole-tree pass, ~0.65s, ~190 MB RSS. Validators check the exit code and parse TS errors by file path to localize to a milestone's domain. The base `include` pulls in the entire `fixtures/ts/**/*.ts` tree plus `internal/engine/runtime/globals.d.ts`, so one pass gates every fixture together. Tsconfig path mappings (`"agent"` / `"ai"` / `"kit"`) resolve to `internal/engine/runtime/*.d.ts`.

2. **Runtime surface**: `go test ./test/fixtures/ -run TestFixtures/<domain> -count=1 -timeout 600s` (or `-timeout 1200s` for the full sweep). Runs each fixture through the real brainkit kernel: deploy → TS type-strip (esbuild-style via `vendor_typescript`) → evaluate in SES Compartment. Per-fixture `expect.json` sidecars carry behavioral assertions. Without `expect.json`, a fixture passes if deploy completes without throwing.

Shell probes (`rg`, `diff`, `grep`, `find`, `jq`) are used as secondary evidence in several contract assertions (e.g. confirming `ConsoleLogger extends MastraLogger` in source, confirming absence of `as any` in fixtures).

## Required testing skills / tools

- `make type-check` (added by the M0 foundation worker)
- `go test` with per-domain `-run` filter
- `rg` / `grep` / `diff` for source-level comparison against canonical
- `git` for clone management
- `tsc` (system TypeScript 5.9.3 or bundle-local)

No browser automation, no TUI, no server. Validators do not need agent-browser, tuistory, or curl.

## Validation Concurrency

**Max concurrent validators: 1 (sequential).** Rationale:

- The user chose sequential execution at planning time.
- The runtime gate depends on shared Podman containers (pgvector, mongodb, libsql-server) that are lazy-started by the Go fixture runner via testcontainers-go. Concurrent Go test processes would either contend for these singletons or multiply container startup cost.
- The type gate is cheap (~0.65s), so parallelism would give negligible speedup for meaningful risk of filesystem races.
- AI-gated fixtures consume OPENAI_API_KEY budget; running them concurrently would fan out the cost.

## Environment setup

- `.env` at repo root contains `OPENAI_API_KEY`, `UPSTASH_API_KEY`, `UPSTASH_REDIS_REST_URL`, `UPSTASH_REDIS_REST_TOKEN`, `GLM_API_KEY`, `MINIMAX_API_KEY`. Validators DO NOT print these values to logs.
- Podman machine `podman-machine-default` should be running (5 CPU, 2 GiB RAM, 100 GiB disk). pgvector, mongo:7, libsql-server containers are spawned on demand by the Go runner.
- `internal/embed/agent/bundle/node_modules/` must exist (created by M0) with `@mastra/*` + `@ai-sdk/*` subtrees.
- Root `package.json` must exist with `typescript@5.9` pinned (created by M0).

## Per-domain test invocation

For each milestone, the user-testing validator:

1. Runs `make type-check` and captures full output.
2. Filters TS errors by file path to the milestone's domain: e.g. for M3 (memory), only errors under `fixtures/ts/memory/**/*.ts` count toward that milestone's assertions.
3. Runs `go test ./test/fixtures/ -run TestFixtures/<domain> -count=1 -timeout 600s > /tmp/fixture-<domain>.log 2>&1`.
4. Reads `/tmp/fixture-<domain>.log` and scans for `PASS` / `FAIL` per subtest.
5. For shell-based assertions: runs the `rg` / `diff` / `grep` commands specified in the assertion's `Evidence` clause and compares output.

Per-fixture tsconfig iteration is **forbidden** — it produces identical output to the single base pass and wastes time.

## Known gotchas

- `fixtures/cross-kit/` and `fixtures/plugin/` are SKIPPED by the general fixtures runner (they have dedicated campaign runners). Ignore them for tsc too unless an assertion explicitly includes them.
- Per-fixture tsconfigs `extends: "../../../tsconfig.base.json"` — they don't override `include`, so a single base pass is authoritative.
- Errors in one fixture can cascade to errors in unrelated fixtures if they share a declaration file. Workers should always run `make type-check` and read ALL errors before claiming a fix is complete.
- `go test ./test/fixtures/` uses Podman for storage-backed fixtures. First run spends 30–60s on container pull/startup; subsequent runs reuse the hot containers.
- AI-gated fixtures will skip cleanly if `OPENAI_API_KEY` is absent — a skip is NOT a failure.

## Evidence capture

For type-gate assertions, capture:
- Full `make type-check` output to `/tmp/type-check.log`.
- Filtered-by-domain error count (grep per fixture path prefix).
- Optional: diff of the domain's `.d.ts` section against canonical (for shell-class assertions).

For runtime-gate assertions, capture:
- Full `go test` output to `/tmp/fixture-<domain>.log`.
- PASS/FAIL count per subtest (parse `--- PASS` and `--- FAIL` lines).
- Count of skips and why (last line of each skipped subtest).

For shell-probe assertions, capture:
- The exact command from the assertion's Evidence field.
- Full stdout + stderr.
- Exit code.
