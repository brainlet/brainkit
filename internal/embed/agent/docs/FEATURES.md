# Mastra Features — Complete Mapping

> Complete inventory of ALL Mastra framework features vs what brainkit exposes at every layer.
> Mastra source: `/Users/davidroman/Documents/code/clones/mastra/packages/core/src/`
> Bundle: `internal/embed/agent/bundle/entry.mjs` → `globalThis.__agent_embed`

---

## 1. Agent

### Mastra Agent Class (`@mastra/core/agent`)

| Feature | Mastra Status | In Bundle | In "agent" module | Notes |
|---------|:------------:|:---------:|:------------------:|-------|
| `new Agent(config)` | Stable | Y | Y | Core agent constructor |
| `agent.generate(messages, opts)` | Stable | Y | Y | Non-streaming generation |
| `agent.stream(messages, opts)` | Stable | Y | Y | Streaming generation |
| `agent.streamUntilIdle(messages, opts)` | Stable | Y | Y | Background-task streaming with continuation turns; Brainkit validates this with a live fixture |
| `agent.resumeStreamUntilIdle(resumeData, opts)` | Stable | Y | Y | Resume-flavored background-task idle stream; still needs a dedicated fixture |
| `agent.network(messages, opts)` | Stable | Y | Y | Supervisor/delegation mode |
| `agent.generateLegacy()` | Deprecated | Y | — | Old API, don't expose |
| `agent.streamLegacy()` | Deprecated | Y | — | Old API, don't expose |
| `agent.listTools()` | Stable | Y | — | Query agent's tools |
| `agent.getModel()` | Stable | Y | — | Get resolved model |
| `agent.getLLM()` | Stable | Y | — | Get LLM instance |
| `agent.getInstructions()` | Stable | Y | — | Get resolved instructions |
| `agent.getDescription()` | Stable | Y | — | Get description |
| `agent.getMemory()` | Stable | Y | — | Get memory instance |
| `agent.getWorkspace()` | Stable | Y | — | Get workspace instance |
| `agent.listAgents()` | Stable | Y | — | List sub-agents |
| `agent.listWorkflows()` | Stable | Y | — | List agent workflows |
| `agent.listScorers()` | Stable | Y | — | List agent scorers |
| `agent.getVoice()` | Stable | Y | — | Get voice config |
| `agent.generateTitleFromUserMessage()` | Stable | Y | — | Auto-title generation |

### Agent Config

| Config Field | Description | Supported |
|-------------|-------------|:---------:|
| `id` | Unique identifier | Y |
| `name` | Display name | Y |
| `description` | Agent description | Y |
| `instructions` | System instructions (string or dynamic) | Y |
| `model` | Language model (static or dynamic) | Y |
| `tools` | Tool definitions (static or dynamic resolver) | Y |
| `toolChoice` | auto/none/required/tool-specific | Y |
| `memory` | Memory instance | Y |
| `agents` | Sub-agents for network/delegation | Y |
| `workflows` | Agent workflows | Y |
| `workspace` | Workspace instance | Y |
| `voice` | Voice configuration | Y |
| `backgroundTasks` | Agent-level background task tool config | Y |
| `maxSteps` | Max tool-call rounds | Y |
| `defaultOptions` | Default call options | Y |
| `inputProcessors` | Input middleware chain | Y |
| `outputProcessors` | Output middleware chain | Y |
| `scorers` | Evaluation scorers | Y |
| `requestContextSchema` | Schema for RequestContext | Y |
| `providerOptions` | Per-provider settings | Y |
| `maxProcessorRetries` | Retry count for processors | Y |

### Agent Execution Options

| Option | Description |
|--------|-------------|
| `maxSteps` | Override max tool-call rounds |
| `memory` | Memory options (thread, resource) |
| `requestContext` | RequestContext for dynamic resolvers |
| `output` | Structured output schema |
| `toolChoice` | Override tool selection |
| `providerOptions` | Per-provider settings |
| `onStepFinish` | Callback per step |
| `onFinish` | Callback when done |
| `experimental_output` | Experimental structured output |

### Agent Network (Delegation)

| Feature | Description |
|---------|-------------|
| `agents` config | Sub-agents become tools |
| `agent.network()` | Supervisor delegates to sub-agents |
| Delegation hooks | `onDelegationStart`, `onDelegationComplete` |
| Message filtering | `messageFilter` to control context passed |
| Iteration hooks | `onIterationComplete` for multi-turn |
| `isTaskComplete` | Custom completion check |

