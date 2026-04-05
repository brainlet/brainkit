# TS Fixtures

166 fixtures across 17 categories. Each fixture is `index.ts` + optional `expect.json`.

Read the category CLAUDE.md before adding or editing fixtures.

## Convention

- `errors/` under any feature = adversarial tests
- Backend name as leaf = runner infers container needs (postgres, mongodb, pgvector, libsql, upstash)
- `output({...})` to emit results, `expect.json` to assert
- Matchers in expect.json: exact value, `"*"` (exists), `"~prefix"` (contains)
- `integration/` under any feature = cross-feature combination

## Classification (from classify.go)

Infrastructure is auto-detected from path segments -- no switch statements to maintain:

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
| memory | 22 | storage/*, semantic-recall/*, generate-title, working-memory/* (13 of 22) | storage/postgres*, storage/mongodb* need containers; storage/upstash needs credential |
| observability | 2 | all | none |
| plugin | 1 | none | none (plugin subprocess) |
| polyfill | 10 | none | none |
| rag | 9 | vector-query-tool only | vector-query-tool needs libsql-server |
| tools | 6 | create-with-schema uses Agent (needs AI key) | none |
| vector | 6 | none | pgvector needs postgres; mongodb needs mongodb; libsql needs libsql-server |
| workflow | 14 | integration/with-agent-step only | none |
