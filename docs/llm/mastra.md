# Mastra (`"agent"` module) — API Reference (brainkit runtime)

Source of truth: `internal/engine/runtime/agent.d.ts`. The `"agent"` module inside `.ts` deployments re-exports Mastra — no wrapping. Mastra types use AI SDK v4 token names in `AgentResult.usage` (`promptTokens` / `completionTokens`); AI SDK v5 functions use v5 names (`inputTokens` / `outputTokens`). Do not cross-pollinate.

```typescript
import {
    Agent, createTool, createWorkflow, createStep,
    Memory, InMemoryStore, LibSQLStore, UpstashStore, PostgresStore, MongoDBStore,
    LibSQLVector, PgVector, MongoDBVector, ModelRouterEmbeddingModel,
    MDocument, GraphRAG,
    createVectorQueryTool, createDocumentChunkerTool, createGraphRAGTool,
    rerank, rerankWithScorer,
    Observability, DefaultExporter, SensitiveDataFilter,
    createScorer, runEvals,
    Workspace, LocalFilesystem, LocalSandbox,
    RequestContext,
    z,
} from "agent";
import { model, embeddingModel, kit, bus } from "kit";
```

`z` is re-exported from `"ai"` — the same Zod instance. Use either import.

---

## Agent

```typescript
class Agent {
    constructor(config: AgentConfig);

    generate(promptOrMessages: string | Message[], options?: AgentCallOptions): Promise<AgentResult>;
    stream(promptOrMessages: string | Message[], options?: AgentCallOptions): Promise<AgentStreamResult>;
    network(promptOrMessages: string | Message[], options?: AgentCallOptions): Promise<AgentResult>;

    // Human-in-the-loop tool-call suspension
    approveToolCallGenerate(opts: { runId: string; toolCallId: string }): Promise<AgentResult>;
    declineToolCallGenerate(opts: { runId: string; toolCallId: string }): Promise<AgentResult>;
    approveToolCallStream(opts: { runId: string; toolCallId: string }): Promise<AgentStreamResult>;
    declineToolCallStream(opts: { runId: string; toolCallId: string }): Promise<AgentStreamResult>;
}
```

`network(...)` runs the agent in supervisor mode, delegating to `config.agents`.

### AgentConfig

```typescript
interface AgentConfig {
    id?: string;
    name: string;
    description?: string;
    instructions:
        | string
        | string[]
        | ((ctx: { requestContext?: RequestContext }) => string | string[] | Promise<string | string[]>);
    model: any;                         // from kit.model(...) or a resolver fn / retry array
    maxRetries?: number;                // default 0
    tools?:
        | Record<string, Tool>
        | ((ctx: { requestContext?: RequestContext }) => Record<string, Tool> | Promise<Record<string, Tool>>);
    workflows?: Record<string, Workflow> | (() => Record<string, Workflow>);
    defaultOptions?: Partial<AgentCallOptions>;
    agents?: Record<string, Agent> | (() => Record<string, Agent>);           // sub-agents (network mode)
    scorers?: Record<string, { scorer: Scorer; sampling?: any }>;
    memory?: Memory | (() => Memory);
    skillsFormat?: "xml" | "json";      // default "xml"
    voice?: any;
    workspace?: Workspace | (() => Workspace | undefined);
    inputProcessors?: any[];            // pre-LLM middleware
    outputProcessors?: any[];           // post-LLM middleware
    maxProcessorRetries?: number;
    providerOptions?: Record<string, Record<string, unknown>>;
    requestContextSchema?: ZodType;
    maxSteps?: number;                  // convenience, forwarded via defaultOptions
    [key: string]: any;
}
```

### AgentCallOptions

