# agent/ Fixtures

Tests the Mastra Agent API: creation, generation, streaming, tool use, memory backends, subagent delegation, HITL approval, callbacks, and integration with workflows.

All agent fixtures require AI (OPENAI_API_KEY) since the `agent` category is in `aiCategories`.

## Fixtures

### generate/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| basic | yes | no | Agent creation and basic `generate()` call; asserts usage metadata and finishReason="stop" |
| active-tools | yes | no | `activeTools` option limits which tools the model can use per call (only `add` exposed, not `multiply`) |
| dynamic-instructions | yes | no | Instructions as a function receiving `requestContext`; two calls with different keywords produce different responses |
| dynamic-model | yes | no | Model as a function reading `requestContext` to select provider/model at runtime |
| dynamic-tools | yes | no | Tools as a function of `requestContext`; different `mode` values expose different tool sets per call |
| instructions-override | yes | no | Per-call `instructions` option overrides the agent's base instructions |
| multi-step | yes | no | Multi-step tool loop: agent calls a `lookup` tool then incorporates its result; asserts `steps.length > 1` |
| options-passthrough | yes | no | Comprehensive generate options: temperature, onStepFinish, onFinish, per-call instructions, maxSteps=1 |
| structured-output | yes | no | `output` option with a Zod schema for structured object extraction (name, age, hobbies) |
| with-context-messages | yes | no | `context` option prepends prior conversation messages so the agent can answer from them |
| with-tools | yes | no | Agent with a locally-defined `add` tool; asserts tool calls appear in result |

### stream/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| basic | yes | no | `agent.stream()` with `textStream` iteration; asserts real-time chunks are received |
| with-tools | yes | no | `agent.stream()` with tool calls mid-stream; consumes textStream and verifies text output |

### memory/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| inmemory | yes | no | Agent with `InMemoryStore` memory; two-call conversation recall across same thread |
| postgres | yes | postgres | Agent with `PostgresStore` memory; two-call recall with real Postgres container |
| mongodb | yes | mongodb | Agent with `MongoDBStore` memory; two-call recall with real MongoDB container |
| libsql | yes | no | Agent with `LibSQLStore` memory; two-call recall using LIBSQL_URL |
| upstash | yes | no (credential) | Agent with `UpstashStore` memory; two-call recall using UPSTASH_REDIS_REST_URL credential |

### subagents/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| basic | yes | no | Supervisor agent with `agents` config delegates math to a sub-agent with a calculator tool; asserts correct answer (105) |
| constrained | yes | no | Supervisor with constrained sub-agents (explorer has view+search, coder has view+search+edit); delegates exploration task |
| network-delegation | yes | no | Supervisor delegates a math question to a named sub-agent via `agents` config; asserts multi-step response |

### hitl/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| bus-approval | yes | no | Bus-based HITL: `generateWithApproval` routes tool approval through a bus topic; Go test runner auto-approves |

### tools/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| with-local-tool | yes | no | Agent uses a locally-defined Zod-schema tool (`add`); asserts exactly 1 tool call in result |
| with-registered-tool | yes | no | Agent uses a platform-registered tool (`tool("multiply")` from Go); asserts tool was used and text returned |

### callbacks/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| on-step-finish | yes | no | `onStepFinish` callback fires for each agent step; asserts callback count > 0 |

### background-tasks/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| cancel-running | yes | no | Live OpenAI-backed `Agent.stream()` dispatches a background task, `BackgroundTaskManager.cancel()` emits `background-task-cancelled`, stores `cancelled`, and aborts the running tool |
| concurrency-queue | yes | no | Real Mastra `createBackgroundTask()` queue semantics with `globalConcurrency: 2` and `perAgentConcurrency: 1`; asserts queued pending task starts after same-agent slot frees |
| lifecycle-callbacks | yes | no | Real Mastra `createBackgroundTask()` lifecycle callbacks; asserts per-task execution/chunk/result/complete/failed hooks and manager `onTaskComplete`/`onTaskFailed` hooks |
| manager-stream-filter-abort | yes | no | Real Mastra `BackgroundTaskManager.stream()` filter behavior for `agentId` and `taskId`, including `AbortSignal` stream closure |
| progress-output | yes | no | Live OpenAI-backed `Agent.streamUntilIdle()` with a background tool that emits progress via `writer.write`; asserts `background-task-output` chunks and completed task result |
| retry-success | yes | no | Live OpenAI-backed `Agent.streamUntilIdle()` with a background tool that fails once then succeeds; asserts two execute attempts, stored `retryCount: 1`, and completed result |
| start-workers-all | yes | no | Real Mastra manager dispatch after plain `mastra.startWorkers()`; Brainkit starts the safe background-task worker while leaving generic scheduler/orchestration loops unclaimed |
| start-workers-surface | yes | no | Plain `mastra.startWorkers()`/`stopWorkers()` returns in QuickJS with the default Mastra worker set present |
| suspend-resume | yes | no | Live OpenAI-backed background tool suspension and out-of-band `BackgroundTaskManager.resume`; asserts suspended storage state, completed result storage, and follow-up recall |
| stream-until-idle | yes | no | Live OpenAI-backed `Agent.streamUntilIdle()` with a Mastra background tool: emits running/completed lifecycle chunks, stores the completed task result, and performs the continuation turn |
| stream-worker-teardown | yes | no | Real Mastra manager stream/worker teardown: stream aborts, manager shutdown, `stopWorkers()`, and Brainkit debug counters prove no active streams, task contexts, abort controllers, or workers remain |
| timeout-failure | yes | no | Live OpenAI-backed `Agent.streamUntilIdle()` with explicit `_background.timeoutMs`; asserts `background-task-failed`, stored `timed_out` state, and timeout error payload |

### channels/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| core | yes | no | Core Mastra channel orchestration: `AgentChannels`, `channels` Agent config, webhook route generation, `Mastra.getChannels()`, channel reaction tools, and `ChatChannelProcessor` request-context prompt injection. This is not concrete Slack/Discord/Telegram provider support |

### integration/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| with-workflow | yes | no | Agent with `workflows` config exposes a Mastra workflow as a tool; agent processes text through the workflow |

### Top-level

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| multi-provider | yes | no | Model resolution across providers; tests OpenAI (always) and Anthropic (if key set) via direct `generateText` |
| request-context | yes | no | `RequestContext` passed to dynamic instructions function; agent responds in character based on context key |
| scorers | yes | no | Per-call `scorers` option with a custom length scorer; tests config acceptance (scoring data may not be fully wired) |
| workspace | yes | no | Agent with `Workspace` + `LocalFilesystem`; tests that workspace construction and assignment works |
