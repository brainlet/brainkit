# Fixtures Test Map

**Purpose:** TS fixture integration tests. Deploy `index.ts`, read output, assert against `expect.json`.
**Count:** 329 fixtures across 23 categories; the general runner discovers 327 because `cross-kit` and `plugin` use specialized runners.
**Runner:** `test/fixtures/runner.go` (WalkDir + path-based classification via `classify.go`)
**Entry:** `TestFixtures` in `fixtures_test.go` calls `runner.RunAll(t)`
**Live AI:** AI fixtures use `BRAINKIT_TEST_LIVE_AI=1`; the harness loads the root `.env` and real provider credentials such as `OPENAI_API_KEY`. Provider support rows require real provider implementations; fixture-local stand-ins only prove Brainkit-owned contracts.
**Assertion:** `helpers.go` -- `"*"` (key exists), `"~prefix"` (contains substring), exact match, float delta 0.01; `deployErrorContains` expects deployment to fail with the given substring.

## Filesystem Count Summary

Refreshed from `find fixtures/ts -name index.ts` on 2026-05-15.
The category counts below are authoritative; the descriptive tables that follow
are human-maintained summaries of representative fixture behavior.

| Category | Fixtures |
|----------|---------:|
| agent | 56 |
| ai | 32 |
| browser | 1 |
| bus | 17 |
| composition | 2 |
| cross-feature | 5 |
| cross-kit | 1 |
| ecosystem | 1 |
| evals | 22 |
| harness | 10 |
| kit | 17 |
| mcp | 2 |
| memory | 26 |
| observability | 9 |
| plugin | 1 |
| polyfill | 17 |
| processors | 16 |
| rag | 22 |
| storage | 1 |
| tools | 15 |
| vector | 9 |
| voice | 15 |
| workflow | 32 |

## Categories

### agent/ (56 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| agent/background-tasks/cancel-running | yes | no | Live OpenAI-backed background task cancellation through `BackgroundTaskManager.cancel`, including cancelled event, stored state, and tool abort signal |
| agent/background-tasks/concurrency-queue | yes | no | Real Mastra background task queue semantics: global/per-agent concurrency limits keep one task pending until a slot frees |
| agent/background-tasks/lifecycle-callbacks | yes | no | Real Mastra background task lifecycle callbacks for completion and failure: per-task hooks plus manager `onTaskComplete`/`onTaskFailed` |
| agent/background-tasks/manager-stream-filter-abort | yes | no | Real Mastra `BackgroundTaskManager.stream()` filters running/completed task events by `agentId` and `taskId`, and closes readers on `AbortSignal` |
| agent/background-tasks/progress-output | yes | no | Live OpenAI-backed `Agent.streamUntilIdle()` with background tool progress output chunks and completed task result |
| agent/background-tasks/retry-success | yes | no | Live OpenAI-backed background task retry: first execute attempt fails, second succeeds, stored `retryCount` is 1 |
| agent/background-tasks/start-workers-all | yes | no | Plain `mastra.startWorkers()` starts Brainkit's safe background-task worker path and real Mastra manager dispatch completes |
| agent/background-tasks/start-workers-surface | yes | no | Plain `mastra.startWorkers()`/`stopWorkers()` returns with the default Mastra worker set present in QuickJS |
| agent/background-tasks/suspend-resume | yes | no | Live OpenAI-backed background tool suspension, out-of-band manager resume, completed result storage, and follow-up recall |
| agent/background-tasks/stream-until-idle | yes | no | Live OpenAI-backed `Agent.streamUntilIdle()` with a Mastra background tool, lifecycle chunks, task persistence, and continuation turn |
| agent/background-tasks/stream-worker-teardown | yes | no | Real Mastra manager stream/worker teardown: stream abort, manager shutdown, `stopWorkers()`, and zero active Brainkit stream/task/worker debug counters |
| agent/background-tasks/timeout-failure | yes | no | Live OpenAI-backed background task `_background.timeoutMs` override emits `background-task-failed` and stores `timed_out` task state |
| agent/callbacks/on-step-finish | yes | no | onStepFinish callback fires during generation |
| agent/channels/core | yes | no | Core Mastra channel orchestration: `AgentChannels`, channel config, route generation, Mastra aggregation, channel reaction tools, and `ChatChannelProcessor` context injection. This is not concrete Slack/Discord/Telegram provider support |
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

### ai/ (32 fixtures)

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

### browser/ (1 fixture)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| browser/session-api | no | no | Deployed TypeScript `kit.browser` lifecycle API reaches modules/browser list/launch/close surface without launching a provider |

