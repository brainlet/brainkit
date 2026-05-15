# Harness Fixtures

Harness fixtures cover Mastra's JS `Harness` class through Brainkit's deployed
TypeScript runtime.

All harness fixtures require live AI (`OPENAI_API_KEY`) because meaningful
Harness behavior drives real `Agent` streams. The runner loads the Brainkit
root `.env`; verify with `BRAINKIT_TEST_LIVE_AI=1`.

## Fixtures

| Path | Needs AI | What it tests |
|------|----------|---------------|
| `browser/basic` | yes | Live OpenAI-backed Harness browser contract: `MastraBrowser` and `BrowserContextProcessor` export surface, Harness `browser` propagation to a mode agent, browser context on `RequestContext`, browser tools added only at execution time, permission-unblocked browser tool call, lifecycle launch/close hooks, and teardown |
| `interactive-tools/basic` | yes | Real Harness interactive tool resolver path: `ask_user`, `submit_plan`, `respondToQuestion`, `respondToPlanApproval`, question/plan events, pending display state, approval and rejection results, and teardown |
| `observational-memory/basic` | yes | Real Harness observational-memory control surface: exported OM helpers, observer/reflector model defaults and switches, threshold persistence in thread metadata, seeded OM record lookup, `loadOMProgress`, OM stream data-part event mapping, display-state updates, activation/title events, and failure abort handling |
| `send-message/basic` | yes | Live OpenAI-backed `Harness.init()`, `sendMessage()`, event emission, thread/session state, display state, and teardown through the deployed `"agent"` module |
| `subagents/basic` | yes | Live OpenAI-backed isolated Harness subagent path: parent model calls built-in `subagent`, child `Agent` resolves through `resolveModel`, `subagent_start`/`subagent_end` events, display subagent snapshots, parent tool lifecycle, and display cleanup |
| `subagents/forked` | yes | Live OpenAI-backed forked Harness subagent path: built-in `subagent` clones the parent memory thread, runs the parent agent on the fork, preserves fork metadata/thread filtering, returns context from cloned history, and records completed display state |
| `task-tools/basic` | yes | Real Harness request-context task tools: `task_write`, `task_update`, `task_complete`, `task_check`, task state mutation, rejection of multiple in-progress tasks, `task_updated` events, display task state, and teardown |
| `tool-approval/basic` | yes | Live OpenAI-backed Harness tool approval: require-approval tool call, `tool_approval_required`, display pending approval, `respondToToolApproval`, resumed stream completion, and display cleanup |
| `tool-suspension/basic` | yes | Live OpenAI-backed Harness tool suspension/resume: tool `suspend()`, `tool_suspended`, `agent_end: suspended`, pending display suspension, `respondToToolSuspension`, resumed stream completion, and display cleanup |
| `workspace/basic` | yes | Live OpenAI-backed Harness workspace integration: static `Workspace` init/ready/destroy events, LocalFilesystem access, workspace tool constants/export surface, built-in `subagent` with `allowedWorkspaceTools`, child agent read-file tool use, subagent display completion, and teardown |
