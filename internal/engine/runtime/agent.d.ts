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

    /** Resume a suspended tool call with approval. HITL pattern. */
    approveToolCallGenerate(opts: { runId: string; toolCallId: string }): Promise<AgentResult>;

    /** Decline a suspended tool call. HITL pattern. */
    declineToolCallGenerate(opts: { runId: string; toolCallId: string }): Promise<AgentResult>;

    /** Resume a suspended tool call with approval (streaming). */
    approveToolCallStream(opts: { runId: string; toolCallId: string }): Promise<AgentStreamResult>;

    /** Decline a suspended tool call (streaming). */
    declineToolCallStream(opts: { runId: string; toolCallId: string }): Promise<AgentStreamResult>;
  }

  export interface AgentConfig {
    /** Agent identifier. Defaults to name if omitted. */
    id?: string;
    /** Display name. */
    name: string;
    /** Description of the agent's purpose and capabilities. */
    description?: string;
    /** Instructions that guide behavior. String, string[], or dynamic resolver. */
    instructions: string | string[] | ((ctx: { requestContext?: RequestContext }) => string | string[] | Promise<string | string[]>);
    /** Language model. Use model() from "kit", or a dynamic resolver, or model-with-retries array. */
    model: any;
    /** Max retries for model calls on failure. @default 0 */
    maxRetries?: number;
    /** Tool definitions. Static map or dynamic resolver. */
    tools?: Record<string, Tool> | ((ctx: { requestContext?: RequestContext }) => Record<string, Tool> | Promise<Record<string, Tool>>);
    /** Workflows the agent can execute. Static or dynamic. */
    workflows?: Record<string, Workflow> | (() => Record<string, Workflow>);
    /** Default options for generate/stream calls. */
    defaultOptions?: Partial<AgentCallOptions>;
    /** Sub-agents for supervisor/network pattern. Static or dynamic. */
    agents?: Record<string, Agent> | (() => Record<string, Agent>);
    /** Scoring configuration for evaluation. */
    scorers?: Record<string, { scorer: Scorer; sampling?: any }>;
    /** Memory module for conversation persistence. */
    memory?: Memory | (() => Memory);
    /** Format for skill injection. @default 'xml' */
    skillsFormat?: "xml" | "json";
    /** Voice settings for speech input/output. */
    voice?: any;
    /** Workspace for file storage and code execution. */
    workspace?: Workspace | (() => Workspace | undefined);
    /** Input processors — middleware before the LLM. */
    inputProcessors?: any[];
    /** Output processors — middleware after the LLM. */
    outputProcessors?: any[];
    /** Max processor retry count per generation. */
    maxProcessorRetries?: number;
    /** Provider-specific options. */
    providerOptions?: Record<string, Record<string, unknown>>;
    /** Schema for validating request context. */
    requestContextSchema?: import("ai").ZodType;
    /** Max steps for tool-call loops (convenience — flows through defaultOptions). */
    maxSteps?: number;
    /** Extra properties passthrough. */
    [key: string]: any;
  }

  export interface AgentCallOptions {
    /** Per-call instructions override. */
    instructions?: string | string[];
    /** Custom system message. */
    system?: string;
    /** Context messages to prepend. */
    context?: Message[];
    /** Memory options: which thread and resource to use. */
    memory?: { thread?: string | { id: string }; resource?: string };
    /** Unique run ID for this execution. */
    runId?: string;
    /** Save messages incrementally per step. @default false */
    savePerStep?: boolean;
    /** Request context for dynamic resolvers. */
    requestContext?: RequestContext;
    /** Override max steps for this call. */
    maxSteps?: number;
    /** Provider-specific options. */
    providerOptions?: Record<string, Record<string, unknown>>;
    /** Callback when a step finishes. */
    onStepFinish?: (event: any) => void | Promise<void>;
    /** Callback when generation finishes. */
    onFinish?: (event: any) => void | Promise<void>;
    /** Callback for each streaming chunk. */
    onChunk?: (event: any) => void | Promise<void>;
    /** Callback when an error occurs during streaming. */
    onError?: (event: any) => void | Promise<void>;
    /** Limit which tools are active. */
    activeTools?: string[];
    /** Abort signal for cancelling execution. */
    abortSignal?: AbortSignal;
    /** Input processors override. */
    inputProcessors?: any[];
    /** Output processors override. */
    outputProcessors?: any[];
    /** Max processor retries override. */
    maxProcessorRetries?: number;
    /** Additional tool sets. */
    toolsets?: Record<string, Record<string, Tool>>;
    /** Tool selection strategy. */
    toolChoice?: "auto" | "none" | "required" | { type: "tool"; toolName: string };
    /** Model-specific settings (temperature, maxTokens, topP, etc). */
    modelSettings?: Record<string, any>;
    /** Per-call scorers for evaluation. */
    scorers?: Record<string, { scorer: any; sampling?: any }>;
    /** Return data needed for scoring. */
    returnScorerData?: boolean;
    /** Require human approval for all tool calls. HITL. */
    requireToolApproval?: boolean;
    /** Automatically resume suspended tools. */
    autoResumeSuspendedTools?: boolean;
    /** Max concurrent tool calls. @default 1 when approval required, 10 otherwise */
    toolCallConcurrency?: number;
    /** Output schema for structured output. */
    output?: import("ai").ZodType;
    /** Extra options passthrough. */
    [key: string]: any;
  }

  export interface AgentResult {
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

  export interface AgentStepResult {
    text: string;
    reasoning?: string;
    toolCalls: Array<{ toolCallId: string; toolName: string; args: Record<string, unknown> }>;
    toolResults: Array<{ toolCallId: string; toolName: string; args: Record<string, unknown>; result: unknown }>;
    finishReason: string;
    usage: { promptTokens: number; completionTokens: number; totalTokens: number };
    stepType: string;
    isContinued: boolean;
  }

  export interface AgentStreamResult {
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

  export interface Message {
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

  export interface ToolConfig {
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
    /** When true, tool calls suspend for human approval before execution. */
    requireApproval?: boolean;
  }

  export interface ToolExecutionContext {
    requestContext?: RequestContext;
  }

  export interface Tool {
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

  export interface WorkflowConfig {
    id: string;
    inputSchema?: import("ai").ZodType;
    outputSchema?: import("ai").ZodType;
    stateSchema?: import("ai").ZodType;
  }

  export interface StepConfig {
    id: string;
    inputSchema?: import("ai").ZodType;
    outputSchema?: import("ai").ZodType;
    stateSchema?: import("ai").ZodType;
    execute?: (context: StepExecutionContext) => Promise<unknown>;
  }

  export interface StepExecutionContext {
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

  export interface WorkflowBuilder {
    then(step: Step): WorkflowBuilder;
    parallel(steps: Step[]): WorkflowBuilder;
    branch(config: BranchConfig | Step[][]): WorkflowBuilder;
    foreach(config: ForEachConfig): WorkflowBuilder;
    /** Loop until condition is met. */
    dountil(step: Step, condition?: (context: StepExecutionContext) => boolean | Promise<boolean>): WorkflowBuilder;
    /** Pause execution for a duration (ms). */
    sleep(ms: number): WorkflowBuilder;
    commit(): Workflow;
  }

  export interface BranchConfig {
    condition: (data: Record<string, unknown>) => boolean;
    trueStep: Step;
    falseStep?: Step;
  }

  export interface ForEachConfig {
    items: string;
    step: Step;
  }

  export interface Step {}

  export interface Workflow {
    createRun(opts?: { runId?: string }): Promise<WorkflowRun>;
  }

  export interface WorkflowRun {
    runId: string;
    start(params: { inputData: Record<string, unknown> }): Promise<WorkflowRunResult>;
    resume(stepOrParams: string | { resumeData: Record<string, unknown>; step?: string }, resumeData?: Record<string, unknown>): Promise<WorkflowRunResult>;
    cancel(): void;
    readonly status: string;
    readonly currentStep: string;
  }

  export interface WorkflowRunResult {
    status: "completed" | "suspended" | "failed" | "success";
    result?: Record<string, unknown>;
    runId?: string;
    steps?: Record<string, StepRunResult>;
  }

  export interface StepRunResult {
    status: string;
    output?: unknown;
  }

  // ── Memory ────────────────────────────────────────────────────

  /** Memory class — conversation persistence (@mastra/memory). */
  export class Memory {
    constructor(config: MemoryConfig);
    /** Create a new thread. resourceId is required. */
    createThread(opts: { resourceId: string; threadId?: string; title?: string; metadata?: Record<string, unknown> }): Promise<Thread>;
    /** Save/upsert a thread. */
    saveThread(opts: { thread: Thread }): Promise<Thread>;
    /** Get thread by ID. Returns null if not found. */
    getThreadById(opts: { threadId: string }): Promise<Thread | null>;
    /** Update thread title or metadata. */
    updateThread(opts: { threadId: string; title?: string; metadata?: Record<string, unknown> }): Promise<Thread>;
    /** List threads with optional filter. */
    listThreads(filter?: { resourceId?: string; page?: number; perPage?: number }): Promise<Thread[]>;
    /** Save messages to a thread. */
    saveMessages(opts: { threadId: string; messages: Message[] }): Promise<void>;
    /** Delete messages from a thread. */
    deleteMessages(opts: { threadId: string }): Promise<void>;
    /** Recall messages from a thread with optional semantic search. */
    recall(opts: { threadId: string; query?: string; resourceId?: string }): Promise<RecallResult>;
    /** Delete a thread and its messages. */
    deleteThread(threadId: string): Promise<void>;
    /** Set storage adapter after construction. */
    setStorage(storage: StorageInstance): void;
    /** Set vector store after construction. */
    setVector(vector: VectorStoreInstance): void;
  }

  export interface Thread {
    id: string;
    resourceId: string;
    title?: string;
    createdAt: Date;
    updatedAt: Date;
    metadata?: Record<string, unknown>;
  }

  export interface RecallResult {
    messages: Message[];
    workingMemory?: string;
  }

  export interface MemoryConfig {
    /** Storage adapter for threads, messages, working memory. */
    storage?: StorageInstance;
    /** Vector database for semantic recall. false to disable. */
    vector?: VectorStoreInstance | false;
    /** Embedding model for semantic recall. String "provider/model" or EmbeddingModel. */
    embedder?: string | any;
    /** Embedding options. */
    embedderOptions?: { dimensions?: number };
    /** Memory behavior options. */
    options?: MemoryOptions;
  }

  export interface MemoryOptions {
    /** Prevent memory from saving new messages. @default false */
    readOnly?: boolean;
    /** Number of recent messages to include. false to disable. @default 10 */
    lastMessages?: number | false;
    /** Semantic recall via vector embeddings. */
    semanticRecall?: boolean | {
      topK?: number;
      messageRange?: number;
      scope?: "thread" | "resource";
    };
    /** Working memory for persistent user data. */
    workingMemory?: boolean | {
      enabled: boolean;
      scope?: "thread" | "resource";
      template?: string;
      schema?: import("ai").ZodType;
      version?: "vnext";
    };
    /** Auto-generate thread titles from first message. */
    generateTitle?: boolean | {
      model?: any;
      instructions?: string;
    };
    /** Observational memory — 3-tier compression. */
    observationalMemory?: boolean | {
      scope?: "thread" | "resource";
      model?: string;
      observation?: { messageTokens?: number };
      reflection?: { observationTokens?: number };
    };
  }

  // ── Storage backends ──────────────────────────────────────────

  /** Marker type for resolved storage instances. */
  export interface StorageInstance {
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
  export interface VectorStoreInstance {
    /** @internal */ readonly __vectorType: string;
    createIndex(opts: { indexName: string; dimension: number; metric?: string }): Promise<void>;
    listIndexes(): Promise<string[]>;
    describeIndex(indexName: string): Promise<any>;
    deleteIndex(indexName: string): Promise<void>;
    upsert(opts: { indexName: string; vectors: VectorEntry[]; metadata?: Record<string, unknown> }): Promise<string[]>;
    query(opts: { indexName: string; queryVector: number[]; topK?: number; filter?: any }): Promise<VectorQueryResult[]>;
  }

  export interface VectorEntry {
    id?: string;
    vector: number[];
    metadata?: Record<string, unknown>;
    [key: string]: any;
  }

  export interface VectorQueryResult {
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

  export interface WorkspaceSearchResult {
    path: string;
    content: string;
    score: number;
  }

  export interface WorkspaceInfo {
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

  export interface WorkspaceConfig {
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

  export interface ChunkOptions {
    strategy?: "recursive" | "character" | "token" | "markdown" | "html";
    size?: number;
    maxSize?: number;
    overlap?: number;
    separator?: string;
    headers?: [string, string][] | Record<string, string>;
    [key: string]: any;
  }

  export interface DocumentChunk {
    text: string;
    metadata: Record<string, unknown>;
  }

  /** Graph RAG for knowledge graph queries. */
  export class GraphRAG {
    constructor(config: { vectorStore: VectorStoreInstance; embedder: import("ai").EmbeddingModel });
    query(query: string, options?: { topK?: number }): Promise<GraphRAGResult>;
  }

  export interface GraphRAGResult {
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

  export interface ObservabilityConfig {
    configs: Record<string, {
      serviceName: string;
      exporters: DefaultExporter[];
    }>;
  }

  export class DefaultExporter {
    constructor(config: { storage: StorageInstance; strategy?: "realtime" | "batch" });
  }

  // ── Evals ─────────────────────────────────────────────────────

  /**
   * Create a custom scorer — builder pattern.
   *
   * @example
   * ```ts
   * const scorer = createScorer({
   *   id: "quality",
   *   name: "Quality Scorer",
   *   description: "Scores output quality",
   * }).preprocess(({ run }) => {
   *   return { normalized: run.output.text.toLowerCase() };
   * }).generateScore(({ results, run }) => {
   *   return results.preprocessStepResult.normalized.length > 10 ? 1 : 0;
   * }).generateReason(({ results }) => {
   *   return results.generateScoreStepResult >= 1 ? "Good" : "Too short";
   * });
   *
   * const result = await scorer.run({
   *   input: [{ role: "user", content: "question" }],
   *   output: { role: "assistant", text: "answer" },
   * });
   * ```
   */
  export function createScorer(config: ScorerConfig): ScorerBuilder;

  export interface ScorerConfig {
    id: string;
    name?: string;
    description?: string;
    /** Scorer type shortcut: "agent" for agent-style input/output. */
    type?: "agent" | { input: import("ai").ZodType; output: import("ai").ZodType };
    /** LLM judge config (for prompt-based scoring). */
    judge?: { model: any; instructions?: string };
  }

  /** Builder for chaining scorer steps. Each method returns a new builder. */
  export interface ScorerBuilder {
    /** Preprocess step — transform input/output before scoring. */
    preprocess(fn: (ctx: { run: ScorerRunContext; results?: any }) => any | Promise<any>): ScorerBuilder;
    /** Analyze step — extract features from preprocessed data. */
    analyze(fn: (ctx: { results: any; run: ScorerRunContext }) => any | Promise<any>): ScorerBuilder;
    /** Generate a numeric score (0-1). Function or prompt-based. */
    generateScore(fnOrConfig: ((ctx: { results: any; run: ScorerRunContext }) => number | Promise<number>) | {
      description: string;
      judge?: { model: any; instructions: string };
      createPrompt: (ctx: { results: any; run: ScorerRunContext }) => string | Promise<string>;
    }): ScorerBuilder;
    /** Generate a text explanation for the score. */
    generateReason(fnOrConfig: ((ctx: { results: any; run: ScorerRunContext }) => string | Promise<string>) | {
      description: string;
      judge?: { model: any; instructions: string };
      createPrompt: (ctx: { results: any; run: ScorerRunContext }) => string | Promise<string>;
    }): ScorerBuilder;
    /** Run the scorer on input/output. */
    run(input: ScorerRunInput): Promise<ScorerRunResult>;
  }

  export interface ScorerRunContext {
    input: any;
    output: any;
  }

  export interface ScorerRunInput {
    input: Array<{ role: string; content: string }>;
    output: { role: string; text: string };
  }

  export interface ScorerRunResult {
    score: number;
    reason?: string;
    runId: string;
    [key: string]: any;
  }

  /** Scorer instance (result of builder chain). */
  export type Scorer = ScorerBuilder;

  /** Run batch evaluations. */
  export function runEvals(config: {
    scorers: Record<string, { scorer: Scorer; sampling?: any }>;
    dataset: Array<{ input: any; output: any }>;
  }): Promise<EvalRunResult>;

  export interface EvalRunResult {
    results: Array<{
      input: any;
      output: any;
      scores: Record<string, ScorerRunResult>;
    }>;
  }

  // ── Observability extras ────────────────────────────────────────

  export class SensitiveDataFilter {
    constructor(config?: { patterns?: RegExp[]; replacement?: string });
  }
}
