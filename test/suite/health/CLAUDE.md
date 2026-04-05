# Health Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests verify kernel health checks, shutdown lifecycle, metrics, and health probes. Most shutdown and probe tests create fresh kernels since they close/drain the kernel or need specific configs. AI probe tests require OPENAI_API_KEY. PgVector probe tests require Podman.

Key conventions:
- Shutdown tests always use fresh kernels (they close the kernel)
- Probe tests with external services call env.RequireAI(t) or env.RequirePodman(t)
- Degraded tests push the kernel to edge conditions then verify health reporting

## Adding a test

1. Add function to the right .go file (checks.go for basic health, shutdown.go for drain/close, metrics.go for metrics, probes.go for external probes, degraded.go for adversarial, shutdown_adv.go for adversarial shutdown)
2. Register in run.go
3. Update TEST_MAP.md
