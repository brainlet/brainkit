/**
 * "agent" module — direct re-exports of Mastra framework.
 * Does NOT re-export AI SDK functions (those come from "ai").
 *
 * @example
 * ```ts
 * import { Agent, createTool, z } from "agent";
 * import { model, kit, bus } from "kit";
 *
 * const myAgent = new Agent({
 *   name: "researcher",
 *   model: model("openai", "gpt-4o-mini"),
 *   instructions: "You research topics thoroughly.",
 *   tools: { search: searchTool },
 * });
 * kit.register("agent", "researcher", myAgent);
 * ```
 */
declare module "agent" {

  // ── Zod (also exported here for convenience with createTool) ──

  export const z: import("ai").Zod;

  // ── Agent ─────────────────────────────────────────────────────

  export class Agent {
    constructor(config: AgentConfig);

    /** Generate a response (non-streaming). */
    generate(promptOrMessages: string | Message[], options?: AgentCallOptions): Promise<AgentResult>;

    /** Stream a response with real-time tokens. */
    stream(promptOrMessages: string | Message[], options?: AgentCallOptions): Promise<AgentStreamResult>;

    /** Supervisor mode — delegates to sub-agents. */
    network(promptOrMessages: string | Message[], options?: AgentCallOptions): Promise<AgentResult>;
  }

  interface AgentConfig {
    /** Display name. */
    name?: string;
    /** Unique ID. */
    id?: string;
    /** Description. */
    description?: string;
    /** System instructions (static string or dynamic resolver). */
    instructions?: string | ((ctx: { requestContext?: RequestContext }) => string | Promise<string>);
    /** Language model (use model() from "kit" to resolve). Accepts static or dynamic resolver. */
    model: any;
    /** Tool definitions (static or dynamic resolver). */
    tools?: Record<string, Tool> | ((ctx: { requestContext?: RequestContext }) => Record<string, Tool> | Promise<Record<string, Tool>>);
    /** Max tool-call rounds (default: 5). */
    maxSteps?: number;
    /** Tool selection strategy. */
    toolChoice?: "auto" | "none" | "required" | { type: "tool"; toolName: string };
    /** Memory instance for conversation persistence. */
    memory?: Memory;
    /** Sub-agents for supervisor/network pattern. */
    agents?: Record<string, Agent>;
    /** Default call options. */
    defaultOptions?: Partial<AgentCallOptions>;
    /** Dynamic model resolver. */
    modelResolver?: (ctx: RequestContext) => import("ai").LanguageModel | Promise<import("ai").LanguageModel>;
    /** Dynamic tools resolver. */
    toolsResolver?: (ctx: RequestContext) => Record<string, Tool> | Promise<Record<string, Tool>>;
    /** Dynamic instructions resolver. */
    instructionsResolver?: (ctx: RequestContext) => string | Promise<string>;
    /** Scorer definitions for evals. */
    scorers?: Scorer[];
    /** Workspace instance. */
    workspace?: Workspace;
    /** Workflow definitions. */
    workflows?: Record<string, Workflow>;
    /** Provider-specific options. */
    providerOptions?: Record<string, Record<string, unknown>>;
  }

  interface AgentCallOptions {
    /** Override max steps for this call. */
    maxSteps?: number;
    /** Memory options. */
    memory?: { thread?: string | { id: string }; resource?: string };
    /** Request context for dynamic resolvers. */
    requestContext?: RequestContext;
    /** Model-specific settings (temperature, etc). */
    modelSettings?: Record<string, any>;
    /** Callback when a step finishes. */
    onStepFinish?: (event: any) => void | Promise<void>;
    /** Callback when generation finishes. */
    onFinish?: (event: any) => void | Promise<void>;
    /** Output schema for structured output. */
    output?: import("ai").ZodType;
    /** Limit which tools are active. */
    activeTools?: string[];
    /** Extra options passthrough. */
    [key: string]: any;
  }

