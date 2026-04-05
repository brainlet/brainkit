# observability/ Fixtures

Tests tracing and span generation: verifies Agent.generate() produces traceId, runId, usage metrics, and step tracking.

## Fixtures

### spans/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| basic | yes | none | Agent with a tool (add) generates answer, confirms traceId, runId, usage.totalTokens, toolCalls count, and steps count are populated |

### trace/

| Fixture | AI | Container | What it tests |
|---------|----|-----------|---------------|
| basic | yes | none | Agent.generate() with simple prompt; verifies traceId and runId are non-empty strings and response contains expected text |