---

## 2. Tools

### Mastra Tool Class (`@mastra/core/tools`)

| Feature | Description | In Bundle |
|---------|-------------|:---------:|
| `createTool(config)` | Create a typed tool | Y |
| `Tool` class | Tool instance (returned by createTool) | Y |
| `tool.execute(input, context)` | Execute the tool | Y |
| `tool.id` | Tool identifier | Y |
| `tool.description` | Tool description | Y |
| `tool.inputSchema` | Zod input schema | Y |
| `tool.outputSchema` | Zod output schema | Y |
| `tool.suspendSchema` | Schema for suspend data (HITL) | Y |
| `tool.resumeSchema` | Schema for resume data (HITL) | Y |
| `tool.requireApproval` | Require approval before execution | Y |
| `ToolStream` | Streaming tool results | Y |
| `isVercelTool` | Check if AI SDK tool | Y |

### Tool Config

| Field | Description |
|-------|-------------|
| `id` | Tool identifier (required) |
| `description` | Human-readable description |
| `inputSchema` | Zod schema for input validation |
| `outputSchema` | Zod schema for output validation |
| `execute` | Async execution function `(input, context) => result` |
| `suspendSchema` | Schema for HITL suspend payload |
| `resumeSchema` | Schema for HITL resume payload |
| `requireApproval` | Boolean or function for approval flow |
| `requestContextSchema` | Schema for per-tool request context |
| `lifecycle` | Lifecycle hooks (onStart, onComplete, etc.) |
| `background` | Background task execution config |

### Tool Execution Context

| Field | Description |
|-------|-------------|
| `context.mastra` | Mastra instance (if registered) |
| `context.requestContext` | Per-request context |
| `context.suspend(data)` | Suspend execution (HITL) |
| `context.resume` | Resume data (after suspend) |
| `context.stream` | Stream writer for tool output streaming |

### Background Tasks

| Feature | Description | In Bundle | Fixture Coverage |
|---------|-------------|:---------:|------------------|
| `new BackgroundTaskManager(config)` | Background task lifecycle manager | Y | `agent/background-tasks/stream-until-idle`, `progress-output`, `suspend-resume`, `timeout-failure`, `cancel-running`, `retry-success`, `concurrency-queue`, `lifecycle-callbacks`, `manager-stream-filter-abort`, `start-workers-all`, `stream-worker-teardown` |
| `createBackgroundTask(manager, options)` | Enqueue and track a background task | Y | covered through agent tool background execution |
| `globalConcurrency` / `perAgentConcurrency` queueing | Queue pending tasks until a global/per-agent slot frees | Y | real Mastra manager fixture |
| `onTaskComplete` / `onTaskFailed` and per-task hooks | Manager and per-task lifecycle callbacks | Y | real Mastra manager fixture |
| `BackgroundTaskManager.stream({ agentId, taskId, abortSignal })` | Filtered background-task event streams, caller-driven stream close, and shutdown-owned stream abort cleanup | Y | real Mastra manager fixture |
| `mastra.startWorkers()` safe worker mapping | Starts Brainkit's supported background-task worker path and a Brainkit-owned workflow-event listener without enabling pull orchestration/scheduler workers in QuickJS; `stopWorkers()` clears tracked worker/listener state | Y | real Mastra manager fixture; scheduled workflow fixture |
| `generateBackgroundTaskSystemPrompt()` | Prompt helper for background tools | Y | type/surface covered |
| `background-task-running` | Running lifecycle chunk | Y | live OpenAI-backed fixture |
| `background-task-output` | Progress/output lifecycle chunk | Y | live OpenAI-backed fixture |
| `background-task-completed` | Completed lifecycle chunk and result injection | Y | live OpenAI-backed fixture |
| `maxRetries` / task retry count | Transient failure retry then completion | Y | live OpenAI-backed fixture |
| `background-task-failed` / `timed_out` | Failure lifecycle chunk and timeout persistence | Y | live OpenAI-backed fixture |
| `background-task-cancelled` / `cancelled` | Cancel lifecycle chunk, storage state, and tool abort propagation | Y | live OpenAI-backed fixture |
| `background-task-suspended/resumed` | Suspend/resume lifecycle | Y | live OpenAI-backed fixture |