```typescript
interface AgentCallOptions {
    instructions?: string | string[];
    system?: string;
    context?: Message[];
    memory?: { thread?: string | { id: string }; resource?: string };
    runId?: string;
    savePerStep?: boolean;              // default false
    requestContext?: RequestContext;
    maxSteps?: number;
    providerOptions?: Record<string, Record<string, unknown>>;
    onStepFinish?: (event: any) => void | Promise<void>;
    onFinish?:     (event: any) => void | Promise<void>;
    onChunk?:      (event: any) => void | Promise<void>;
    onError?:      (event: any) => void | Promise<void>;
    activeTools?: string[];
    abortSignal?: AbortSignal;
    inputProcessors?: any[];
    outputProcessors?: any[];
    maxProcessorRetries?: number;
    toolsets?: Record<string, Record<string, Tool>>;
    toolChoice?: "auto" | "none" | "required" | { type: "tool"; toolName: string };
    modelSettings?: Record<string, any>;     // temperature, maxTokens, topP, …
    scorers?: Record<string, { scorer: any; sampling?: any }>;
    returnScorerData?: boolean;
    requireToolApproval?: boolean;
    autoResumeSuspendedTools?: boolean;
    toolCallConcurrency?: number;            // default 1 w/ approval, 10 otherwise
    output?: ZodType;                        // structured output
    [key: string]: any;
}
```

### AgentResult (v4 usage names)

```typescript
interface AgentResult {
    text: string;
    reasoningText?: string;
    object?: unknown;                    // populated when options.output set
    toolCalls:   Array<{ toolCallId: string; toolName: string; args: Record<string, unknown> }>;
    toolResults: Array<{ toolCallId: string; toolName: string; args: Record<string, unknown>; result: unknown }>;
    finishReason: string;
    usage: { promptTokens: number; completionTokens: number; totalTokens: number };   // v4 names
    steps: AgentStepResult[];
    response: { id: string; modelId: string; timestamp: string };
    runId?: string;
    traceId?: string;
    suspendPayload?: unknown;            // HITL: present when a tool call suspended
    providerMetadata?: Record<string, unknown>;
}

interface AgentStepResult {
    text: string;
    reasoning?: string;
    toolCalls:   Array<{ toolCallId: string; toolName: string; args: Record<string, unknown> }>;
    toolResults: Array<{ toolCallId: string; toolName: string; args: Record<string, unknown>; result: unknown }>;
    finishReason: string;
    usage: { promptTokens: number; completionTokens: number; totalTokens: number };
    stepType: string;
    isContinued: boolean;
}

interface AgentStreamResult {
    textStream: AsyncIterable<string>;
    fullStream: AsyncIterable<import("ai").StreamPart>;
    text:         Promise<string>;
    usage:        Promise<{ promptTokens: number; completionTokens: number; totalTokens: number }>;
    finishReason: Promise<string>;
    toolCalls:    Promise<Array<{ toolCallId: string; toolName: string; args: Record<string, unknown> }>>;
    toolResults:  Promise<Array<{ toolCallId: string; toolName: string; args: Record<string, unknown>; result: unknown }>>;
    steps:        Promise<AgentStepResult[]>;
}

interface Message {
    role: "system" | "user" | "assistant" | "tool";
    content: import("ai").MessageContent;   // string | ContentPart[]
}
```

### Register with the Kit

```typescript
const a = new Agent({
    name: "researcher",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You research topics thoroughly.",
    tools: { search },
});
kit.register("agent", "researcher", a);

bus.on("ask", async (msg) => {
    const r = await a.generate(String(msg.payload?.prompt ?? ""));
    msg.reply({ text: r.text, usage: r.usage });       // v4-shaped usage
});
```

---

## Mastra (container)

```typescript
class Mastra {
    constructor(config: MastraConfig);
    getAgent(name: string): Agent;
    getWorkflow(name: string): Workflow;
    getStorage(): MastraStorage | undefined;
}

interface MastraConfig {
    agents?: Record<string, Agent>;
    workflows?: Record<string, Workflow>;
    storage?: MastraStorage;           // InMemoryStore, LibSQLStore, PostgresStore, …
    vectors?: Record<string, MastraVector>;
    observability?: ObservabilityConfig;
}
```

`Mastra` is the container that binds a set of agents + their storage + their
workflow registry. Use it — not a bare `Agent` — whenever the flow needs
persistent run state.

### When to wrap an Agent in Mastra — required for:

