# kit/ Fixtures

Tests core kit APIs: output(), registry, filesystem, lifecycle, storage pool, and error paths for registration, secrets, tools, and security validation.

## Fixtures

### errors/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| deploy-throws-init | no | none | Verifies `output()` works before any error in deployment init phase |
| error-code-inspection | no | none | Calling nonexistent tool returns descriptive error; `bus.publish` returns replyTo; `bus.emit` does not throw on valid topic |
| file-url-blocked | no | none | `LibSQLStore` and `LibSQLVector` with `file:` URLs throw VALIDATION_ERROR; `http:` and `libsql:` URLs are not blocked by validation |
| multi-tool-register | no | none | Registers 5 tools in a loop via `kit.register("tool", ...)`, verifies all 5 appear in `tools.list()` |
| register-invalid-type | no | none | `kit.register("banana", ...)` throws with descriptive error listing valid types (tool, agent, workflow, memory) |
| secrets-operations | no | none | `secrets.get()` for nonexistent key returns empty string, not null or error |
| tool-lifecycle | no | none | Full tool lifecycle: `createTool` + `kit.register` + `tools.list` (find by shortName) + `tools.call` returns correct result |

### fs/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| list-stat | no | none | `fs.writeFileSync` + `fs.statSync` (isDirectory false) + `fs.readdirSync` (file appears in listing) |
| operations | no | none | Full filesystem lifecycle: write, read, list, stat (has size), delete via `fs.unlinkSync` |
| read-write | no | none | `fs.writeFileSync` then `fs.readFileSync` roundtrip; content matches exactly |

### lifecycle/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| deploy-teardown | no | none | `kit.register` two resources, `kit.list` confirms count, `kit.unregister` removes one, count decreases |

### output/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| basic | no | none | `output()` sets module result with string, number, and nested object fields |

### registry/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| has-list | no | none | `registry.has("provider", "nonexistent")` returns false; `registry.list("provider")` and `registry.list("storage")` return arrays |
| operations | no | none | Same as has-list: confirms registry query APIs return expected types for nonexistent entries |
| resolve | no | none | `registry.resolve("provider", "nonexistent")` returns null |

### storage-pool/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| default | no | none | `storage("default")` resolves a store from the kernel resource pool; has a name property |
| memory | no | none | `storage("mem")` resolves an InMemoryStore from the resource pool; has a name property |
