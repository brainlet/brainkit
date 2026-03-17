/**
 * Kit Runtime Type Definitions
 *
 * These types define the developer-facing API for `.ts` files running
 * on the brainlet platform. Everything is imported from "kit".
 *
 * @example
 * ```ts
 * import { agent, createWorkflow, createStep, z, output } from "kit";
 * ```
 *
 * @see brainkit-maps/brainkit/DESIGN.md for the full architecture
 * @see brainkit-maps/references/sdk/DESIGN.md for the API surface design
 */
declare module "kit" {

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

  /**
   * Define a constrained subagent type for the supervisor pattern.
   * Used with `subagents` config on agent() — creates a meta-tool that spawns
   * fresh agents per invocation with filtered tool sets.
   *
   * @example
   * ```ts
   * const explorer = createSubagent({
   *   id: "explore",
   *   instructions: "You explore codebases. Read files, search, but never write.",
   *   allowedTools: ["view", "search", "find_files"],
   *   model: "openai/gpt-4o-mini",
   * });
   *
   * const coder = createSubagent({
   *   id: "execute",
   *   instructions: "You write code. Read, edit, and run commands.",
   *   allowedTools: ["view", "search", "find_files", "edit", "write", "run"],
   *   model: "openai/gpt-4o",
   * });
   *
   * const lead = agent({
   *   model: "openai/gpt-4o",
   *   instructions: "Delegate tasks to your team.",
   *   tools: { view: viewTool, search: searchTool, edit: editTool, write: writeTool, run: runTool, find_files: findTool },
   *   subagents: [explorer, coder],
   * });
   * ```
   */
  export function createSubagent(def: SubagentDefinition): SubagentDef;

  interface SubagentDefinition {
    /** Unique ID — used as the agentType in the meta-tool. */
    id: string;
    /** Display name. Default: same as id. */
    name?: string;
    /** System instructions for this subagent type. */
    instructions: string;
    /** Tool IDs this subagent can use (filtered from parent's tools). */
    allowedTools: string[];
    /** Model for this subagent type. Default: parent's model. */
    model?: string;
    /** Max tool-call rounds. Default: 50. */
    maxSteps?: number;
  }

  interface SubagentDef {
    readonly id: string;
    readonly name: string;
    readonly instructions: string;
    readonly allowedTools: string[];
    readonly model?: string;
    readonly maxSteps: number;
  }

  type SubagentEvent =
    | { type: "start"; agentType: string; task: string }
    | { type: "text_delta"; agentType: string; text: string }
    | { type: "tool_start"; agentType: string; toolName: string; args?: any }
    | { type: "tool_end"; agentType: string; toolName: string; isError: boolean }
    | { type: "end"; agentType: string; durationMs: number; isError: boolean };

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

  // ── Vector Stores ───────────────────────────────────────────────

  /** LibSQL vector store — for semantic recall and RAG. */
  export const LibSQLVector: LibSQLVectorConstructor;

  /** PostgreSQL pgvector store — for semantic recall and RAG. */
  export const PgVector: VectorStoreConstructor;

  /** MongoDB Atlas vector store — for semantic recall and RAG. */
  export const MongoDBVector: VectorStoreConstructor;

  interface LibSQLVectorConstructor {
    /** Brainkit mode — auto-connects to Kit's embedded storage. */
    new (config: { id?: string; storage?: string }): VectorStore;
    /** Mastra mode — explicit URL (passthrough). */
    new (config: { id?: string; url: string; authToken?: string }): VectorStore;
  }

  interface VectorStoreConstructor {
    new (config: { id?: string; url?: string; connectionString?: string; [key: string]: any }): VectorStore;
  }

  interface VectorStore {
    createIndex(opts: { indexName: string; dimension: number; metric?: "cosine" | "euclidean" | "dotProduct" }): Promise<void>;
    listIndexes(): Promise<string[]>;
    describeIndex(indexName: string): Promise<{ dimension: number; metric: string; count: number }>;
    deleteIndex(indexName: string): Promise<void>;
    upsert(opts: { indexName: string; vectors: number[][]; metadata?: Record<string, any>[]; ids?: string[] }): Promise<string[]>;
    query(opts: { indexName: string; queryVector: number[]; topK?: number; filter?: any; includeVector?: boolean }): Promise<{ id: string; score: number; metadata?: any; vector?: number[] }[]>;
    deleteVectors(opts: { indexName: string; ids?: string[]; filter?: any }): Promise<void>;
  }

  // ── Workspace ───────────────────────────────────────────────

