# TS Fixtures

329 fixtures across 23 categories. Each fixture is `index.ts` + optional `expect.json`.
The general `test/fixtures` runner discovers 327 of them because `cross-kit`
and `plugin` are intentionally excluded and run through specialized runners.

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
  "partial": "~hello",       // prefix ~ — assert.Contains
  "deployErrorContains": "Could not resolve" // expected deploy-time failure
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
2. If it needs AI, verify with `BRAINKIT_TEST_LIVE_AI=1`; the runner loads the
   root `.env` and uses the real `OPENAI_API_KEY`. Do not use fake models as
   proof for Mastra behavior fixtures.
3. For provider features, distinguish contract fixtures from provider support.
   A small in-fixture implementation can prove Brainkit-owned plumbing, but it
   does not close a concrete Mastra provider row. Browser provider work, for
   example, must use the real provider package and a real browser/CDP lifecycle
   before `@mastra/agent-browser`, `@mastra/stagehand`, or
   `@mastra/browser-viewer` can be marked supported.
4. If it needs containers, verify with Podman running

## Classification (from classify.go)

| Path segment | Infrastructure |
|-------------|----------------|
| `postgres`, `postgres-scram`, `pgvector` | Postgres container |
| `mongodb`, `mongodb-scram` | MongoDB container |
| `libsql` (under vector/ only) | libsql-server container |
| `upstash` | UPSTASH_REDIS_REST_URL credential |
| Category `agent`, `ai`, `harness`, `observability`, `composition`, `voice`, `processors` | OPENAI_API_KEY |
| Category `browser` | modules/browser mounted |
| Category `memory` + segment `storage` | OPENAI_API_KEY |
| Segment `with-agent-step`, `agent-stream-data-persistence`, `agent-stream-data-transient`, `agent-stream-writer`, `agent-stream-subagent-writer`, `create-with-schema`, `vector-query-tool`, `with-llm-judge`, `semantic-recall`, `generate-title`, `working-memory`, `rerank`, `graph-rag`, `prebuilt` | OPENAI_API_KEY |
| Category `mcp` | In-process MCP server |

## Categories

| Category | Count | Needs AI | Needs Containers |
|----------|-------|----------|-----------------|
| agent | 56 | all | memory/mongodb, memory/postgres need containers; memory/upstash needs credential |
| ai | 32 | all | none |
| browser | 1 | none | modules/browser mounted |
| bus | 17 | none | none |
| composition | 2 | all | none |
| cross-feature | 5 | none | none |
| cross-kit | 1 | none | none |
| ecosystem | 1 | none | none |
| evals | 22 | prebuilt LLM scorers and with-llm-judge | none |
| harness | 10 | all | none |
| kit | 17 | none | none |
| mcp | 2 | none | MCP server (in-process) |
| memory | 26 | semantic recall, working memory, title generation, storage | storage/postgres*, storage/mongodb* need containers |
| observability | 9 | all | none |
| plugin | 1 | none | none |
| polyfill | 17 | none | none |
| processors | 16 | all by category | none |
| rag | 22 | vector-query-tool, graph-rag, rerank | vector-query-tool needs libsql-server |
| storage | 1 | none | none |
| tools | 15 | create-with-schema, agent-stream-data-persistence, agent-stream-data-transient, agent-stream-writer, agent-stream-subagent-writer | none |
| vector | 9 | none | pgvector -> postgres, mongodb -> mongodb, libsql -> libsql-server |
| voice | 15 | all by category | provider-specific credentials for live provider paths |
| workflow | 32 | with-agent-step only | none |

## Category Documentation

Each active category should have its own `CLAUDE.md` except categories that are
intentionally tiny and covered elsewhere. Current category docs include
agent, ai, browser, bus, composition, cross-feature, cross-kit, ecosystem, evals,
harness, kit, mcp, memory, observability, plugin, polyfill, processors, rag,
tools, vector, voice, and workflow.
