/**
 * Brainlet Runtime Type Definitions
 *
 * These types define the developer-facing API for `.ts` files running
 * on the brainlet platform. Everything is imported from "brainlet".
 *
 * @example
 * ```ts
 * import { agent, createWorkflow, createStep, z, output } from "brainlet";
 * ```
 *
 * @see brainkit-maps/brainkit/DESIGN.md for the full architecture
 * @see brainkit-maps/references/sdk/DESIGN.md for the API surface design
 */
declare module "brainlet" {

  // ═══════════════════════════════════════════════════════════════
  // LOCAL — direct JS in the Kit's runtime, no bus, no RBAC
  // ═══════════════════════════════════════════════════════════════

  // ── Agent ──────────────────────────────────────────────────────

  /**
   * Create a persistent agent in this Kit.
   *
   * @example
   * ```ts
   * const researcher = agent({
   *   model: "openai/gpt-4o-mini",
   *   instructions: "You research topics thoroughly.",
   *   tools: { search: searchTool },
   *   memory: {
   *     thread: "session-1",
   *     resource: "user-1",
   *     storage: new InMemoryStore({ id: "mem" }),
   *   },
   * });
   * const result = await researcher.generate("Find papers on RLHF");
   * ```
   */
  export function agent(config: AgentConfig): Agent;

  /**
   * Define a tool in this Kit.
   * Local tools are only visible within this Kit unless explicitly registered.
   *
   * @example
   * ```ts
   * const calculator = createTool({
   *   name: "add",
   *   description: "Adds two numbers",
   *   schema: z.object({ a: z.number(), b: z.number() }),
   *   execute: async ({ a, b }) => ({ result: a + b }),
   * });
   * ```
   */
  export function createTool(config: ToolConfig): MastraTool;

  /** Zod schema builder — use for tool schemas, workflow schemas, etc. */
  export const z: Zod;

  // ── AI ─────────────────────────────────────────────────────────

  /**
   * Direct LLM calls — LOCAL, same runtime as agents. No bus round-trip.
   *
   * @example
   * ```ts
   * const result = await ai.generate({
   *   model: "openai/gpt-4o-mini",
   *   prompt: "Translate to French: Hello world",
   * });
   * ```
   */
  export const ai: {
    /** Generate text from a prompt. */
    generate(params: AIGenerateParams): Promise<AIGenerateResult>;

    /** Stream text with real-time token delivery. */
    stream(params: AIStreamParams): Promise<StreamResult>;

    /** Generate an embedding vector for a single value. */
    embed(params: AIEmbedParams): Promise<AIEmbedResult>;

    /** Generate embedding vectors for multiple values. */
    embedMany(params: AIEmbedManyParams): Promise<AIEmbedManyResult>;
  };

  // ── Workflows ──────────────────────────────────────────────────

  /**
   * Create a workflow — a multi-step pipeline with typed data flow.
   *
   * @example
   * ```ts
   * const workflow = createWorkflow({
   *   id: "my-pipeline",
   *   inputSchema: z.object({ message: z.string() }),
   *   outputSchema: z.object({ result: z.string() }),
   * })
   *   .then(formatStep)
   *   .then(processStep)
   *   .commit();
   *
   * const run = await workflow.createRun();
   * const result = await run.start({ inputData: { message: "hello" } });
   * ```
   */
  export function createWorkflow(config: WorkflowConfig): WorkflowBuilder;

  /**
   * Create a workflow step — the atomic building block.
   *
   * @example
   * ```ts
   * const formatStep = createStep({
   *   id: "format",
   *   inputSchema: z.object({ message: z.string() }),
   *   outputSchema: z.object({ formatted: z.string() }),
   *   execute: async ({ inputData }) => {
   *     return { formatted: inputData.message.toUpperCase() };
   *   },
   * });
   * ```
   */
  export function createStep(config: StepConfig): Step;

  // ── Memory ─────────────────────────────────────────────────────

  /**
   * Create a Memory instance with a storage provider.
   * Usually you don't need this directly — pass storage config to agent() instead.
   */
  export function createMemory(config: MemoryConfig): MemoryInstance;

  /** The Memory class from @mastra/memory. */
  export const Memory: MemoryConstructor;

  // ── Storage Providers ──────────────────────────────────────────