| Flow                                  | Why                                                                                 |
|---------------------------------------|-------------------------------------------------------------------------------------|
| `approveToolCallGenerate`             | Loads workflow snapshot from `this.#mastra.getStorage().getStore("workflows")`.     |
| `declineToolCallGenerate`             | Same resume path.                                                                   |
| `resumeGenerate` / `resumeStream`     | Same resume path.                                                                   |
| Workflows with agents as steps        | `agent` step resolves via `mastra.getAgent(name)`.                                  |
| Sub-agent networks with shared memory | Shared storage + run context live on the Mastra instance.                           |

Bare `Agent` silently returns the original `{finishReason: "suspended"}` if
any of the above are called without a Mastra-attached agent — the snapshot
lookup short-circuits and no error is thrown.

### Two patterns

```typescript
// Simple — one-shot generate, no suspend/resume
const a = new Agent({ name: "simple", model: model(...), ... });
const r = await a.generate("hi");

// Durable — HITL, workflows, resumable runs
const mastra = new Mastra({
    agents: { assistant: new Agent({ name: "assistant", model: model(...), ... }) },
    storage: new InMemoryStore(),
});
const a = mastra.getAgent("assistant");
const r = await a.generate("hi", { requireToolApproval: true });
if (r.finishReason === "suspended") {
    const final = await a.approveToolCallGenerate({
        runId: r.runId,
        toolCallId: r.suspendPayload!.toolCallId,
    });
}
```

See `fixtures/ts/agent/hitl/bus-approval/index.ts` for the full HITL wiring.

---

## Tools: `createTool`

```typescript
function createTool(config: ToolConfig): Tool;

interface ToolConfig {
    id: string;
    description?: string;
    inputSchema?:  ZodType;
    outputSchema?: ZodType;
    execute?: (input: any, context?: ToolExecutionContext) => Promise<unknown>;
    suspendSchema?: ZodType;
    resumeSchema?:  ZodType;
    requireApproval?: boolean;       // suspend before running — HITL
}

interface ToolExecutionContext {
    requestContext?: RequestContext;
}

interface Tool {
    id: string;
    description?: string;
    execute?: (input: Record<string, unknown>, context?: ToolExecutionContext) => Promise<unknown>;
}
```

Mastra tools are class-like — not the plain objects of AI SDK's `tool()`. The Mastra runtime sometimes wraps the call as `{ context, runtimeContext }`, sometimes passes raw args — normalise defensively:

```typescript
const calc = createTool({
    id: "add",
    description: "Add two numbers",
    inputSchema: z.object({ a: z.number(), b: z.number() }),
    execute: async (args) => {
        const { a, b } = (args && args.context) || args || {};
        return { result: a + b };
    },
});
kit.register("tool", "add", calc);
```

---

## Workflows

```typescript
function createWorkflow(config: WorkflowConfig): WorkflowBuilder;
function createStep(config: StepConfig): Step;

interface WorkflowConfig {
    id: string;
    inputSchema?:  ZodType;
    outputSchema?: ZodType;
    stateSchema?:  ZodType;
}

interface StepConfig {
    id: string;
    inputSchema?:  ZodType;
    outputSchema?: ZodType;
    stateSchema?:  ZodType;
    execute?: (context: StepExecutionContext) => Promise<unknown>;
}

interface StepExecutionContext {
    inputData: Record<string, any>;
    mapiData?: Record<string, unknown>;
    state: Record<string, any>;
    setState(keyOrUpdates: string | Record<string, any>, value?: any): void;
    getStepResult(stepId: string): any;
    suspend(data?: unknown): void;                  // HITL pause
    resumeData?: Record<string, any>;
}

interface WorkflowBuilder {
    then(step: Step): WorkflowBuilder;
    parallel(steps: Step[]): WorkflowBuilder;
    branch(config: BranchConfig | Step[][]): WorkflowBuilder;
    foreach(config: ForEachConfig): WorkflowBuilder;
    dountil(step: Step, condition?: (ctx: StepExecutionContext) => boolean | Promise<boolean>): WorkflowBuilder;
    sleep(ms: number): WorkflowBuilder;
    commit(): Workflow;
}

interface BranchConfig {
    condition: (data: Record<string, unknown>) => boolean;
    trueStep: Step;
    falseStep?: Step;
}

interface ForEachConfig {
    items: string;
    step: Step;
}

interface Workflow {
    createRun(opts?: { runId?: string }): Promise<WorkflowRun>;
}

interface WorkflowRun {
    runId: string;
    start(params: { inputData: Record<string, unknown> }): Promise<WorkflowRunResult>;
    resume(stepOrParams: string | { resumeData: Record<string, unknown>; step?: string },
           resumeData?: Record<string, unknown>): Promise<WorkflowRunResult>;
    cancel(): void;
    readonly status: string;
    readonly currentStep: string;
}

interface WorkflowRunResult {
    status: "completed" | "suspended" | "failed" | "success";
    result?: Record<string, unknown>;
    runId?: string;
    steps?: Record<string, StepRunResult>;
}

interface StepRunResult { status: string; output?: unknown; }
```

