# Mastra Framework — API Reference for brainkit

> `import { Agent, createTool, createWorkflow, createStep, Memory, ... } from "agent";`
> Types verified against actual Mastra source and brainkit fixtures.

## Agent

```typescript
class Agent {
    constructor(config: AgentConfig);
    generate(promptOrMessages: string | Message[], options?: AgentCallOptions): Promise<AgentResult>;
    stream(promptOrMessages: string | Message[], options?: AgentCallOptions): Promise<AgentStreamResult>;
}
```

### AgentConfig

```typescript
interface AgentConfig {
    name?: string;
    id?: string;
    description?: string;
    instructions?: string | ((ctx: { requestContext?: RequestContext }) => string | Promise<string>);
    model: LanguageModel;  // use model("openai", "gpt-4o-mini") from "kit"
    tools?: Record<string, Tool> | ((ctx: { requestContext?: RequestContext }) => Record<string, Tool>);
    maxSteps?: number;     // default: 5
    toolChoice?: "auto" | "none" | "required" | { type: "tool"; toolName: string };
    memory?: Memory;
    agents?: Record<string, Agent>;  // sub-agents for delegation
    defaultOptions?: Partial<AgentCallOptions>;
}
```

### AgentCallOptions

```typescript
interface AgentCallOptions {
    modelSettings?: { temperature?: number; maxTokens?: number; topP?: number; };
    activeTools?: string[];
    toolChoice?: "auto" | "none" | "required";
    instructions?: string;
    maxSteps?: number;
    structuredOutput?: { schema: ZodType };
    onStepFinish?: (step: StepResult) => void;
    onFinish?: (result: AgentResult) => void;
    onError?: (error: { error: Error }) => void;
    memory?: { thread?: { id: string }; resource?: string };
    threadId?: string;
    resourceId?: string;
    requestContext?: RequestContext;
    abortSignal?: AbortSignal;
    requireToolApproval?: boolean;
}
```

### AgentResult

```typescript
interface AgentResult {
    text: string;
    reasoning?: string;
    toolCalls: ToolCall[];
    toolResults: ToolResult[];
    finishReason: FinishReason;
    usage: Usage;
    steps: StepResult[];
    response: ResponseMeta;
    runId?: string;
    suspendPayload?: { toolCallId: string; toolName: string; args: any };
}
```

## createTool

```typescript
function createTool(config: ToolConfig): Tool;

interface ToolConfig {
    id: string;
    description: string;
    inputSchema?: ZodType;
    outputSchema?: ZodType;
    requireApproval?: boolean;
    execute: (input: { context: any }, opts?: { requestContext?: any }) => Promise<any>;
}
```

**Rule:** `createTool()` creates a tool object. It does NOT register it. Call `kit.register("tool", name, tool)` to make it discoverable.

## createWorkflow / createStep

```typescript
function createWorkflow(config: WorkflowConfig): WorkflowBuilder;
function createStep(config: StepConfig): Step;

interface WorkflowConfig {
    id: string;
    inputSchema: ZodType;
    outputSchema: ZodType;
}

interface StepConfig {
    id: string;
    inputSchema: ZodType;
    outputSchema: ZodType;
    execute: (ctx: { inputData: any; suspend?: (data: any) => Promise<void> }) => Promise<any>;
}

interface WorkflowBuilder {
    then(step: Step): WorkflowBuilder;
    branch(conditions: [predicate, Step][]): WorkflowBuilder;
    parallel(steps: Step[]): WorkflowBuilder;
    commit(): Workflow;
}

interface Workflow {
    createRun(): Promise<WorkflowRun>;
}

interface WorkflowRun {
    start(opts: { inputData: any }): Promise<WorkflowResult>;
}

interface WorkflowResult {
    status: "success" | "failed" | "suspended";
    result: any;
}
```

## Memory

```typescript
class Memory {
    constructor(config: { storage: StorageInstance });
}
```

Used on Agent: `new Agent({ memory: new Memory({ storage: store }) })`.

## Storage Backends

```typescript
class InMemoryStore { constructor(); }
class LibSQLStore { constructor(opts: { id: string; url?: string; authToken?: string; storage?: string }); }
class PostgresStore { constructor(opts: { id?: string; connectionString: string }); }
class MongoDBStore { constructor(opts: { id?: string; uri: string; dbName: string }); }
class UpstashStore { constructor(opts: { id?: string; url: string; token: string }); }
```