Brainkit embeds a same-process inline background-task executor for per-task
executor closures. This keeps the Mastra lifecycle events and result injection
real inside QuickJS without routing stream-returning agent workflows through
Mastra's generic orchestration worker. Plain `mastra.startWorkers()` is mapped
to Brainkit's safe background-task worker path and an in-process workflow-event
push listener in this runtime. Generic pull orchestration and scheduler worker
process loops still need explicit Brainkit module owners before they are
claimed as supported.
The embedded bridge tracks background-task streams, abort controllers, active
task contexts, and worker instances so fixtures can assert teardown rather than
only checking that APIs returned.

---

## 3. Workflows

### Mastra Workflow (`@mastra/core/workflows`)

| Feature | Description | In Bundle |
|---------|-------------|:---------:|
| `createWorkflow(config)` | Create a workflow | Y |
| `createStep(config)` | Create a workflow step | Y |
| `workflow.then(step)` | Chain steps sequentially | Y |
| `workflow.parallel(steps)` | Run steps in parallel | Y |
| `workflow.branch(config)` | Conditional branching | Y |
| `workflow.forEach(config)` | Loop over items | Y |
| `workflow.commit()` | Finalize workflow definition | Y |
| `workflow.createRun(opts)` | Create a run instance | Y |
| `createWorkflow({ schedule })` | Declarative Mastra workflow schedule; auto-promotes to evented workflow | Y |
| `WorkflowScheduler.tick()` | Scheduler tick publishes due workflow events through Brainkit's in-process workflow-event listener | Y |
| `run.start({inputData})` | Start execution | Y |
| `run.resume({resumeData, step})` | Resume after suspend | Y |
| `run.cancel()` | Cancel execution | Y |
| `run.status` | Current status | Y |
| `run.runId` | Run identifier | Y |
| `mapVariable()` | Map data between steps | Y |

### Workflow Config

| Field | Description |
|-------|-------------|
| `id` | Workflow identifier |
| `inputSchema` | Zod schema for input |
| `outputSchema` | Zod schema for output |
| `schedule` | Single or array Mastra schedule config (`cron`, `timezone`, `inputData`, `initialState`, `requestContext`, `metadata`) |

### Scheduled Workflows

Brainkit supports the embedded Mastra scheduled workflow happy path through:

- `fixtures/ts/workflow/scheduled/basic`: a workflow declared with `schedule`
  auto-promotes to the evented engine, registers a durable Mastra schedule row,
  fires a due tick, runs through the Brainkit-owned workflow-event listener,
  records trigger history, and tears down listener plus scheduler state.
- `fixtures/ts/workflow/scheduled/timezone`: a daily `09:00`
  `America/New_York` schedule proves Croner timezone conversion works in
  QuickJS via Brainkit's `Intl.DateTimeFormat(...).formatToParts()` bridge.
- `fixtures/ts/workflow/scheduled/multi`: an array-form scheduled workflow
  proves one persisted row per stable schedule id, per-entry input/context/
  metadata preservation, trigger history for both due rows, and listener/
  scheduler teardown.
- `fixtures/ts/workflow/scheduled/pause-resume`: paused due rows are skipped
  without trigger history or `nextFireAt` advancement, then fire once resumed
  to `active`.
- `fixtures/ts/workflow/scheduled/redeploy-diff`: declarative redeploy
  registration preserves paused status while applying cron/target/metadata
  changes, migrates single-form rows to array-form rows, deletes stale `wf_`
  rows, and preserves user-created schedules.

### Step Config

| Field | Description |
|-------|-------------|
| `id` | Step identifier |
| `inputSchema` | Zod schema for step input |
| `outputSchema` | Zod schema for step output |
| `execute` | Async function `({inputData, mapiData}) => result` |

### Workflow Run Result

| Field | Description |
|-------|-------------|
| `status` | "completed" / "suspended" / "failed" |
| `result` | Output data |
| `runId` | Run identifier |
| `steps` | Step results map |

---

## 4. Memory

### Mastra Memory (`@mastra/memory`)

| Feature | Description | In Bundle |
|---------|-------------|:---------:|
| `new Memory(config)` | Create memory instance | Y |
| `memory.createThread(opts)` | Create conversation thread | Y |
| `memory.getThreadById({threadId})` | Get thread by ID | Y |
| `memory.listThreads(filter)` | List threads | Y |
| `memory.saveMessages({threadId, messages})` | Save messages to thread | Y |
| `memory.recall({threadId, query})` | Semantic recall from thread | Y |
| `memory.deleteThread(threadId)` | Delete a thread | Y |
| `memory.updateThread({threadId, ...})` | Update thread metadata | Y |
| `memory.saveThread({...})` | Save/create thread | Y |
| `memory.getWorkingMemory({threadId})` | Get working memory state | Y |
| `memory.updateWorkingMemory({...})` | Update working memory | Y |
| `memory.updateMessages({...})` | Update existing messages | Y |
| `memory.deleteMessages({...})` | Delete messages | Y |
| `memory.cloneThread({...})` | Clone a thread | Y |
| `memory.listTools()` | Memory-related tools (working memory) | Y |
| `memory.getSystemMessage()` | Get memory system prompt | Y |
| `MockMemory` | In-memory mock for testing | Y |