```typescript
const s1 = createStep({
    id: "normalize",
    inputSchema:  z.object({ message: z.string() }),
    outputSchema: z.object({ message: z.string() }),
    execute: async ({ inputData }) => ({ message: inputData.message.trim() }),
});

const wf = createWorkflow({
    id: "pipeline",
    inputSchema:  z.object({ message: z.string() }),
    outputSchema: z.object({ message: z.string() }),
}).then(s1).commit();

kit.register("workflow", "pipeline", wf);
```

Suspend/resume: any step can call `ctx.suspend({ why: … })`. The run's `status` becomes `"suspended"`; call `run.resume({ resumeData: … })` (or `run.resume(stepId, resumeData)`) to continue.

---

## Memory

```typescript
class Memory {
    constructor(config: MemoryConfig);

    createThread(opts: { resourceId: string; threadId?: string; title?: string; metadata?: Record<string, unknown> }): Promise<Thread>;
    saveThread(opts: { thread: Thread }): Promise<Thread>;
    getThreadById(opts: { threadId: string }): Promise<Thread | null>;
    updateThread(opts: { threadId: string; title?: string; metadata?: Record<string, unknown> }): Promise<Thread>;
    listThreads(filter?: { resourceId?: string; page?: number; perPage?: number }): Promise<Thread[]>;

    saveMessages(opts: { threadId: string; messages: Message[] }): Promise<void>;
    deleteMessages(opts: { threadId: string }): Promise<void>;
    recall(opts: { threadId: string; query?: string; resourceId?: string }): Promise<RecallResult>;

    deleteThread(threadId: string): Promise<void>;
    setStorage(storage: StorageInstance): void;
    setVector(vector: VectorStoreInstance): void;
}

interface Thread {
    id: string;
    resourceId: string;
    title?: string;
    createdAt: Date;
    updatedAt: Date;
    metadata?: Record<string, unknown>;
}

interface RecallResult {
    messages: Message[];
    workingMemory?: string;
}

interface MemoryConfig {
    storage?: StorageInstance;
    vector?:  VectorStoreInstance | false;     // false disables semantic recall
    embedder?: string | any;                   // "provider/model-id" or EmbeddingModel
    embedderOptions?: { dimensions?: number };
    options?: MemoryOptions;
}

interface MemoryOptions {
    readOnly?: boolean;                        // default false
    lastMessages?: number | false;             // default 10
    semanticRecall?: boolean | {
        topK?: number;
        messageRange?: number;
        scope?: "thread" | "resource";
    };
    workingMemory?: boolean | {
        enabled: boolean;
        scope?: "thread" | "resource";
        template?: string;
        schema?: ZodType;
        version?: "vnext";
    };
    generateTitle?: boolean | { model?: any; instructions?: string };
    observationalMemory?: boolean | {
        scope?: "thread" | "resource";
        model?: string;
        observation?: { messageTokens?: number };
        reflection?:  { observationTokens?: number };
    };
}

kit.register("memory", "main", new Memory({
    storage: new LibSQLStore({ url: "file:./mem.db" }),
    vector:  new LibSQLVector({ connectionUrl: "file:./mem.db" }),
    embedder: "openai/text-embedding-3-small",
    options: { lastMessages: 20, semanticRecall: { topK: 5 } },
}));
```

---

## Storage backends

All storage classes implement `StorageInstance`:

```typescript
interface StorageInstance { readonly __storageType: string; }

class InMemoryStore { constructor(config?: { id?: string }); }
class LibSQLStore   { constructor(config: { id?: string; url?: string; authToken?: string; storage?: string }); }
class UpstashStore  { constructor(config: { id?: string; url: string; token: string }); }
class PostgresStore { constructor(config: { id?: string; connectionString: string }); init(): Promise<void>; }
class MongoDBStore  { constructor(config: { id?: string; uri?: string; url?: string; dbName?: string }); init(): Promise<void>; }
```

`LibSQLStore` file-URL guard: `opts.url` must include `file:` (throws `BrainkitError("VALIDATION_ERROR")` otherwise). Use `"file:./data.db"` or `"file::memory:"`.

`PostgresStore` / `MongoDBStore` require `init()` before first use (creates tables / indexes).

---

## Vector stores

```typescript
interface VectorStoreInstance {
    readonly __vectorType: string;
    createIndex(opts: { indexName: string; dimension: number; metric?: string }): Promise<void>;
    listIndexes(): Promise<string[]>;
    describeIndex(indexName: string): Promise<any>;
    deleteIndex(indexName: string): Promise<void>;
    upsert(opts: { indexName: string; vectors: VectorEntry[]; metadata?: Record<string, unknown> }): Promise<string[]>;
    query(opts:  { indexName: string; queryVector: number[]; topK?: number; filter?: any }): Promise<VectorQueryResult[]>;
}

interface VectorEntry {
    id?: string;
    vector: number[];
    metadata?: Record<string, unknown>;
    [key: string]: any;
}

interface VectorQueryResult {
    id: string;
    score: number;
    metadata?: Record<string, unknown>;
    vector?: number[];
}

class LibSQLVector  { constructor(config: { id?: string; connectionUrl?: string; url?: string; authToken?: string; storage?: string }); }
class PgVector      { constructor(config: { id?: string; connectionString: string }); }
class MongoDBVector { constructor(config: { id?: string; uri: string; dbName?: string }); }
```

`LibSQLVector` file-URL guard: `opts.connectionUrl` (NOT `opts.url`) must include `file:` — throws `BrainkitError("VALIDATION_ERROR")` otherwise. `url` is tolerated as a secondary field for Mastra parity but the validation is against `connectionUrl`.

---

## Embedding router

```typescript
class ModelRouterEmbeddingModel {
    constructor(modelId: string);   // e.g. "openai/text-embedding-3-small"
}
```

Used where Mastra accepts an embedder string. For direct calls prefer `embeddingModel("openai", "text-embedding-3-small")` from `"kit"`.

---

## RAG

```typescript
class MDocument {
    static fromText(text: string, metadata?: Record<string, unknown>): MDocument;
    static fromMarkdown(markdown: string, metadata?: Record<string, unknown>): MDocument;
    chunk(options?: ChunkOptions): Promise<DocumentChunk[]>;
}

interface ChunkOptions {
    strategy?: "recursive" | "character" | "token" | "markdown" | "html";
    size?: number;
    maxSize?: number;
    overlap?: number;
    separator?: string;
    headers?: [string, string][] | Record<string, string>;
    [key: string]: any;
}

interface DocumentChunk {
    text: string;
    metadata: Record<string, unknown>;
}

class GraphRAG {
    constructor(config: { vectorStore: VectorStoreInstance; embedder: import("ai").EmbeddingModel });
    query(query: string, options?: { topK?: number }): Promise<GraphRAGResult>;
}

interface GraphRAGResult {
    answer: string;
    sources: Array<{ text: string; score: number }>;
}

function createVectorQueryTool(config: {
    vectorStore: VectorStoreInstance;
    indexName: string;
    embedder: import("ai").EmbeddingModel;
    topK?: number;
    description?: string;
}): Tool;

function createDocumentChunkerTool(config: {
    vectorStore: VectorStoreInstance;
    indexName: string;
    embedder: import("ai").EmbeddingModel;
    chunkOptions?: ChunkOptions;
}): Tool;

function createGraphRAGTool(config: {
    graphRag: GraphRAG;
    description?: string;
}): Tool;

function rerank(config: {
    results: Array<{ text: string; score: number }>;
    query: string;
    topK?: number;
}): Promise<Array<{ text: string; score: number }>>;

function rerankWithScorer(config: {
    results: Array<{ text: string; score: number }>;
    query: string;
    scorer: Scorer;
    topK?: number;
}): Promise<Array<{ text: string; score: number }>>;
```

