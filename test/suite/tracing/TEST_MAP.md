# Tracing Test Map

**Purpose:** Verifies distributed tracing: span creation for commands, handlers, tool calls, and deploys; bus-based trace queries; sample rate control; trace context propagation; and no-op behavior without a trace store.
**Tests:** 10 functions across 1 file
**Entry point:** `tracing_test.go` → `Run(t, env)`
**Campaigns:** transport (amqp, redis, postgres, nats, sqlite), fullstack (nats_postgres_rbac)

## Files

### spans.go — Trace span creation, querying, and propagation

| Function | Purpose |
|----------|---------|
| testCommandRequestCreatesSpan | Runs tools.list via EvalTS, queries MemoryTraceStore, verifies at least one trace was recorded |
| testHandlerCreatesSpan | Deploys .ts handler, sends it a message, queries traces, verifies traces with root spans exist |
| testQueryViaBus | Manually creates a span, publishes TraceListMsg via bus, verifies traces returned in response |
| testNoStoreNoOp | Creates minimal kernel (no trace store), publishes ToolListMsg, verifies response arrives (tracing is transparent no-op) |
| testToolCallCreatesSpan | Calls "echo" tool via SDK, queries traces, verifies tool call created trace spans |
| testDeployCreatesSpan | Deploys .ts, queries traces, verifies a span with name "kit.deploy:traced-deploy.ts" and correct source exists |
| testQueryBySource | Deploys .ts handler, sends a message, queries TraceListMsg for all traces, verifies non-empty response |
| testEmptyStore | Queries empty store directly (verifies empty list), queries nonexistent traceID via bus (verifies "spans" in response) |
| testSampleRate | Creates kernel with TraceSampleRate=0.0, calls tool, verifies no panic (traces may or may not be recorded) |
| testTraceContextPropagates | Stamps traceId/parentSpanId/traceSampled into publish context, subscribes to the topic, verifies all 3 metadata fields propagate |

## Cross-references

- **Campaigns:** transport/{amqp,redis,postgres,nats,sqlite}_test.go, fullstack/nats_postgres_rbac_test.go
- **Related domains:** tools (tool call tracing), deploy (deploy tracing)
- **Fixtures:** none (tracing tests are Go-driven)