  /**
   * Workspace — auto-injects filesystem, sandbox, skills, and search tools into agents.
   *
   * @example
   * ```ts
   * const ws = new Workspace({
   *   id: "my-workspace",
   *   filesystem: new LocalFilesystem({ basePath: "./project" }),
   *   sandbox: new LocalSandbox({ workingDirectory: "./project" }),
   *   bm25: true,
   *   vectorStore: new LibSQLVector({ id: "ws-vectors" }),
   *   embedder: async (text) => { const r = await ai.embed({ model: "openai/text-embedding-3-small", value: text }); return r.embedding; },
   * });
   * const a = agent({ model: "openai/gpt-4o-mini", workspace: ws });
   * ```
   */
  export const Workspace: WorkspaceConstructor;

  /** Local filesystem provider backed by Go bridges (jsbridge/fs.go). */
  export const LocalFilesystem: LocalFilesystemConstructor;

  /** Local sandbox provider backed by Go bridges (jsbridge/exec.go). */
  export const LocalSandbox: LocalSandboxConstructor;

  interface WorkspaceConstructor {
    new (config: WorkspaceConfig): WorkspaceInstance;
  }

  interface WorkspaceConfig {
    id?: string;
    name?: string;
    /** Filesystem provider. */
    filesystem: FilesystemInstance;
    /** Sandbox provider for command execution. */
    sandbox?: SandboxInstance;
    /** Enable BM25 keyword search. */
    bm25?: boolean | { k1?: number; b?: number };
    /** Vector store for semantic search. Requires `embedder`. */
    vectorStore?: VectorStore;
    /** Embedding function for vector search. Required if `vectorStore` is set. */
    embedder?: (text: string) => Promise<number[]>;
    /** Custom index name for vector search. Default: auto-generated from workspace ID. */
    searchIndexName?: string;
    /** Per-tool configuration — rename, enable/disable, require approval. */
    tools?: WorkspaceToolsConfig;
    /** Skills directory paths for SKILL.md discovery. */
    skills?: string[];
    /** LSP configuration for post-edit diagnostics. */
    lsp?: boolean | LSPConfig;
  }

  /** Per-tool configuration for workspace tools. */
  type WorkspaceToolsConfig = {
    /** Default enabled state for all tools. */
    enabled?: boolean;
    /** Default approval requirement for all tools. */
    requireApproval?: boolean;
  } & Record<string, WorkspaceToolConfig>;

  interface WorkspaceToolConfig {
    /** Whether this tool is enabled. Default: true. */
    enabled?: boolean;
    /** Custom name exposed to the LLM (e.g. rename "read_file" to "view"). */
    name?: string;
    /** Require user approval before execution. */
    requireApproval?: boolean;
    /** For write tools: require read_file before write_file. */
    requireReadBeforeWrite?: boolean;
    /** Max output tokens for this tool's response. */
    maxOutputTokens?: number;
  }

  interface LSPConfig {
    /** Diagnostic timeout in ms. Default: 5000. */
    diagnosticTimeout?: number;
    /** Server init timeout in ms. Default: 15000. */
    initTimeout?: number;
    /** Disable specific LSP servers. */
    disableServers?: string[];
    /** Custom binary paths. */
    binaryOverrides?: Record<string, string>;
    /** Package runner for missing binaries. */
    packageRunner?: string;
  }

  interface WorkspaceInstance {
    /** Initialize the workspace (creates indexes, starts LSP servers). */
    init(): Promise<void>;
    /** Destroy the workspace (stops LSP servers, cleans up). */
    destroy(): Promise<void>;
    /** Search workspace content. Mode auto-detected based on config. */
    search(query: string, options?: WorkspaceSearchOptions): Promise<WorkspaceSearchResult[]>;
    /** Index a file for search. */
    index(filePath: string, content: string): Promise<void>;
    /** Get workspace info (id, name, base path). */
    getInfo(): any;
    /** Get workspace instructions for agent context. */
    getInstructions(): string;
    /** Get the current tools configuration. */
    getToolsConfig(): WorkspaceToolsConfig | undefined;
    /** Update tools configuration at runtime (e.g. switch modes). */
    setToolsConfig(config: WorkspaceToolsConfig | undefined): void;
    /** The filesystem provider. */
    filesystem: FilesystemInstance;
  }

  interface WorkspaceSearchOptions {
    /** Number of results to return. Default: 10. */
    topK?: number;
    /** Minimum score threshold. */
    minScore?: number;
    /** Search mode. Auto-detected if omitted (hybrid if both BM25+vector, else whichever is available). */
    mode?: "bm25" | "vector" | "hybrid";
    /** Weight for vector scores in hybrid mode (0-1). Default: 0.5. */
    vectorWeight?: number;
    /** Metadata filter (vector search only). */
    filter?: Record<string, any>;
  }