  interface AgentResult {
    text: string;
    reasoningText?: string;
    object?: unknown;
    toolCalls: Array<{ toolCallId: string; toolName: string; args: Record<string, unknown> }>;
    toolResults: Array<{ toolCallId: string; toolName: string; args: Record<string, unknown>; result: unknown }>;
    finishReason: string;
    usage: { promptTokens: number; completionTokens: number; totalTokens: number };
    steps: AgentStepResult[];
    response: { id: string; modelId: string; timestamp: string };
    runId?: string;
    traceId?: string;
    suspendPayload?: unknown;
    providerMetadata?: Record<string, unknown>;
  }

  interface AgentStepResult {
    text: string;
    reasoning?: string;
    toolCalls: Array<{ toolCallId: string; toolName: string; args: Record<string, unknown> }>;
    toolResults: Array<{ toolCallId: string; toolName: string; args: Record<string, unknown>; result: unknown }>;
    finishReason: string;
    usage: { promptTokens: number; completionTokens: number; totalTokens: number };
    stepType: string;
    isContinued: boolean;
  }

  interface AgentStreamResult {
    /** Async iterable of text chunks. */
    textStream: AsyncIterable<string>;
    /** Async iterable of typed stream parts. */
    fullStream: AsyncIterable<import("ai").StreamPart>;
    /** Promise: final complete text. */
    text: Promise<string>;
    /** Promise: token usage. */
    usage: Promise<{ promptTokens: number; completionTokens: number; totalTokens: number }>;
    /** Promise: finish reason. */
    finishReason: Promise<string>;
    /** Promise: all tool calls. */
    toolCalls: Promise<Array<{ toolCallId: string; toolName: string; args: Record<string, unknown> }>>;
    /** Promise: all tool results. */
    toolResults: Promise<Array<{ toolCallId: string; toolName: string; args: Record<string, unknown>; result: unknown }>>;
    /** Promise: all steps. */
    steps: Promise<AgentStepResult[]>;
  }

  interface Message {
    role: "system" | "user" | "assistant" | "tool";
    content: import("ai").MessageContent;
  }

  // ── Tools ─────────────────────────────────────────────────────

  /**
   * Create a Mastra tool definition.
   *
   * @example
   * ```ts
   * const calculator = createTool({
   *   id: "add",
   *   description: "Add two numbers",
   *   inputSchema: z.object({ a: z.number(), b: z.number() }),
   *   execute: async ({ a, b }) => ({ result: a + b }),
   * });
   * kit.register("tool", "add", calculator);
   * ```
   */
  export function createTool(config: ToolConfig): Tool;

  interface ToolConfig {
    /** Tool ID / name. */
    id: string;
    /** Human-readable description. */
    description?: string;
    /** Input schema (Zod). */
    inputSchema?: import("ai").ZodType;
    /** Output schema (Zod). */
    outputSchema?: import("ai").ZodType;
    /** Execute function. Input is the Zod-validated data. */
    execute?: (input: any, context?: ToolExecutionContext) => Promise<unknown>;
    /** Suspend schema (for tool approval workflows). */
    suspendSchema?: import("ai").ZodType;
    /** Resume schema. */
    resumeSchema?: import("ai").ZodType;
  }

  interface ToolExecutionContext {
    requestContext?: RequestContext;
  }

  interface Tool {
    id: string;
    description?: string;
    execute?: (input: Record<string, unknown>, context?: ToolExecutionContext) => Promise<unknown>;
  }

  // ── Workflows ─────────────────────────────────────────────────

  /**
   * Create a Mastra workflow — multi-step pipeline with typed data flow.
   *
   * @example
   * ```ts
   * const wf = createWorkflow({
   *   id: "my-pipeline",
   *   inputSchema: z.object({ message: z.string() }),
   *   outputSchema: z.object({ result: z.string() }),
   * }).then(step1).then(step2).commit();
   * ```
   */
  export function createWorkflow(config: WorkflowConfig): WorkflowBuilder;