### Memory Config

| Field | Description |
|-------|-------------|
| `storage` | Storage backend (InMemoryStore, LibSQLStore, etc.) |
| `vector` | Vector store for semantic recall (optional) |
| `embedder` | Embedding function/model (required if vector) |
| `options.lastMessages` | Number of recent messages to include |
| `options.semanticRecall` | Semantic recall configuration |
| `options.workingMemory` | Working memory configuration |
| `options.generateTitle` | Auto-generate thread titles |
| `options.observationalMemory` | 3-tier memory compression |

### Observational Memory

| Feature | Description |
|---------|-------------|
| Observer agent | Compresses messages → observations |
| Reflector agent | Compresses observations → reflections |
| Thresholds | Configurable token thresholds for compression |
| Custom model | Per-tier model override |

---

## 5. Storage Backends

| Backend | Package | In Bundle | Constructor |
|---------|---------|:---------:|-------------|
| `InMemoryStore` | `@mastra/core/storage` | Y | `new InMemoryStore()` |
| `LibSQLStore` | `@mastra/libsql` | Y | `new LibSQLStore({url, authToken})` |
| `UpstashStore` | `@mastra/upstash` | Y | `new UpstashStore({url, token})` |
| `PostgresStore` | `@mastra/pg` | Y | `new PostgresStore({connectionString})` |
| `MongoDBStore` | `@mastra/mongodb` | Y | `new MongoDBStore({uri, dbName})` |

### Storage API (all backends)

| Method | Description |
|--------|-------------|
| `createThread(thread)` | Create a thread |
| `getThreadById({threadId})` | Get thread |
| `listThreads(filter)` | List threads |
| `deleteThread(threadId)` | Delete thread |
| `saveMessages({threadId, messages})` | Save messages |
| `getMessages({threadId})` | Get messages |
| `persistWorkflowSnapshot(snapshot)` | Workflow state persistence |
| `getWorkflowSnapshot({runId})` | Get workflow state |

---

## 6. Vector Stores

| Store | Package | In Bundle | Constructor |
|-------|---------|:---------:|-------------|
| `LibSQLVector` | `@mastra/libsql` | Y | `new LibSQLVector({connectionUrl, authToken})` |
| `PgVector` | `@mastra/pg` | Y | `new PgVector({connectionString})` |
| `MongoDBVector` | `@mastra/mongodb` | Y | `new MongoDBVector({uri, dbName})` |

### Vector Store API

| Method | Description |
|--------|-------------|
| `createIndex({indexName, dimension, metric})` | Create vector index |
| `listIndexes()` | List index names |
| `describeIndex(name)` | Get index info |
| `deleteIndex(name)` | Delete index |
| `upsert({indexName, vectors})` | Insert/update vectors |
| `query({indexName, queryVector, topK})` | Similarity search |
| `deleteVectors({indexName, ids})` | Delete vectors |

---

## 7. Evals (Scorers)

### Scorer Infrastructure (`@mastra/core/evals`)

| Feature | Description | In Bundle |
|---------|-------------|:---------:|
| `createScorer(config)` | Create custom scorer | Y |
| `runEvals(config)` | Run evaluation suite | Y |

### Pre-built Scorers (`@mastra/evals/scorers/prebuilt`)

**Rule-based (no LLM needed):**

| Scorer | Description | In Bundle |
|--------|-------------|:---------:|
| `createCompletenessScorer` | Check output completeness | Y |
| `createTextualDifferenceScorer` | Text diff scoring | Y |
| `createKeywordCoverageScorer` | Keyword coverage check | Y |
| `createContentSimilarityScorer` | Content similarity | Y |
| `createToneScorer` | Tone analysis | Y |

**LLM-based (require judge model):**

