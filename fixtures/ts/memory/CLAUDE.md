# memory/ Fixtures

Tests the Mastra Memory API: thread CRUD, message persistence, recall, working memory, semantic recall with vector stores, observational memory, read-only mode, title generation, and storage backend compatibility.

Memory fixtures under `storage/` and the `semantic-recall/`, `working-memory/`, and `generate-title/` segments all require AI. Thread and message fixtures without those segments do not require AI.

## Fixtures

### threads/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| create | no | no | `createThread` + `getThreadById`; asserts thread created with valid id and retrievable |
| delete | no | no | `createThread` then `deleteThread`; asserts thread is null after deletion |
| get-by-id | no | no | `getThreadById` for existing and non-existent ids; asserts found/missing correctly |
| list | no | no | Creates 3 threads across 2 resources; asserts all found with distinct ids |
| management | no | no | Full thread management API on LibSQLStore: saveThread, getThreadById, listThreads, updateThread, saveMessages, recall, deleteMessages, deleteThread |

### messages/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| save-and-recall | no | no | `saveMessages` then `recall` on InMemoryStore; asserts messages are persisted and retrievable |

### working-memory/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| basic | yes | no | Agent with working memory template on LibSQLStore+LibSQLVector; learns facts across 3 calls and recalls from working memory (requires LIBSQL_URL) |
| schema | yes | no | Agent with working memory enabled (no template) on InMemoryStore; learns name+city across calls, asserts recall |

### semantic-recall/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| basic | yes | no | Memory with LibSQLStore+LibSQLVector+embedder; agent learns a fact then retrieves it via semantic vector search (requires LIBSQL_URL) |
| resource-scope | no | no | Semantic recall config with `scope: "resource"` on InMemoryStore without vector store; asserts config rejects or requires vector store |

### storage/

All storage fixtures require AI (the `memory/storage` path triggers `needs.AI = true` in classify.go).

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| inmemory | yes | no | Agent with InMemoryStore; two-call conversation recall (name + workplace) |
| libsql | yes | no | Agent with LibSQLStore via LIBSQL_URL; two-call recall of a programming language preference |
| libsql-local | yes | no | Agent with LibSQLStore via embedded bridge URL; two-call recall of color + dog name |
| postgres | yes | postgres | Agent with PostgresStore; two-call recall of favorite animal |
| postgres-scram | yes | postgres | Agent with PostgresStore over SCRAM-SHA-256 auth; two-call recall of favorite number |
| mongodb | yes | mongodb | Agent with MongoDBStore (no auth); two-call recall of favorite color |
| mongodb-scram | yes | mongodb | Agent with MongoDBStore over SCRAM-SHA-256 auth; two-call recall of favorite city |
| upstash | yes | no (credential) | Agent with UpstashStore via UPSTASH_REDIS_REST_URL+TOKEN; two-call recall of favorite language |

### observational/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| observational | yes | no | Agent with `observationalMemory: true` on InMemoryStore; learns name+city, asserts recall in second call |

### read-only/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| read-only | yes | no | Writable agent writes history, then read-only agent reads same thread; asserts read-only agent can recall but not persist |

### generate-title/

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| generate-title | yes | no | Memory with `generateTitle: true`; agent generates and the thread gets an auto-created title from conversation content |

### Top-level

| Fixture | AI | Container | What it tests |
|---------|-----|-----------|---------------|
| libsql-local-debug | yes | no | Debug fixture: LibSQLStore with LIBSQL_URL; direct agent two-call recall testing the storage path end-to-end |
