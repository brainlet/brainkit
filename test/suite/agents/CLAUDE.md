# Agents Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests follow the standard suite pattern: `func testXxx(t *testing.T, env *suite.TestEnv)` registered in `run.go`. AI-dependent tests call `env.RequireAI(t)` to skip when OPENAI_API_KEY is absent. Each AI test deploys a .ts file, verifies output via `globalThis.__module_result`, and tears down.

## Adding a test

1. Add function to the right .go file (lifecycle.go for CRUD, ai.go for AI agent ops, surface.go for AI SDK surface)
2. Register in run.go
3. Update TEST_MAP.md
