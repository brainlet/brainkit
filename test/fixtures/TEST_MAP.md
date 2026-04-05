# Fixtures Test Map

**Purpose:** TS fixture integration tests. Deploy `index.ts`, read output, assert against `expect.json`.
**Count:** 166 fixtures across 17 categories
**Runner:** `test/fixtures/runner.go` (WalkDir + path-based classification via `classify.go`)
**Entry:** `TestFixtures` in `fixtures_test.go` calls `runner.RunAll(t)`
**Assertion:** `helpers.go` -- `"*"` (key exists), `"~prefix"` (contains substring), exact match, float delta 0.01

## Categories

### agent/ (30 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| agent/callbacks/on-step-finish | yes | no | onStepFinish callback fires during generation |
| agent/generate/active-tools | yes | no | Agent generation with active tool selection |
| agent/generate/basic | yes | no | Basic agent generate -- usage stats, finish reason |
| agent/generate/dynamic-instructions | yes | no | Dynamic instructions produce different outputs per call |
| agent/generate/dynamic-model | yes | no | Dynamic model selection at generation time |
| agent/generate/dynamic-tools | yes | no | Dynamic tool registration (add, multiply) |
| agent/generate/instructions-override | yes | no | Instruction override at generate time |
| agent/generate/multi-step | yes | no | Multi-step generation with tool use |
| agent/generate/options-passthrough | yes | no | Temperature, instructions, maxSteps pass through correctly |
| agent/generate/structured-output | yes | no | Structured output schema enforcement |
| agent/generate/with-context-messages | yes | no | Context messages (knows "blue" from prior message) |
| agent/generate/with-tools | yes | no | Agent generation with tool calls |
| agent/hitl/bus-approval | yes | no | Human-in-the-loop: bus-based approval flow (Go auto-approver) |
| agent/integration/with-workflow | yes | no | Agent integrated with workflow step |
| agent/memory/inmemory | yes | no | Agent with in-memory storage remembers across turns |
| agent/memory/libsql | yes | libsql-server | Agent memory persistence with LibSQL backend |
| agent/memory/mongodb | yes | mongodb | Agent memory persistence with MongoDB backend |
| agent/memory/postgres | yes | postgres | Agent memory persistence with Postgres backend |
| agent/memory/upstash | yes | UPSTASH credential | Agent memory persistence with Upstash backend |
| agent/multi-provider | yes | no | Multi-provider agent (OpenAI verification) |
| agent/request-context | yes | no | Request context injection into agent |
| agent/scorers | yes | no | Agent output scoring |
| agent/stream/basic | yes | no | Basic agent streaming with real-time tokens |
| agent/stream/with-tools | yes | no | Agent streaming with tool calls |
| agent/subagents/basic | yes | no | Basic sub-agent delegation |
| agent/subagents/constrained | yes | no | Constrained sub-agent (limited capabilities) |
| agent/subagents/network-delegation | yes | no | Network-based sub-agent delegation |
| agent/tools/with-local-tool | yes | no | Agent with locally-defined tool (1 tool call) |
| agent/tools/with-registered-tool | yes | no | Agent with Go-registered tool (multiply) |
| agent/workspace | yes | no | Agent workspace support |

### ai/ (21 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| ai/embed/many | yes | no | Batch embedding (3 inputs, all vectors, usage) |
| ai/embed/single | yes | no | Single embedding (values, 1536 dimensions) |
| ai/generate-object/array | yes | no | Object generation returning array |
| ai/generate-object/basic | yes | no | Object generation (name, age, hobbies) |
| ai/generate-object/enum | yes | no | Object generation with enum constraint |
| ai/generate-text/basic | yes | no | Basic text generation -- usage, finish reason |
| ai/generate-text/conversation | yes | no | Multi-turn conversation (remembers name, city) |
| ai/generate-text/max-tokens | yes | no | Max tokens limit enforcement |
| ai/generate-text/multi-step | yes | no | Multi-step generation with tool calls |
| ai/generate-text/stop-sequences | yes | no | Stop sequence enforcement |
| ai/generate-text/temperature | yes | no | Temperature 0 produces deterministic output |
| ai/generate-text/with-system | yes | no | System message injection |
| ai/generate-text/with-tools | yes | no | Text generation with tool calls |
| ai/middleware/wrap-model | yes | no | Model middleware wrapping |
| ai/stream-object/basic | yes | no | Streaming object generation (name, age, hobbies) |
| ai/stream-text/basic | yes | no | Basic text streaming with real-time tokens |
| ai/stream-text/full-stream | yes | no | Full stream with text deltas |
| ai/stream-text/on-chunk | yes | no | onChunk callback during streaming |
| ai/stream-text/on-finish | yes | no | onFinish callback after streaming |
| ai/stream-text/with-tools | yes | no | Streaming with tool calls (usage stats) |
| ai/tool/with-suspend | yes | no | Tool suspension and resume with approval flow |

