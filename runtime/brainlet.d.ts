/**
 * Brainlet Runtime Type Definitions
 *
 * These types define the developer-facing API for `.ts` files running
 * on the brainlet platform. Everything is imported from "brainlet".
 *
 * ⚠️  This file reflects what is IMPLEMENTED AND TESTED.
 *     Commented-out sections are designed but not yet wired.
 *     They will be uncommented as each feature is integrated.
 *
 * @see brainkit-maps/brainkit/DESIGN.md for the full architecture
 * @see brainkit-maps/references/sdk/DESIGN.md for the API surface design
 */
declare module "brainlet" {

  // ═══════════════════════════════════════════════════════════════
  // LOCAL — intra-sandbox, direct Mastra, no bus, no RBAC
  // ═══════════════════════════════════════════════════════════════

  /**
   * Create a persistent agent in this sandbox.
   * The agent lives in the JS runtime and can be called multiple times.
   *
   * @example
   * ```ts
   * const researcher = agent({
   *   model: "openai/gpt-4o-mini",
   *   instructions: "You research topics thoroughly.",
   *   tools: { search: searchTool },
   * });
   * const result = await researcher.generate("Find papers on RLHF");
   * ```
   */
  export function agent(config: AgentConfig): Agent;

  /**
   * Define a tool in this sandbox.
   * Local tools are only visible within this sandbox unless explicitly registered.
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

  /** Zod schema builder — use for tool input schemas. */
  export const z: Zod;

  /**
   * Direct LLM calls — LOCAL, same runtime as agents. No bus round-trip.
   * Use for quick text generation without the full agent loop.
   *
   * @example
   * ```ts
   * const result = await ai.generate({
   *   model: "openai/gpt-4o-mini",
   *   prompt: "Translate to French: Hello world",
   * });
   * console.log(result.text);
   * ```
   */
  export const ai: {
    /** Generate text from a prompt. */
    generate(params: AIGenerateParams): Promise<AIGenerateResult>;

    /** Stream text with real-time token delivery. */
    stream(params: AIStreamParams): Promise<StreamResult>;

    // ── Not yet wired (needs Go bridge for embedding) ──────────
    // embed(params: AIEmbedParams): Promise<AIEmbedResult>;
    // embedMany(params: AIEmbedManyParams): Promise<AIEmbedManyResult>;
  };

  // ═══════════════════════════════════════════════════════════════
  // PLATFORM — through bus, Go bridges, interceptors apply
  // ═══════════════════════════════════════════════════════════════

  /**
   * Tool registry — call tools from any namespace.
   * Tools can be registered by plugins, .ts files, WASM modules, or the platform.
   *
   * @example
   * ```ts
   * // Call a tool by short name (namespace resolution applies)
   * const rows = await tools.call("db_query", { sql: "SELECT * FROM users" });
   *
   * // Call a fully qualified tool
   * const data = await tools.call("plugin.postgres@1.0.0.db_query", { sql: "..." });
   * ```
   */
  export const tools: {
    /** Call a tool by name. Namespace resolution: caller → user → platform → plugin. */
    call(name: string, input?: any): Promise<any>;

    // ── Not yet wired ──────────────────────────────────────────
    // list(namespace?: string): Promise<ToolInfo[]>;
    // register(name: string, config: ToolRegisterConfig): Promise<void>;
  };

  /**
   * Look up a registered tool by name and return a Mastra-compatible tool object.
   * Use this to pass platform/plugin tools to an agent's tool config.
   *
   * @example
   * ```ts
   * const coder = agent({
   *   model: "openai/gpt-4o-mini",
   *   tools: { db: tool("db_query"), search: tool("search") },
   * });
   * ```
   */
  // export function tool(name: string): MastraTool;  // Not yet wired (needs tools.resolve handler)

  /**
   * Platform bus — pub/sub events and request/response.
   *
   * @example
   * ```ts
   * bus.publish("pipeline.complete", { result: output });
   * bus.subscribe("data.new", async (msg) => { ... });
   * ```
   */
  export const bus: {
    /** Send a message to a topic (fire and forget). */
    send(topic: string, payload?: any): Promise<void>;

    /** Alias for send. */
    publish(topic: string, payload?: any): Promise<void>;

    /** Send a request and wait for a response. */
    request(topic: string, payload?: any): Promise<any>;

    // ── Not yet wired ──────────────────────────────────────────
    // subscribe(topic: string, handler: (msg: BusMessage) => void): string;
  };

  /**
   * Sandbox context — identity and namespace of the current sandbox.
   *
   * @example
   * ```ts
   * console.log(sandbox.id);        // "a1b2c3..."
   * console.log(sandbox.namespace); // "user" or "agent.team-1"
   * console.log(sandbox.callerID);  // "user.my-script"
   * ```
   */
  export const sandbox: SandboxContext;