---

## Observability

```typescript
class Observability {
    constructor(config: ObservabilityConfig);
}

interface ObservabilityConfig {
    configs: Record<string, {
        serviceName: string;
        exporters: DefaultExporter[];
    }>;
}

class DefaultExporter {
    constructor(config: { storage: StorageInstance; strategy?: "realtime" | "batch" });
}

class SensitiveDataFilter {
    constructor(config?: { patterns?: RegExp[]; replacement?: string });
}
```

Traces flow through the kit's own tracing module when the deployment lives inside brainkit; Mastra's `Observability` is for cases where you want a parallel Mastra-native trace sink.

---

## Evals (`createScorer` + `runEvals`)

```typescript
function createScorer(config: ScorerConfig): ScorerBuilder;

interface ScorerConfig {
    id: string;
    name?: string;
    description?: string;
    type?: "agent" | { input: ZodType; output: ZodType };
    judge?: { model: any; instructions?: string };       // LLM-as-judge config
}

interface ScorerBuilder {
    preprocess(fn: (ctx: { run: ScorerRunContext; results?: any }) => any | Promise<any>): ScorerBuilder;
    analyze(fn:    (ctx: { results: any; run: ScorerRunContext }) => any | Promise<any>): ScorerBuilder;
    generateScore(fnOrConfig:
        | ((ctx: { results: any; run: ScorerRunContext }) => number | Promise<number>)
        | { description: string; judge?: { model: any; instructions: string };
            createPrompt: (ctx: { results: any; run: ScorerRunContext }) => string | Promise<string> }
    ): ScorerBuilder;
    generateReason(fnOrConfig:
        | ((ctx: { results: any; run: ScorerRunContext }) => string | Promise<string>)
        | { description: string; judge?: { model: any; instructions: string };
            createPrompt: (ctx: { results: any; run: ScorerRunContext }) => string | Promise<string> }
    ): ScorerBuilder;
    run(input: ScorerRunInput): Promise<ScorerRunResult>;
}

interface ScorerRunContext { input: any; output: any; }
interface ScorerRunInput   { input: Array<{ role: string; content: string }>; output: { role: string; text: string }; }
interface ScorerRunResult  { score: number; reason?: string; runId: string; [key: string]: any; }

type Scorer = ScorerBuilder;

function runEvals(config: {
    scorers: Record<string, { scorer: Scorer; sampling?: any }>;
    dataset: Array<{ input: any; output: any }>;
}): Promise<EvalRunResult>;

interface EvalRunResult {
    results: Array<{
        input: any;
        output: any;
        scores: Record<string, ScorerRunResult>;
    }>;
}
```

Builder chain: `.preprocess()` → `.analyze()` → `.generateScore()` → `.generateReason()` → `.run()`. Only `generateScore` is required; all steps are optional otherwise.

---

## Workspace + sandboxes

```typescript
class Workspace {
    constructor(config: WorkspaceConfig);
    init(): Promise<void>;
    destroy(): Promise<void>;
    search(query: string, options?: { limit?: number }): Promise<WorkspaceSearchResult[]>;
    index(filePath: string, content: string): Promise<void>;
    getInfo(): WorkspaceInfo;
    getInstructions(): string;
}

interface WorkspaceSearchResult { path: string; content: string; score: number; }
interface WorkspaceInfo         { id: string; name?: string; basePath: string; }

class LocalFilesystem {
    constructor(config: { basePath: string; allowedPaths?: string[]; contained?: boolean });
}

class LocalSandbox {
    constructor(config?: { workingDirectory?: string; env?: Record<string, string>; defaultShell?: string });
}

interface WorkspaceConfig {
    id?: string;
    name?: string;
    filesystem: LocalFilesystem;
    sandbox?: LocalSandbox;
    bm25?: boolean | { k1?: number; b?: number };
    vectorStore?: VectorStoreInstance;
    embedder?: (text: string) => Promise<number[]>;
    searchIndexName?: string;
    tools?: Record<string, Tool>;
    skills?: string[];
    lsp?: boolean | { command: string; args?: string[] };
}
```