  /** Create a workflow step. */
  export function createStep(config: StepConfig): Step;

  interface WorkflowConfig {
    id: string;
    inputSchema?: import("ai").ZodType;
    outputSchema?: import("ai").ZodType;
    stateSchema?: import("ai").ZodType;
  }

  interface StepConfig {
    id: string;
    inputSchema?: import("ai").ZodType;
    outputSchema?: import("ai").ZodType;
    stateSchema?: import("ai").ZodType;
    execute?: (context: StepExecutionContext) => Promise<unknown>;
  }

  interface StepExecutionContext {
    inputData: Record<string, any>;
    mapiData?: Record<string, unknown>;
    /** Shared workflow state (read). */
    state: Record<string, any>;
    /** Update shared workflow state. */
    setState(keyOrUpdates: string | Record<string, any>, value?: any): void;
    /** Get result of a previous step by step ID. */
    getStepResult(stepId: string): any;
    /** Suspend the workflow (HITL pattern). */
    suspend(data?: unknown): void;
    /** Data passed when resuming a suspended workflow. */
    resumeData?: Record<string, any>;
  }

  interface WorkflowBuilder {
    then(step: Step): WorkflowBuilder;
    parallel(steps: Step[]): WorkflowBuilder;
    branch(config: BranchConfig | Step[][]): WorkflowBuilder;
    forEach(config: ForEachConfig): WorkflowBuilder;
    /** Loop until condition is met. */
    dountil(step: Step, condition?: (context: StepExecutionContext) => boolean | Promise<boolean>): WorkflowBuilder;
    /** Pause execution for a duration (ms). */
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

  interface Step {}

  interface Workflow {
    createRun(opts?: { runId?: string }): Promise<WorkflowRun>;
  }

  interface WorkflowRun {
    runId: string;
    start(params: { inputData: Record<string, unknown> }): Promise<WorkflowRunResult>;
    resume(stepOrParams: string | { resumeData: Record<string, unknown>; step?: string }, resumeData?: Record<string, unknown>): Promise<WorkflowRunResult>;
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

  interface StepRunResult {
    status: string;
    output?: unknown;
  }

  // ── Memory ────────────────────────────────────────────────────

  /** Memory class — conversation persistence. */
  export class Memory {
    constructor(config: MemoryConfig);
    createThread(opts?: { resourceId?: string }): Promise<{ id: string }>;
    saveThread(opts: { thread: { id: string; resourceId?: string; title?: string; createdAt?: string; [key: string]: any } }): Promise<Thread>;
    getThreadById(opts: { threadId: string }): Promise<Thread | null>;
    updateThread(opts: { threadId: string; title?: string }): Promise<Thread>;
    listThreads(filter?: { resourceId?: string }): Promise<Thread[]>;
    saveMessages(opts: { threadId: string; messages: Message[] }): Promise<void>;
    deleteMessages(opts: { threadId: string }): Promise<void>;
    recall(opts: { threadId: string; query?: string; resourceId?: string }): Promise<RecallResult>;
    deleteThread(threadId: string): Promise<void>;
  }

  interface Thread {
    id: string;
    resourceId?: string;
    title?: string;
    createdAt: string;
    updatedAt: string;
  }

  interface RecallResult {
    messages: Message[];
    workingMemory?: string;
  }

  interface MemoryConfig {
    storage?: StorageInstance;
    vector?: VectorStoreInstance | false;
    embedder?: any;
    options?: {
      lastMessages?: number;
      semanticRecall?: boolean | { topK?: number; messageRange?: number };
      workingMemory?: boolean | { enabled: boolean; template?: string };
      generateTitle?: boolean;
      observationalMemory?: boolean | { enabled: boolean };
    };
  }

  // ── Storage backends ──────────────────────────────────────────

  /** Marker type for resolved storage instances. */
  interface StorageInstance {
    /** @internal */ readonly __storageType: string;
  }

