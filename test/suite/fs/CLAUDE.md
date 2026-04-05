# FS Tests

Read `TEST_MAP.md` before editing any test in this directory.

All tests use `env.Kernel.EvalTS()` to run JS/TS code that exercises the Node.js fs polyfill. Each test operates in the kernel's FSRoot sandbox. Path traversal tests verify workspace escape prevention.

## Adding a test

1. Add function to operations.go
2. Register in run.go
3. Update TEST_MAP.md