### bus/ (13 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| bus/emit-fire-and-forget | no | no | Fire-and-forget emit (2 events) |
| bus/errors/concurrent-publish | no | no | 50 concurrent publishes all succeed |
| bus/errors/large-payload | no | no | Large payload (50KB) publish and reply |
| bus/errors/schedule-unschedule | no | no | Schedule 5 items, verify all have IDs |
| bus/errors/send-no-heartbeat-adv | no | no | Send without heartbeat: final reply + 2 chunks |
| bus/errors/sendto | no | no | SendTo publishes with replyTo |
| bus/errors/streaming-protocol-adv | no | no | Streaming protocol handler registration |
| bus/mailbox-on | no | no | Mailbox on/reply pattern (question/answer) |
| bus/publish-reply | no | no | Publish with replyTo, correlationId, and reply |
| bus/send-to-service | no | no | SendToService with reply (greeting) |
| bus/send-to-shard | no | no | SendToShard with stateful increment |
| bus/streaming-send-reply | no | no | Streaming send/reply (4 chunks + final) |
| bus/subscribe-basic | no | no | Basic subscribe (subscription ID, 2 messages) |

### composition/ (2 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| composition/agent-workflow-memory | yes | no | Agent + workflow + memory composition (Go reverse tool) |
| composition/multi-module-integration | yes | no | Multi-module integration (bus + agent) |

### cross-feature/ (5 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| cross-feature/agent-with-bus | no | no | Agent availability + tool creation via bus |
| cross-feature/deploy-with-secrets | no | no | Deploy reads secrets (empty value check) |
| cross-feature/deploy-with-tools | no | no | Tool call during deploy init |
| cross-feature/multi-service-chain | no | no | Multi-service publish chain (replyTo, correlation) |
| cross-feature/schedule-triggers-handler | no | no | Schedule/unschedule lifecycle |

### cross-kit/ (1 fixture)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| cross-kit/publish-to-remote | no | no | Cross-kit publish (namespace, tool, self-call, bus) |

**Note:** cross-kit and plugin categories are skipped by the general runner (see `skipCategories`). They run through campaign-specific runners.

### evals/ (5 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| evals/batch/run-evals | yes | no | Batch evaluation runner (2 scored, keyword positive) |
| evals/scorer/basic | yes | no | Basic scorer (score in range, has runId) |
| evals/scorer/with-llm-judge | yes | no | LLM-as-judge scorer |
| evals/scorer/with-preprocess | yes | no | Scorer with preprocessing (positive score in range) |
| evals/scorer/with-reason | yes | no | Scorer with reason (score=1, has reason string) |