  // ── Not yet wired ──────────────────────────────────────────────
  // export const wasm: {
  //   compile(source: string, opts?: WASMCompileOpts): Promise<WASMModule>;
  //   run(module: WASMModule, input?: any): Promise<any>;
  //   validate(module: WASMModule): Promise<void>;
  //   deploy(module: WASMModule, trigger: TriggerConfig): Promise<void>;
  // };
  //
  // export const agents: {
  //   request(agentId: string, prompt: string): Promise<AgentResult>;
  //   send(agentId: string, message: string): Promise<void>;
  //   stream(agentId: string, prompt: string): AsyncIterable<string>;
  //   spawn(name: string, config: AgentConfig): Promise<AgentHandle>;
  //   broadcast(team: string, message: string): Promise<void>;
  //   list(): Promise<AgentInfo[]>;
  // };
  //
  // export function workflow(name: string): WorkflowBuilder;
  //
  // export const memory: {
  //   createThread(opts: ThreadOpts): Promise<Thread>;
  //   recall(opts: RecallOpts): Promise<RecallResult>;
  //   save(opts: SaveOpts): Promise<void>;
  // };
  //
  // export const knowledge: { ... };
  // export const fs: { ... };
  // export const http: { ... };
  // export const comms: { ... };
  // export const tasks: { ... };
  // export const triggers: { ... };
  // export const spend: { ... };

  // ═══════════════════════════════════════════════════════════════
  // Types
  // ═══════════════════════════════════════════════════════════════

  // ── Agent ────────────────────────────────────────────────────

  interface AgentConfig {
    /** Display name for the agent. */
    name?: string;

    /** Model identifier: "provider/model-id" (e.g., "openai/gpt-4o-mini"). */
    model: string;

    /** System instructions for the agent. */
    instructions?: string;

    /** Description of the agent's purpose. */
    description?: string;

    /** Tools available to the agent. Keys are tool names. */
    tools?: Record<string, MastraTool>;

    /** Maximum tool-calling iterations per generate/stream call. Default: 5. */
    maxSteps?: number;
  }

  interface Agent {
    /** Generate a complete response. */
    generate(prompt: string | Message[], options?: GenerateOptions): Promise<GenerateResult>;

    /** Stream a response with real-time token delivery. */
    stream(prompt: string | Message[], options?: StreamOptions): Promise<StreamResult>;
  }

  interface GenerateOptions {
    maxSteps?: number;
  }

  interface StreamOptions {
    maxSteps?: number;
  }

  interface GenerateResult {
    /** The generated text. */
    text: string;

    /** Reasoning text (if model supports it). */
    reasoning: string;

    /** Token usage. */
    usage: Usage;

    /** Why generation stopped. */
    finishReason: FinishReason;

    /** Tools that were called during generation. */
    toolCalls: ToolCall[];

    /** Results from tool executions. */
    toolResults: ToolResult[];

    /** Individual steps (for multi-step tool use). */
    steps: StepResult[];
  }

  /** StreamResult is the object returned by agent.stream(). */
  interface StreamResult {
    /** Async iterable of text chunks. */
    textStream: AsyncIterable<string>;

    /** Async iterable of all stream events. */
    fullStream: AsyncIterable<StreamPart>;

    /** Resolves to the complete text after stream finishes. */
    text: Promise<string>;

    /** Resolves to token usage after stream finishes. */
    usage: Promise<Usage>;

    /** Resolves to finish reason after stream finishes. */
    finishReason: Promise<FinishReason>;
  }

  // ── Tool ─────────────────────────────────────────────────────

  interface ToolConfig {
    /** Tool name / identifier. */
    name: string;

    /** Human-readable description (shown to the LLM). */
    description?: string;

    /** Zod schema for the tool's input parameters. */
    schema?: ZodObject;

    /** The function that executes when the tool is called. */
    execute: (input: any) => Promise<any> | any;
  }

  /** Opaque Mastra tool object. Pass to agent config or tool registry. */
  interface MastraTool {
    readonly _registryTool?: boolean;
  }

  // ── AI ───────────────────────────────────────────────────────

  interface AIGenerateParams {
    /** Model identifier: "provider/model-id". */
    model: string;

    /** The prompt to generate from. */
    prompt?: string;

    /** System prompt. */
    system?: string;
  }

  /** Parameters for ai.stream(). Same as AIGenerateParams. */
  interface AIStreamParams extends AIGenerateParams {}

  interface AIGenerateResult {
    /** The generated text. */
    text: string;

    /** Reasoning text (if model supports it). */
    reasoning: string;

    /** Token usage. */
    usage: Usage;

    /** Why generation stopped. */
    finishReason: FinishReason;

    /** Response metadata. */
    response: ResponseMeta;
  }

  // ── Shared Types ─────────────────────────────────────────────

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

  interface StepResult {
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
    /** Unique sandbox identifier. */
    readonly id: string;

    /** Sandbox namespace (e.g., "user", "agent.team-1"). */
    readonly namespace: string;

    /** Identity used for bus messages from this sandbox. */
    readonly callerID: string;
  }

  // ── Zod (minimal surface for tool schemas) ───────────────────

  interface ZodObject {
    describe(description: string): ZodObject;
    optional(): ZodObject;
  }

  interface Zod {
    string(): ZodObject;
    number(): ZodObject;
    boolean(): ZodObject;
    object(shape: Record<string, ZodObject>): ZodObject;
    array(element: ZodObject): ZodObject;
    any(): ZodObject;
    enum(values: string[]): ZodObject;
  }
}
