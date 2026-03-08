// Ported from: packages/core/src/loop/workflows/
//
// Package workflows contains the workflow-based loop execution engine.
// This is a stub package; full implementation will follow once dependent
// packages (agent, stream, workflows core) are ported.
package workflows

// TODO: Port the following files from packages/core/src/loop/workflows/:
//
// Root files:
//   - errors.ts        -> errors.go       (custom error types for workflow loop)
//   - run-state.ts     -> run_state.go    (RunState tracking for workflow execution)
//   - schema.ts        -> schema.go       (Zod schemas for workflow step I/O)
//   - stream.ts        -> stream.go       (workflowLoopStream - main entry point)
//
// agentic-execution/ subdirectory:
//   - index.ts                  -> agentic_execution.go       (workflow definition and step wiring)
//   - is-task-complete-step.ts  -> is_task_complete_step.go   (completion check step)
//   - llm-execution-step.ts     -> llm_execution_step.go     (LLM call step, ~46KB)
//   - llm-mapping-step.ts       -> llm_mapping_step.go       (model selection and mapping step)
//   - tool-call-step.ts         -> tool_call_step.go         (tool execution step)
//
// agentic-loop/ subdirectory:
//   - index.ts                  -> agentic_loop.go           (outer agentic loop workflow)
//
// Test files (not ported, covered by testutils package):
//   - llm-mapping-step.test.ts
//   - tool-call-step.test.ts
