# AI and Agents

brainkit embeds the Vercel AI SDK and Mastra framework. AI calls (generateText, streamText, embed) are direct — no bus messages, no handlers, no wrappers. Agent creation uses Mastra's `Agent` class directly.

## AI SDK — Direct Calls

### generateText

```typescript
// fixtures/ts/ai/generate-text-basic/index.ts
const result = await generateText({
    model: model("openai", "gpt-4o-mini"),
    prompt: "What is 2+2? Reply with just the number.",
    maxTokens: 10,
});

output({
    text: result.text,               // "4"
    finishReason: result.finishReason, // "stop"
    usage: result.usage,              // {promptTokens, completionTokens, totalTokens}
});
```

### streamText

```typescript
// fixtures/ts/ai/stream-text-basic/index.ts
const result = await streamText({
    model: model("openai", "gpt-4o-mini"),
    prompt: "Count from 1 to 5.",
});

const chunks: string[] = [];
for await (const chunk of result.textStream) {
    chunks.push(chunk);
}

const fullText = await result.text;
const usage = await result.usage;
```

### generateObject

```typescript
// fixtures/ts/ai/generate-object-basic/index.ts
const result = await generateObject({
    model: model("openai", "gpt-4o-mini"),
    prompt: "Generate a person with name and age.",
    schema: z.object({
        name: z.string(),
        age: z.number(),
    }),
});

output(result.object); // {name: "Alice", age: 30}
```

### embed / embedMany

```typescript
// fixtures/ts/ai/embed-single/index.ts
const result = await embed({
    model: embeddingModel("openai", "text-embedding-3-small"),
    value: "Hello world",
});

output({
    dimensions: result.embedding.length,  // 1536
    hasUsage: !!result.usage,
});
```

```typescript
// fixtures/ts/ai/embed-many/index.ts
const result = await embedMany({
    model: embeddingModel("openai", "text-embedding-3-small"),
    values: ["Hello", "World", "Foo"],
});

output({ count: result.embeddings.length }); // 3
```

### Tool use with AI SDK

```typescript
// fixtures/ts/ai/generate-text-with-tools/index.ts
const result = await generateText({
    model: model("openai", "gpt-4o-mini"),
    prompt: "What is 6 times 7?",
    tools: {
        multiply: {
            description: "Multiply two numbers",
            parameters: z.object({ a: z.number(), b: z.number() }),
            execute: async ({ a, b }) => a * b,
        },
    },
    maxSteps: 3,
});

output({
    text: result.text,        // "42"
    hasToolCalls: result.toolCalls.length > 0,
    hasSteps: result.steps.length > 0,
});
```

## Agents — Mastra

### Creating an Agent

```typescript
// fixtures/ts/agent/generate-basic/index.ts
const myAgent = new Agent({
    name: "my-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Reply with exactly: AGENT_WORKS",
});

kit.register("agent", "my-agent", myAgent);

const result = await myAgent.generate("Say the magic word");
output({
    text: result.text,
    hasUsage: !!result.usage,
    finishReason: result.finishReason,
});
```

**Important:** `new Agent({...})` creates the agent. `kit.register("agent", name, agent)` makes it visible in the agent registry (agents.list, agents.discover). Creating without registering is valid — the agent works but isn't discoverable.

### Agent with Tools

```typescript
// fixtures/ts/agent/generate-with-tools/index.ts
const searchTool = createTool({
    id: "search",
    description: "Search the web",
    inputSchema: z.object({ query: z.string() }),
    execute: async ({ context: { query } }) => {
        return { results: [`Result for: ${query}`] };
    },
});

const agent = new Agent({
    name: "researcher",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Use the search tool to answer questions.",
    tools: { search: searchTool },
    maxSteps: 5,
});
```

### Agent Streaming

```typescript
// fixtures/ts/agent/stream-basic/index.ts
const agent = new Agent({
    name: "streamer",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Be concise.",
});

const stream = await agent.stream("Count to 5");

const chunks: string[] = [];
for await (const chunk of stream.textStream) {
    chunks.push(chunk);
}

const text = await stream.text;
const usage = await stream.usage;
```

### Agent with Go-Registered Tools

```typescript
// fixtures/ts/agent/with-registered-tool/index.ts
// "multiply" was registered in Go before this .ts file runs
const multiplyTool = tool("multiply");

const agent = new Agent({
    name: "math-bot",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Use the multiply tool.",
    tools: { multiply: multiplyTool },
});

const result = await agent.generate("What is 6 times 7?", { maxSteps: 3 });
```

### Agent with Memory