| Scorer | Description | In Bundle |
|--------|-------------|:---------:|
| `createHallucinationScorer` | Detect hallucinations | Y |
| `createFaithfulnessScorer` | Faithfulness to context | Y |
| `createAnswerRelevancyScorer` | Answer relevance | Y |
| `createAnswerSimilarityScorer` | Answer similarity | Y |
| `createBiasScorer` | Bias detection | Y |
| `createToxicityScorer` | Toxicity detection | Y |
| `createContextPrecisionScorer` | Context precision | Y |
| `createContextRelevanceScorerLLM` | Context relevance (LLM) | Y |
| `createNoiseSensitivityScorerLLM` | Noise sensitivity | Y |
| `createPromptAlignmentScorerLLM` | Prompt alignment | Y |
| `createToolCallAccuracyScorerLLM` | Tool call accuracy | Y |

---

## 8. Processors (Input/Output Middleware)

### Built-in Processors (`@mastra/core/processors`)

| Processor | Category | Description | In Bundle |
|-----------|----------|-------------|:---------:|
| `ModerationProcessor` | Security | Content moderation | Y |
| `PromptInjectionDetector` | Security | Detect prompt injection | Y |
| `PIIDetector` | Security | Detect PII in content | Y |
| `SystemPromptScrubber` | Security | Remove system prompt leaks | Y |
| `UnicodeNormalizer` | Data | Normalize unicode text | Y |
| `LanguageDetector` | Data | Detect input language | Y |
| `TokenLimiterProcessor` | Stream | Limit token output | Y |
| `BatchPartsProcessor` | Stream | Batch stream parts | Y |
| `StructuredOutputProcessor` | Stream | Enforce structured output | Y |
| `ToolCallFilter` | Tool | Filter tool calls | Y |
| `ToolSearchProcessor` | Tool | Search-based tool selection | Y |
| `AgentsMDInjector` | Instructions | Inject AGENTS.md reminders | Y |
| `SkillsProcessor` | Instructions | Inject skill metadata | Y |
| `SkillSearchProcessor` | Tool | Search and load skills on demand | Y |
| `WorkspaceInstructionsProcessor` | Workspace | Inject workspace instructions | Y |
| `ResponseCache` | Cache | Cache LLM step responses via a server cache | Y |
| `buildResponseCacheKey` | Cache | Deterministic response-cache key helper | Y |
| `InMemoryServerCache` | Cache | Local/dev server cache backend | Y |
| `MastraServerCache` | Cache | Abstract server cache base | Y |

---

## 9. RAG (Retrieval-Augmented Generation)

### Document Processing (`@mastra/rag`)

| Feature | Description | In Bundle |
|---------|-------------|:---------:|
| `MDocument` | Document class for chunking | Y |
| `MDocument.fromText(text)` | Create from plain text | Y |
| `MDocument.fromMarkdown(md)` | Create from markdown | Y |
| `document.chunk(options)` | Chunk document | Y |
| `GraphRAG` | Knowledge graph RAG | Y |

### RAG Tools

| Tool | Description | In Bundle |
|------|-------------|:---------:|
| `createVectorQueryTool(config)` | Vector similarity search tool | Y |
| `createDocumentChunkerTool(config)` | Document chunking tool | Y |
| `createGraphRAGTool(config)` | Graph RAG query tool | Y |

### Reranking

| Feature | Description | In Bundle |
|---------|-------------|:---------:|
| `rerank(config)` | Rerank results | Y |
| `rerankWithScorer(config)` | Rerank using custom scorer | Y |

---

## 10. Workspace

### Workspace Class (`@mastra/core/workspace`)

| Feature | Description | In Bundle |
|---------|-------------|:---------:|
| `new Workspace(config)` | Create workspace | Y |
| `workspace.init()` | Initialize (create indexes, start LSP) | Y |
| `workspace.destroy()` | Cleanup | Y |
| `workspace.search(query, opts)` | Search content | Y |
| `workspace.index(path, content)` | Index a file | Y |
| `workspace.getInfo()` | Get workspace info | Y |
| `workspace.getInstructions()` | Get workspace instructions | Y |

### Providers

| Provider | Description | In Bundle |
|----------|-------------|:---------:|
| `LocalFilesystem` | Local file access via Go bridges | Y |
| `LocalSandbox` | Local command execution via Go bridges | Y |
| `CompositeFilesystem` | Multi-root filesystem | Y |

### Workspace Tools