## Vector Backends

```typescript
class LibSQLVector {
    constructor(opts: { id: string; connectionUrl?: string; authToken?: string; storage?: string });
    createIndex(name: string, dimension: number, metric?: string): Promise<void>;
    upsert(index: string, vectors: VectorRecord[]): Promise<void>;
    query(index: string, embedding: number[], topK?: number): Promise<QueryResult[]>;
    listIndexes(): Promise<IndexInfo[]>;
    deleteIndex(name: string): Promise<void>;
}

class PgVector {
    constructor(opts: { id: string; connectionString: string });
    // same methods as LibSQLVector
}

class MongoDBVector {
    constructor(opts: { id: string; uri: string; dbName: string });
    // same methods as LibSQLVector
}
```

## RAG

```typescript
class MDocument {
    static fromText(text: string): MDocument;
    static fromMarkdown(md: string): MDocument;
    chunk(opts: { strategy: "recursive" | "markdown" | "token"; size: number; overlap?: number }): Promise<MDocument[]>;
    getText(): string;
}

class GraphRAG {
    constructor(opts: { model: LanguageModel; vectorStore: VectorStore; indexName: string; embeddingModel: EmbeddingModel });
    addDocuments(docs: MDocument[]): Promise<void>;
    query(query: string): Promise<any>;
}

function createVectorQueryTool(opts: { vectorStoreName: string; indexName: string; model: EmbeddingModel }): Tool;
function createDocumentChunkerTool(opts: { strategy: string; size: number; overlap?: number }): Tool;
function createGraphRAGTool(opts: { graphRag: GraphRAG; description: string }): Tool;
function rerank(results: any[], query: string, opts: { model: LanguageModel; topK?: number }): Promise<any[]>;
function rerankWithScorer(results: any[], query: string, opts: { scorer: Function; topK?: number }): Promise<any[]>;
```

## Evals

```typescript
function createScorer(opts: { name: string; description?: string }): ScorerBuilder;

interface ScorerBuilder {
    preprocess(fn: (ctx: { output: string }) => any): ScorerBuilder;
    generateScore(fn: ((ctx: { output: string; expectedOutput?: string; preprocessResult?: any }) => number) | { model: LanguageModel; instructions: string; outputSchema: ZodType }): ScorerBuilder;
    generateReason(fn: (ctx: { score: number }) => string): ScorerBuilder;
}

function runEvals(opts: {
    agent: Agent;
    data: { input: string; expectedOutput?: string }[];
    scorers: ScorerBuilder[];
    concurrency?: number;
}): Promise<{ scores: Record<string, number>; summary: { totalItems: number } }>;
```

## Observability

```typescript
class Observability {
    constructor(opts: { configs: Record<string, { serviceName: string; exporters: Exporter[] }> });
}

class DefaultExporter {
    constructor(opts: { storage: StorageInstance; strategy?: "realtime" | "batch" });
}
```

## Other Exports

```typescript
class RequestContext {
    constructor(data?: Record<string, any>);
    get(key: string): any;
}

class Workspace {
    constructor(opts: { id: string; filesystem: LocalFilesystem; sandbox?: LocalSandbox; bm25?: boolean });
}

class LocalFilesystem {
    constructor(opts: { basePath: string; allowedPaths?: string[] });
}

class LocalSandbox {
    constructor(opts: { workingDirectory: string });
}

class ModelRouterEmbeddingModel { /* for multi-model routing */ }

const z: ZodInstance; // Zod v4 — same instance as in "ai" module
```

## Shared Types

```typescript
type FinishReason = "stop" | "length" | "content-filter" | "tool-calls" | "error" | "suspended" | "other";
interface Usage { promptTokens: number; completionTokens: number; totalTokens: number; }
interface ResponseMeta { id: string; modelId: string; timestamp: Date; }
interface ToolCall { toolCallId: string; toolName: string; args: Record<string, unknown>; }
interface ToolResult { toolCallId: string; toolName: string; args: any; result: any; }
interface StepResult { text: string; reasoning?: string; toolCalls: ToolCall[]; toolResults: ToolResult[]; finishReason: FinishReason; usage: Usage; stepType: string; isContinued: boolean; }
```