  interface WorkspaceSearchResult {
    id: string;
    content: string;
    score: number;
    scoreDetails?: { vector?: number; bm25?: number };
    metadata?: Record<string, any>;
    lineRange?: { start: number; end: number };
  }

  interface LocalFilesystemConstructor {
    new (config: { basePath: string; allowedPaths?: string[]; contained?: boolean }): FilesystemInstance;
  }

  interface LocalSandboxConstructor {
    new (config: { workingDirectory?: string; env?: Record<string, string>; defaultShell?: string }): SandboxInstance;
  }

  interface FilesystemInstance {
    readFile(path: string): Promise<string>;
    writeFile(path: string, content: string): Promise<void>;
    stat(path: string): Promise<any>;
    readdir(path: string): Promise<string[]>;
    mkdir(path: string, options?: { recursive?: boolean }): Promise<void>;
    rm(path: string, options?: { recursive?: boolean }): Promise<void>;
    /** Update allowed paths at runtime. */
    setAllowedPaths?(paths: string[]): void;
  }

  interface SandboxInstance {
    exec(command: string): Promise<{ stdout: string; stderr: string; exitCode: number }>;
  }

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
  /**
   * Create a custom scorer using a pipeline pattern.
   *
   * @example Simple function scorer
   * ```ts
   * const scorer = createScorer({ id: "my-scorer", description: "..." })
   *   .generateScore(({ run }) => run.output.includes("hello") ? 1.0 : 0.0)
   *   .generateReason(({ score }) => score === 1 ? "Found" : "Missing");
   * ```
   *
   * @example LLM-based judge scorer
   * ```ts
   * const scorer = createScorer({
   *   id: "helpfulness",
   *   description: "Rate helpfulness",
   *   judge: { model: "openai/gpt-4o-mini", instructions: "You evaluate helpfulness." },
   * })
   *   .generateScore({
   *     description: "Rate from 0 to 1",
   *     createPrompt: ({ run }) => `Rate: "${run.output}" for "${run.input}"`,
   *   })
   *   .generateReason({
   *     description: "Explain rating",
   *     createPrompt: ({ run, score }) => `Explain why score=${score}`,
   *   });
   * ```
   */
  export function createScorer(config: ScorerConfig): ScorerBuilder;

  interface ScorerConfig {
    id: string;
    description: string;
    /** LLM judge — creates an internal agent for prompt-based scoring. */
    judge?: { model: string; instructions?: string };
  }

  interface ScorerBuilder {
    /** Function-based score step. */
    generateScore(fn: (context: { run: ScorerRun; results: Record<string, any> }) => number | Promise<number>): ScorerBuilder;
    /** Prompt-based score step (requires judge config). */
    generateScore(config: { description: string; createPrompt: (context: { run: ScorerRun; results: Record<string, any> }) => string }): ScorerBuilder;
    /** Function-based reason step. */
    generateReason(fn: (context: { run: ScorerRun; results: Record<string, any>; score: number }) => string | Promise<string>): ScorerBuilder;
    /** Prompt-based reason step (requires judge config). */
    generateReason(config: { description: string; createPrompt: (context: { run: ScorerRun; results: Record<string, any>; score: number }) => string }): ScorerBuilder;
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

  /**
   * Batch evaluation — run scorers against a dataset.
   *
   * @example
   * ```ts
   * const results = await runEvals({
   *   target: myAgent,
   *   data: [
   *     { input: "What is 2+2?", groundTruth: "4" },
   *     { input: "Capital of France?", groundTruth: "Paris" },
   *   ],
   *   scorers: [relevanceScorer, accuracyScorer],
   *   concurrency: 3,
   * });
   * console.log(results.scores); // { relevance: { average: 0.85 }, accuracy: { average: 0.9 } }
   * ```
   */
  export function runEvals(config: RunEvalsConfig): Promise<RunEvalsResult>;

  interface RunEvalsConfig {
    /** Agent to evaluate. Accepts brainkit wrapped agents or raw Mastra agents. */
    target: Agent;
    /** Dataset items — each has input (prompt) and optional groundTruth. */
    data: RunEvalsDataItem[];
    /** Scorers to run against each item's output. */
    scorers: ScorerBuilder[];
    /** Options passed to every target.generate() call. */
    targetOptions?: Partial<GenerateOptions>;
    /** Called after each item is scored. */
    onItemComplete?: (params: { item: RunEvalsDataItem; targetResult: any; scorerResults: Record<string, any> }) => void | Promise<void>;
    /** Max concurrent evaluations. Default: 1. */
    concurrency?: number;
  }