| Feature | Description | In Bundle | Fixture Coverage |
|---------|-------------|:---------:|------------------|
| `WORKSPACE_TOOLS` / `WORKSPACE_TOOLS_PREFIX` | Stable tool-id constants for workspace tool allowlists | Y | `fixtures/ts/harness/workspace/basic`, `fixtures/ts/polyfill/agent-surface` |
| `createWorkspaceTools(workspace)` | Build Mastra workspace tools bound to a Workspace | Y | `fixtures/ts/harness/workspace/basic` |
| `readFileTool`, `writeFileTool`, `editFileTool`, `listFilesTool`, `deleteFileTool`, `fileStatTool`, `mkdirTool`, `searchTool`, `indexContentTool`, `executeCommandTool` | Built-in workspace tool instances | Y | surface covered by `fixtures/ts/polyfill/agent-surface`; read-file execution covered through `fixtures/ts/harness/workspace/basic` |
| `requireWorkspace`, `requireFilesystem`, `requireSandbox`, `resolveToolConfig` | Workspace tool helper utilities | Y | surface covered by `fixtures/ts/polyfill/agent-surface` |

Brainkit precomputes workspace tool schemas before SES lockdown, matching the
Harness built-in tool treatment. `fixtures/ts/harness/workspace/basic` proved
that without this, Mastra workspace tool preparation can fail during late
schema traversal under QuickJS/SES.

### Workspace Config

| Field | Description |
|-------|-------------|
| `id` | Workspace identifier |
| `name` | Display name |
| `filesystem` | Filesystem provider (required) |
| `sandbox` | Sandbox provider (optional) |
| `bm25` | BM25 keyword search (boolean or config) |
| `vectorStore` | Vector store for semantic search |
| `embedder` | Embedding function |
| `tools` | Per-tool configuration (rename, enable, approval) |
| `skills` | Skill directory paths |
| `lsp` | LSP configuration for diagnostics |

---

## 11. Observability

| Feature | Package | In Bundle |
|---------|---------|:---------:|
| `Observability` | `@mastra/observability` | Y |
| `DefaultExporter` | `@mastra/observability` | Y |
| `SensitiveDataFilter` | `@mastra/observability` | Y |

---

## 12. Harness

| Feature | Description | In Bundle | Fixture Coverage |
|---------|-------------|:---------:|------------------|
| `Harness` | Orchestration layer | Y | `fixtures/ts/harness/send-message/basic`, `fixtures/ts/harness/subagents/basic`, `fixtures/ts/harness/subagents/forked`, `fixtures/ts/harness/tool-approval/basic`, `fixtures/ts/harness/tool-suspension/basic`, `fixtures/ts/harness/workspace/basic`, `fixtures/ts/harness/browser/basic`, `fixtures/ts/harness/observational-memory/basic` |
| `defaultDisplayState` / `defaultOMProgressState` | Harness display-state defaults | Y | `fixtures/ts/harness/observational-memory/basic` |
| `assignTaskIds` / `parseSubagentMeta` | Harness helper exports | Y | `fixtures/ts/harness/observational-memory/basic` |
| `askUserTool` | Built-in tool for user questions | Y | `fixtures/ts/harness/interactive-tools/basic` |
| `submitPlanTool` | Built-in tool for plan submission | Y | `fixtures/ts/harness/interactive-tools/basic` |
| `taskWriteTool` | Built-in tool for task management | Y | `fixtures/ts/harness/task-tools/basic` |
| `taskUpdateTool` | Built-in tool for task updates | Y | `fixtures/ts/harness/task-tools/basic` |
| `taskCompleteTool` | Built-in tool for task completion | Y | `fixtures/ts/harness/task-tools/basic` |
| `taskCheckTool` | Built-in tool for task checking | Y | `fixtures/ts/harness/task-tools/basic` |

### Harness Features

| Feature | Description |
|---------|-------------|
| Modes | Multiple agent modes with model switching |
| Threads | Thread management (create, switch, delete, clone) |
| Tool approval | Approval workflow for tool execution |
| Plan approval | Plan submission and approval |
| State management | Custom state schema with display state |
| Permissions | Category-based permission rules |
| Subagents | Constrained subagent management |
| Browser | Harness-level browser propagation to mode agents |
| Observational memory | OM integration with model switching |
| Token tracking | Usage tracking across sessions |

Brainkit currently proves the live Harness happy path with
`fixtures/ts/harness/send-message/basic`: a real OpenAI-backed `Agent`,
Harness `init()`, `sendMessage()`, event emission, thread/session state,
persisted memory messages, display-idle state, and teardown. The embedded
compatibility layer also waits for Brainkit's delayed first stream chunk before
returning from idle `Harness.sendMessage()` calls.

