# Tools Test Map

**Purpose:** Verifies tool listing, resolution, calling (echo, add), error handling for missing tools, input abuse, E2E pipeline (deploy tool from .ts, call, teardown), and transport-level roundtrip.
**Tests:** 12 functions across 4 files
**Entry point:** `tools_test.go` → `Run(t, env)`
**Campaigns:** transport (amqp, redis, postgres, nats, sqlite), fullstack (nats_postgres_rbac, amqp_postgres_vector, redis_mongodb)

## Files

### registry.go — Core tool list/resolve/call operations

| Function | Purpose |
|----------|---------|
| testToolsList | Publishes ToolListMsg, verifies response contains "echo" and "add" tools by ShortName |
| testToolsResolveEcho | Publishes ToolResolveMsg for "echo", verifies ShortName, Description, and non-nil InputSchema |
| testToolsResolveNotFound | Publishes ToolResolveMsg for "nonexistent", verifies error field in response |
| testToolsCallEcho | Publishes ToolCallMsg for "echo" with message "hello world", verifies result contains echoed value |
| testToolsCallAdd | Publishes ToolCallMsg for "add" with a=17, b=25, verifies result sum=42 |
| testToolsCallNotFound | Publishes ToolCallMsg for "nonexistent", verifies error field in response |

### input_abuse.go — Tools input abuse

| Function | Purpose |
|----------|---------|
| testInputAbuseCallNonexistent | Calls nonexistent tool via SendAndReceive, verifies NOT_FOUND response code |
| testInputAbuseWrongInputType | Calls "echo" with string input instead of object, verifies response (not hang) within 5s |
| testInputAbuseEmptyToolName | Calls with empty tool name, verifies error response |
| testInputAbuseOversizedInput | Calls "echo" with 100KB message value, verifies response (not crash) within 10s |

### e2e.go — Tool pipeline end-to-end

| Function | Purpose |
|----------|---------|
| testToolPipeline | Deploys .ts that creates "greeter-tool-adv", verifies it appears in tools.list, calls it with name "Brainkit", verifies greeting "Hello, Brainkit!", tears down, verifies cleanup |

### backend_advanced.go — Transport-level tool tests

| Function | Purpose |
|----------|---------|
| testToolCallRoundtrip | Publishes ToolCallMsg for "echo" with "roundtrip-suite", subscribes raw, verifies payload contains the message |

## Cross-references

- **Campaigns:** transport/{amqp,redis,postgres,nats,sqlite}_test.go, fullstack/{nats_postgres_rbac,amqp_postgres_vector,redis_mongodb}_test.go
- **Related domains:** registry (tool registration), workflows (tools.call inside steps), security (tool name collision)
- **Fixtures:** tool-related TS fixtures