### bus/ (17 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| bus/emit-fire-and-forget | no | no | Fire-and-forget emit (2 events) |
| bus/errors/concurrent-publish | no | no | 50 concurrent publishes all succeed |
| bus/errors/large-payload | no | no | Large payload (50KB) fire-and-forget publish |
| bus/errors/schedule-unschedule | no | no | Schedule 5 items, verify all have IDs |
| bus/errors/send-no-heartbeat-adv | no | no | Send without heartbeat: final reply + 2 chunks |
| bus/errors/sendto | no | no | SendTo is service-addressed fire-and-forget |
| bus/errors/streaming-protocol-adv | no | no | Streaming protocol handler registration |
| bus/mailbox-on | no | no | Mailbox on/reply pattern (question/answer) |
| bus/publish-reply | no | no | Request/reply uses `bus.call`; handlers answer with `msg.reply` |
| bus/send-to-service | no | no | `bus.callService` request/reply (greeting) |
| bus/streaming-send-reply | no | no | `bus.callStream` receives 3 chunks + final |
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
| cross-feature/multi-service-chain | no | no | Multi-service fire-and-forget publish chain |
| cross-feature/schedule-triggers-handler | no | no | Schedule/unschedule lifecycle |

### cross-kit/ (1 fixture)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| cross-kit/publish-to-remote | no | no | Cross-kit publish (namespace, tool, self-call, bus) |

**Note:** cross-kit and plugin categories are skipped by the general runner (see `skipCategories`). They run through campaign-specific runners.

### ecosystem/ (1 fixture)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| ecosystem/bare-npm-rejected | no | no | Source package deployment rejects bare npm imports until a named resolver profile exists |

### evals/ (22 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| evals/batch/run-evals | yes | no | Batch evaluation runner (2 scored, keyword positive) |
| evals/scorer/basic | yes | no | Basic scorer (score in range, has runId) |
| evals/scorer/with-llm-judge | yes | no | LLM-as-judge scorer |
| evals/scorer/with-preprocess | yes | no | Scorer with preprocessing (positive score in range) |
| evals/scorer/with-reason | yes | no | Scorer with reason (score=1, has reason string) |

### harness/ (10 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| harness/browser/basic | yes | no | Live OpenAI-backed Harness browser contract: `MastraBrowser`/`BrowserContextProcessor` export surface, Harness browser propagation to a mode agent, browser context on `RequestContext`, execution-time browser tool availability, permission-unblocked browser tool call, lifecycle hooks, and teardown |
| harness/interactive-tools/basic | yes | no | Real Harness interactive tools through deployed TypeScript: `ask_user`, `submit_plan`, `respondToQuestion`, `respondToPlanApproval`, question/plan events, display pending state, approval and rejection results |
| harness/observational-memory/basic | yes | no | Harness observational-memory control surface: exported OM helpers, observer/reflector model defaults and switches, threshold persistence, seeded OM record lookup, `loadOMProgress`, OM stream data-part events, display-state updates, activation/title events, and failure abort handling |
| harness/send-message/basic | yes | no | Live OpenAI-backed Mastra `Harness` through deployed TypeScript: init, `sendMessage`, events, thread/session state, display idle state, and teardown |
| harness/subagents/basic | yes | no | Live OpenAI-backed isolated Harness subagent path: parent model calls built-in `subagent`, child agent resolves through `resolveModel`, subagent events/display snapshots, parent tool lifecycle, and display cleanup |
| harness/subagents/forked | yes | no | Live OpenAI-backed forked Harness subagent path: built-in `subagent` clones the parent memory thread, runs the parent agent on the fork, preserves fork metadata/thread filtering, returns cloned-history context, and records completed display state |
| harness/task-tools/basic | yes | no | Real Harness task tools through deployed TypeScript: task write/update/complete/check, task state mutation, multiple-in-progress rejection, task events, display task state, and teardown |
| harness/tool-approval/basic | yes | no | Live OpenAI-backed Harness approval path: require-approval tool call, `tool_approval_required`, display pending approval, `respondToToolApproval`, resumed stream completion, and display cleanup |
| harness/tool-suspension/basic | yes | no | Live OpenAI-backed Harness suspension path: tool `suspend()`, `tool_suspended`, `agent_end: suspended`, display pending suspension, `respondToToolSuspension`, resumed stream completion, and display cleanup |
| harness/workspace/basic | yes | no | Live OpenAI-backed Harness workspace integration: static workspace lifecycle events, LocalFilesystem access, workspace tool export surface, built-in `subagent` with `allowedWorkspaceTools`, child workspace read-file tool use, display completion, and teardown |

### kit/ (17 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| kit/errors/deploy-throws-init | no | no | Deploy that throws during init (beforeThrow fires) |
| kit/errors/error-code-inspection | no | no | Error code inspection for missing tool, fire-and-forget publish, emit |
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

### memory/ (26 fixtures)

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

### observability/ (9 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| observability/spans/basic | yes | no | Basic span creation (has answer, traceId) |
| observability/trace/basic | yes | no | Basic trace creation (works, has traceId) |

