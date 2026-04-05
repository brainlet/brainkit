# plugin/ Fixtures

Tests calling tools provided by plugin subprocesses: the test runner starts a plugin that registers tools, and the fixture calls them via `tools.call()`.

## Fixtures

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| call-plugin-tool | no | none | Calls plugin-provided "echo" and "concat" tools via `tools.call()`; verifies both return correct results through the plugin subprocess bridge |
