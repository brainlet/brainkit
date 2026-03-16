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
  export function createTool(config: ToolConfig): Tool;

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

    /**
     * Generate a structured object from a prompt using a schema.
     *
     * @example
     * ```ts
     * const result = await ai.generateObject({
     *   model: "openai/gpt-4o-mini",
     *   schema: z.object({ name: z.string(), age: z.number() }),
     *   prompt: "Generate a fictional person",
     * });
     * console.log(result.object); // { name: "Alice", age: 30 }
     * ```
     */
    generateObject(params: AIGenerateObjectParams): Promise<AIGenerateObjectResult>;

    /**
     * Stream a structured object with partial updates.
     *
     * @example
     * ```ts
     * const result = ai.streamObject({
     *   model: "openai/gpt-4o-mini",
     *   schema: z.object({ items: z.array(z.string()) }),
     *   prompt: "List 5 programming languages",
     * });
     * for await (const partial of result.partialObjectStream) {
     *   console.log(partial); // { items: ["Python"] }, { items: ["Python", "Go"] }, ...
     * }
     * ```
     */
    streamObject(params: AIStreamObjectParams): AIStreamObjectResult;
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

  /** Memory class — create instances for agent conversation persistence. */
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

    /**
     * Register a tool on the platform. Visible to other Kits sharing the same ToolRegistry.
     * The tool is registered under the caller's namespace.
     */
    register(name: string, config: { description?: string; inputSchema?: any }): Promise<void>;

    /** List all tools visible to this Kit. */
    list(namespace?: string): Promise<ToolInfo[]>;
  };

  interface ToolInfo {
    name: string;
    shortName: string;
    namespace: string;
    description: string;
  }

  /**
   * Look up a registered tool by name and return a tool object for agent use.
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
  export function tool(name: string): Tool;

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
    /**
     * Subscribe to messages matching a topic pattern.
     * Returns a subscription ID for unsubscribing.
     *
     * @example
     * ```ts
     * const subId = bus.subscribe("data.*", (msg) => {
     *   console.log(msg.topic, msg.payload);
     * });
     * // later:
     * bus.unsubscribe(subId);
     * ```
     */
    subscribe(topic: string, handler: (msg: BusMessage) => void): string;
    /** Remove a subscription. */
    unsubscribe(subscriptionId: string): void;
  };

  interface BusMessage {
    topic: string;
    callerID: string;
    payload: any;
    traceID?: string;
  }

  /**
   * WASM operations — compile AssemblyScript and execute WASM modules.
   *
   * @example
   * ```ts
   * const mod = await wasm.compile('export function run(): i32 { return 42; }');
   * const result = await wasm.run(mod, {});
   * ```
   */
  /**
   * Direct AI SDK generateText — for advanced use.
   * Most developers should use ai.generate() instead.
   * Accepts an AI SDK model object (from resolving "provider/model-id").
   */
  export function generateText(params: any): Promise<any>;

  /**
   * Direct AI SDK streamText — for advanced use.
   * Most developers should use ai.stream() instead.
   */
  export function streamText(params: any): any;

  /**
   * Direct AI SDK generateObject — for advanced use.
   * Most developers should use ai.generateObject() instead.
   */
  export function generateObject(params: any): Promise<any>;

  /**
   * Direct AI SDK streamObject — for advanced use.
   * Most developers should use ai.streamObject() instead.
   */
  export function streamObject(params: any): any;

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

  // ── RequestContext ──────────────────────────────────────────

  /**
   * RequestContext — key-value context for dynamic config resolvers.
   * Pass to generate()/stream() to provide runtime state to resolver functions.
   *
   * @example
   * ```ts
   * const ctx = new RequestContext([["model", "openai/gpt-4o"], ["persona", "coding"]]);
   * const result = await myAgent.generate("hello", { requestContext: ctx });
   * ```
   */
  export const RequestContext: RequestContextConstructor;

  interface RequestContextConstructor {
    new (entries?: Iterable<[string, unknown]> | Record<string, unknown>): RequestContextInstance;
  }

  interface RequestContextInstance {
    get(key: string): unknown;
    set(key: string, value: unknown): void;
    has(key: string): boolean;
    delete(key: string): boolean;
    clear(): void;
    readonly all: Record<string, unknown>;
  }

  // ── RAG ─────────────────────────────────────────────────────

  /**
   * MDocument — document ingestion and chunking.
   *
   * @example
   * ```ts
   * const doc = MDocument.fromText("long text...");
   * const chunks = await doc.chunk({ strategy: "recursive", maxSize: 500, overlap: 50 });
   * ```
   */
  export const MDocument: MDocumentConstructor;

  interface MDocumentConstructor {
    fromText(text: string, metadata?: Record<string, any>): MDocumentInstance;
    fromHTML(html: string, metadata?: Record<string, any>): MDocumentInstance;
    fromMarkdown(markdown: string, metadata?: Record<string, any>): MDocumentInstance;
    fromJSON(json: string, metadata?: Record<string, any>): MDocumentInstance;
  }

  interface MDocumentInstance {
    chunk(params?: ChunkParams): Promise<Chunk[]>;
    getDocs(): Chunk[];
    getText(): string[];
    getMetadata(): Record<string, any>[];
  }

  interface Chunk {
    text: string;
    metadata: Record<string, any>;
  }

  type ChunkStrategy =
    | "recursive" | "character" | "token" | "markdown"
    | "html" | "json" | "latex" | "sentence" | "semantic-markdown";

  interface ChunkParams {
    strategy?: ChunkStrategy;
    maxSize?: number;
    overlap?: number;
    separator?: string;
    separators?: string[];
    headers?: [string, string][];
    sections?: [string, string][];
    language?: string;
    encodingName?: string;
    modelName?: string;
    minSize?: number;
    joinThreshold?: number;
    sentenceEnders?: string[];
    returnEachLine?: boolean;
    stripHeaders?: boolean;
    ensureAscii?: boolean;
    convertLists?: boolean;
  }

  /** Create a vector query tool for agent use. */
  export function createVectorQueryTool(options: VectorQueryToolOptions): Tool;
  /** Create a document chunker tool for agent use. */
  export function createDocumentChunkerTool(options: { doc: MDocumentInstance; params?: ChunkParams }): Tool;
  /** Create a graph RAG tool for agent use. */
  export function createGraphRAGTool(options: GraphRAGToolOptions): Tool;

  interface VectorQueryToolOptions {
    vectorStore: any;
    indexName: string;
    model: string;
    enableFilter?: boolean;
    includeVectors?: boolean;
    includeSources?: boolean;
    reranker?: any;
  }

  interface GraphRAGToolOptions extends VectorQueryToolOptions {
    graphOptions?: {
      dimension?: number;
      randomWalkSteps?: number;
      restartProb?: number;
      threshold?: number;
    };
  }

  /** GraphRAG — knowledge graph from vector embeddings. */
  export const GraphRAG: GraphRAGConstructor;

  interface GraphRAGConstructor {
    new (dimension?: number, threshold?: number): GraphRAGInstance;
  }

  interface GraphRAGInstance {
    createGraph(chunks: { text: string; metadata: Record<string, any> }[], embeddings: { vector: number[] }[]): void;
    query(params: { query: number[]; topK?: number; randomWalkSteps?: number; restartProb?: number }): any[];
  }

  /** Rerank results using LLM scoring. */
  export function rerank(results: any[], query: string, model: any, options?: RerankOptions): Promise<any[]>;
  /** Rerank results using a custom relevance scorer. */
  export function rerankWithScorer(params: { results: any[]; query: string; scorer: any; options?: RerankOptions }): Promise<any[]>;

  interface RerankOptions {
    weights?: { semantic?: number; vector?: number; position?: number };
    queryEmbedding?: number[];
    topK?: number;
  }

  /**
   * Create a workflow run with suspend/resume support.
   * Injects storage for snapshot persistence and tracks the run for later resume.
   * Use this instead of workflow.createRun() when you need suspend/resume.
   *
   * @example
   * ```ts
   * const workflow = createWorkflow({...}).then(step1).then(step2).commit();
   * const run = await createWorkflowRun(workflow);
   * const result = await run.start({ inputData: { ... } });
   * ```
   */
  export function createWorkflowRun(workflow: Workflow, opts?: { runId?: string; resourceId?: string }): Promise<WorkflowRun>;

  /**
   * Resume a suspended workflow run.
   * Use after a workflow step called `suspend()` and returned status "suspended".
   *
   * @example
   * ```ts
   * const run = await createWorkflowRun(workflow);
   * const result = await run.start({ inputData: { ... } });
   * if (result.status === "suspended") {
   *   const final = await resumeWorkflow(run.runId, "approval-step", {
   *     approved: true,
   *   });
   * }
   * ```
   */
  export function resumeWorkflow(
    runId: string,
    stepId: string | undefined,
    resumeData: any
  ): Promise<WorkflowResult>;

  // ── Evals / Scorers ──────────────────────────────────────────

  /**
   * Create a custom scorer using a pipeline pattern.
   * Scorer context: `{ run: { input, output, groundTruth }, results: {} }`
   * generateReason context: `{ run, results, score }`
   *
   * @example
   * ```ts
   * const scorer = createScorer({ id: "my-scorer", description: "..." })
   *   .generateScore(({ run }) => run.output.includes("hello") ? 1.0 : 0.0)
   *   .generateReason(({ score }) => score === 1 ? "Found" : "Missing");
   * const result = await scorer.run({ input: "...", output: "..." });
   * ```
   */
  export function createScorer(config: { id: string; description: string }): ScorerBuilder;

  interface ScorerBuilder {
    /** Add the score generation step — must return a number (0.0-1.0 by convention). */
    generateScore(fn: (context: { run: ScorerRun; results: Record<string, any> }) => number | Promise<number>): ScorerBuilder;
    /** Add a reason generation step — explain the score. */
    generateReason(fn: (context: { run: ScorerRun; results: Record<string, any>; score: number }) => string | Promise<string>): ScorerBuilder;
    /** Add a preprocessing step. */
    preprocess(fn: (context: { run: ScorerRun; results: Record<string, any> }) => any): ScorerBuilder;
    /** Add an analysis step. */
    analyze(fn: (context: { run: ScorerRun; results: Record<string, any> }) => any): ScorerBuilder;
    /** Execute the scorer pipeline. */
    run(input: ScorerRun): Promise<ScorerResult>;
  }

  interface ScorerRun {
    input?: any;
    output: any;
    groundTruth?: any;
    runId?: string;
  }

  interface ScorerResult {
    runId: string;
    score: number;
    reason?: string;
    input?: any;
    output: any;
  }

  // ═══════════════════════════════════════════════════════════════
  // Types
  // ═══════════════════════════════════════════════════════════════

  // ── Agent Types ────────────────────────────────────────────────

  /** Resolver context passed to dynamic config functions. */
  interface ResolverContext {
    requestContext: RequestContextInstance;
  }

  /** A value or a function that computes it at generate/stream time. */
  type DynamicArg<T> = T | ((ctx: ResolverContext) => T | Promise<T>);

  interface AgentConfig {
    name?: string;
    /** Model: static string or dynamic resolver. */
    model: DynamicArg<string>;
    /** Instructions: static string or dynamic resolver. */
    instructions?: DynamicArg<string>;
    description?: string;
    /** Tools: static map or dynamic resolver. */
    tools?: DynamicArg<Record<string, Tool>>;
    /** Memory config — enables conversation persistence. */
    memory?: AgentMemoryConfig;
    maxSteps?: number;
    /** Input processors — middleware that transforms messages before the LLM. */
    inputProcessors?: InputProcessor[];
    /** Output processors — middleware that transforms or blocks LLM output. */
    outputProcessors?: OutputProcessor[];
    /** Max retries when a processor requests retry via tripwire. Default: 3. */
    maxProcessorRetries?: number;
    /** Scorers — auto-evaluate responses after each generate()/stream() call (fire-and-forget). */
    scorers?: Record<string, { scorer: ScorerBuilder; sampling?: { type: "none" } | { type: "ratio"; rate: number } }>;
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
    /** Observational memory — 3-tier compression for infinite context. */
    observationalMemory?: boolean | ObservationalMemoryConfig;
  }

  interface SemanticRecallConfig {
    topK?: number;
    messageRange?: number;
  }

  interface WorkingMemoryConfig {
    enabled?: boolean;
    scope?: "thread" | "resource";
  }

  /** Observational memory config — 3-tier compression (messages → observations → reflections). */
  interface ObservationalMemoryConfig {
    /** Model for both observer and reflector. Default: "google/gemini-2.5-flash". */
    model?: string;
    /** Observation config (messages → observations). */
    observation?: {
      /** Model override for observer agent. */
      model?: string;
      /** Token threshold before observer triggers. Default: 30000. */
      messageTokens?: number;
      /** Model settings for observer. */
      modelSettings?: { temperature?: number; maxOutputTokens?: number };
      /** Max tokens per observation batch. Default: 10000. */
      maxTokensPerBatch?: number;
      /**
       * Async buffer interval as fraction of messageTokens. Default: 0.2 (20%).
       * Set to `false` to disable async buffering (synchronous observation only).
       */
      bufferTokens?: number | false;
      /** Fraction of messages to keep after activation. Default: 0.8 (keep 20%). */
      bufferActivation?: number;
      /** Emergency sync threshold multiplier. Default: 1.2x messageTokens. */
      blockAfter?: number;
      /** Custom instruction appended to observer system prompt. */
      instruction?: string;
    };
    /** Reflection config (observations → reflections). */
    reflection?: {
      /** Model override for reflector agent. */
      model?: string;
      /** Token threshold before reflector triggers. Default: 40000. */
      observationTokens?: number;
      /** Model settings for reflector. */
      modelSettings?: { temperature?: number; maxOutputTokens?: number };
      /** Buffer activation fraction. Default: 0.5. */
      bufferActivation?: number;
      /** Emergency sync threshold multiplier. Default: 1.2x observationTokens. */
      blockAfter?: number;
      /** Custom instruction appended to reflector system prompt. */
      instruction?: string;
    };
    /** Scope: "thread" (per-conversation) or "resource" (per-user, experimental). Default: "thread". */
    scope?: "thread" | "resource";
    /** Allow flexible token allocation between observations and messages. Default: false. */
    shareTokenBudget?: boolean;
  }

  interface Agent {
    generate(prompt: string | Message[], options?: GenerateOptions): Promise<GenerateResult>;
    stream(prompt: string | Message[], options?: StreamOptions): Promise<StreamResult>;
  }

  interface GenerateOptions {
    maxSteps?: number;
    memory?: { threadId?: string; resourceId?: string };
    /** RequestContext for dynamic config resolvers. */
    requestContext?: RequestContextInstance;
    /** Model settings — temperature, maxTokens, etc. */
    modelSettings?: {
      temperature?: number;
      maxTokens?: number;
      topP?: number;
      topK?: number;
      presencePenalty?: number;
      frequencyPenalty?: number;
      stopSequences?: string[];
      seed?: number;
    };
    /** Filter which tools are active for this call. */
    activeTools?: string[];
    /** Tool selection strategy. */
    toolChoice?: "auto" | "none" | "required" | { type: "tool"; toolName: string };
    /** Require human approval before tool calls. */
    requireToolApproval?: boolean;
    /** Concurrent tool call limit. */
    toolCallConcurrency?: number;
    /** Abort signal for cancellation. */
    abortSignal?: AbortSignal;
    /** Structured output schema. */
    structuredOutput?: { schema: ZodType; schemaName?: string; schemaDescription?: string };
    /** Callbacks. */
    onStepFinish?: (step: any) => void;
    onFinish?: (result: any) => void;
    onChunk?: (chunk: any) => void;
    onError?: (error: any) => void;
    /** Per-call input/output processors. */
    inputProcessors?: InputProcessor[];
    outputProcessors?: OutputProcessor[];
    /** Per-call scorers. */
    scorers?: Record<string, any>;
  }

  interface StreamOptions extends GenerateOptions {}

  interface GenerateResult {
    text: string;
    reasoning: string;
    usage: Usage;
    totalUsage: Usage;
    finishReason: FinishReason;
    toolCalls: ToolCall[];
    toolResults: ToolResult[];
    steps: AgentStepResult[];
    sources: any[];
    files: any[];
    warnings: any[];
    response: ResponseMeta;
    traceId?: string;
  }

  interface StreamResult {
    textStream: AsyncIterable<string>;
    fullStream: AsyncIterable<StreamPart>;
    text: Promise<string>;
    usage: Promise<Usage>;
    finishReason: Promise<FinishReason>;
    reasoning: Promise<string | undefined>;
    toolCalls: Promise<ToolCall[]>;
    toolResults: Promise<ToolResult[]>;
    steps: Promise<AgentStepResult[]>;
    sources: Promise<any[]>;
    response: Promise<ResponseMeta>;
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
    runtime?: any;
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
    /** Tool identifier. */
    id: string;
    description?: string;
    /** Zod schema for the tool's input parameters. */
    inputSchema?: ZodObject;
    execute: (input: any) => Promise<any> | any;
  }

  interface Tool {
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

  interface AIGenerateObjectParams {
    model: string;
    prompt?: string;
    system?: string;
    messages?: Message[];
    /** Zod schema or JSON schema defining the output structure. */
    schema: ZodType | any;
    schemaName?: string;
    schemaDescription?: string;
    /** Generation mode: "auto" (default), "json", or "tool". */
    mode?: "auto" | "json" | "tool";
    /** Output type: "object" (default), "array", "enum", or "no-schema". */
    output?: "object" | "array" | "enum" | "no-schema";
    /** For output: "enum" — the list of allowed values. */
    enum?: string[];
  }

  interface AIGenerateObjectResult {
    /** The generated object matching the schema. */
    object: any;
    usage: Usage;
    finishReason: FinishReason;
    warnings: any[];
    response: ResponseMeta;
  }

  interface AIStreamObjectParams extends AIGenerateObjectParams {}

  interface AIStreamObjectResult {
    /** Async iterable of partial objects as they're generated. */
    partialObjectStream: AsyncIterable<any>;
    /** Promise resolving to the final complete object. */
    object: Promise<any>;
    usage: Promise<Usage>;
    finishReason: Promise<FinishReason>;
    warnings: Promise<any[]>;
    response: Promise<ResponseMeta>;
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
    /** Observational memory — 3-tier compression for infinite context. */
    observationalMemory?: boolean | ObservationalMemoryConfig;
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

  // ── Processor Types ──────────────────────────────────────────

  /** Input processor — transforms messages before the LLM sees them. */
  interface InputProcessor {
    readonly id: string;
    /** Transform input messages before the first LLM call. */
    processInput?(args: { messages: any[]; abort: AbortFn; [key: string]: any }): any;
    /** Transform messages before each step in the agentic loop. */
    processInputStep?(args: { messages: any[]; abort: AbortFn; [key: string]: any }): any;
  }

  /** Output processor — transforms or blocks LLM output. */
  interface OutputProcessor {
    readonly id: string;
    /** Transform individual stream chunks. */
    processOutputStream?(args: { chunk: any; abort: AbortFn; [key: string]: any }): any;
    /** Transform or block output after each LLM step. */
    processOutputStep?(args: { messages: any[]; abort: AbortFn; [key: string]: any }): any;
    /** Transform final output messages. */
    processOutputResult?(args: { messages: any[]; abort: AbortFn; [key: string]: any }): any;
  }

  /** Abort function — throws a TripWire to halt processing. */
  type AbortFn = (reason?: string, options?: { retry?: boolean; metadata?: any }) => never;

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