### plugin/ (1 fixture)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| plugin/call-plugin-tool | no | no | Call plugin tools (echo, concat) |

**Note:** Skipped by the general runner. Runs through campaign plugin runner.

### polyfill/ (17 fixtures)

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

### rag/ (22 fixtures)

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

### processors/ (16 fixtures)

See the filesystem count summary for current count. Processor fixture behavior
is covered by `fixtures/ts/processors/*`.

### storage/ (1 fixture)

See the filesystem count summary for current count. Storage fixture behavior is
covered by `fixtures/ts/storage/*`.

### tools/ (15 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| tools/call-from-ts | no | no | Call Go tool from TS (uppercase "HELLO BRAINLET") |
| tools/call-go-tool | no | no | Call Go tool (echo + sum=42) |
| tools/abort-signal | no | no | Tool execution context receives an already-aborted AbortSignal |
| tools/agent-stream-data-persistence | yes | no | Live OpenAI-backed agent stream persists non-transient `data-*` tool chunks into memory |
| tools/agent-stream-data-transient | yes | no | Live OpenAI-backed agent stream omits transient `data-*` chunks from memory while persisting non-transient chunks |
| tools/agent-stream-subagent-writer | yes | no | Live OpenAI-backed parent/sub-agent fullStream bubbles sub-agent tool `writer.custom()` `data-*` chunks |
| tools/agent-stream-writer | yes | no | Live OpenAI-backed agent fullStream tool writer chunks: `writer.write()` -> `tool-output`, `writer.custom()` -> direct `data-*` chunk |
| tools/create-basic | no | no | Create tool in TS (sum=42, registered) |
| tools/create-with-schema | yes | no | Create tool with JSON schema |
| tools/register-list | no | no | Register and list tools |
| tools/register-unregister | no | no | Register, find, unregister, verify removed |
| tools/require-approval | no | no | Tool-level approval suspend/approve flow |
| tools/runtime-context | no | no | RuntimeContext alias + tool execution context access |
| tools/with-output-schema | no | no | Tool output schema result shape |
| tools/with-streaming | no | no | Direct tool stream writer progress chunks |

### vector/ (9 fixtures)

| Path | Needs AI | Needs Container | What it tests |
|------|----------|-----------------|---------------|
| vector/create-upsert-query/libsql | no | libsql-server | LibSQL vector create/upsert/query |
| vector/create-upsert-query/mongodb | no | mongodb | MongoDB vector create/upsert/query |
| vector/create-upsert-query/pgvector | no | postgres | PgVector create/upsert/query |
| vector/methods/libsql | no | libsql-server | LibSQL vector methods (allPassed) |
| vector/methods/mongodb | no | mongodb | MongoDB vector methods (upserted=2) |
| vector/methods/pgvector | no | postgres | PgVector methods (resultCount=2) |

### voice/ (15 fixtures)

See the filesystem count summary for current count. Voice fixture behavior is
covered by `fixtures/ts/voice/*`.

### workflow/ (32 fixtures)

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
| workflow/run/stream-writer | no | no | `run.stream()` delivers `writer.custom()` events and records `writer.write()` bubbling behavior |
| workflow/runtime-context | no | no | Workflow step reads per-run context through RuntimeContext/RequestContext |
| workflow/scheduled/basic | no | no | Declarative `createWorkflow({ schedule })` registers a schedule row, fires through the Brainkit-owned workflow event listener, records trigger history, and tears down scheduler/listener state |
| workflow/scheduled/multi | no | no | Array-form scheduled workflow registers `wf_<workflow>__<schedule>` rows, preserves per-entry schedule data, fires both due schedules, records trigger history, and tears down scheduler/listener state |
| workflow/scheduled/pause-resume | no | no | Paused due schedule rows are skipped without trigger history or `nextFireAt` advancement, then resumed active rows fire through the workflow-event listener |
| workflow/scheduled/redeploy-diff | no | no | Declarative redeploy diffing preserves paused status while updating config, migrates single-form to array-form rows, removes stale `wf_` rows, and preserves user-created schedule rows |
| workflow/scheduled/timezone | no | no | Timezone-aware scheduled workflow stores `America/New_York`, computes initial/recomputed fire times at 09:00 in that zone, fires through the workflow-event listener, and tears down scheduler/listener state |
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
- AI categories (always need OPENAI_API_KEY): agent, ai, observability, composition, voice, processors
- AI segments (need OPENAI_API_KEY anywhere in path): with-agent-step, agent-stream-data-persistence, agent-stream-data-transient, agent-stream-writer, agent-stream-subagent-writer, create-with-schema, vector-query-tool, with-llm-judge, semantic-recall, generate-title, working-memory, rerank, graph-rag, prebuilt