  interface RunEvalsDataItem {
    /** Input prompt or messages for the agent. */
    input: string | Message[];
    /** Expected/reference output for comparison scorers. */
    groundTruth?: string;
    /** Request context for this item. */
    requestContext?: Record<string, any>;
  }

  interface RunEvalsResult {
    /** Per-scorer aggregated scores (averages across all items). */
    scores: Record<string, { average: number; values: number[] }>;
    /** Summary metadata. */
    summary: { totalItems: number };
  }

  /** Pre-built scorers — rule-based and LLM-based. */
  export const scorers: {
    // Rule-based (no LLM needed)
    completeness(opts?: any): ScorerBuilder;
    textualDifference(opts?: any): ScorerBuilder;
    keywordCoverage(opts?: any): ScorerBuilder;
    contentSimilarity(opts?: any): ScorerBuilder;
    tone(opts?: any): ScorerBuilder;
    // LLM-based (require model)
    hallucination(opts: { model: string }): ScorerBuilder;
    faithfulness(opts: { model: string }): ScorerBuilder;
    answerRelevancy(opts: { model: string }): ScorerBuilder;
    answerSimilarity(opts: { model: string }): ScorerBuilder;
    bias(opts: { model: string }): ScorerBuilder;
    toxicity(opts: { model: string }): ScorerBuilder;
    contextPrecision(opts: { model: string }): ScorerBuilder;
    contextRelevance(opts: { model: string }): ScorerBuilder;
    noiseSensitivity(opts: { model: string }): ScorerBuilder;
    promptAlignment(opts: { model: string }): ScorerBuilder;
    toolCallAccuracy(opts: { model: string }): ScorerBuilder;
  };

  // ── MCP ─────────────────────────────────────────────────────────

  /** MCP Client — access tools from external MCP servers. */
  export const mcp: {
    /** List tools from an MCP server (or all servers if no name given). */
    listTools(serverName?: string): Promise<any[]>;
    /** Call a tool on an MCP server. */
    callTool(serverName: string, toolName: string, args?: any): Promise<any>;
  };

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
    id?: string;
    /** Model: static string or dynamic resolver. */
    model: DynamicArg<string>;
    /** Instructions: static string or dynamic resolver. */
    instructions?: DynamicArg<string>;
    description?: string;
    /** Tools: static map or dynamic resolver. */
    tools?: DynamicArg<Record<string, Tool>>;
    /** Memory config — enables conversation persistence. Can be a config object or a Memory instance. */
    memory?: AgentMemoryConfig | MemoryInstance;
    /** Maximum agentic loop steps (tool call rounds). */
    maxSteps?: number;
    /** Default options applied to every generate()/stream() call. */
    defaultOptions?: Partial<GenerateOptions>;
    /** Zod schema for validating requestContext entries. */
    requestContextSchema?: ZodType;
    /** Input processors — middleware that transforms messages before the LLM. */
    inputProcessors?: DynamicArg<InputProcessor[]>;
    /** Output processors — middleware that transforms or blocks LLM output. */
    outputProcessors?: DynamicArg<OutputProcessor[]>;
    /** Max retries when a processor requests retry via tripwire. Default: 3. */
    maxProcessorRetries?: number;
    /** Scorers — auto-evaluate responses after each generate()/stream() call (fire-and-forget). */
    scorers?: DynamicArg<Record<string, { scorer: ScorerBuilder; sampling?: { type: "none" } | { type: "ratio"; rate: number } }>>;
    /** Sub-agents for delegation/supervisor pattern.
     * Each agent becomes a tool named `agent-<name>` that the LLM can call.
     *
     * @example
     * ```ts
     * const researcher = agent({ model: "openai/gpt-4o-mini", instructions: "Research topics." });
     * const coder = agent({ model: "openai/gpt-4o-mini", instructions: "Write code." });
     *
     * const supervisor = agent({
     *   model: "openai/gpt-4o",
     *   instructions: "You are a tech lead. Delegate to your team.",
     *   agents: { researcher, coder },
     * });
     *
     * // Supervisor sees agent-researcher and agent-coder as tools
     * await supervisor.generate("Research RLHF then implement it");
     * // Or use network mode for multi-step delegation
     * await supervisor.network("Build a REST API with tests");
     * ```
     */
    agents?: DynamicArg<Record<string, Agent>>;
    /** Constrained subagent types — creates a `subagent` meta-tool.
     * Each type defines instructions + allowedTools. The LLM calls
     * `subagent({ agentType, task })` which spawns a fresh agent per invocation
     * with filtered tools from the parent's tool set.
     */
    subagents?: SubagentDef[];
    /** Callback for subagent lifecycle events (for UI rendering). */
    onSubagentEvent?: (event: SubagentEvent) => void;
    /** Workflows available as tools. Each becomes `workflow-<name>`. */
    workflows?: Record<string, any>;
    /** Delegation hooks for sub-agent calls. */
    delegation?: {
      /** Called before delegating to a sub-agent. Return { allowed: false } to reject. */
      onDelegationStart?: (ctx: { agentId: string; input: any }) => { allowed?: boolean; modifiedInput?: any } | void;
      /** Called after sub-agent completes. */
      onDelegationComplete?: (ctx: { agentId: string; output: any }) => void;
      /** Filter parent conversation before passing to sub-agent. */
      messageFilter?: (messages: Message[]) => Message[];
    };
    /** Workspace — auto-injects filesystem, sandbox, skills, search tools into the agent.
     * Can be a static Workspace instance or a dynamic factory resolved per-request.
     *
     * @example
     * ```ts
     * // Static workspace
     * workspace: new Workspace({ filesystem: new LocalFilesystem({ basePath: "./project" }) })
     *
     * // Dynamic workspace (resolved per generate/stream call)
     * workspace: ({ requestContext }) => {
     *   const path = requestContext.get("projectPath");
     *   return new Workspace({ filesystem: new LocalFilesystem({ basePath: path }) });
     * }
     * ```
     */
    workspace?: WorkspaceInstance | DynamicArg<WorkspaceInstance | undefined>;
    /** Voice — TTS/STT provider for speak/listen capabilities. */
    voice?: any;
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
    /** Scope: "thread" (default) or "resource". */
    scope?: "thread" | "resource";
    /** Minimum similarity threshold (0-1). */
    threshold?: number;
  }