```typescript
// fixtures/ts/agent/with-memory-inmemory/index.ts
const store = new InMemoryStore();
const mem = new Memory({ storage: store });

const agent = new Agent({
    name: "memory-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Remember the user's name.",
    memory: mem,
});

await agent.generate("My name is David", { threadId: "t1", resourceId: "user-1" });
const result = await agent.generate("What's my name?", { threadId: "t1", resourceId: "user-1" });
```

### Agent Networks (Sub-Agents)

```typescript
// fixtures/ts/agent/subagents-basic/index.ts
const mathAgent = new Agent({
    name: "math",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You are a math expert. Compute what's asked.",
});

const supervisor = new Agent({
    name: "supervisor",
    model: model("openai", "gpt-4o-mini"),
    instructions: "Delegate math questions to the math agent.",
    agents: { math: mathAgent },
    maxSteps: 5,
});

const result = await supervisor.generate("What is 123 * 456?");
```

Each sub-agent becomes a tool (`agent-math`) that the supervisor can call.

## Workflows

```typescript
// fixtures/ts/workflow/then-basic/index.ts
const step1 = createStep({
    id: "uppercase",
    inputSchema: z.object({ text: z.string() }),
    outputSchema: z.object({ upper: z.string() }),
    execute: async ({ inputData }) => ({ upper: inputData.text.toUpperCase() }),
});

const step2 = createStep({
    id: "exclaim",
    inputSchema: z.object({ upper: z.string() }),
    outputSchema: z.object({ result: z.string() }),
    execute: async ({ inputData }) => ({ result: inputData.upper + "!!!" }),
});

const wf = createWorkflow({
    id: "my-workflow",
    inputSchema: z.object({ text: z.string() }),
    outputSchema: z.object({ result: z.string() }),
}).then(step1).then(step2).commit();

const run = await wf.createRun();
const result = await run.start({ inputData: { text: "hello" } });
// result.status: "success"
// result.result: { result: "HELLO!!!" }
```

### Branching

```typescript
// fixtures/ts/workflow/branch/index.ts
const wf = createWorkflow({
    id: "branching",
    inputSchema: z.object({ type: z.string() }),
    outputSchema: z.any(),
})
.then(classifyStep)
.branch([
    [({ inputData }) => inputData.classification === "urgent", urgentStep],
    [({ inputData }) => inputData.classification === "normal", normalStep],
])
.commit();
```

### Parallel

```typescript
// fixtures/ts/workflow/parallel/index.ts
const wf = createWorkflow({...})
.then(fetchStep)
.parallel([analyzeStep, summarizeStep, extractStep])
.then(mergeStep)
.commit();
```

## Evals

### createScorer

```typescript
// fixtures/ts/evals/create-scorer/index.ts
const accuracy = createScorer({
    name: "accuracy",
    description: "Checks if output contains expected answer",
}).generateScore(({ output, expectedOutput }) => {
    return output.toLowerCase().includes(expectedOutput.toLowerCase()) ? 1 : 0;
});
```

### runEvals

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
    totalItems: results.summary.totalItems,
    scores: results.scores,
});
```

### LLM Judge Pattern

```typescript
// fixtures/ts/evals/llm-judge/index.ts
const helpfulness = createScorer({
    name: "helpfulness",
    description: "LLM judges helpfulness",
}).generateScore({
    model: model("openai", "gpt-4o-mini"),
    instructions: "Rate helpfulness 0-1. Return JSON: {score: number}",
    outputSchema: z.object({ score: z.number() }),
});
```

## Model Resolution

`model(provider, modelId)` resolves from AI providers auto-detected via `os.Getenv`:

```typescript
model("openai", "gpt-4o-mini")         // OpenAI
model("anthropic", "claude-sonnet-4-20250514") // Anthropic
model("google", "gemini-2.0-flash")    // Google
model("groq", "llama-3.3-70b")        // Groq
model("deepseek", "deepseek-chat")     // DeepSeek

embeddingModel("openai", "text-embedding-3-small") // 1536 dims
embeddingModel("openai", "text-embedding-3-large") // 3072 dims
```

Only providers whose API keys are present in the environment are available. Calling `model("anthropic", "...")` without `ANTHROPIC_API_KEY` set returns a string identifier (no API key, calls will fail).

## What's Real vs Mastra Upstream

Everything documented here is tested in brainkit's fixture suite. Features that exist in Mastra but aren't tested through brainkit (processors, voice, some workspace features) are NOT documented here — they may or may not work. If it's in `fixtures/ts/`, it works. If it's not, treat it as unverified.
