# Gateway Tests

Read `TEST_MAP.md` before editing any test in this directory.

Tests use `gwStart()`, `gwStartWithStream()`, or `gwSetup()` helpers to start a gateway on a random port. Most tests create fresh kernels via `suite.Full(t)` since they need specific gateway configurations. HTTP assertions use standard `http.Get`/`http.Post` and the `gwGet`/`gwPost` helper functions.

Key conventions:
- Gateway tests deploy .ts handlers then register HTTP routes via `gw.Handle()` or `gw.HandleStream()`
- SSE tests read the event-stream body and parse SSE event/data lines
- Attack tests verify resilience: no panics, no leaks, kernel stays alive

## Adding a test

1. Add function to the right .go file (routes.go for core, stream.go for SSE config, advanced.go for adversarial, errors.go for error handling, attacks.go for security)
2. Register in run.go
3. Update TEST_MAP.md
