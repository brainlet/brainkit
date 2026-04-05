# cross-kit/ Fixtures

Tests cross-Kit awareness: namespace isolation, tool registration visible to other kits, and bus operation on shared transport (NATS).

## Fixtures

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| publish-to-remote | no | none | Registers a `createTool` on Kit A, verifies `kit.namespace` is populated, bus pub/sub works within the kit, and `tools.call()` round-trips through the registered tool |