  interface WorkingMemoryConfig {
    enabled?: boolean;
    /** Scope: "thread" (per-conversation) or "resource" (per-user, default). */
    scope?: "thread" | "resource";
    /** Markdown template for working memory structure. */
    template?: string;
    /** Read-only — agent can read but not update. */
    readOnly?: boolean;
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
    /** Supervisor/network mode — delegates to registered sub-agents.
     * The routing agent sees sub-agents as tools and decides what to delegate.
     * Continues until completion (LLM decides, or isTaskComplete scorer passes).
     */
    network(prompt: string | Message[], options?: GenerateOptions): Promise<GenerateResult>;
    /** Direct access to the Memory instance. null if no memory configured.
     * Use for thread management, message queries, working memory, etc.
     *
     * @example
     * ```ts
     * const a = agent({ model: "openai/gpt-4o-mini", memory: { thread: "t1", storage: store } });
     * const threads = await a.memory.listThreads({ resourceId: "user-1" });
     * const thread = await a.memory.getThreadById({ threadId: "t1" });
     * await a.memory.deleteMessages(["msg-1", "msg-2"]);
     * ```
     */
    memory: MemoryInstance | null;
  }

  interface GenerateOptions {
    maxSteps?: number;
    /** Memory thread/resource for this call. */
    memory?: { thread?: string | { id: string }; resource?: string; threadId?: string; resourceId?: string; options?: { readOnly?: boolean } };
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
      maxRetries?: number;
    };
    /** Filter which tools are active for this call. */
    activeTools?: string[];
    /** Additional tool sets for this call. */
    toolsets?: Record<string, Record<string, Tool>>;
    /** Client-side tools. */
    clientTools?: Record<string, Tool>;
    /** Tool selection strategy. */
    toolChoice?: "auto" | "none" | "required" | { type: "tool"; toolName: string };
    /** Require human approval before tool calls. */
    requireToolApproval?: boolean;
    /** Concurrent tool call limit. */
    toolCallConcurrency?: number;
    /** Abort signal for cancellation. */
    abortSignal?: AbortSignal;
    /** Stop condition — e.g. stepCountIs(3) from 'ai'. */
    stopWhen?: any;
    /** Called before each agentic loop step — can override model, toolChoice per step. */
    prepareStep?: (ctx: { stepNumber: number; model: any }) => Promise<{ model?: any; toolChoice?: any } | void> | void;
    /** Structured output schema. */
    structuredOutput?: {
      schema: ZodType;
      schemaName?: string;
      schemaDescription?: string;
      /** Model override for structured output extraction. */
      model?: string;
      /** Custom instructions for extraction. */
      instructions?: string;
      /** Error handling: "throw" (default) or "warn". */
      errorStrategy?: "throw" | "warn";
    };
    /** Additional context messages prepended to the conversation. */
    context?: Message[];
    /** Whether to return detailed scoring data. */
    returnScorerData?: boolean;
    /** Save messages after each step (not just at end). Default: false. */
    savePerStep?: boolean;
    /** Unique run ID for tracking. */
    runId?: string;
    /** Provider-specific options (e.g. { openai: { reasoningEffort: "high" } }). */
    providerOptions?: Record<string, Record<string, any>>;
    /** Telemetry settings. */
    telemetry?: { isEnabled?: boolean; recordInputs?: boolean; recordOutputs?: boolean; functionId?: string };
    /** Tracing options. */
    tracingOptions?: { metadata?: Record<string, any>; requestContextKeys?: string[]; traceId?: string };
    /** Override instructions for this call. */
    instructions?: string;
    /** System message for this call. */
    system?: string;
    /** Automatically resume suspended tools without human approval. */
    autoResumeSuspendedTools?: boolean;
    /** Include raw provider chunks in stream output. */
    includeRawChunks?: boolean;
    /** Callbacks. */
    onStepFinish?: (step: any) => void | Promise<void>;
    onFinish?: (result: any) => void | Promise<void>;
    onChunk?: (chunk: any) => void | Promise<void>;
    onError?: (error: { error: Error | string }) => void | Promise<void>;
    onAbort?: () => void | Promise<void>;
    /** Called after each LLM iteration. Return { continue: false } to stop. */
    onIterationComplete?: (ctx: any) => { continue?: boolean; feedback?: string } | void;
    /** Per-call input/output processors. */
    inputProcessors?: InputProcessor[];
    outputProcessors?: OutputProcessor[];
    /** Max processor retries (overrides agent config). */
    maxProcessorRetries?: number;
    /** Per-call scorers. */
    scorers?: Record<string, any>;
    /** Task completion pattern (supervisor). Scorers check if the task is done. */
    isTaskComplete?: {
      scorers: Array<{ scorer: any; threshold?: number }>;
      strategy?: "all" | "any";
    };
    /** Delegation hooks — intercept sub-agent and workflow calls. */
    delegation?: {
      onDelegationStart?: (ctx: { agentId: string; input: any }) => { allowed?: boolean; modifiedInput?: any } | void;
      onDelegationComplete?: (ctx: { agentId: string; output: any }) => void;
    };
  }

  interface StreamOptions extends GenerateOptions {}

  interface GenerateResult {
    text: string;
    reasoning: string;
    /** Structured output object (when using structuredOutput option). */
    object?: any;
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
    runId?: string;
    /** Provider-specific metadata. */
    providerMetadata?: Record<string, any>;
    /** Suspend payload (when a tool or workflow suspended). */
    suspendPayload?: any;
  }

  interface StreamResult {
    textStream: AsyncIterable<string>;
    fullStream: AsyncIterable<StreamPart>;
    text: Promise<string>;
    /** Structured output object (when using structuredOutput option). */
    object?: Promise<any>;
    usage: Promise<Usage>;
    totalUsage: Promise<Usage>;
    finishReason: Promise<FinishReason>;
    reasoning: Promise<string | undefined>;
    toolCalls: Promise<ToolCall[]>;
    toolResults: Promise<ToolResult[]>;
    steps: Promise<AgentStepResult[]>;
    sources: Promise<any[]>;
    files: Promise<any[]>;
    warnings: Promise<any[]>;
    response: Promise<ResponseMeta>;
    traceId?: string;
    runId?: string;
    error?: Promise<any>;
    /** Provider-specific metadata. */
    providerMetadata?: Promise<Record<string, any>>;
    /** Scoring data (when returnScorerData is true). */
    scoringData?: Promise<any>;
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
    /** Tool identifier. Also accepts `name` as alias. */
    id?: string;
    /** Alias for id. */
    name?: string;
    description?: string;
    /** Zod schema for the tool's input parameters. Also accepts `schema` as alias. */
    inputSchema?: ZodType;
    /** Alias for inputSchema. */
    schema?: ZodType;
    /** Zod schema for the tool's output. */
    outputSchema?: ZodType;
    /** Schema for suspend payload (HITL tools). */
    suspendSchema?: ZodType;
    /** Schema for resume data (HITL tools). */
    resumeSchema?: ZodType;
    /** Require human approval before execution. Emits tool-call-approval in stream. */
    requireApproval?: boolean | ((toolCallId: string, args: any) => boolean | Promise<boolean>);
    /** Transform execute output before sending to the model. */
    toModelOutput?: (output: any) => any;
    /** Provider-specific options. */
    providerOptions?: Record<string, any>;
    /** MCP annotations. */
    mcp?: { annotations?: { title?: string; readOnlyHint?: boolean; destructiveHint?: boolean; idempotentHint?: boolean; openWorldHint?: boolean }; _meta?: any };
    /** Schema for validating requestContext entries. */
    requestContextSchema?: ZodType;
    /** Lifecycle hooks. */
    onInputStart?: (ctx: { toolCallId: string; messages: any[]; abortSignal?: AbortSignal }) => void;
    onInputDelta?: (ctx: { toolCallId: string; delta: any; abortSignal?: AbortSignal }) => void;
    onInputAvailable?: (ctx: { toolCallId: string; input: any; abortSignal?: AbortSignal }) => void;
    onOutput?: (ctx: { toolCallId: string; output: any }) => void;
    /** Execute the tool. */
    execute: (input: any, context?: ToolExecuteContext) => Promise<any> | any;
  }

  interface ToolExecuteContext {
    /** Request context (from generate options). */
    requestContext?: RequestContextInstance;
    /** Tracing context for spans. */
    tracingContext?: any;
    /** Abort signal. */
    abortSignal?: AbortSignal;
    /** Suspend execution (HITL tools in workflows). */
    suspend?: (payload?: any) => Promise<never>;
    /** Resume data (when resuming after suspend). */
    resumeData?: any;
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

  interface MemoryThread {
    id: string;
    title?: string;
    resourceId?: string;
    createdAt?: string;
    updatedAt?: string;
    metadata?: Record<string, any>;
  }

  interface MemoryRecallResult {
    messages: any[];
    usage?: any;
    total?: number;
    page?: number;
    perPage?: number;
    hasMore?: boolean;
  }

  interface MemoryInstance {
    // ── Thread Management ────────────────────────────────────────

    /** Create a new thread. */
    createThread(opts: { resourceId: string; threadId?: string; title?: string; metadata?: any }): Promise<MemoryThread>;
    /** Get a thread by ID. */
    getThreadById(opts: { threadId: string }): Promise<MemoryThread | null>;
    /** List threads with pagination and filtering. */
    listThreads(opts?: { resourceId?: string; page?: number; perPage?: number }): Promise<{ threads: MemoryThread[]; total: number }>;
    /** Save/upsert a thread. */
    saveThread(opts: { thread: { id?: string; title?: string; resourceId?: string; metadata?: any; createdAt?: string; updatedAt?: string } }): Promise<MemoryThread>;
    /** Update thread title or metadata. */
    updateThread(opts: { id: string; title: string; metadata: Record<string, any>; memoryConfig?: any }): Promise<MemoryThread>;
    /** Delete a thread and all its messages. */
    deleteThread(threadId: string): Promise<void>;

    // ── Thread Cloning ───────────────────────────────────────────

    /** Clone a thread (messages, working memory, observations). */
    cloneThread(opts: { sourceThreadId: string; newThreadId?: string; resourceId?: string; title?: string; metadata?: any; options?: { filterMessages?: (msg: any) => boolean } }): Promise<{ thread: MemoryThread; clonedMessages: any[] }>;
    /** Check if a thread is a clone. */
    isClone(thread: MemoryThread): boolean;
    /** Get the source thread for a clone. */
    getSourceThread(opts: { threadId: string }): Promise<MemoryThread | null>;
    /** List all clones of a source thread. */
    listClones(opts: { sourceThreadId: string }): Promise<MemoryThread[]>;
    /** Get the full clone history chain (oldest → newest). */
    getCloneHistory(threadId: string): Promise<MemoryThread[]>;

    // ── Message Management ───────────────────────────────────────

    /** Retrieve messages + observations for a thread (the main recall API). */
    recall(opts: { threadId: string; resourceId?: string; vectorSearchString?: string; perPage?: number | false; page?: number; orderBy?: "asc" | "desc"; filter?: any }): Promise<MemoryRecallResult>;
    /** Save messages to a thread. */
    saveMessages(opts: { messages: any[] }): Promise<void>;
    /** Update existing messages. Auto-syncs vector embeddings if semantic recall is enabled. */
    updateMessages(opts: { messages: Array<{ id: string } & Record<string, any>> }): Promise<any[]>;
    /** Delete messages by ID. */
    deleteMessages(messageIds: string[] | Array<{ id: string }>): Promise<void>;
    /** Get messages by their IDs. */
    listMessagesById(opts: { messageIds: string[] }): Promise<any[]>;

    // ── Working Memory ───────────────────────────────────────────

    /** Get working memory for a thread/resource. */
    getWorkingMemory(opts: { threadId: string; resourceId?: string }): Promise<string | null>;
    /** Update working memory content. */
    updateWorkingMemory(opts: { threadId: string; resourceId?: string; content: string }): Promise<void>;
    /** Get the working memory template (markdown or JSON schema). */
    getWorkingMemoryTemplate(opts?: { memoryConfig?: any }): Promise<{ format: "json" | "markdown"; content: string } | null>;
    /** Build a formatted system message including working memory and tool instructions. */
    getSystemMessage(opts: { threadId: string; resourceId?: string }): Promise<string | null>;

    // ── Configuration & Introspection ────────────────────────────

    /** Get the memory configuration. */
    getConfig(): any;
    /** Get the merged thread config with defaults applied. */
    getMergedThreadConfig(config?: any): any;
    /** List tools auto-provided by memory (e.g. updateWorkingMemory tool). */
    listTools(): any[];
    /** Get input processors (includes OM processor if configured). */
    getInputProcessors(configured?: InputProcessor[], context?: any): Promise<InputProcessor[]>;
    /** Get output processors (includes OM processor if configured). */
    getOutputProcessors(configured?: OutputProcessor[], context?: any): Promise<OutputProcessor[]>;

    // ── Runtime Reconfiguration ──────────────────────────────────

    /** Attach or change the storage adapter. */
    setStorage(storage: StorageProvider): void;
    /** Attach or change the vector store for semantic recall. */
    setVector(vector: VectorStore): void;
    /** Set the embedding model for semantic recall. */
    setEmbedder(embedder: string | any): void;
  }

  interface MemoryConstructor {
    new (config: { storage?: StorageProvider; options?: any; vector?: any; embedder?: string | any }): MemoryInstance;
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
    /** Brainkit mode — auto-connects to Kit's embedded storage. */
    new (config: { id: string; storage?: string }): StorageProvider;
    /** Mastra mode — explicit URL (passthrough). */
    new (config: { id: string; url: string; authToken?: string; maxRetries?: number; initialBackoffMs?: number; disableInit?: boolean }): StorageProvider;
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

  // === HARNESS ===

  /** Create a Harness orchestrator (called from Go via Kit.InitHarness). */
  export function createHarness(config: string | HarnessJSConfig): boolean;

  interface HarnessJSConfig {
    id: string;
    resourceId?: string;
    modes: HarnessModeConfig[];
    stateSchema?: Record<string, any>;
    initialState?: Record<string, any>;
    subagents?: HarnessSubagentJSConfig[];
    omConfig?: HarnessOMJSConfig;
    defaultPermissions?: Record<string, string>;
  }

  interface HarnessModeConfig {
    id: string;
    name?: string;
    default?: boolean;
    defaultModelId?: string;
    color?: string;
    agentName?: string;
  }

  interface HarnessSubagentJSConfig {
    id: string;
    allowedTools?: string[];
    defaultModelId?: string;
    instructions?: string;
  }

  interface HarnessOMJSConfig {
    defaultObserverModel?: string;
    defaultReflectorModel?: string;
    observationThreshold?: number;
    reflectionThreshold?: number;
  }

  /** Event types emitted by the Harness (41 total). */
  type HarnessEventType =
    | "agent_start" | "agent_end"
    | "mode_changed" | "model_changed"
    | "thread_changed" | "thread_created" | "thread_deleted"
    | "message_start" | "message_update" | "message_end"
    | "tool_start" | "tool_approval_required" | "tool_input_start" | "tool_input_delta" | "tool_input_end" | "tool_update" | "tool_end" | "shell_output"
    | "ask_question" | "plan_approval_required" | "plan_approved"
    | "subagent_start" | "subagent_text_delta" | "subagent_tool_start" | "subagent_tool_end" | "subagent_end" | "subagent_model_changed"
    | "om_status" | "om_observation_start" | "om_observation_end" | "om_observation_failed"
    | "om_reflection_start" | "om_reflection_end" | "om_reflection_failed"
    | "om_buffering_start" | "om_buffering_end" | "om_buffering_failed"
    | "om_activation" | "om_model_changed"
    | "workspace_status_changed" | "workspace_ready" | "workspace_error"
    | "state_changed" | "display_state_changed" | "task_updated" | "usage_update" | "follow_up_queued" | "info" | "error";

  interface HarnessEvent {
    type: HarnessEventType;
    threadId?: string;
    runId?: string;
    toolCallId?: string;
    toolName?: string;
    messageId?: string;
    questionId?: string;
    planId?: string;
    modeId?: string;
    modelId?: string;
    agentType?: string;
    text?: string;
    content?: string;
    question?: string;
    plan?: string;
    error?: string;
    message?: string;
    delta?: string;
    title?: string;
    args?: Record<string, any>;
    result?: any;
    usage?: { promptTokens: number; completionTokens: number; totalTokens: number };
    tasks?: Array<{ title: string; description?: string; status: string }>;
    options?: string[];
    changedKeys?: string[];
    isError?: boolean;
    fatal?: boolean;
    status?: string;
    category?: string;
    finishReason?: string;
    stream?: string;
    data?: string;
    duration?: number;
    [key: string]: any;
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
