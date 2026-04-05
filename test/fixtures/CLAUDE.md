# TS Fixtures

TypeScript fixture runner. Each fixture is `index.ts` + optional `expect.json` in a nested directory.

## How it works

1. `filepath.WalkDir` discovers every `index.ts` under `fixtures/ts/`
2. Path segments classify infrastructure needs (containers, AI keys, credentials)
3. Runner deploys the `.ts`, reads `globalThis.__module_result`, asserts against `expect.json`

## Directory convention

```
fixtures/ts/<category>/<subcategory>/<variant>/
  index.ts        — the fixture code
  expect.json     — expected output (optional)
```

- `errors/` under any feature = adversarial/error path tests
- Backend name as leaf directory (`postgres/`, `libsql/`, `pgvector/`) = runner infers container needs
- `integration/` under any feature = cross-feature combination

## Adding a fixture

1. Create directory: `fixtures/ts/<category>/<feature>/<variant>/`
2. Add `index.ts` with your test code. Use `output({...})` to emit results.
3. Add `expect.json` with expected keys. Matchers: `"*"` (exists), `"~prefix"` (contains), exact match.
4. Run: `go test ./test/fixtures/ -run 'TestFixtures/<category>/<feature>/<variant>'`

## Path-based classification

The runner auto-detects infrastructure from path segments:

| Segment | Needs |
|---------|-------|
| `postgres`, `pgvector` | Postgres container |
| `mongodb`, `mongodb-scram` | MongoDB container |
| `libsql` (under vector/) | libsql-server container |
| `upstash` | UPSTASH_REDIS_REST_URL credential |
| `agent`, `ai`, `composition` | OPENAI_API_KEY |

No switch statements to update. Add a new backend → add path segment to the map in `classify.go`.

## Runner API (for campaigns)

```go
runner := fixtures.NewRunner(fixtures.FixturesRoot(t))
runner.RunAll(t)                                          // standalone
runner.RunMatching(t, "memory/storage/postgres*")         // from campaign
```