### kit/ (17 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| kit/errors/deploy-throws-init | no | no | Deploy that throws during init (beforeThrow fires) |
| kit/errors/error-code-inspection | no | no | Error code inspection for missing tool, replyTo, emit |
| kit/errors/file-url-blocked | no | no | file:// URL blocking for store/vector/http/libsql |
| kit/errors/multi-tool-register | no | no | Register 5 tools, verify all found |
| kit/errors/register-invalid-type | no | no | Invalid type registration error message |
| kit/errors/secrets-operations | no | no | Secrets operations (empty result) |
| kit/errors/tool-lifecycle | no | no | Tool register, find, call lifecycle (doubled=42) |
| kit/fs/list-stat | no | no | FS list and stat operations |
| kit/fs/operations | no | no | FS write, read, find, size, delete |
| kit/fs/read-write | no | no | FS read/write roundtrip |
| kit/lifecycle/deploy-teardown | no | no | Deploy then teardown (removed=true) |
| kit/output/basic | no | no | Basic output (hello world, number 42) |
| kit/registry/has-list | no | no | Registry has/list (nonexistent=false, providers/storages arrays) |
| kit/registry/operations | no | no | Registry operations (has, providers, storages) |
| kit/registry/resolve | no | no | Registry resolve (missing returns null) |
| kit/storage-pool/default | no | no | Default storage pool resolution |
| kit/storage-pool/memory | no | no | Memory storage pool resolution |

### mcp/ (2 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| mcp/agent-with-mcp-tool | yes | no | Agent with MCP tool (echo tool, 1 tool count) |
| mcp/call-tool | no | no | MCP tool listing (1 tool) |

### memory/ (22 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| memory/generate-title | yes | no | Thread title generation |
| memory/libsql-local-debug | yes | no | LibSQL local debug memory (remembers) |
| memory/messages/save-and-recall | no | no | Message save and recall |
| memory/observational | yes | no | Observational memory |
| memory/read-only | yes | no | Read-only memory (knows "mango") |
| memory/semantic-recall/basic | yes | no | Semantic recall (remembers Rust) |
| memory/semantic-recall/resource-scope | yes | no | Semantic recall with resource scoping |
| memory/storage/inmemory | yes | no | In-memory storage (remembers name, work) |
| memory/storage/libsql | yes | libsql-server | LibSQL storage persistence |
| memory/storage/libsql-local | yes | no | LibSQL local storage (remembers color, dog) |
| memory/storage/mongodb | yes | mongodb | MongoDB storage persistence |
| memory/storage/mongodb-scram | yes | mongodb | MongoDB SCRAM auth storage persistence |
| memory/storage/postgres | yes | postgres | Postgres storage persistence |
| memory/storage/postgres-scram | yes | postgres | Postgres SCRAM auth storage persistence |
| memory/storage/upstash | yes | UPSTASH credential | Upstash storage persistence |
| memory/threads/create | no | no | Thread create/fetch/id match |
| memory/threads/delete | no | no | Thread deletion |
| memory/threads/get-by-id | no | no | Thread get by ID (found, missing, correctId) |
| memory/threads/list | no | no | Thread list (3 created, all found, distinct IDs) |
| memory/threads/management | no | no | Thread management operations |
| memory/working-memory/basic | yes | no | Working memory (knows name) |
| memory/working-memory/schema | yes | no | Working memory with schema (knows Bob) |

### observability/ (2 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| observability/spans/basic | yes | no | Basic span creation (has answer, traceId) |
| observability/trace/basic | yes | no | Basic trace creation (works, has traceId) |

### plugin/ (1 fixture)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| plugin/call-plugin-tool | no | no | Call plugin tools (echo, concat) |

**Note:** Skipped by the general runner. Runs through campaign plugin runner.

### polyfill/ (10 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| polyfill/buffer/pool-size | no | no | Buffer poolSize, encoding checks, byteLength, compare |
| polyfill/crypto/getfips | no | no | crypto.getFips, ciphers, timingSafeEqual |
| polyfill/dns/lookup | no | no | DNS sync and async lookup |
| polyfill/events/max-listeners | no | no | EventEmitter maxListeners, captureRejections |
| polyfill/exec/sync | no | no | execSync, execFileSync, spawnSync |
| polyfill/os/release | no | no | os.release (not stub), cpus, EOL |
| polyfill/process/extras | no | no | process.emitWarning, uid, gid, hrtime, nextTick |
| polyfill/stream/readable-from | no | no | Readable.from (3 items), pipe (2 items) |
| polyfill/util/types | no | no | util.types checks (Date, RegExp, Map, Set, TypedArray, Buffer) |
| polyfill/zlib/deflate-inflate | no | no | zlib deflate/inflate, gzip/gunzip, async, constants |