  export class InMemoryStore implements StorageInstance {
    readonly __storageType: "memory";
    constructor(config?: { id?: string });
  }

  export class LibSQLStore implements StorageInstance {
    readonly __storageType: "libsql";
    constructor(config: { id?: string; url?: string; authToken?: string; storage?: string });
  }

  export class UpstashStore implements StorageInstance {
    readonly __storageType: "upstash";
    constructor(config: { id?: string; url: string; token: string });
  }

  export class PostgresStore implements StorageInstance {
    readonly __storageType: "postgres";
    constructor(config: { id?: string; connectionString: string });
    init(): Promise<void>;
  }

  export class MongoDBStore implements StorageInstance {
    readonly __storageType: "mongodb";
    constructor(config: { id?: string; uri?: string; url?: string; dbName?: string });
    init(): Promise<void>;
  }

  // ── Vector stores ─────────────────────────────────────────────

  /** Marker type for resolved vector store instances. */
  interface VectorStoreInstance {
    /** @internal */ readonly __vectorType: string;
    createIndex(opts: { indexName: string; dimension: number; metric?: string }): Promise<void>;
    listIndexes(): Promise<string[]>;
    describeIndex(indexName: string): Promise<any>;
    deleteIndex(indexName: string): Promise<void>;
    upsert(opts: { indexName: string; vectors: VectorEntry[]; metadata?: Record<string, unknown> }): Promise<string[]>;
    query(opts: { indexName: string; queryVector: number[]; topK?: number; filter?: any }): Promise<VectorQueryResult[]>;
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

  export class LibSQLVector implements VectorStoreInstance {
    readonly __vectorType: "libsql";
    constructor(config: { id?: string; connectionUrl?: string; url?: string; authToken?: string; storage?: string });
    createIndex(opts: { indexName: string; dimension: number; metric?: string }): Promise<void>;
    listIndexes(): Promise<string[]>;
    describeIndex(indexName: string): Promise<any>;
    deleteIndex(indexName: string): Promise<void>;
    upsert(opts: { indexName: string; vectors: VectorEntry[]; metadata?: Record<string, unknown> }): Promise<string[]>;
    query(opts: { indexName: string; queryVector: number[]; topK?: number; filter?: any }): Promise<VectorQueryResult[]>;
  }

  export class PgVector implements VectorStoreInstance {
    readonly __vectorType: "pgvector";
    constructor(config: { id?: string; connectionString: string });
    createIndex(opts: { indexName: string; dimension: number; metric?: string }): Promise<void>;
    listIndexes(): Promise<string[]>;
    describeIndex(indexName: string): Promise<any>;
    deleteIndex(indexName: string): Promise<void>;
    upsert(opts: { indexName: string; vectors: VectorEntry[]; metadata?: Record<string, unknown> }): Promise<string[]>;
    query(opts: { indexName: string; queryVector: number[]; topK?: number; filter?: any }): Promise<VectorQueryResult[]>;
  }

  export class MongoDBVector implements VectorStoreInstance {
    readonly __vectorType: "mongodb";
    constructor(config: { id?: string; uri: string; dbName?: string });
    createIndex(opts: { indexName: string; dimension: number; metric?: string }): Promise<void>;
    listIndexes(): Promise<string[]>;
    describeIndex(indexName: string): Promise<any>;
    deleteIndex(indexName: string): Promise<void>;
    upsert(opts: { indexName: string; vectors: VectorEntry[]; metadata?: Record<string, unknown> }): Promise<string[]>;
    query(opts: { indexName: string; queryVector: number[]; topK?: number; filter?: any }): Promise<VectorQueryResult[]>;
  }

  // ── Embedding model router ────────────────────────────────────

  /** Routes "provider/model-id" strings to embedding model instances. */
  export class ModelRouterEmbeddingModel {
    constructor(modelId: string);
  }

