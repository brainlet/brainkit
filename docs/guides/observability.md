# Observability

brainkit provides logging, tracing, and evaluation scoring. Logs are routed through a Go handler. Traces use Mastra's Observability system. Evals use Mastra's scorer framework.

## Logging

Every deployed .ts file gets a per-source tagged console:

```typescript
// Inside my-service.ts
console.log("starting");    // [my-service.ts] [log] starting
console.warn("slow query"); // [my-service.ts] [warn] slow query
console.error("failed");    // [my-service.ts] [error] failed
```

### LogHandler

By default, logs go to `log.Printf`. Override with a custom handler:

```go
k, err := brainkit.NewKernel(brainkit.KernelConfig{
    LogHandler: func(entry brainkit.LogEntry) {
        // entry.Source:  "my-service.ts" or "kernel"
        // entry.Level:   "log", "warn", "error", "debug", "info"
        // entry.Message: the log text
        // entry.Time:    time.Time

        // Route to your logging system
        slog.Info(entry.Message,
            "source", entry.Source,
            "level", entry.Level,
        )
    },
})
```

`LogHandler` is called concurrently from multiple goroutines. It MUST be goroutine-safe.

### Log sources

| Source format | Origin |
|---------------|--------|
| `my-service.ts` | .ts Compartment console.log/warn/error |
| `kernel` | Internal Kernel operations |

## Observability (Mastra Tracing)

Mastra's `Observability` class provides trace collection with exporters:

```typescript
// From kit_runtime.js — auto-configured during Kernel init
const obs = new Observability({
    configs: {
        default: {
            serviceName: "brainkit",
            exporters: [new DefaultExporter({
                storage: store,
                strategy: "realtime", // or "batch"
            })],
        },
    },
});
```

### Configuration from Go

```go
k, err := kit.NewKernel(kit.KernelConfig{
    Observability: kit.ObservabilityConfig{
        Enabled:     &enabled,     // default: true
        Strategy:    "realtime",   // "realtime" or "batch"
        ServiceName: "my-service", // default: "brainkit"
    },
})
```

When enabled, agent.generate, agent.stream, and workflow runs automatically emit trace spans to the configured exporter.

### DefaultExporter

The `DefaultExporter` writes traces to the Kit's default storage (the InMemoryStore created during runtime init). Traces include:

- Agent name, model, instructions hash
- Tool calls with args and results
- Step-by-step execution trace
- Token usage per step
- Total duration

## Evaluation Scoring

### createScorer

Build a custom scorer with optional preprocessing and reasoning:

```typescript
// fixtures/ts/evals/create-scorer/index.ts
const accuracy = createScorer({
    name: "accuracy",
    description: "Checks if output matches expected",
}).generateScore(({ output, expectedOutput }) => {
    return output.toLowerCase().includes(expectedOutput.toLowerCase()) ? 1 : 0;
});
```

With preprocessing and reasoning:

```typescript
// fixtures/ts/evals/with-preprocess/index.ts
const scorer = createScorer({
    name: "quality",
}).preprocess(({ output }) => {
    return { cleaned: output.trim().toLowerCase() };
}).generateScore(({ output, preprocessResult }) => {
    return preprocessResult.cleaned.length > 10 ? 1 : 0;
}).generateReason(({ score }) => {
    return score === 1 ? "Output is substantial" : "Output too short";
});
```

### LLM-as-Judge

```typescript
// fixtures/ts/evals/llm-judge/index.ts
const helpfulness = createScorer({
    name: "helpfulness",
}).generateScore({
    model: model("openai", "gpt-4o-mini"),
    instructions: "Rate helpfulness 0-1. Return JSON: {score: number}",
    outputSchema: z.object({ score: z.number() }),
});
```

### runEvals — Batch Evaluation

Run a dataset through an agent and score every response:

```typescript
// fixtures/ts/evals/batch/index.ts
const results = await runEvals({
    agent: myAgent,
    data: [
        { input: "What is 2+2?", expectedOutput: "4" },
        { input: "Capital of France?", expectedOutput: "paris" },
    ],
    scorers: [accuracy],
});

output({
    totalItems: results.summary.totalItems,  // 2
    scores: results.scores,                   // { accuracy: 1.0 }
});
```

## Metrics

The Watermill middleware tracks message processing metrics:

```go
// Available internally — not yet exposed as a public API
metrics := kernel.host.Metrics()
// metrics.Published: map[topic]count
// metrics.Handled: map[topic]count
// metrics.Errors: map[topic]count
```

Scaling strategies consume these metrics for auto-scaling decisions.

## What's Not Yet Available

| Feature | Status |
|---------|--------|
| OpenTelemetry export | OTel stubs exist (no-op), real export not wired |
| Trace query API | Traces stored but no query endpoint |
| Custom exporters | DefaultExporter only — custom exporters would need bundle integration |
| Metrics endpoint | Internal only — no HTTP /metrics |
| Health check endpoint | Probing exists for providers, not as an HTTP endpoint |