Brainkit also proves Harness tool approval with
`fixtures/ts/harness/tool-approval/basic`: a live OpenAI-backed `sendMessage()`
run reaches `tool_approval_required`, `respondToToolApproval()` approves the
tool, the approval resume stream is consumed, and display state returns idle.
The embedded compatibility layer consumes approval/decline resume streams
directly because upstream Harness does not process the returned
`approveToolCall()` stream on Brainkit's QuickJS subscription path.

Brainkit proves Harness tool suspension with
`fixtures/ts/harness/tool-suspension/basic`: a live OpenAI-backed
`sendMessage()` run executes a tool that calls `suspend()`, emits
`tool_suspended`, ends the first run as `suspended`, resumes through
`respondToToolSuspension()`, and completes with display state cleaned up.

Brainkit proves Harness subagents with two live fixtures:
`fixtures/ts/harness/subagents/basic` covers the isolated child-agent path where
the parent model calls the built-in `subagent` tool and the child agent resolves
through `resolveModel`; `fixtures/ts/harness/subagents/forked` covers the forked
path where the built-in tool clones the parent memory thread, runs the parent
agent on the fork, hides fork threads by default, and preserves fork metadata.

Brainkit proves Harness workspace integration with
`fixtures/ts/harness/workspace/basic`: static `Workspace` init/ready/destroy
events, LocalFilesystem access, workspace tool constants and helpers exported
through deployed TypeScript, a real OpenAI-backed Harness subagent constrained
with `allowedWorkspaceTools`, and the child agent reading a secret from disk via
Mastra's workspace read-file tool.

Brainkit proves the core Harness browser contract with
`fixtures/ts/harness/browser/basic`: `MastraBrowser` and
`BrowserContextProcessor` are exported through the curated `"agent"` module and
deployed TypeScript endowments, a Harness-level browser propagates to a static
mode agent, browser context is written to `RequestContext`, browser tools remain
absent from `agent.listTools()` but execute during a live OpenAI-backed run, and
the browser provider lifecycle is closed explicitly. Concrete provider packages
(`@mastra/agent-browser`, `@mastra/stagehand`, BrowserViewer), browser process
ownership, CDP, and screencast remain future module/resolver work.

Brainkit proves the Harness observational-memory control surface with
`fixtures/ts/harness/observational-memory/basic`: Harness helper exports
(`defaultDisplayState`, `defaultOMProgressState`, `assignTaskIds`,
`parseSubagentMeta`), observer/reflector default model resolution and switch
events, threshold persistence in thread metadata, seeded OM record lookup,
`loadOMProgress()`, OM stream data-part event mapping, display-state updates,
activation/title events, and OM failure abort handling.

Brainkit also proves built-in Harness tool behavior with
`fixtures/ts/harness/task-tools/basic` and
`fixtures/ts/harness/interactive-tools/basic`: task state mutation, structured
task checking, multiple in-progress rejection, `ask_user` question resolution,
and `submit_plan` approval/rejection resolution through real Harness request
contexts.

---

## 13. Additional Utilities

### RequestContext (`@mastra/core/request-context`)

| Feature | Description | In Bundle |
|---------|-------------|:---------:|
| `new RequestContext(entries?)` | Create context | Y |
| `ctx.get(key)` | Get value | Y |
| `ctx.set(key, value)` | Set value | Y |
| `ctx.has(key)` | Check existence | Y |

### ModelRouterEmbeddingModel (`@mastra/core/llm`)

| Feature | Description | In Bundle |
|---------|-------------|:---------:|
| `new ModelRouterEmbeddingModel(id)` | Resolve embedding model by "provider/model" string | Y |

### Zod (`zod`)

| Feature | Description | In Bundle |
|---------|-------------|:---------:|
| `z` | Zod schema builder | Y |
| `toJSONSchema` | Convert Zod to JSON Schema | Y |

---

## 14. What's NOT in the Bundle

Features available in Mastra but NOT imported in the agent-embed bundle:

