# cross-feature/ Fixtures

Tests interactions between two or more brainkit features (agent+bus, deploy+secrets, deploy+tools, bus+scheduling) within a single kit instance.

## Fixtures

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| agent-with-bus | no | none | Verifies Agent constructor is available alongside `createTool()` and `bus.publish()` -- confirms agent and bus modules coexist |
| deploy-with-secrets | no | none | Reads `secrets.get()` during deployment init; confirms nonexistent key returns empty string |
| deploy-with-tools | no | none | Calls `tools.call("echo", ...)` during deployment init phase; confirms Go-registered tools are callable at init time |
| multi-service-chain | no | none | Tests `bus.publish()` routing with replyTo and correlationId; validates the publish mechanism for multi-service chains |
| schedule-triggers-handler | no | none | Calls `bus.schedule("in 1h", ...)` to create a scheduled message, verifies schedule ID is returned, then `bus.unschedule()` cleans up |
