# composition/ Fixtures

Tests full-stack compositions that combine multiple brainkit subsystems (Agent, AI SDK, tools, workflows, memory, bus) in a single .ts deployment.

## Fixtures

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| agent-workflow-memory | yes | none | Combines `generateText()` (direct AI call), `tools.call()` (Go tool via bus), and `createTool()` (local Zod-schema tool) in one deployment |
| multi-module-integration | yes | none | Exercises all five subsystems together: `createWorkflow` + `createStep` pipeline, `tools.call()` registered tool, `Memory` with `InMemoryStore`, `Agent.generate()`, and `bus.publish()` -- verifies all interoperate |