  /** In-memory storage — fast, ephemeral. Good for testing. */
  export const InMemoryStore: InMemoryStoreConstructor;

  /** LibSQL/Turso storage — HTTP-based, serverless-friendly. */
  export const LibSQLStore: LibSQLStoreConstructor;

  /** Upstash Redis storage — HTTP-based, serverless-friendly. */
  export const UpstashStore: UpstashStoreConstructor;

  /** PostgreSQL storage — TCP-based via jsbridge/net.go. */
  export const PostgresStore: PostgresStoreConstructor;

  /** MongoDB storage — TCP-based via jsbridge/net.go. */
  export const MongoDBStore: MongoDBStoreConstructor;

  // ═══════════════════════════════════════════════════════════════
  // PLATFORM — through bus, Go bridges, interceptors apply
  // ═══════════════════════════════════════════════════════════════

  /**
   * Tool registry — call registered tools from any namespace.
   *
   * @example
   * ```ts
   * const rows = await tools.call("db_query", { sql: "SELECT * FROM users" });
   * ```
   */
  export const tools: {
    /** Call a tool by name. Namespace resolution: caller → user → platform → plugin. */
    call(name: string, input?: any): Promise<any>;
  };

  /**
   * Look up a registered tool by name and return a Mastra-compatible tool object.
   * Use to pass platform/plugin tools to an agent's tool config.
   *
   * @example
   * ```ts
   * const coder = agent({
   *   model: "openai/gpt-4o-mini",
   *   tools: { db: tool("db_query"), search: tool("search") },
   * });
   * ```
   */
  export function tool(name: string): MastraTool;

  /**
   * Platform bus — pub/sub events and request/response.
   *
   * @example
   * ```ts
   * bus.publish("pipeline.complete", { result: output });
   * const response = await bus.request("tools.resolve", { name: "db_query" });
   * ```
   */
  export const bus: {
    /** Send a message to a topic (fire and forget). */
    send(topic: string, payload?: any): Promise<void>;
    /** Alias for send. */
    publish(topic: string, payload?: any): Promise<void>;
    /** Send a request and wait for a response. */
    request(topic: string, payload?: any): Promise<any>;
  };

  /**
   * WASM operations — compile AssemblyScript and execute WASM modules.
   *
   * @example
   * ```ts
   * const mod = await wasm.compile('export function run(): i32 { return 42; }');
   * const result = await wasm.run(mod, {});
   * ```
   */
  export const wasm: {
    /** Compile AssemblyScript source to WASM. */
    compile(source: string, opts?: WASMCompileOpts): Promise<WASMModule>;
    /** Execute a compiled WASM module. */
    run(module: WASMModule, input?: any): Promise<WASMRunResult>;
  };

  /** Sandbox context — identity and namespace of this Kit. */
  export const sandbox: SandboxContext;

  /**
   * Set the module's output value. Passes results back to Go.
   *
   * @example
   * ```ts
   * output({ text: result.text, tokens: result.usage.totalTokens });
   * ```
   */
  export function output(value: any): void;

  // ═══════════════════════════════════════════════════════════════
  // Types
  // ═══════════════════════════════════════════════════════════════

  // ── Agent Types ────────────────────────────────────────────────

  interface AgentConfig {
    name?: string;
    /** Model identifier: "provider/model-id" (e.g., "openai/gpt-4o-mini"). */
    model: string;
    instructions?: string;
    description?: string;
    /** Tools available to the agent. Keys are tool names. */
    tools?: Record<string, MastraTool>;
    /** Memory config — enables conversation persistence. */
    memory?: AgentMemoryConfig;
    maxSteps?: number;
  }

  interface AgentMemoryConfig {
    /** Thread ID for conversation grouping. */
    thread: string | { id: string };
    /** Resource ID for scoping memory. */
    resource?: string;
    /** Storage provider instance. Default: InMemoryStore. */
    storage?: StorageProvider;
    /** Number of recent messages to include in context. */
    lastMessages?: number;
    /** Enable semantic recall (requires vector store). */
    semanticRecall?: boolean | SemanticRecallConfig;
    /** Enable working memory. */
    workingMemory?: boolean | WorkingMemoryConfig;
  }

  interface SemanticRecallConfig {
    topK?: number;
    messageRange?: number;
  }

  interface WorkingMemoryConfig {
    enabled?: boolean;
    scope?: "thread" | "resource";
  }

