# TS Fixtures

166 fixtures across 17 categories. Each fixture is `index.ts` + optional `expect.json`.

Read the category CLAUDE.md before adding or editing fixtures in that category.

## Conventions — READ BEFORE EDITING

### Directory structure is the API
The runner discovers fixtures via `filepath.WalkDir`. The **path determines infrastructure needs**. No switch statements, no hardcoded lists. The path IS the classification.

```
fixtures/ts/<category>/<subcategory>/<variant>/
  index.ts        ← deployed to kernel
  expect.json     ← asserts against output() (optional)
  tsconfig.json   ← IDE only, ignored by runner
```

### Path conventions (enforced by classify.go)
- **Backend name as leaf directory** (`postgres/`, `libsql/`, `pgvector/`, `mongodb/`) → runner auto-starts the right container
- **`errors/`** under any feature → adversarial/error path tests
- **`integration/`** under any feature → cross-feature combination
- Depth grows naturally: feature → function → variant → backend

### output() is how fixtures emit results
```typescript
output({ myKey: true, count: 42, message: "hello" });
```
Runner reads `globalThis.__module_result` after deploy and compares to expect.json.

### expect.json matchers
```json
{
  "exact_bool": true,        // exact match
  "exact_number": 42,        // ±0.01 delta
  "exact_string": "hello",   // exact match
  "just_exists": "*",        // wildcard — key must exist
  "partial": "~hello"        // prefix ~ — assert.Contains
}
```

### No hardcoded URLs
Use environment variables. The runner/campaigns set these:
- `POSTGRES_URL`, `MONGODB_URL`, `LIBSQL_URL`, `LIBSQL_VECTOR_URL`, `UPSTASH_REDIS_REST_URL`

### Adding a new backend
1. Add the segment name to the map in `test/fixtures/classify.go`
2. Create fixture directories with the backend name as leaf
3. No runner code changes needed

### Adding a fixture
1. Create: `fixtures/ts/<category>/<feature>/<variant>/`
2. Write `index.ts` — use `output({...})`
3. Write `expect.json` — keys to assert
4. Update the category CLAUDE.md
5. Run: `go test ./test/fixtures/ -run 'TestFixtures/<category>/<feature>/<variant>'`

### After editing a fixture
1. Run: `go test ./test/fixtures/ -run 'TestFixtures/<path>'`
2. If it needs AI, verify with `OPENAI_API_KEY` set
3. If it needs containers, verify with Podman running

## Classification (from classify.go)

| Path segment | Infrastructure |
|-------------|----------------|
| `postgres`, `postgres-scram`, `pgvector` | Postgres container |
| `mongodb`, `mongodb-scram` | MongoDB container |
| `libsql` (under vector/ only) | libsql-server container |
| `upstash` | UPSTASH_REDIS_REST_URL credential |
| Category `agent`, `ai`, `observability`, `composition` | OPENAI_API_KEY |
| Category `memory` + segment `storage` | OPENAI_API_KEY |
| Segment `with-agent-step`, `vector-query-tool`, `with-llm-judge`, `semantic-recall`, `generate-title`, `working-memory` | OPENAI_API_KEY |
| Category `mcp` | In-process MCP server |

## Categories

| Category | Count | Needs AI | Needs Containers |
|----------|-------|----------|-----------------|
| agent | 30 | all | memory/mongodb, memory/postgres need containers; memory/upstash needs credential |
| ai | 21 | all | none |
| bus | 13 | none | none |
| composition | 2 | all | none |
| cross-feature | 5 | none | none |
| cross-kit | 1 | none | none |
| evals | 5 | with-llm-judge only | none |
| kit | 17 | none | none |
| mcp | 2 | none | MCP server (in-process) |
| memory | 22 | 13 of 22 | storage/postgres*, storage/mongodb* need containers |
| observability | 2 | all | none |
| plugin | 1 | none | none |
| polyfill | 10 | none | none |
| rag | 9 | vector-query-tool only | vector-query-tool needs libsql-server |
| tools | 6 | create-with-schema only | none |
| vector | 6 | none | pgvector→postgres, mongodb→mongodb, libsql→libsql-server |
| workflow | 14 | with-agent-step only | none |
