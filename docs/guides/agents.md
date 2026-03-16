# Agents in Brainkit

Agents are the core unit of AI execution. An agent wraps a language model with instructions, tools, memory, and processors. Brainkit agents support delegation to sub-agents, dynamic configuration, and supervisor patterns.

---

## Quick Start

```ts
import { agent } from "brainlet";

const a = agent({
  model: "openai/gpt-4o-mini",
  instructions: "You are a helpful assistant.",
});

const result = await a.generate("Hello!");
console.log(result.text);
```

---

## Agent Config

```ts
const a = agent({
  // Required
  model: "openai/gpt-4o-mini",       // or dynamic: ({ requestContext }) => "openai/gpt-4o"

  // Identity
  name: "my-agent",
  id: "agent-001",
  description: "A helpful assistant",

  // Behavior
  instructions: "You are helpful.",    // or dynamic: ({ requestContext }) => "..."
  maxSteps: 10,                        // max tool-call rounds per generate()
  defaultOptions: { modelSettings: { temperature: 0.7 } },

  // Tools
  tools: { search: searchTool },       // or dynamic: ({ requestContext }) => ({...})

  // Memory
  memory: {
    thread: "session-1",
    resource: "user-1",
    storage: store,
  },

  // Sub-agents (supervisor pattern)
  agents: { researcher, coder },

  // Processors
  inputProcessors: [unicodeNormalizer, tokenLimiter],
  outputProcessors: [moderationProcessor],

  // Workspace
  workspace: ws,

  // Evaluation
  scorers: { quality: { scorer: qualityScorer } },
});
```

---

## Generate & Stream

```ts
// Full response
const result = await a.generate("What is 2+2?");
result.text          // "4"
result.usage         // { promptTokens, completionTokens, totalTokens }
result.toolCalls     // [{ toolName, args, result }]
result.steps         // step-by-step execution trace

// Streaming
const stream = await a.stream("Count to 10");
for await (const chunk of stream.textStream) {
  process.stdout.write(chunk);
}
```

### Per-Call Options

Every `generate()` and `stream()` call accepts options that override agent defaults:

```ts
await a.generate("Hello", {
  // Model settings
  modelSettings: { temperature: 0.9, maxTokens: 500 },

  // Tool control
  activeTools: ["search"],              // only these tools available
  toolChoice: "required",              // force tool use
  toolCallConcurrency: 3,             // parallel tool calls

  // Per-call overrides
  instructions: "Be a pirate. Say ARRR.",
  maxSteps: 1,

  // Structured output
  structuredOutput: {
    schema: z.object({ answer: z.number() }),
  },

  // Callbacks
  onStepFinish: (step) => console.log("Step done"),
  onFinish: (result) => console.log("Done:", result.text),
  onError: ({ error }) => console.error(error),

  // Memory
  memory: { thread: { id: "custom-thread" }, resource: "user-2" },

  // Cancellation
  abortSignal: controller.signal,
});
```

---

## Agent Networks (Supervisor Pattern)

Register sub-agents on a supervisor. Each sub-agent becomes a tool (`agent-<name>`) the supervisor can delegate to.

```ts
const researcher = agent({
  model: "openai/gpt-4o-mini",
  instructions: "Research topics thoroughly. Cite sources.",
});

const coder = agent({
  model: "openai/gpt-4o-mini",
  instructions: "Write clean, tested code.",
  tools: { execute: execTool },
});

const supervisor = agent({
  model: "openai/gpt-4o",
  instructions: "You are a tech lead. Delegate research to the researcher and coding to the coder.",
  agents: { researcher, coder },
  maxSteps: 10,
});

// Supervisor sees agent-researcher and agent-coder as tools
const result = await supervisor.generate("Build a REST API for a todo app");
```

### Network Mode

For multi-step delegation loops where the supervisor coordinates several sub-agents:

```ts
const result = await supervisor.network("Research RLHF, then implement a training loop", {
  maxSteps: 20,
});
```

Network mode continues until the supervisor decides the task is complete or `maxSteps` is reached.

### Delegation Hooks

Control what sub-agents can see and do:

```ts
const supervisor = agent({
  model: "openai/gpt-4o",
  agents: { researcher, coder },
  delegation: {
    // Called before each delegation — can reject or modify
    onDelegationStart: ({ agentId, input }) => {
      if (agentId === "coder" && input.includes("delete")) {
        return { allowed: false }; // reject dangerous delegations
      }
    },
    // Filter what conversation history the sub-agent sees
    messageFilter: (messages) => messages.slice(-5), // only last 5 messages
    // Called after delegation completes
    onDelegationComplete: ({ agentId, output }) => {
      console.log(`${agentId} finished: ${output.substring(0, 50)}`);
    },
  },
});
```

### Dynamic Sub-Agents

Sub-agents can be resolved dynamically per-request:

```ts
const supervisor = agent({
  model: "openai/gpt-4o",
  agents: ({ requestContext }) => {
    const team = requestContext.get("team");
    return team === "engineering"
      ? { coder, reviewer }
      : { writer, editor };
  },
});
```

---

## Memory Access

When an agent has memory configured, access it directly for thread/message management:

```ts
const a = agent({
  model: "openai/gpt-4o-mini",
  memory: { thread: "t1", resource: "user-1", storage: store },
});

// Thread management
const threads = await a.memory.listThreads({ resourceId: "user-1" });
const thread = await a.memory.getThreadById({ threadId: "t1" });
await a.memory.updateThread({ id: "t1", title: "New title", metadata: {} });
await a.memory.deleteThread("t1");

// Message management
const recalled = await a.memory.recall({ threadId: "t1" });
await a.memory.deleteMessages(["msg-1", "msg-2"]);
```

---

## Dynamic Configuration

Model, instructions, and tools can be functions that resolve at call time:

```ts
const a = agent({
  model: ({ requestContext }) => {
    const tier = requestContext.get("tier");
    return tier === "premium" ? "openai/gpt-4o" : "openai/gpt-4o-mini";
  },
  instructions: ({ requestContext }) => {
    const lang = requestContext.get("language");
    return `Respond in ${lang}. Be helpful.`;
  },
  tools: ({ requestContext }) => {
    const mode = requestContext.get("mode");
    return mode === "readonly" ? { search: searchTool } : { search: searchTool, write: writeTool };
  },
});

// Pass context per call
await a.generate("Hello", {
  requestContext: new RequestContext({ tier: "premium", language: "French", mode: "readonly" }),
});
```

---

## What's Not Supported

| Feature | Notes |
|---------|-------|
| Stored Agents CRUD | Create/Get/List/Update/Delete persistent agent definitions via storage |
| Voice (TTS/STT) | 13 providers in Mastra — not yet bundled |
| Agent.approveToolCall() | Tool suspension approval flow |
| Agent.resumeStream() | Resume a suspended stream |