  interface Agent {
    generate(prompt: string | Message[], options?: GenerateOptions): Promise<GenerateResult>;
    stream(prompt: string | Message[], options?: StreamOptions): Promise<StreamResult>;
  }

  interface GenerateOptions {
    maxSteps?: number;
    memory?: { threadId?: string; resourceId?: string };
  }

  interface StreamOptions {
    maxSteps?: number;
    memory?: { threadId?: string; resourceId?: string };
  }

  interface GenerateResult {
    text: string;
    reasoning: string;
    usage: Usage;
    finishReason: FinishReason;
    toolCalls: ToolCall[];
    toolResults: ToolResult[];
    steps: AgentStepResult[];
  }

  interface StreamResult {
    textStream: AsyncIterable<string>;
    fullStream: AsyncIterable<StreamPart>;
    text: Promise<string>;
    usage: Promise<Usage>;
    finishReason: Promise<FinishReason>;
  }

  // ── Workflow Types ─────────────────────────────────────────────

  interface WorkflowConfig {
    id: string;
    inputSchema: ZodType;
    outputSchema: ZodType;
    stateSchema?: ZodObject;
  }

  interface StepConfig {
    id: string;
    inputSchema: ZodType;
    outputSchema: ZodType;
    stateSchema?: ZodType;
    resumeSchema?: ZodType;
    suspendSchema?: ZodType;
    execute: (context: StepExecuteContext) => Promise<any>;
  }

  interface StepExecuteContext {
    inputData: any;
    state?: any;
    setState?: (newState: any) => Promise<void>;
    suspend?: (payload?: any) => Promise<never>;
    resumeData?: any;
    suspendData?: any;
    abortSignal?: AbortSignal;
    mastra?: any;
    bail?: (payload: any) => any;
  }

  interface Step {
    readonly id: string;
  }

  interface WorkflowBuilder {
    then(step: Step | Workflow): WorkflowBuilder;
    parallel(steps: Step[]): WorkflowBuilder;
    branch(conditions: [((ctx: any) => Promise<boolean> | boolean), Step][]): WorkflowBuilder;
    foreach(step: Step, options?: { concurrency?: number }): WorkflowBuilder;
    map(fn: (ctx: any) => Promise<any>): WorkflowBuilder;
    dountil(step: Step, condition: (ctx: any) => Promise<boolean> | boolean): WorkflowBuilder;
    dowhile(step: Step, condition: (ctx: any) => Promise<boolean> | boolean): WorkflowBuilder;
    sleep(ms: number | ((ctx: any) => number | Promise<number>)): WorkflowBuilder;
    commit(): Workflow;
  }

  interface Workflow {
    readonly id: string;
    createRun(): Promise<WorkflowRun>;
  }

  interface WorkflowRun {
    readonly runId: string;

    /** Execute the workflow synchronously. */
    start(params: WorkflowStartParams): Promise<WorkflowResult>;

    /** Execute in the background, returns runId immediately. */
    startAsync(params: WorkflowStartParams): Promise<{ runId: string }>;

    /** Execute with real-time streaming events. */
    stream(params: WorkflowStartParams): WorkflowStreamResult;

    /** Resume a suspended workflow. */
    resume(params: { resumeData: any; step?: Step | string }): Promise<WorkflowResult>;

    /** Cancel a running workflow. */
    cancel(): Promise<{ message: string }>;
  }

  interface WorkflowStartParams {
    inputData: any;
    initialState?: any;
  }

  interface WorkflowResult {
    status: "success" | "failed" | "suspended" | "tripwire" | "paused" | "canceled" | "waiting";
    result?: any;
    error?: any;
    input?: any;
    steps?: Record<string, any>;
    suspendPayload?: any;
    suspended?: string[];
    traceId?: string;
  }

  interface WorkflowStreamResult {
    fullStream: AsyncIterable<WorkflowStreamEvent>;
    result: Promise<WorkflowResult>;
    status: string;
    usage: Promise<Usage>;
  }

  interface WorkflowStreamEvent {
    type: string;
    payload: any;
  }

  // ── Tool Types ─────────────────────────────────────────────────

  interface ToolConfig {
    name: string;
    description?: string;
    schema?: ZodObject;
    execute: (input: any) => Promise<any> | any;
  }

  interface MastraTool {
    readonly _registryTool?: boolean;
  }