### rag/ (9 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| rag/chunk/markdown | no | no | Markdown chunking (multiple chunks) |
| rag/chunk/text | no | no | Text chunking (multiple chunks) |
| rag/chunk/token | no | no | Token-based chunking (multiple chunks) |
| rag/document-chunker-tool | no | no | Document chunker tool creation |
| rag/graph-rag | no | no | Graph RAG availability |
| rag/mdocument-parsing | no | no | Markdown document parsing (text created) |
| rag/rerank/basic | no | no | Basic reranking |
| rag/rerank/functional | no | no | Functional reranking |
| rag/vector-query-tool | yes | no | Vector query tool (has results, needs AI for embedding) |

### tools/ (6 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| tools/call-from-ts | no | no | Call Go tool from TS (uppercase "HELLO BRAINLET") |
| tools/call-go-tool | no | no | Call Go tool (echo + sum=42) |
| tools/create-basic | no | no | Create tool in TS (sum=42, registered) |
| tools/create-with-schema | yes | no | Create tool with JSON schema |
| tools/register-list | no | no | Register and list tools |
| tools/register-unregister | no | no | Register, find, unregister, verify removed |

### vector/ (6 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| vector/create-upsert-query/libsql | no | libsql-server | LibSQL vector create/upsert/query |
| vector/create-upsert-query/mongodb | no | mongodb | MongoDB vector create/upsert/query |
| vector/create-upsert-query/pgvector | no | postgres | PgVector create/upsert/query |
| vector/methods/libsql | no | libsql-server | LibSQL vector methods (allPassed) |
| vector/methods/mongodb | no | mongodb | MongoDB vector methods (upserted=2) |
| vector/methods/pgvector | no | postgres | PgVector methods (resultCount=2) |

### workflow/ (14 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| workflow/basic-then | no | no | Basic .then() chaining (success) |
| workflow/branch | no | no | Workflow branching (correct path) |
| workflow/errors/error-handling | no | no | Error handling (isFailed + success fallback) |
| workflow/foreach | no | no | ForEach iteration (success) |
| workflow/hooks/after | no | no | After-step hook (correct value) |
| workflow/integration/with-agent-step | yes | no | Workflow with agent step (has answer) |
| workflow/loop-dountil | no | no | DoUntil loop (success) |
| workflow/nested | no | no | Nested workflows (is42, success) |
| workflow/parallel | no | no | Parallel step execution (success) |
| workflow/sleep | no | no | Sleep step (success) |
| workflow/state/get-step-result | no | no | Get previous step result (correct=true, fromStep1=42) |
| workflow/state/shared | no | no | Shared state across steps (success) |
| workflow/suspend-resume/basic | no | no | Workflow suspend and resume (success) |
| workflow/suspend-resume/with-data | no | no | Suspend/resume with data (approved, approver=david) |

## Go tool registration

Some fixtures require Go-side tool registration (in `runner.go:registerFixtureTools`):

| Fixture Path | Tool Name | What it does |
|-------------|-----------|--------------|
| tools/call-from-ts | uppercase | Converts text to uppercase |
| agent/tools/with-registered-tool | multiply | Multiplies two numbers |
| agent/hitl/bus-approval | (subscriber) | Auto-approves via `sdk.Reply` on `test.approvals` |
| composition/agent-workflow-memory | reverse | Reverses a string |

## Cross-references

- Storage campaigns call `RunMatching(t, "memory/storage/postgres*", "agent/memory/postgres")`
- Vector campaigns call `RunMatching(t, "vector/*/pgvector")`
- `cross-kit/` and `plugin/` are in `skipCategories` -- not run by general runner
- Classification logic is in `classify.go` -- scans all path segments for infrastructure markers
- AI categories (always need OPENAI_API_KEY): agent, ai, observability, composition
- AI segments (need OPENAI_API_KEY anywhere in path): with-agent-step, vector-query-tool, with-llm-judge, semantic-recall, generate-title, working-memory