  // ── RequestContext ─────────────────────────────────────────────

  /** Key-value context for dynamic config resolvers. */
  export class RequestContext {
    constructor(entries?: Array<[string, string]>);
    get(key: string): string | undefined;
    set(key: string, value: string): void;
    has(key: string): boolean;
  }

  // ── Workspace ─────────────────────────────────────────────────

  export class Workspace {
    constructor(config: WorkspaceConfig);
    init(): Promise<void>;
    destroy(): Promise<void>;
    search(query: string, options?: { limit?: number }): Promise<WorkspaceSearchResult[]>;
    index(filePath: string, content: string): Promise<void>;
    getInfo(): WorkspaceInfo;
    getInstructions(): string;
  }

  interface WorkspaceSearchResult {
    path: string;
    content: string;
    score: number;
  }

  interface WorkspaceInfo {
    id: string;
    name?: string;
    basePath: string;
  }

  export class LocalFilesystem {
    constructor(config: { basePath: string; allowedPaths?: string[]; contained?: boolean });
  }

  export class LocalSandbox {
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

  // ── RAG ───────────────────────────────────────────────────────

  /** Document class for RAG chunking. */
  export class MDocument {
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

  /** Graph RAG for knowledge graph queries. */
  export class GraphRAG {
    constructor(config: { vectorStore: VectorStoreInstance; embedder: import("ai").EmbeddingModel });
    query(query: string, options?: { topK?: number }): Promise<GraphRAGResult>;
  }

  interface GraphRAGResult {
    answer: string;
    sources: Array<{ text: string; score: number }>;
  }

  export function createVectorQueryTool(config: {
    vectorStore: VectorStoreInstance;
    indexName: string;
    embedder: import("ai").EmbeddingModel;
    topK?: number;
    description?: string;
  }): Tool;

  export function createDocumentChunkerTool(config: {
    vectorStore: VectorStoreInstance;
    indexName: string;
    embedder: import("ai").EmbeddingModel;
    chunkOptions?: ChunkOptions;
  }): Tool;

  export function createGraphRAGTool(config: {
    graphRag: GraphRAG;
    description?: string;
  }): Tool;

  export function rerank(config: {
    results: Array<{ text: string; score: number }>;
    query: string;
    topK?: number;
  }): Promise<Array<{ text: string; score: number }>>;

  export function rerankWithScorer(config: {
    results: Array<{ text: string; score: number }>;
    query: string;
    scorer: Scorer;
    topK?: number;
  }): Promise<Array<{ text: string; score: number }>>;

  // ── Observability ─────────────────────────────────────────────

  export class Observability {
    constructor(config: ObservabilityConfig);
  }

  interface ObservabilityConfig {
    configs: Record<string, {
      serviceName: string;
      exporters: DefaultExporter[];
    }>;
  }

  export class DefaultExporter {
    constructor(config: { storage: StorageInstance; strategy?: "realtime" | "batch" });
  }

  // ── Evals ─────────────────────────────────────────────────────

  /** Create a custom scorer for agent evaluation. */
  export function createScorer(config: {
    id?: string;
    name?: string;
    description?: string;
    judge?: { model: import("ai").LanguageModel };
    execute?: (input: ScorerInput) => Promise<ScorerResult>;
  }): Scorer;

  interface ScorerInput {
    input: string;
    output: string;
    groundTruth?: string;
  }

  interface ScorerResult {
    score: number;
    details?: Record<string, unknown>;
  }

  /** Run evaluations against an agent. */
  export function runEvals(config: {
    target: Agent;
    scorers: Scorer[];
    dataset: Array<{ input: string; output?: string; groundTruth?: string }>;
  }): Promise<EvalRunResult>;

  interface EvalRunResult {
    results: Array<{
      input: string;
      output: string;
      scores: Record<string, ScorerResult>;
    }>;
  }

  interface Scorer {
    run(input: ScorerInput): Promise<ScorerResult>;
  }
}
