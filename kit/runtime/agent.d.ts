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
    /** System instructions. */
    instructions?: string;
    /** Language model (use model() from "kit" to resolve). */
    model: any;
    /** Tool definitions. */
    tools?: Record<string, any>;
    /** Max tool-call rounds (default: 5). */
    maxSteps?: number;
    /** Tool selection strategy. */
    toolChoice?: "auto" | "none" | "required" | { type: "tool"; toolName: string };
    /** Memory instance for conversation persistence. */
    memory?: MemoryInstance;
    /** Sub-agents for supervisor/network pattern. */
    agents?: Record<string, Agent>;
    /** Default call options. */
    defaultOptions?: Partial<AgentCallOptions>;
    /** Dynamic model resolver. */
    modelResolver?: (ctx: any) => any | Promise<any>;
    /** Dynamic tools resolver. */
    toolsResolver?: (ctx: any) => Record<string, any> | Promise<Record<string, any>>;
    /** Dynamic instructions resolver. */
    instructionsResolver?: (ctx: any) => string | Promise<string>;
    /** Input processors. */
    inputProcessors?: any[];
    /** Output processors. */
    outputProcessors?: any[];
    /** Scorer definitions for evals. */
    scorers?: any[];
    /** Workspace instance. */
    workspace?: WorkspaceInstance;
    /** Voice configuration. */
    voice?: any;
    /** Workflow definitions. */
    workflows?: Record<string, any>;
    /** Provider-specific options. */
    providerOptions?: Record<string, Record<string, any>>;
  }

  interface AgentCallOptions {
    /** Override max steps for this call. */
    maxSteps?: number;
    /** Memory options. */
    memory?: { thread?: string | { id: string }; resource?: string };
    /** Request context for dynamic resolvers. */
    requestContext?: RequestContextInstance;
  }

  interface AgentResult {
    text: string;
    reasoningText?: string;
    object?: any;
    toolCalls: Array<{ toolCallId: string; toolName: string; args: any }>;
    toolResults: Array<{ toolCallId: string; toolName: string; args: any; result: any }>;
    finishReason: string;
    usage: { promptTokens: number; completionTokens: number; totalTokens: number };
    steps: Array<{
      text: string;
      reasoning?: string;
      toolCalls: any[];
      toolResults: any[];
      finishReason: string;
      usage: any;
      stepType: string;
      isContinued: boolean;
    }>;
    response: { id: string; modelId: string; timestamp: string };
    runId?: string;
    suspendPayload?: any;
    providerMetadata?: any;
  }

  interface AgentStreamResult {
    /** Async iterable of text chunks. */
    textStream: AsyncIterable<string>;
    /** Async iterable of typed stream parts. */
    fullStream: AsyncIterable<any>;
    /** Promise: final complete text. */
    text: Promise<string>;
    /** Promise: token usage. */
    usage: Promise<any>;
    /** Promise: finish reason. */
    finishReason: Promise<string>;
    /** Promise: all tool calls. */
    toolCalls: Promise<any[]>;
    /** Promise: all tool results. */
    toolResults: Promise<any[]>;
    /** Promise: all steps. */
    steps: Promise<any[]>;
  }

  interface Message {
    role: "system" | "user" | "assistant" | "tool";
    content: any;
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
    inputSchema?: any;
    /** Output schema (Zod). */
    outputSchema?: any;
    /** Execute function. */
    execute?: (input: any, context?: any) => Promise<any>;
    /** Suspend schema (for tool approval workflows). */
    suspendSchema?: any;
    /** Resume schema. */
    resumeSchema?: any;
  }

  interface Tool {
    id: string;
    description?: string;
    execute?: (input: any, context?: any) => Promise<any>;
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
    inputSchema?: any;
    outputSchema?: any;
  }

  interface StepConfig {
    id: string;
    inputSchema?: any;
    outputSchema?: any;
    execute?: (context: { inputData: any; mapiData?: any }) => Promise<any>;
  }

  interface WorkflowBuilder {
    then(step: Step): WorkflowBuilder;
    parallel(steps: Step[]): WorkflowBuilder;
    branch(config: any): WorkflowBuilder;
    forEach(config: any): WorkflowBuilder;
    commit(): Workflow;
  }

  interface Step {}

  interface Workflow {
    createRun(opts?: any): Promise<WorkflowRun>;
  }

  interface WorkflowRun {
    runId: string;
    start(params: { inputData: any }): Promise<WorkflowRunResult>;
    resume(params: { resumeData: any; step?: string }): Promise<WorkflowRunResult>;
    cancel(): void;
    readonly status: string;
    readonly currentStep: string;
  }

  interface WorkflowRunResult {
    status: "completed" | "suspended" | "failed";
    result?: any;
    runId?: string;
    steps?: Record<string, any>;
  }

  // ── Memory ────────────────────────────────────────────────────

  /** Memory class — conversation persistence. */
  export class Memory {
    constructor(config: MemoryConfig);
    createThread(opts?: any): Promise<{ id: string }>;
    getThreadById(opts: { threadId: string }): Promise<any>;
    listThreads(filter?: any): Promise<any>;
    saveMessages(opts: { threadId: string; messages: any[] }): Promise<void>;
    recall(opts: { threadId: string; query?: string; resourceId?: string }): Promise<any>;
    deleteThread(threadId: string): Promise<void>;
  }

  type MemoryInstance = Memory;

  interface MemoryConfig {
    storage?: any;
    vector?: any | false;
    embedder?: any;
    options?: {
      lastMessages?: number;
      semanticRecall?: any;
      workingMemory?: any;
      generateTitle?: any;
      observationalMemory?: any;
    };
  }

  // ── Storage backends ──────────────────────────────────────────

  export class InMemoryStore {
    constructor(config?: { id?: string });
  }

  export class LibSQLStore {
    constructor(config: { id?: string; url?: string; authToken?: string; storage?: string });
  }

  export class UpstashStore {
    constructor(config: { id?: string; url: string; token: string });
  }

  export class PostgresStore {
    constructor(config: { id?: string; connectionString: string });
  }

  export class MongoDBStore {
    constructor(config: { id?: string; uri: string; dbName?: string });
  }

  // ── Vector stores ─────────────────────────────────────────────

  export class LibSQLVector {
    constructor(config: { id?: string; connectionUrl?: string; url?: string; authToken?: string; storage?: string });
    createIndex(opts: { indexName: string; dimension: number; metric?: string }): Promise<void>;
    listIndexes(): Promise<string[]>;
    deleteIndex(indexName: string): Promise<void>;
    upsert(opts: { indexName: string; vectors: any[] }): Promise<any>;
    query(opts: { indexName: string; queryVector: number[]; topK?: number }): Promise<any[]>;
  }

  export class PgVector {
    constructor(config: { id?: string; connectionString: string });
    createIndex(opts: { indexName: string; dimension: number; metric?: string }): Promise<void>;
    listIndexes(): Promise<string[]>;
    deleteIndex(indexName: string): Promise<void>;
    upsert(opts: { indexName: string; vectors: any[] }): Promise<any>;
    query(opts: { indexName: string; queryVector: number[]; topK?: number }): Promise<any[]>;
  }

  export class MongoDBVector {
    constructor(config: { id?: string; uri: string; dbName?: string });
    createIndex(opts: { indexName: string; dimension: number; metric?: string }): Promise<void>;
    listIndexes(): Promise<string[]>;
    deleteIndex(indexName: string): Promise<void>;
    upsert(opts: { indexName: string; vectors: any[] }): Promise<any>;
    query(opts: { indexName: string; queryVector: number[]; topK?: number }): Promise<any[]>;
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

  type RequestContextInstance = RequestContext;

  // ── Workspace ─────────────────────────────────────────────────

  export class Workspace {
    constructor(config: WorkspaceConfig);
    init(): Promise<void>;
    destroy(): Promise<void>;
    search(query: string, options?: any): Promise<any[]>;
    index(filePath: string, content: string): Promise<void>;
    getInfo(): any;
    getInstructions(): string;
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
    vectorStore?: LibSQLVector | PgVector | MongoDBVector;
    embedder?: (text: string) => Promise<number[]>;
    searchIndexName?: string;
    tools?: any;
    skills?: string[];
    lsp?: boolean | any;
  }

  type WorkspaceInstance = Workspace;

  // ── RAG ───────────────────────────────────────────────────────

  /** Document class for RAG chunking. */
  export class MDocument {
    static fromText(text: string, metadata?: any): MDocument;
    static fromMarkdown(markdown: string, metadata?: any): MDocument;
    chunk(options?: any): Promise<any[]>;
  }

  /** Graph RAG for knowledge graph queries. */
  export class GraphRAG {
    constructor(config: any);
    query(query: string, options?: any): Promise<any>;
  }

  export function createVectorQueryTool(config: any): any;
  export function createDocumentChunkerTool(config: any): any;
  export function createGraphRAGTool(config: any): any;
  export function rerank(config: any): Promise<any>;
  export function rerankWithScorer(config: any): Promise<any>;

  // ── Observability ─────────────────────────────────────────────

  export class Observability {
    constructor(config: any);
  }

  export class DefaultExporter {
    constructor(config: any);
  }

  // ── Evals ─────────────────────────────────────────────────────

  /** Create a custom scorer for agent evaluation. */
  export function createScorer(config: {
    id?: string;
    name?: string;
    description?: string;
    judge?: { model: any };
    execute?: (input: any) => Promise<{ score: number; details?: any }>;
  }): Scorer;

  /** Run evaluations against an agent. */
  export function runEvals(config: {
    target: Agent;
    scorers: Scorer[];
    dataset: Array<{ input: string; output?: string; groundTruth?: string }>;
  }): Promise<any>;

  interface Scorer {
    run(input: any): Promise<{ score: number; details?: any }>;
  }
}