| Feature | Package | Reason |
|---------|---------|--------|
| `Mastra` class (full) | `@mastra/core/mastra` | brainkit IS the runtime, not Mastra |
| MCP server classes | `@mastra/core/mcp` | brainkit has its own MCP client (`internal/mcp`) |
| Voice providers (beyond default) | `@mastra/core/voice` | Not yet needed |
| Server/deployer | `@mastra/core/server` | brainkit is not a web server |
| Auth | `@mastra/core/auth` | brainkit handles auth differently |
| DI container | `@mastra/core/di` | Not needed in embedded context |
| Bundler | `@mastra/core/bundler` | Not applicable |
| Editor | `@mastra/core/editor` | Not applicable |
| Events system | `@mastra/core/events` | brainkit has its own bus |
| Integration | `@mastra/core/integration` | Not yet needed |
| Filesystem-based storage | `@mastra/core/storage` | brainkit uses libsql/pg bridges |
| A2A protocol | `@mastra/core/a2a` | Not yet needed |
| TTS providers | `@mastra/core/tts` | Not yet needed |
| Tool builder pattern | `@mastra/core/tools/tool-builder` | Experimental |

---

## 15. Go Wrapper (`internal/embed/agent`) Coverage

The Go wrapper provides a typed Go API for creating agents and calling generate/stream.

| Go Method | Mastra Feature | Supported | Notes |
|-----------|---------------|:---------:|-------|
| `Sandbox.CreateAgent(cfg)` | `new Agent(config)` | Y | Subset of config fields |
| `Agent.Generate(ctx, params)` | `agent.generate()` | Y | Prompt, Messages, MaxSteps |
| `Agent.Stream(ctx, params)` | `agent.stream()` | Y | OnToken callback, blocks until done |
| `Agent.Close()` | Cleanup | Y | Removes from JS registry |

### Go Agent Config (subset of Mastra)

| Field | Supported | Notes |
|-------|:---------:|-------|
| `Name` | Y | |
| `Model` | Y | "provider/model-id" string |
| `Instructions` | Y | |
| `Tools` | Y | Go callbacks via QuickJS bridge |
| `MaxSteps` | Y | Default: 5 |
| `Description` | Y | |
| `ToolChoice` | Y | |
| Memory | **NO** | Go agent has no memory config |
| Agents (network) | **NO** | Go can't configure sub-agents |
| Processors | **NO** | Go can't configure processors |
| Workspace | **NO** | Go can't configure workspace |
| Dynamic resolvers | **NO** | Go can't pass JS functions |

### Go Agent Limitations

- `Agent.Stream()` blocks until stream completes — no real-time chunk access from Go
- Go tools use synchronous QuickJS callbacks (no async tool execution)
- No access to `agent.network()` from Go
- No observational memory from Go
- No structured output from Go

---

## 16. Summary — What brainkit exposes from Mastra

### In the `"agent"` module (direct re-exports)

| Export | Source |
|--------|--------|
| `Agent` | `@mastra/core/agent` |
| `createTool` | `@mastra/core/tools` |
| `createWorkflow`, `createStep` | `@mastra/core/workflows` |
| `Memory` | `@mastra/memory` |
| `InMemoryStore` | `@mastra/core/storage` |
| `LibSQLStore`, `LibSQLVector` | `@mastra/libsql` |
| `UpstashStore` | `@mastra/upstash` |
| `PostgresStore`, `PgVector` | `@mastra/pg` |
| `MongoDBStore`, `MongoDBVector` | `@mastra/mongodb` |
| `ModelRouterEmbeddingModel` | `@mastra/core/llm` |
| `RequestContext` | `@mastra/core/request-context` |
| `Workspace`, `LocalFilesystem`, `LocalSandbox`, `CompositeFilesystem`, `WORKSPACE_TOOLS`, `createWorkspaceTools`, workspace tool helpers | `@mastra/core/workspace` |
| `MastraBrowser`, `BrowserContextProcessor` | `@mastra/core/browser` |
| `MDocument`, `GraphRAG` | `@mastra/rag` |
| `createVectorQueryTool`, `createDocumentChunkerTool`, `createGraphRAGTool` | `@mastra/rag` |
| `rerank`, `rerankWithScorer` | `@mastra/rag` |
| `Observability`, `DefaultExporter` | `@mastra/observability` |
| `createScorer`, `runEvals` | `@mastra/core/evals` |

### In Compartment endowments (available to deployed .ts code)

All of the above plus:
- 5 rule-based scorers (createCompletenessScorer, etc.)
- 11 LLM-based scorers (createHallucinationScorer, etc.)
- 11 processors (ModerationProcessor, PIIDetector, etc.)
- Harness + built-in tools (askUserTool, submitPlanTool, etc.)
- Workspace tool constants/helpers used by deployed TypeScript packages
- Browser base/context processor used by deployed TypeScript packages
- `SensitiveDataFilter` from observability
