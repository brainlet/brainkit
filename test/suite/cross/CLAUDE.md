# Cross Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests verify interactions across surfaces (Go, TS, plugin) and across Kit instances. Many tests require Podman (container-backed NATS) and call `env.RequirePodman(t)` to skip otherwise. Plugin tests use `testutil.BuildTestPlugin(t)` to compile a subprocess plugin binary. Cross-kit tests iterate over `testutil.AllBackends(t)`.

Key conventions:
- Namespace names include "-cross" suffix to avoid collisions
- Subprocess plugin tests need NATS containers (Podman required)
- Discovery tests use in-memory static provider (no containers)

## Adding a test

1. Add function to the right .go file (crosskit.go for TS<->Go, plugins.go for plugin, node_commands.go for Node commands, discovery.go for discovery, plugin_surface.go for plugin surface, backend_matrix.go for cross-Kit)
2. Register in run.go
3. Update TEST_MAP.md