  // ── AI Types ───────────────────────────────────────────────────

  interface AIGenerateParams {
    model: string;
    prompt?: string;
    system?: string;
  }

  interface AIStreamParams extends AIGenerateParams {}

  interface AIGenerateResult {
    text: string;
    reasoning: string;
    usage: Usage;
    finishReason: FinishReason;
    response: ResponseMeta;
  }

  interface AIEmbedParams {
    /** Embedding model: "provider/model-id" (e.g., "openai/text-embedding-3-small"). */
    model: string;
    /** The text to embed. */
    value: string;
  }

  interface AIEmbedResult {
    /** The embedding vector. */
    embedding: number[];
    /** Token usage. */
    usage: Usage;
  }

  interface AIEmbedManyParams {
    model: string;
    /** Array of texts to embed. */
    values: string[];
  }

  interface AIEmbedManyResult {
    /** Array of embedding vectors (same order as values). */
    embeddings: number[][];
    usage: Usage;
  }

  // ── Memory Types ───────────────────────────────────────────────

  interface MemoryConfig {
    /** Storage provider instance. */
    storage?: StorageProvider;
    lastMessages?: number;
    semanticRecall?: boolean | SemanticRecallConfig;
    workingMemory?: boolean | WorkingMemoryConfig;
    generateTitle?: boolean;
  }

  interface MemoryInstance {
    createThread(opts: { threadId: string; resourceId?: string }): Promise<any>;
    getThreadById(opts: { threadId: string }): Promise<any>;
    saveMessages(opts: { threadId: string; messages: Message[] }): Promise<void>;
    getMessages(opts: { threadId: string; limit?: number }): Promise<Message[]>;
  }

  interface MemoryConstructor {
    new (config: { storage?: StorageProvider; options?: any; vector?: boolean }): MemoryInstance;
  }

  // ── Storage Provider Types ─────────────────────────────────────

  /** Base interface for all storage providers. */
  interface StorageProvider {
    init(): Promise<void>;
  }

  interface InMemoryStoreConstructor {
    new (config: { id: string }): StorageProvider;
  }

  interface LibSQLStoreConstructor {
    new (config: { id: string; url: string; authToken?: string }): StorageProvider;
  }

  interface UpstashStoreConstructor {
    new (config: { id: string; url: string; token: string }): StorageProvider;
  }

  interface PostgresStoreConstructor {
    new (config: { id: string; connectionString: string }): StorageProvider;
  }

  interface MongoDBStoreConstructor {
    new (config: { id: string; url: string; dbName: string }): StorageProvider;
  }

  // ── WASM Types ─────────────────────────────────────────────────

  interface WASMCompileOpts {
    optimizeLevel?: number;
    shrinkLevel?: number;
    debug?: boolean;
    runtime?: "stub" | "incremental" | "minimal";
  }

  interface WASMModule {
    moduleId: string;
  }

  interface WASMRunResult {
    exitCode: number;
    value?: number;
  }

  // ── Shared Types ───────────────────────────────────────────────

  interface Message {
    role: "system" | "user" | "assistant" | "tool";
    content: string | any;
  }

  interface Usage {
    promptTokens: number;
    completionTokens: number;
    totalTokens: number;
  }

  type FinishReason =
    | "stop"
    | "length"
    | "content-filter"
    | "tool-calls"
    | "error"
    | "other";

  interface ToolCall {
    toolCallId: string;
    toolName: string;
    args: any;
  }

  interface ToolResult {
    toolCallId: string;
    toolName: string;
    args: any;
    result: any;
  }

  interface AgentStepResult {
    text: string;
    reasoning: string;
    finishReason: FinishReason;
    usage: Usage;
    toolCalls: ToolCall[];
    toolResults: ToolResult[];
  }

  interface StreamPart {
    type: string;
    [key: string]: any;
  }

  interface ResponseMeta {
    id: string;
    modelId: string;
    timestamp: string;
  }

  interface SandboxContext {
    readonly id: string;
    readonly namespace: string;
    readonly callerID: string;
  }

  // ── Zod Types ──────────────────────────────────────────────────

  interface ZodType<T = any> {
    describe(description: string): ZodType<T>;
    optional(): ZodType<T | undefined>;
    default(value: T): ZodType<T>;
    nullable(): ZodType<T | null>;
    transform<U>(fn: (val: T) => U): ZodType<U>;
    refine(check: (val: T) => boolean, message?: string): ZodType<T>;
  }