Wire a workspace into an agent via `agent.config.workspace`. Sandboxed shell + file surface; only enable when the deployment is trusted.

---

## RequestContext

```typescript
class RequestContext {
    constructor(entries?: Array<[string, string]>);
    get(key: string): string | undefined;
    set(key: string, value: string): void;
    has(key: string): boolean;
}
```

Pass a `RequestContext` via `options.requestContext`; dynamic `instructions` / `tools` / `agents` resolvers receive it.

---

## Voice

```typescript
// Voice providers — endowed via the "agent" module.
import { OpenAIVoice, CompositeVoice } from "agent";

// One provider for both speak + listen.
const voice = new OpenAIVoice();                      // uses OPENAI_API_KEY
const stream = await voice.speak("hello",            // → Node Readable
    { responseFormat: "mp3" });
const text   = await voice.listen(stream,             // → string transcript
    { filetype: "mp3" });

// Or mix providers per leg.
const split = new CompositeVoice({
    speakProvider:  new OpenAIVoice(),
    listenProvider: new OpenAIVoice(),
});

// Wire on an Agent.
const agent = new Agent({
    name: "voice-agent",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You answer concisely.",
    voice,
});
// agent.voice.speak(...) / agent.voice.listen(...) are available.
```

Playback uses the **web-standard `Audio` polyfill**: `new
Audio(streamOrBufferOrPath).play()`. Resolves URL / path /
Buffer / Uint8Array / Blob / Node Readable / Web ReadableStream.
Route the bytes by wiring `Config.Audio` on the Go side
(`brainkit/audio/local.New()` for desktop speakers, `audio.Func(fn)`
for bus / HTTP fan-out, `audio.Composite(...)` for multi-sink).
Without a sink, `play()` resolves silently so portable agent code
runs unchanged on headless kits. See `examples/voice-agent/` for
the full round trip.

## HITL (human-in-the-loop) recap

- **Tool-level:** `createTool({ requireApproval: true })` — the tool call suspends with `AgentResult.suspendPayload` populated and `runId` / `toolCallId` captured. Resume with `agent.approveToolCallGenerate({ runId, toolCallId })` or `declineToolCallGenerate(...)`. Streaming variants exist.
- **Workflow-level:** inside a step's `execute`, call `ctx.suspend(data)` — the run's `status` becomes `"suspended"`. Resume with `run.resume({ resumeData })`.
- **Generic wrapper:** `generateWithApproval` from `"kit"` (see `ts-runtime.md`) — externalises approval over the brainkit bus for any handler.

---

## Example — agent + memory + tool + bus

```typescript
import {
    Agent, createTool,
    Memory, LibSQLStore, LibSQLVector,
    z,
} from "agent";
import { model, kit, bus } from "kit";

const search = createTool({
    id: "search",
    description: "Search the knowledge base",
    inputSchema: z.object({ q: z.string() }),
    execute: async (args) => {
        const { q } = (args && args.context) || args;
        return { hits: ["…results for " + q] };
    },
});

const memory = new Memory({
    storage: new LibSQLStore({ url: "file:./mem.db" }),
    vector:  new LibSQLVector({ connectionUrl: "file:./mem.db" }),
    embedder: "openai/text-embedding-3-small",
    options:  { lastMessages: 20, semanticRecall: { topK: 5 } },
});

const agent = new Agent({
    name: "researcher",
    model: model("openai", "gpt-4o-mini"),
    instructions: "You research topics thoroughly, citing sources.",
    tools: { search },
    memory,
});

kit.register("agent", "researcher", agent);

bus.on("ask", async (msg) => {
    const { prompt, threadId, user } = msg.payload || {};
    const r = await agent.generate(String(prompt || ""), {
        memory: { thread: threadId, resource: user },
    });
    msg.reply({
        text: r.text,
        usage: r.usage,                     // v4 names
        runId: r.runId,
    });
});
```