  interface ZodString extends ZodType<string> {
    min(len: number, message?: string): ZodString;
    max(len: number, message?: string): ZodString;
    email(message?: string): ZodString;
    url(message?: string): ZodString;
    uuid(message?: string): ZodString;
    regex(pattern: RegExp, message?: string): ZodString;
    startsWith(prefix: string): ZodString;
    endsWith(suffix: string): ZodString;
    trim(): ZodString;
    toLowerCase(): ZodString;
    toUpperCase(): ZodString;
    describe(description: string): ZodString;
    optional(): ZodType<string | undefined>;
    default(value: string): ZodString;
    nullable(): ZodType<string | null>;
  }

  interface ZodNumber extends ZodType<number> {
    min(value: number, message?: string): ZodNumber;
    max(value: number, message?: string): ZodNumber;
    int(message?: string): ZodNumber;
    positive(message?: string): ZodNumber;
    negative(message?: string): ZodNumber;
    nonnegative(message?: string): ZodNumber;
    finite(message?: string): ZodNumber;
    describe(description: string): ZodNumber;
    optional(): ZodType<number | undefined>;
    default(value: number): ZodNumber;
    nullable(): ZodType<number | null>;
  }

  interface ZodBoolean extends ZodType<boolean> {
    describe(description: string): ZodBoolean;
    optional(): ZodType<boolean | undefined>;
    default(value: boolean): ZodBoolean;
  }

  interface ZodObject<T extends Record<string, any> = Record<string, any>> extends ZodType<T> {
    passthrough(): ZodObject<T>;
    strip(): ZodObject<T>;
    strict(): ZodObject<T>;
    merge<U extends Record<string, any>>(other: ZodObject<U>): ZodObject<T & U>;
    pick<K extends keyof T>(keys: Record<K, true>): ZodObject<Pick<T, K>>;
    omit<K extends keyof T>(keys: Record<K, true>): ZodObject<Omit<T, K>>;
    partial(): ZodObject<Partial<T>>;
    extend<U extends Record<string, ZodType>>(shape: U): ZodObject;
    describe(description: string): ZodObject<T>;
    optional(): ZodType<T | undefined>;
  }

  interface ZodArray<T = any> extends ZodType<T[]> {
    min(len: number, message?: string): ZodArray<T>;
    max(len: number, message?: string): ZodArray<T>;
    nonempty(message?: string): ZodArray<T>;
    describe(description: string): ZodArray<T>;
    optional(): ZodType<T[] | undefined>;
  }

  interface ZodEnum<T extends string = string> extends ZodType<T> {
    describe(description: string): ZodEnum<T>;
    optional(): ZodType<T | undefined>;
  }

  interface Zod {
    string(): ZodString;
    number(): ZodNumber;
    boolean(): ZodBoolean;
    bigint(): ZodType<bigint>;
    date(): ZodType<Date>;
    object<T extends Record<string, ZodType>>(shape: T): ZodObject;
    array<T extends ZodType>(element: T): ZodArray;
    tuple(items: ZodType[]): ZodType;
    record(keyType: ZodType, valueType: ZodType): ZodType;
    map(keyType: ZodType, valueType: ZodType): ZodType;
    set(element: ZodType): ZodType;
    enum<T extends readonly [string, ...string[]]>(values: T): ZodEnum<T[number]>;
    nativeEnum<T extends Record<string, string | number>>(enumObj: T): ZodType;
    union(types: ZodType[]): ZodType;
    discriminatedUnion(discriminator: string, types: ZodObject[]): ZodType;
    intersection(left: ZodType, right: ZodType): ZodType;
    any(): ZodType<any>;
    unknown(): ZodType<unknown>;
    void(): ZodType<void>;
    never(): ZodType<never>;
    null(): ZodType<null>;
    undefined(): ZodType<undefined>;
    literal<T extends string | number | boolean>(value: T): ZodType<T>;
    optional<T extends ZodType>(type: T): ZodType;
    nullable<T extends ZodType>(type: T): ZodType;
    coerce: {
      string(): ZodString;
      number(): ZodNumber;
      boolean(): ZodBoolean;
      date(): ZodType<Date>;
      bigint(): ZodType<bigint>;
    };
  }
}
