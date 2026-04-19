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

  // ── Foundational schema + argument types ──────────────────────
  //
  // Types that permeate the Mastra API. Kept loose here because
  // brainkit doesn't distinguish Standard Schema / Zod / JSON
  // Schema at the type level — Mastra does, but the distinction
  // rarely matters at `.ts` call sites inside a deployment.

  /**
   * Mastra's internal "standard schema" envelope — either a Zod
   * schema, a JSON-schema shape, or anything that validates via
   * `.parse`. Used on Tool / Workflow / Agent inputSchema +
   * outputSchema + structuredOutput slots.
   */
  export type StandardSchemaWithJSON<TIn = any, TOut = TIn> = import("ai").ZodType | {
    parse?: (input: unknown) => TOut;
    safeParse?: (input: unknown) => { success: boolean; data?: TOut; error?: unknown };
    jsonSchema?: Record<string, unknown>;
    _type?: TOut;
    _input?: TIn;
    [key: string]: unknown;
  };

  /** Extract the validated input type from a StandardSchema. */
  export type InferStandardSchemaInput<T> =
    T extends { _input: infer I } ? I :
    T extends { parse: (input: unknown) => infer O } ? O :
    unknown;

  /** Extract the output type from a StandardSchema. */
  export type InferStandardSchemaOutput<T> =
    T extends { _type: infer O } ? O :
    T extends { parse: (input: unknown) => infer O } ? O :
    unknown;

  /** User-facing schema — accepts StandardSchema or inline shape. */
  export type PublicSchema<T = unknown> = StandardSchemaWithJSON<T, T>;
  export type InferPublicSchema<T> = InferStandardSchemaOutput<T>;

  /**
   * `DynamicArgument` covers the pattern of "either a value or a
   * function that takes a RequestContext and returns the value".
   * Appears on every AgentConfig slot that resolves at call time.
   */
  export type DynamicArgument<T, TRequestContext = unknown> =
    | T
    | ((ctx: { requestContext?: RequestContext<TRequestContext extends Record<string, any> ? TRequestContext : Record<string, any>> }) => T | Promise<T>);

  // ── MCP + Vercel tool types ───────────────────────────────────

  /** Annotations attached to an MCP-sourced tool. */
  export interface ToolAnnotations {
    title?: string;
    readOnlyHint?: boolean;
    destructiveHint?: boolean;
    idempotentHint?: boolean;
    openWorldHint?: boolean;
  }

  /** Properties carried by a tool that came from an MCP server. */
  export interface MCPToolProperties {
    toolType: "mcp";
    annotations: ToolAnnotations;
    /** Server-supplied metadata. */
    _meta?: Record<string, unknown>;
  }

  /** AI SDK v4 tool shape. */
  export interface VercelTool {
    description?: string;
    parameters: import("ai").ZodType | Record<string, unknown>;
    execute?: (args: any, options?: any) => any | Promise<any>;
  }

  /** AI SDK v5 tool shape — the ESM + newer-return-type variant. */
  export interface VercelToolV5 {
    description?: string;
    inputSchema: import("ai").ZodType | Record<string, unknown>;
    execute?: (args: any, options?: any) => any | Promise<any>;
  }

  /** Provider-defined tool (Anthropic computer-use, OpenAI image, etc.). */
  export interface ProviderDefinedTool {
    type: "provider-defined";
    id: string;
    args: Record<string, unknown>;
    execute?: (args: any) => any | Promise<any>;
  }

  // ── Agent ─────────────────────────────────────────────────────

  /**
   * Mastra-shaped Agent class. Generics mirror the upstream
   * definition so IDE inference on `agent.id` / tool maps
   * works the same way.
   *
   * - `TAgentId` — narrows `agent.id` to the literal name.
   * - `TTools` — preserves the tool bag shape for type-safe
   *   `listTools()` returns.
   * - `TOutput` — the structured output type when configured.
   * - `TRequestContext` — the RequestContext value shape.
   *
   * All four default to loose types so existing `new Agent({...})`
   * call sites keep compiling.
   */
  export class Agent<
    TAgentId extends string = string,
    TTools extends Record<string, Tool> = Record<string, Tool>,
    TOutput = undefined,
    TRequestContext extends Record<string, any> = Record<string, any>,
  > {
    constructor(config: AgentConfig);
    readonly id: TAgentId;
    readonly name: string;

    /** Generate a response (non-streaming). */
    generate(promptOrMessages: string | Message[], options?: AgentCallOptions): Promise<AgentResult>;

    /** Legacy generate path — pre-`maxSteps` option shape. */
    generateLegacy(promptOrMessages: string | Message[], options?: AgentCallOptions): Promise<AgentResult>;

    /** Stream a response with real-time tokens. */
    stream(promptOrMessages: string | Message[], options?: AgentCallOptions): Promise<AgentStreamResult>;

    /** Legacy stream path — kept for pre-v0.20 callers. */
    streamLegacy(promptOrMessages: string | Message[], options?: AgentCallOptions): Promise<AgentStreamResult>;

    /** Supervisor mode — delegates to sub-agents. */
    network(promptOrMessages: string | Message[], options?: AgentCallOptions): Promise<AgentResult>;

    /** Resume a suspended generate run with fresh input. HITL pattern. */
    resumeGenerate(resumeData: unknown, options: AgentCallOptions & { runId: string; toolCallId?: string }): Promise<AgentResult>;

    /** Resume a suspended stream run with fresh input. HITL pattern. */
    resumeStream(resumeData: unknown, options: AgentCallOptions & { runId: string; toolCallId?: string }): Promise<AgentStreamResult>;

    /** Resume a suspended network run. */
    resumeNetwork(resumeData: unknown, options: AgentCallOptions & { runId: string }): Promise<AgentResult>;

    /** Approve a suspended tool call (non-streaming). HITL pattern. */
    approveToolCallGenerate(opts: AgentCallOptions & { runId: string; toolCallId: string }): Promise<AgentResult>;

    /** Decline a suspended tool call (non-streaming). */
    declineToolCallGenerate(opts: AgentCallOptions & { runId: string; toolCallId: string }): Promise<AgentResult>;

    /** Approve a suspended tool call (streaming). HITL pattern. */
    approveToolCall(opts: AgentCallOptions & { runId: string; toolCallId: string }): Promise<AgentStreamResult>;

    /** Decline a suspended tool call (streaming). */
    declineToolCall(opts: AgentCallOptions & { runId: string; toolCallId: string }): Promise<AgentStreamResult>;

    /** Approve a suspended tool call inside a network run. */
    approveNetworkToolCall(opts: AgentCallOptions & { runId: string; toolCallId: string }): Promise<AgentResult>;

    /** Decline a suspended tool call inside a network run. */
    declineNetworkToolCall(opts: AgentCallOptions & { runId: string; toolCallId: string }): Promise<AgentResult>;

    /** Attached voice provider (set via AgentConfig.voice). */
    readonly voice: MastraVoice;

    // ── Accessors ──────────────────────────────────────────────────

    /** Returns the configured voice provider (resolved if dynamic). */
    getVoice(opts?: { requestContext?: RequestContext }): Promise<MastraVoice | undefined>;
    /** Returns the configured memory (resolved if dynamic). */
    getMemory(opts?: { requestContext?: RequestContext }): Promise<Memory | undefined>;
    /** Returns true if memory is configured on this agent. */
    hasOwnMemory(): boolean;
    /** Returns the configured workspace (resolved if dynamic). */
    getWorkspace(opts?: { requestContext?: RequestContext }): Promise<Workspace | undefined>;
    /** Returns true if a workspace is configured on this agent. */
    hasOwnWorkspace(): boolean;
    /** Returns the agent's instructions (resolved if dynamic). */
    getInstructions(opts?: { requestContext?: RequestContext }): Promise<string | string[]>;
    /** Returns the agent description (empty string if unset). */
    getDescription(): string;
    /** Returns the tool map (resolved if dynamic). */
    listTools(opts?: { requestContext?: RequestContext }): Promise<Record<string, Tool>>;
    /** Returns workflows attached to this agent. */
    listWorkflows(opts?: { requestContext?: RequestContext }): Promise<Record<string, Workflow>>;
    /** Returns scorers attached to this agent. */
    listScorers(opts?: { requestContext?: RequestContext }): Promise<Record<string, { scorer: Scorer; sampling?: any }>>;
    /** Returns sub-agents for network/delegation patterns. */
    listAgents(opts?: { requestContext?: RequestContext }): Promise<Record<string, Agent>>;
    /** Returns input processors (including memory-derived). */
    listInputProcessors(requestContext?: RequestContext): Promise<any[]>;
    /** Returns output processors (including memory-derived). */
    listOutputProcessors(requestContext?: RequestContext): Promise<any[]>;
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
    /** Voice provider — OpenAIVoice / CompositeVoice / OpenAIRealtimeVoice. */
    voice?: MastraVoice;
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
    /** Output schema for structured output. Legacy single-call form. */
    output?: import("ai").ZodType;
    /**
     * Typed structured output for `agent.generate` / `agent.stream`.
     * Takes precedence over `output` when both are set. On `stream`,
     * the result exposes `object: Promise<T>` + `objectStream: AsyncIterable<Partial<T>>`.
     */
    structuredOutput?: {
      schema: import("ai").ZodType;
    };
    /**
     * Stop condition(s) — halt generation when met. Accepts a
     * single condition or an array evaluated in order. Common
     * helpers: `stepCountIs(n)`, `hasToolCall("name")`.
     */
    stopWhen?: any | any[];
    /**
     * Tools provided by the client (separate from server tools).
     * Used when the model runs on a client and the Kit mediates.
     */
    clientTools?: Record<string, Tool>;
    /**
     * Per-step modifier — called before each LLM step with the
     * step's model / tools / messages and returns overrides.
     */
    prepareStep?: (step: any) => any | Promise<any>;
    /**
     * Task-completion scorer — stops the loop when the scorer
     * says the task is done (LLM-judge or code-based).
     */
    isTaskComplete?: any;
    /** Include provider raw chunks in the stream. */
    includeRawChunks?: boolean;
    /** Fired after each outer iteration completes. */
    onIterationComplete?: (event: any) => void | Promise<void>;
    /**
     * Sub-agent delegation lifecycle hooks
     * (onDelegationStart / onDelegationComplete / messageFilter).
     */
    delegation?: {
      onDelegationStart?: (ctx: any) => any | Promise<any>;
      onDelegationComplete?: (ctx: any) => any | Promise<any>;
      messageFilter?: (ctx: any) => any | Promise<any>;
    };
    /** Per-call tracing overrides. */
    tracingOptions?: {
      enabled?: boolean;
      serviceName?: string;
      attributes?: Record<string, unknown>;
      [key: string]: any;
    };
    /** Called when the run is aborted (distinct from onError). */
    onAbort?: (event: any) => void | Promise<void>;
    /**
     * Network-mode routing configuration.
     * Controls how a supervisor agent picks between sub-agents.
     */
    routing?: {
      strategy?: "round-robin" | "model" | "first-match";
      [key: string]: any;
    };
    /**
     * Completion scorer chain for network mode — decides when
     * the whole multi-agent task is done.
     */
    completion?: any;
    /** Extra options passthrough. */
    [key: string]: any;
  }

  /** Alias — Mastra's formal name for the options shape. */
  export type AgentExecutionOptionsBase<OUTPUT = unknown> = AgentCallOptions & {
    structuredOutput?: { schema: import("ai").ZodType };
    _output?: OUTPUT;
  };

  /** Network-mode call options. */
  export type NetworkOptions<OUTPUT = unknown> = AgentCallOptions & {
    routing?: AgentCallOptions["routing"];
    completion?: AgentCallOptions["completion"];
    _output?: OUTPUT;
  };

  /**
   * Shape returned by `agent.generate(...)`. Mastra's internal
   * name is `FullOutput<OUTPUT>`; brainkit exports both names.
   */
  export interface AgentResult {
    /** Final reply text. */
    text: string;
    /** Reasoning traces if the model emitted any. */
    reasoning?: string;
    reasoningText?: string;
    /** Parsed structured output when `structuredOutput.schema` was set. */
    object?: unknown;
    /** Every tool call across every step. */
    toolCalls: Array<{ toolCallId: string; toolName: string; args: Record<string, unknown> }>;
    /** Results for each toolCall. */
    toolResults: Array<{ toolCallId: string; toolName: string; args: Record<string, unknown>; result: unknown }>;
    finishReason: string;
    usage: Usage;
    steps: AgentStepResult[];
    /** Model response metadata. */
    response: {
      id: string;
      modelId: string;
      timestamp: string;
      messages?: Message[];
    };
    request?: { body?: unknown };
    runId?: string;
    traceId?: string;
    /** Populated when a tool required approval and the run suspended. */
    suspendPayload?: unknown;
    /** Sources (citations) when the provider returns them. */
    sources?: Array<{ id: string; url?: string; title?: string; [key: string]: unknown }>;
    /** Files produced by the model (image generation, etc). */
    files?: Array<{ mediaType: string; data: Uint8Array | string; [key: string]: unknown }>;
    /** Provider warnings for unsupported features. */
    warnings?: Array<{ type: string; message: string; [key: string]: unknown }>;
    providerMetadata?: Record<string, unknown>;
    /** True if a moderation / PII / injection processor tripwired. */
    tripwire?: boolean;
    tripwireReason?: string;
    /** Data the scorer ran on, if `returnScorerData: true`. */
    scoringData?: unknown;
  }

  /** Alias matching Mastra's public type name. */
  export type FullOutput<OUTPUT = unknown> = AgentResult & { object?: OUTPUT };

  /** Token usage across a run. */
  export interface Usage {
    /** Legacy field — equivalent to `inputTokens`. */
    promptTokens: number;
    /** Legacy field — equivalent to `outputTokens`. */
    completionTokens: number;
    totalTokens: number;
    inputTokens?: number;
    outputTokens?: number;
    reasoningTokens?: number;
    cachedInputTokens?: number;
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

  /**
   * Shape returned by `agent.stream(...)`. Mastra's internal
   * name is `MastraModelOutput<OUTPUT>`; brainkit exports both
   * names. Same fields are available as promises for the
   * awaitable-final variant or as async iterables for the
   * streaming variant — consumers pick whichever matches
   * their loop.
   */
  export interface AgentStreamResult<OUTPUT = unknown> {
    // ── Async iterables ────────────────────────────────────
    /** Text chunks, one per LLM token emission. */
    textStream: AsyncIterable<string>;
    /** Typed stream parts (`text-delta`, `tool-call`, `object-result`, `finish`, etc.). */
    fullStream: AsyncIterable<import("ai").StreamPart>;
    /** Individual tool calls as they happen. */
    toolCallsStream?: AsyncIterable<{ toolCallId: string; toolName: string; args: Record<string, unknown> }>;
    /** Tool results as they resolve. */
    toolResultsStream?: AsyncIterable<{ toolCallId: string; toolName: string; result: unknown }>;
    /** Reasoning trace deltas. */
    reasoningStream?: AsyncIterable<string>;
    /** Source citations as they surface. */
    sourcesStream?: AsyncIterable<{ id: string; url?: string; title?: string }>;
    /** Files the model produces (image generation etc). */
    filesStream?: AsyncIterable<{ mediaType: string; data: Uint8Array | string }>;
    /**
     * Progressive partials of the structured output. Only populated
     * when `structuredOutput.schema` was set on the call.
     */
    objectStream?: AsyncIterable<Partial<OUTPUT>>;

    // ── Promises (resolve after stream finishes) ───────────
    text: Promise<string>;
    reasoning?: Promise<string>;
    /** Final parsed structured object. Set when `structuredOutput.schema` was provided. */
    object?: Promise<OUTPUT>;
    usage: Promise<Usage>;
    finishReason: Promise<string>;
    toolCalls: Promise<Array<{ toolCallId: string; toolName: string; args: Record<string, unknown> }>>;
    toolResults: Promise<Array<{ toolCallId: string; toolName: string; args: Record<string, unknown>; result: unknown }>>;
    steps: Promise<AgentStepResult[]>;
    sources?: Promise<Array<{ id: string; url?: string; title?: string }>>;
    files?: Promise<Array<{ mediaType: string; data: Uint8Array | string }>>;
    warnings?: Promise<Array<{ type: string; message: string }>>;
    response?: Promise<{ id: string; modelId: string; timestamp: string; messages?: Message[] }>;
    request?: Promise<{ body?: unknown }>;
    providerMetadata?: Promise<Record<string, unknown>>;
    tripwire?: Promise<boolean>;
    tripwireReason?: Promise<string>;
    scoringData?: Promise<unknown>;

    // ── Methods ────────────────────────────────────────────
    /** Await + collect every field into a FullOutput-shaped object. */
    getFullOutput(): Promise<FullOutput<OUTPUT>>;
    /** Drain the stream without observing; resolves when finished. */
    consumeStream(): Promise<void>;

    // ── HTTP / framework adapters (mirror Mastra's API so
    // users can opt into chunked-transfer responses without
    // reaching outside "agent"). brainkit's gateway wraps
    // these; .ts code on the Kit side rarely needs them. ──
    toDataStream?(): ReadableStream;
    toDataStreamResponse?(): Response;
    toTextStreamResponse?(): Response;
    pipeDataStreamToResponse?(response: any): Promise<void>;
    pipeTextStreamToResponse?(response: any): Promise<void>;
  }

  /** Alias matching Mastra's public type name. */
  export type MastraModelOutput<OUTPUT = unknown> = AgentStreamResult<OUTPUT>;

  /** Network-mode streaming result. */
  export type MastraAgentNetworkStream<OUTPUT = unknown> = AgentStreamResult<OUTPUT>;

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

  /**
   * Context handed to a tool's `execute` function. Covers every
   * shape Mastra passes at runtime — HITL suspend/resume, the
   * streaming writer slot, and the back-references to agent /
   * workflow / MCP contexts.
   */
  export interface ToolExecutionContext {
    requestContext?: RequestContext;
    abortSignal?: AbortSignal;
    mastra?: Mastra;
    /** Writable stream a tool can pipe progress updates into. */
    writer?: any;
    /** Workspace binding (when the agent has a workspace). */
    workspace?: Workspace;
    /** Present when invoked from an Agent execution. */
    agent?: {
      toolCallId: string;
      threadId?: string;
      resourceId?: string;
      suspend?: (payload: unknown) => Promise<never>;
      resumeData?: unknown;
      writableStream?: any;
    };
    /** Present when invoked from a Workflow step. */
    workflow?: {
      runId: string;
      stepId: string;
      suspend?: (payload: unknown) => Promise<never>;
    };
    /** Present when invoked through MCP. */
    mcp?: {
      toolCallId: string;
      [key: string]: unknown;
    };
  }

  export interface Tool {
    id: string;
    description?: string;
    inputSchema?: import("ai").ZodType;
    outputSchema?: import("ai").ZodType;
    suspendSchema?: import("ai").ZodType;
    resumeSchema?: import("ai").ZodType;
    /** HITL gate — when true, calls suspend for human approval. */
    requireApproval?: boolean;
    /** Provider-specific tool options (OpenAI, Anthropic, etc.). */
    providerOptions?: Record<string, Record<string, unknown>>;
    /** Present when this tool was sourced from an MCP server. */
    mcp?: MCPToolProperties;
    /** User-supplied MCP metadata. */
    mcpMetadata?: Record<string, unknown>;
    /** Few-shot input examples. */
    inputExamples?: unknown[];
    /** Normalize the tool output before it reaches the model. */
    toModelOutput?: (result: unknown) => unknown;
    execute?: (input: Record<string, unknown>, context?: ToolExecutionContext) => Promise<unknown>;
  }

  /**
   * Broader alias — Mastra accepts AI SDK v4 / v5 tools and
   * provider-defined tools alongside native Mastra tools. The
   * agent's tool resolver picks the right execution path.
   */
  export type ToolAction = Tool | VercelTool | VercelToolV5 | ProviderDefinedTool;
  /** Tool bag shape accepted by `AgentConfig.tools`. */
  export type ToolsInput = Record<string, ToolAction>;

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

  /**
   * Anything step-shaped Mastra's builder accepts — a formal
   * `Step`, an inline `async ({inputData}) => ...` executor,
   * or a wrapped tool / agent. Loose on purpose — Mastra's
   * runtime resolves the callable at execution time.
   */
  export type StepLike = Step | ((context: StepExecutionContext & { inputData: any }) => any | Promise<any>);

  export interface WorkflowBuilder {
    then(step: StepLike): WorkflowBuilder;
    parallel(steps: StepLike[]): WorkflowBuilder;
    /**
     * Branch on a predicate or a pair list of
     * `[condition, step]`. Accepts loose shapes — the runtime
     * does the discrimination.
     */
    branch(config: BranchConfig | Array<[StepLike, StepLike]> | StepLike[][]): WorkflowBuilder;
    foreach(config: ForEachConfig | StepLike): WorkflowBuilder;
    /** Loop while condition is true (check runs before each step). */
    dowhile(step: StepLike, condition?: (context: StepExecutionContext) => boolean | Promise<boolean>): WorkflowBuilder;
    /** Loop until condition is met (check runs after each step). */
    dountil(step: StepLike, condition?: (context: StepExecutionContext) => boolean | Promise<boolean>): WorkflowBuilder;
    /** Pause execution for a duration (ms). */
    sleep(ms: number | { duration: number }): WorkflowBuilder;
    /** Pause execution until a specific date/time. Mastra passes the Date directly. */
    sleepUntil(date: Date | ((ctx: any) => Date | Promise<Date>)): WorkflowBuilder;
    /**
     * Remap inputs / outputs between steps — useful when the
     * previous step emits a different shape than the next step
     * expects.
     */
    map(config: Record<string, unknown> | ((data: unknown) => unknown)): WorkflowBuilder;
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

  /**
   * Step definition — each step is a typed IO node in a
   * workflow. Mastra parameterizes with 8 generics; brainkit
   * keeps the public `Step` alias loose + provides
   * `StepDefinition` for authors who need the full shape.
   */
  export interface Step {
    readonly id?: string;
    readonly description?: string;
    readonly inputSchema?: import("ai").ZodType;
    readonly outputSchema?: import("ai").ZodType;
    readonly suspendSchema?: import("ai").ZodType;
    readonly resumeSchema?: import("ai").ZodType;
    readonly retries?: number;
  }

  /** Concrete step shape returned by createStep(...). */
  export interface StepDefinition<TInput = unknown, TOutput = unknown> extends Step {
    execute(context: StepExecutionContext & { inputData: TInput }): Promise<TOutput>;
  }

  /**
   * A completed workflow. Mastra parameterizes this class with
   * 8 generics; brainkit keeps the public alias loose because
   * most deployments rarely touch the generics — use
   * `WorkflowBuilder` chaining for compile-time type inference.
   */
  export interface Workflow<
    TInput = any,
    TOutput = any,
    TState = any,
    TRequestContext = any,
  > {
    readonly id?: string;
    readonly inputSchema?: import("ai").ZodType;
    readonly outputSchema?: import("ai").ZodType;
    readonly stateSchema?: import("ai").ZodType;
    readonly steps?: Record<string, Step>;
    /** True once `.commit()` has been called. */
    readonly committed?: boolean;
    createRun(opts?: { runId?: string }): Promise<WorkflowRun<TInput, TOutput>>;
    /** Shortcut — createRun + start. */
    execute?(input: TInput, opts?: { runId?: string; requestContext?: RequestContext<TRequestContext extends Record<string, any> ? TRequestContext : Record<string, any>> }): Promise<WorkflowRunResult<TOutput>>;
    /** List historical runs (when a persistence store is wired). */
    listWorkflowRuns?(opts?: { limit?: number; offset?: number }): Promise<WorkflowRunResult<TOutput>[]>;
    /** List active (in-flight) runs. */
    listActiveWorkflowRuns?(): Promise<WorkflowRun<TInput, TOutput>[]>;
    /** Delete one persisted run. */
    deleteWorkflowRunById?(runId: string): Promise<void>;
    /** Fetch one persisted run. */
    getWorkflowRunById?(runId: string): Promise<WorkflowRunResult<TOutput> | null>;
    /** Restart every active run. */
    restartAllActiveWorkflowRuns?(): Promise<void>;
    /** Scorers attached to this workflow. */
    listScorers?(): Promise<Record<string, Scorer>>;
    // Phantom markers — not runtime-accessible.
    readonly _input?: TInput;
    readonly _output?: TOutput;
    readonly _state?: TState;
    readonly _requestContext?: TRequestContext;
  }

  export interface WorkflowRun<TInput = unknown, TOutput = unknown> {
    runId: string;
    start(params: { inputData: TInput; requestContext?: RequestContext }): Promise<WorkflowRunResult<TOutput>>;
    /** Async variant — kick off + return a handle. */
    startAsync?(params: { inputData: TInput; requestContext?: RequestContext }): Promise<WorkflowRun<TInput, TOutput>>;
    /** Streaming run — emits each step as it completes. */
    stream?(params: { inputData: TInput; requestContext?: RequestContext }): AsyncIterable<unknown>;
    /** Observe another run's stream. */
    observeStream?(runId: string): AsyncIterable<unknown>;
    /** Legacy streaming path. */
    observeStreamLegacy?(runId: string): AsyncIterable<unknown>;
    resume(stepOrParams: string | { resumeData: Record<string, unknown>; step?: string }, resumeData?: Record<string, unknown>): Promise<WorkflowRunResult<TOutput>>;
    /** Resume + stream rather than resolve. */
    resumeStream?(opts: { step?: string; resumeData: Record<string, unknown> }): AsyncIterable<unknown>;
    /** Legacy stream path for resumed runs. */
    streamLegacy?(params: { inputData: TInput }): AsyncIterable<unknown>;
    /** Re-run this run from scratch with the original input. */
    restart?(): Promise<WorkflowRunResult<TOutput>>;
    /** Branch from a prior step + try an alternative path. */
    timeTravel?(opts: { stepId: string; inputData?: Record<string, unknown> }): Promise<WorkflowRunResult<TOutput>>;
    timeTravelStream?(opts: { stepId: string; inputData?: Record<string, unknown> }): AsyncIterable<unknown>;
    /** Attach a handler that fires on every step completion. */
    watch?(handler: (event: { stepId: string; status: string; result?: unknown }) => void | Promise<void>): () => void;
    watchAsync?(handler: (event: { stepId: string; status: string; result?: unknown }) => void | Promise<void>): () => void;
    cancel(): void;
    readonly status: string;
    readonly currentStep: string;
  }

  /**
   * Workflow run result. Shape matches Mastra's discriminated
   * status set — consumers can narrow on `.status` to inspect
   * the success payload / suspend payload / failure reason.
   */
  export interface WorkflowRunResult<TOutput = unknown> {
    status: "completed" | "suspended" | "failed" | "success";
    /** Final output (on success/completed). */
    result?: TOutput;
    runId?: string;
    steps?: Record<string, StepRunResult>;
    /** Populated when status === "suspended". */
    suspended?: { stepId: string; payload?: unknown };
    /** Populated when status === "failed". */
    error?: { message: string; cause?: unknown };
  }

  export interface StepRunResult<TOutput = any> {
    status: "completed" | "suspended" | "failed" | "skipped";
    output?: TOutput;
    /** Populated when status === "suspended". */
    suspended?: unknown;
    /** Populated when status === "failed". */
    error?: { message: string };
  }

  /** Discriminated union matching Mastra's StepResult export. */
  export type StepResult<TOutput = unknown> =
    | { status: "completed"; output: TOutput }
    | { status: "suspended"; suspended: unknown }
    | { status: "failed"; error: { message: string } }
    | { status: "skipped" };

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
    /** Set by the storage layer on save; optional on construction. */
    createdAt?: Date;
    /** Set by the storage layer on save; optional on construction. */
    updatedAt?: Date;
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
    /**
     * Semantic recall via vector embeddings. `boolean` toggles
     * the feature with defaults; the object form tunes per-kit.
     */
    semanticRecall?: boolean | SemanticRecallConfig;
    /**
     * Working memory for persistent user data. Discriminated
     * union matching Mastra source: enabled:false | template |
     * schema variant.
     */
    workingMemory?: WorkingMemory;
    /** Auto-generate thread titles from first message. */
    generateTitle?: boolean | {
      model?: any;
      instructions?: string;
    };
    /** Observational memory — 3-tier compression. */
    observationalMemory?: boolean | ObservationalMemoryOptions;
  }

  /** Full semantic-recall configuration. */
  export interface SemanticRecallConfig {
    /** How many semantically-similar messages to include. */
    topK?: number;
    /**
     * Window of messages around each hit — either a scalar
     * (before == after) or an explicit pair.
     */
    messageRange?: number | { before: number; after: number };
    /** `thread` = per-thread scope; `resource` = whole user. */
    scope?: "thread" | "resource";
    /** Minimum score to include a hit. */
    threshold?: number;
    /** Named index for the vector store. */
    indexName?: string;
    /** Backend-specific index config (metric, hnsw / ivf knobs). */
    indexConfig?: VectorIndexConfig;
  }

  /**
   * Vector index configuration — shape shared by storage-backed
   * memory, RAG, and any user-managed index.
   */
  export interface VectorIndexConfig {
    type?: "ivfflat" | "hnsw" | "flat";
    metric?: "cosine" | "euclidean" | "dotproduct";
    ivf?: { lists?: number };
    hnsw?: { m?: number; efConstruction?: number };
  }

  /**
   * Working memory — discriminated union:
   * - `{ enabled: false }` — disabled.
   * - `{ enabled: true; template }` — freeform Markdown template.
   * - `{ enabled: true; schema }` — typed Zod schema.
   */
  export type WorkingMemory =
    | { enabled: false }
    | {
        enabled: true;
        /** Optional freeform markdown template. */
        template?: string;
        /** Optional typed schema (Mastra narrows extracted data). */
        schema?: import("ai").ZodType;
        scope?: "thread" | "resource";
        version?: "vnext";
      };

  /**
   * Observational memory — Mastra's "journal" style persistence
   * that the model writes to autonomously.
   */
  export interface ObservationalMemoryOptions {
    enabled?: boolean;
    scope?: "thread" | "resource";
    model?: any;
    observation?: {
      model?: any;
      messageTokens?: number;
      modelSettings?: any;
      maxTokensPerBatch?: number;
      bufferTokens?: number;
      instruction?: string;
      threadTitle?: string;
    };
    reflection?: {
      model?: any;
      observationTokens?: number;
      modelSettings?: any;
      instruction?: string;
    };
    shareTokenBudget?: boolean;
    retrieval?: {
      topK?: number;
      threshold?: number;
    };
  }

  // ── Storage backends ──────────────────────────────────────────

  /** Marker type for resolved storage instances. */
  export interface StorageInstance {
    /** @internal */ readonly __storageType: string;
    /** Friendly name — set by kit.Storage() on resolution. */
    readonly name?: string;
  }

  export class InMemoryStore implements StorageInstance {
    readonly __storageType: "memory";
    readonly name?: string;
    constructor(config?: { id?: string });
    /**
     * Access a domain-specific sub-store — `memory`,
     * `workflow`, `evals`, `scores`, `audit`, etc. Matches
     * Mastra's `MastraStorage.getStore` contract.
     */
    getStore(storeName: string): Promise<any | undefined>;
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
    upsert(opts: { indexName: string; vectors: VectorEntry[]; metadata?: Record<string, unknown>; ids?: string[] }): Promise<string[]>;
    query(opts: { indexName: string; queryVector: number[]; topK?: number; filter?: VectorFilter; includeVector?: boolean }): Promise<QueryResult[]>;
    /** Upsert a single entry by id. Some backends alias this to upsert. */
    updateIndexById?(indexName: string, id: string, update: { vector?: number[]; metadata?: Record<string, unknown> }): Promise<void>;
    /** Delete a single entry. */
    deleteIndexById?(indexName: string, id: string): Promise<void>;
  }

  /**
   * Filter DSL accepted by every Mastra vector store.
   * Operators match MongoDB's convention; brainkit vector
   * backends implement the subset their engine supports.
   */
  export type VectorFilter = Record<string, FilterOperator | string | number | boolean | null | (string | number | boolean | null)[]>;

  export type FilterOperator =
    | { $eq: unknown }
    | { $ne: unknown }
    | { $gt: number | string | Date }
    | { $gte: number | string | Date }
    | { $lt: number | string | Date }
    | { $lte: number | string | Date }
    | { $in: unknown[] }
    | { $nin: unknown[] }
    | { $exists: boolean }
    | { $regex: string }
    | { $contains: unknown }
    | { $size: number }
    | { $and: VectorFilter[] }
    | { $or: VectorFilter[] }
    | { $not: VectorFilter };

  /** Single hit from vector `.query(...)`. Alias for VectorQueryResult. */
  export type QueryResult = VectorQueryResult;

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
    /** Some backends (Pinecone, Chroma) return the source document verbatim. */
    document?: string;
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
    /** Opens the underlying MongoDB driver connection. */
    connect(): Promise<void>;
    /** Closes the underlying connection. */
    disconnect(): Promise<void>;
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

  /**
   * Key-value context carried through agent / tool / workflow
   * execution. Mastra uses this for per-call configuration
   * (tenant id, locale, feature flags). The generic narrows
   * `.get` / `.set` to the declared value type.
   *
   * @example
   * ```ts
   * type Ctx = { userId: string; tenantId: string };
   * const rc = new RequestContext<Ctx>();
   * rc.set("userId", "abc");        // typed: value must be string
   * const id = rc.get("userId");     // typed: string | undefined
   * ```
   */
  export class RequestContext<Values extends Record<string, any> = any> {
    constructor(initial?: Iterable<[keyof Values, Values[keyof Values]]> | RequestContext<Values>);

    set<K extends keyof Values>(key: K, value: Values[K]): this;
    get<K extends keyof Values, R = Values[K]>(key: K): R | undefined;
    has<K extends keyof Values>(key: K): boolean;
    delete<K extends keyof Values>(key: K): boolean;
    clear(): void;
    keys(): IterableIterator<keyof Values>;
    values(): IterableIterator<Values[keyof Values]>;
    entries(): IterableIterator<[keyof Values, Values[keyof Values]]>;
    /** NOTE: method (not a number property) — matches Mastra's source. */
    size(): number;
    forEach(cb: (value: Values[keyof Values], key: keyof Values, ctx: this) => void, thisArg?: unknown): void;
    toJSON(): Record<string, any>;
    /** Snapshot of every value. */
    readonly all: Values;
    [Symbol.iterator](): IterableIterator<[keyof Values, Values[keyof Values]]>;
  }

  /**
   * Reserved keys Mastra interprets specifically when they
   * appear on a RequestContext. Using the constants keeps code
   * robust against Mastra renaming the underlying string.
   */
  export const MASTRA_RESOURCE_ID_KEY: "mastra__resourceId";
  export const MASTRA_THREAD_ID_KEY: "mastra__threadId";

  // ── Mastra top-level class ─────────────────────────────────────

  /**
   * Mastra coordinator. brainkit deployments don't construct
   * this directly (the Kit owns agent / workflow / storage
   * registration), but several AgentConfig fields declare
   * `mastra?: Mastra` — the type alias keeps those slots
   * structured instead of `any`.
   *
   * Not runtime-instantiable inside a brainkit deployment.
   */
  export class Mastra {
    readonly agents: Record<string, Agent>;
    readonly workflows: Record<string, Workflow>;
    getAgent(name: string): Agent | undefined;
    listAgents(): Record<string, Agent>;
    getWorkflow(name: string): Workflow | undefined;
    listWorkflows(): Record<string, Workflow>;
    getVector(name: string): any;
    getStorage(): any;
    getMemory(): Memory | undefined;
    getLogger(): any;
  }

  // ── Workspace ─────────────────────────────────────────────────

  export class Workspace {
    constructor(config?: WorkspaceConfig);
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
    constructor(config?: { basePath?: string; allowedPaths?: string[]; contained?: boolean });
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

  /**
   * Document class for RAG chunking. Every factory returns an
   * `MDocument` instance whose `chunk()` method emits chunks
   * ready for embedding + upsert.
   */
  export class MDocument {
    static fromText(text: string, metadata?: Record<string, unknown>): MDocument;
    static fromMarkdown(markdown: string, metadata?: Record<string, unknown>): MDocument;
    static fromHTML(html: string, metadata?: Record<string, unknown>): MDocument;
    static fromJSON(jsonString: string, metadata?: Record<string, unknown>): MDocument;

    chunk(options?: ChunkingConfig): Promise<DocumentChunk[]>;
    getText(): string[];
    getDocs(): DocumentChunk[];
    getMetadata(): Record<string, unknown>[];
  }

  /**
   * Chunking configuration — discriminated union so IDE
   * completion narrows on strategy-specific knobs. Mirror of
   * Mastra's chunking strategies in `@mastra/rag`.
   */
  export type ChunkingConfig =
    | { strategy?: "recursive"; size?: number; maxSize?: number; overlap?: number; separators?: string[]; separator?: string; extract?: ChunkExtractConfig }
    | { strategy: "character"; size?: number; maxSize?: number; overlap?: number; separator?: string; extract?: ChunkExtractConfig }
    | { strategy: "token"; size?: number; maxSize?: number; overlap?: number; encoding?: string; extract?: ChunkExtractConfig }
    | { strategy: "markdown"; headers?: [string, string][] | Record<string, string>; size?: number; maxSize?: number; overlap?: number; extract?: ChunkExtractConfig }
    | { strategy: "html"; headers?: [string, string][] | Record<string, string>; size?: number; maxSize?: number; overlap?: number; extract?: ChunkExtractConfig }
    | { strategy: "sentence"; size?: number; maxSize?: number; overlap?: number; extract?: ChunkExtractConfig }
    | {
        strategy: "semantic-markdown";
        size?: number;
        maxSize?: number;
        overlap?: number;
        joinThreshold?: number;
        encodingName?: string;
        modelName?: string;
        allowedSpecial?: "all" | Set<string>;
        disallowedSpecial?: "all" | Set<string>;
        extract?: ChunkExtractConfig;
      }
    | { strategy: "json"; size?: number; maxSize?: number; overlap?: number; convertLists?: boolean; extract?: ChunkExtractConfig }
    | { strategy: "latex"; size?: number; maxSize?: number; overlap?: number; extract?: ChunkExtractConfig };

  export interface ChunkExtractConfig {
    /** Extract keywords from each chunk (default false). */
    keywords?: boolean | { count?: number };
    /** Extract summary. */
    summary?: boolean | { prompt?: string };
    /** Extract questions a chunk could answer. */
    questions?: boolean | { count?: number };
    /** Extract title. */
    title?: boolean | { prompt?: string };
  }

  /** Legacy alias; prefer ChunkingConfig in new code. */
  export type ChunkOptions = ChunkingConfig;

  export interface DocumentChunk {
    text: string;
    metadata: Record<string, unknown>;
  }

  /**
   * Graph RAG — builds a knowledge graph over a chunk set then
   * queries it with random-walk restart for multi-hop recall.
   */
  export class GraphRAG {
    constructor(config: { dimension: number; threshold?: number });
    /** Build the graph from pre-chunked docs + their embeddings. */
    createGraph(chunks: DocumentChunk[], embeddings: number[][]): Promise<void>;
    query(opts: {
      queryEmbedding: number[];
      topK?: number;
      randomWalkSteps?: number;
      restartProb?: number;
    }): Promise<GraphRAGResult>;
  }

  export interface GraphRAGResult {
    answer: string;
    sources: Array<{ text: string; score: number }>;
  }

  /**
   * Create a Mastra vector-query tool. Two shapes:
   *   - `{ vectorStore: VectorStoreInstance, ... }` — direct instance (the
   *     path brainkit consumers use because there's no Mastra registry
   *     inside the SES compartment).
   *   - `{ vectorStoreName: string, ... }` — Mastra registry lookup (only
   *     works when a Mastra instance is configured; not applicable inside
   *     a brainkit deployment).
   * `model` is the embedding model used to vectorize the incoming query.
   */
  export function createVectorQueryTool(
    config:
      | {
          vectorStore: VectorStoreInstance;
          indexName: string;
          model: import("ai").EmbeddingModel;
          topK?: number;
          description?: string;
          enableFilter?: boolean;
          reranker?: { scorer?: Scorer; model?: import("ai").EmbeddingModel; weights?: RerankWeights; topK?: number };
        }
      | {
          vectorStoreName: string;
          indexName: string;
          model: import("ai").EmbeddingModel;
          topK?: number;
          description?: string;
          enableFilter?: boolean;
        },
  ): Tool;

  export function createDocumentChunkerTool(config: {
    vectorStore: VectorStoreInstance;
    indexName: string;
    model: import("ai").EmbeddingModel;
    chunkOptions?: ChunkOptions;
  }): Tool;

  export function createGraphRAGTool(config: {
    graphRag: GraphRAG;
    description?: string;
  }): Tool;

  /**
   * Positional rerank. Accepts the Mastra query-result shape returned by
   * `VectorStoreInstance.query` and a relevance model.
   */
  export function rerank(
    results: QueryResult[],
    query: string,
    model: import("ai").EmbeddingModel,
    options?: { topK?: number; weights?: RerankWeights },
  ): Promise<RerankResult[]>;

  /**
   * Rerank via a custom scorer (LLM-as-judge or code scorer). Weights
   * combine semantic relevance, vector similarity, and position.
   */
  export function rerankWithScorer(config: {
    results: QueryResult[];
    query: string;
    scorer: Scorer | RelevanceScoreProvider;
    options?: {
      weights?: RerankWeights;
      queryEmbedding?: number[];
      topK?: number;
    };
  }): Promise<RerankResult[]>;

  /** Weights for the combined reranking score. Must sum to 1.0. */
  export interface RerankWeights {
    semantic?: number;
    vector?: number;
    position?: number;
  }

  export interface RerankResult {
    result: QueryResult;
    score: number;
    details: {
      semantic: number;
      vector: number;
      position: number;
      queryAnalysis?: { magnitude: number; dominantFeatures: number[] };
    };
  }

  /** Relevance score provider — Mastra's shipped scorers (Cohere, etc.). */
  export interface RelevanceScoreProvider {
    score(query: string, text: string): Promise<number> | number;
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

  /**
   * Batch-evaluate an Agent or Workflow against a dataset, applying
   * one or more scorers concurrently. Returns aggregate + per-item
   * scores.
   */
  export function runEvals(config: RunEvalsOptions): Promise<RunEvalsResult>;

  export interface RunEvalsOptions {
    /** The target to evaluate. */
    target: Agent | Workflow;
    /** Test cases. */
    data: RunEvalsDataItem[];
    /**
     * Scorers to run. Flat array applies every scorer to the raw
     * output. For agents, the `AgentScorerConfig` object lets you
     * separate agent-output scorers from trajectory scorers.
     */
    scorers: Scorer[] | AgentScorerConfig | WorkflowScorerConfig;
    /** Options forwarded to the target during execution. */
    targetOptions?: AgentCallOptions | Record<string, any>;
    /** Concurrent test cases. Default: 1. */
    concurrency?: number;
    /** Callback fired after each item completes. */
    onItemComplete?: (ctx: {
      item: RunEvalsDataItem;
      targetResult: any;
      scorerResults: Record<string, ScorerRunResult>;
    }) => void;
  }

  export interface RunEvalsDataItem {
    input: string | string[] | Message[] | any;
    groundTruth?: any;
    expectedTrajectory?: any;
    requestContext?: any;
    tracingContext?: any;
    startOptions?: any;
  }

  export interface AgentScorerConfig {
    agent?: Scorer[];
    trajectory?: Scorer[];
  }

  export interface WorkflowScorerConfig {
    workflow?: Scorer[];
    trajectory?: Scorer[];
    steps?: Record<string, Scorer[]>;
  }

  export interface RunEvalsResult {
    scores: Record<string, number>;
    summary: {
      totalItems: number;
      completedItems?: number;
      failedItems?: number;
    };
    results: Array<{
      item: RunEvalsDataItem;
      targetResult: any;
      scorerResults: Record<string, ScorerRunResult>;
    }>;
  }

  // ── Prebuilt scorer factories (@mastra/evals/scorers/prebuilt) ────
  //
  // These ship with Mastra but become usable inside a .ts deployment
  // only when the brainkit bundle re-exports them (session 05 lands
  // the endowment wiring). Types declared up front so the IDE sees
  // them once the runtime surface catches up.

  export interface LLMJudgeScorerOptions {
    model: any;
    options?: { uncertaintyWeight?: number; scale?: number };
  }

  export function createAnswerRelevancyScorer(opts: LLMJudgeScorerOptions): Scorer;
  export function createAnswerSimilarityScorer(opts: {
    model: any;
    options?: {
      requireGroundTruth?: boolean;
      semanticThreshold?: number;
      exactMatchBonus?: number;
      missingPenalty?: number;
      contradictionPenalty?: number;
      extraInfoPenalty?: number;
      scale?: number;
    };
  }): Scorer;
  export function createFaithfulnessScorer(opts: {
    model: any;
    options?: { scale?: number; context?: string[] };
  }): Scorer;
  export function createBiasScorer(opts: { model: any; options?: { scale?: number } }): Scorer;
  export function createHallucinationScorer(opts: {
    model: any;
    options?: { scale?: number; context?: string[]; getContext?: (p: any) => string[] | Promise<string[]> };
  }): Scorer;
  export function createToxicityScorer(opts: LLMJudgeScorerOptions): Scorer;
  export function createContextPrecisionScorer(opts: {
    model: any;
    options: { scale?: number; context?: string[]; contextExtractor?: (input: any, output: any) => string[] };
  }): Scorer;
  // Mastra suffixes the LLM-judge variants with `LLM`; brainkit
  // mirrors that so consumers can tell the judge variant apart
  // from the code-only scorers at a glance.
  export function createContextRelevanceScorerLLM(opts: {
    model: any;
    options: {
      scale?: number;
      context?: string[];
      contextExtractor?: (input: any, output: any) => string[];
      penalties?: {
        unusedHighRelevanceContext?: number;
        missingContextPerItem?: number;
        maxMissingContextPenalty?: number;
      };
    };
  }): Scorer;
  export function createNoiseSensitivityScorerLLM(opts: {
    model: any;
    options: {
      baselineResponse: string;
      noisyQuery: string;
      noiseType?: string;
      scoring?: {
        impactWeights?: Record<string, number>;
        penalties?: { majorIssuePerItem?: number; maxMajorIssuePenalty?: number };
        discrepancyThreshold?: number;
      };
    };
  }): Scorer;
  export function createPromptAlignmentScorerLLM(opts: {
    model: any;
    options?: { scale?: number; evaluationMode?: "user" | "system" | "both" };
  }): Scorer;
  export function createToolCallAccuracyScorerLLM(opts: { model: any; availableTools: any[] }): Scorer;

  export function createCompletenessScorer(): Scorer;
  export function createContentSimilarityScorer(opts?: { ignoreCase?: boolean; ignoreWhitespace?: boolean }): Scorer;
  export function createKeywordCoverageScorer(): Scorer;
  export function createTextualDifferenceScorer(): Scorer;
  export function createToneScorer(opts?: { referenceTone?: string }): Scorer;

  // ── Observability extras ────────────────────────────────────────

  export class SensitiveDataFilter {
    constructor(config?: { patterns?: RegExp[]; replacement?: string });
  }

  // ── Voice providers ────────────────────────────────────────────

  /**
   * Audio bytes produced or consumed by a voice provider.
   * Mastra historically uses NodeJS.ReadableStream; brainkit's
   * polyfill surface also accepts Uint8Array + Int16Array on
   * send(), so the type here is intentionally loose.
   */
  export type VoiceAudioStream = any;

  /**
   * Common option shapes shared across providers. Individual
   * providers extend these with their own knobs (emotion,
   * voiceId, language, etc.).
   */
  export interface VoiceSpeakOptions {
    speaker?: string;
    responseFormat?: "mp3" | "opus" | "aac" | "flac" | "wav" | "pcm" | string;
    [key: string]: any;
  }
  export interface VoiceListenOptions {
    filetype?: "mp3" | "mp4" | "mpeg" | "mpga" | "m4a" | "wav" | "webm";
    language?: string;
    [key: string]: any;
  }

  /**
   * `MastraVoice` is the abstract base class every provider
   * extends. The contract is `speak()` + `listen()`; realtime
   * providers add `connect()` / `send()` / `on()`. Exported as
   * a class so custom provider subclasses type-check against
   * it directly: `class MyVoice extends MastraVoice { ... }`.
   */
  export abstract class MastraVoice {
    constructor(config?: any);
    abstract speak(input: string | VoiceAudioStream, options?: VoiceSpeakOptions): Promise<VoiceAudioStream> | void;
    abstract listen(audio: VoiceAudioStream, options?: VoiceListenOptions): Promise<string>;
    getSpeakers?(): Promise<Array<{ voiceId: string; name?: string; [key: string]: any }>>;
    addInstructions?(instructions: string): void;
    addTools?(tools: Record<string, unknown>): void;
    updateConfig?(config: any): void;
    close?(): void;
  }

  /**
   * OpenAI's whisper (STT) + TTS provider. Without config the
   * constructor uses `process.env.OPENAI_API_KEY`.
   */
  export class OpenAIVoice implements MastraVoice {
    constructor(config?: {
      speechModel?: { name?: string; apiKey?: string; options?: any };
      listeningModel?: { name?: string; apiKey?: string; options?: any };
      speaker?: string;
    });
    speak(
      input: string | VoiceAudioStream,
      options?: { speaker?: string; responseFormat?: "mp3" | "opus" | "aac" | "flac" | "wav" | "pcm"; [key: string]: any },
    ): Promise<VoiceAudioStream>;
    listen(
      audio: VoiceAudioStream,
      options?: { filetype?: "mp3" | "mp4" | "mpeg" | "mpga" | "m4a" | "wav" | "webm"; [key: string]: any },
    ): Promise<string>;
  }

  /**
   * Route speak() and listen() through different providers so
   * you can mix-and-match (e.g. OpenAI TTS + a separate STT).
   */
  export class CompositeVoice implements MastraVoice {
    constructor(config: {
      speakProvider?: MastraVoice;
      listenProvider?: MastraVoice;
      realtimeProvider?: MastraVoice;
    });
    speak(input: string | VoiceAudioStream, options?: any): Promise<VoiceAudioStream>;
    listen(audio: VoiceAudioStream, options?: any): Promise<string>;
  }

  /**
   * OpenAI Realtime voice — bidirectional WebSocket session.
   * Requires `globalThis.WebSocket` (brainkit ships a client
   * polyfill in `internal/jsbridge/websocket.go`).
   */
  export class OpenAIRealtimeVoice implements MastraVoice {
    constructor(config?: {
      realtimeConfig?: {
        model?: string;
        apiKey?: string;
        url?: string;
        options?: {
          sessionConfig?: {
            turn_detection?: {
              type?: "server_vad";
              threshold?: number;
              silence_duration_ms?: number;
              prefix_padding_ms?: number;
            };
            [key: string]: any;
          };
        };
      };
      speaker?: string;
    });

    /** Open the WebSocket session. */
    connect(options?: { runtimeContext?: any }): Promise<void>;
    /** Close the WebSocket session. */
    disconnect(): void;
    close(): void;

    /** Push microphone audio to the model. */
    send(audio: VoiceAudioStream | Int16Array, eventId?: string): Promise<void>;
    /** Trigger a TTS response. */
    speak(input: string, options?: { speaker?: string; [key: string]: any }): Promise<VoiceAudioStream>;
    /** Single-shot listen path (buffers internally). */
    listen(audio: VoiceAudioStream, options?: any): Promise<string>;

    /** Update session config mid-flight (voice, VAD, instructions). */
    updateSession(config: any): void;
    updateConfig(config: any): void;

    /**
     * Realtime events:
     *   "speaker"   - a new reply audio stream (Node Readable of PCM)
     *   "writing"   - partial / final transcript ({text, role, response_id})
     *   "speaking"  - each decoded audio chunk ({audio, response_id})
     *   "session.created" / "session.updated"
     *   "response.created" / "response.done"
     *   "error"     - provider errors
     */
    on(event: "speaker", listener: (stream: VoiceAudioStream) => void): this;
    on(event: "writing", listener: (ev: { text: string; role: "user" | "assistant"; response_id?: string }) => void): this;
    on(event: "speaking", listener: (ev: { audio: any; response_id?: string }) => void): this;
    on(event: "error", listener: (err: { message: string; code?: string; details?: any }) => void): this;
    on(event: string, listener: (...args: any[]) => void): this;
    off(event: string, listener: (...args: any[]) => void): this;
    emit(event: string, ...args: any[]): boolean;
  }

  // ── Additional voice providers ────────────────────────────────
  //
  // Every provider below is a thin class wrapping the matching
  // `@mastra/voice-<provider>` package. Each extends MastraVoice
  // with its own speak / listen + provider-specific constructor
  // options. Types here are intentionally looser than the
  // underlying packages — providers rename config fields
  // between releases, so we document the shape without
  // committing to version-specific field lists.

  /** Azure Cognitive Services — speak + listen. */
  export class AzureVoice implements MastraVoice {
    constructor(config?: {
      speechModel?: { name?: string; apiKey?: string; region?: string; style?: string; pitch?: string; rate?: string; [key: string]: any };
      listeningModel?: { name?: string; apiKey?: string; region?: string; language?: string; [key: string]: any };
      speaker?: string;
    });
    speak(input: string | VoiceAudioStream, options?: VoiceSpeakOptions): Promise<VoiceAudioStream>;
    listen(audio: VoiceAudioStream, options?: VoiceListenOptions): Promise<string>;
  }

  /** ElevenLabs — high-quality TTS + STT. */
  export class ElevenLabsVoice implements MastraVoice {
    constructor(config?: {
      speechModel?: { name?: string; apiKey?: string; [key: string]: any };
      listeningModel?: { name?: string; apiKey?: string; [key: string]: any };
      speaker?: string;
    });
    speak(input: string | VoiceAudioStream, options?: VoiceSpeakOptions & { voiceId?: string; emotion?: string }): Promise<VoiceAudioStream>;
    listen(audio: VoiceAudioStream, options?: VoiceListenOptions): Promise<string>;
  }

  /** Cloudflare Workers AI — edge TTS. */
  export class CloudflareVoice implements MastraVoice {
    constructor(config?: {
      speechModel?: { name?: string; accountId?: string; apiToken?: string; [key: string]: any };
      speaker?: string;
    });
    speak(input: string | VoiceAudioStream, options?: VoiceSpeakOptions): Promise<VoiceAudioStream>;
    listen(audio: VoiceAudioStream, options?: VoiceListenOptions): Promise<string>;
  }

  /** Deepgram — TTS + speech-to-text. */
  export class DeepgramVoice implements MastraVoice {
    constructor(config?: {
      speechModel?: { name?: string; apiKey?: string; tone?: string; [key: string]: any };
      listeningModel?: { name?: string; apiKey?: string; format?: string; language?: string; [key: string]: any };
      speaker?: string;
    });
    speak(input: string | VoiceAudioStream, options?: VoiceSpeakOptions): Promise<VoiceAudioStream>;
    listen(audio: VoiceAudioStream, options?: VoiceListenOptions): Promise<string>;
  }

  /** PlayAI — natural-sounding TTS. No STT. */
  export class PlayAIVoice implements MastraVoice {
    constructor(config?: {
      speechModel?: { name?: string; apiKey?: string; userId?: string; speed?: number; [key: string]: any };
      speaker?: string;
    });
    speak(input: string | VoiceAudioStream, options?: VoiceSpeakOptions & { speed?: number }): Promise<VoiceAudioStream>;
    listen(audio: VoiceAudioStream, options?: VoiceListenOptions): Promise<string>;
  }
  /** Map of PlayAI voice presets available without an API call. */
  export const PLAYAI_VOICES: Record<string, { id: string; name: string; [key: string]: any }>;

  /** Speechify — accessibility-focused TTS. No STT. */
  export class SpeechifyVoice implements MastraVoice {
    constructor(config?: {
      speechModel?: { name?: string; apiKey?: string; speed?: number; [key: string]: any };
      speaker?: string;
    });
    speak(input: string | VoiceAudioStream, options?: VoiceSpeakOptions & { speed?: number }): Promise<VoiceAudioStream>;
    listen(audio: VoiceAudioStream, options?: VoiceListenOptions): Promise<string>;
  }

  /** Sarvam — Indic-language specialized speak + listen. */
  export class SarvamVoice implements MastraVoice {
    constructor(config?: {
      speechModel?: { name?: string; apiKey?: string; language?: string; [key: string]: any };
      listeningModel?: { name?: string; apiKey?: string; language?: string; [key: string]: any };
      speaker?: string;
    });
    speak(input: string | VoiceAudioStream, options?: VoiceSpeakOptions): Promise<VoiceAudioStream>;
    listen(audio: VoiceAudioStream, options?: VoiceListenOptions): Promise<string>;
  }

  /** Murf — studio-quality TTS. No STT. */
  export class MurfVoice implements MastraVoice {
    constructor(config?: {
      speechModel?: { name?: string; apiKey?: string; emotion?: string; [key: string]: any };
      speaker?: string;
    });
    speak(input: string | VoiceAudioStream, options?: VoiceSpeakOptions & { emotion?: string }): Promise<VoiceAudioStream>;
    listen(audio: VoiceAudioStream, options?: VoiceListenOptions): Promise<string>;
  }

  // Note: @mastra/voice-google (classic) pulls @google-cloud/speech
  // which requires a full gRPC-over-HTTP2 polyfill brainkit does
  // not ship. Users needing Google TTS/STT can use Gemini Live
  // (not currently bundled — tracked separately) or reach OpenAI
  // / Deepgram / ElevenLabs for comparable capabilities.

  // ── Processors ─────────────────────────────────────────────────
  //
  // Prebuilt input/output processors from @mastra/core/processors.
  // Each implements the Processor contract; most are model-gated
  // (pass a `model: LanguageModel`). TripWire-style processors
  // (moderation, injection, PII) can abort the run with a
  // TripwireResult.

  /** Base Processor interface shared by input + output processors. */
  export interface Processor<TId extends string = string> {
    readonly id: TId;
    readonly name: string;
    readonly version?: string;
    processInput?(args: ProcessInputArgs): Promise<void> | void;
    processInputStep?(args: ProcessInputStepArgs): Promise<void> | void;
    processOutputStream?(args: ProcessOutputStreamArgs): AsyncIterable<any>;
    processOutputResult?(args: ProcessOutputResultArgs): Promise<void> | void;
    processOutputStep?(args: ProcessOutputStepArgs): Promise<void> | void;
  }
  export type InputProcessor = Processor;
  export type OutputProcessor = Processor;
  /** Either a processor or a processor-typed Workflow. */
  export type InputProcessorOrWorkflow = Processor | Workflow;
  export type OutputProcessorOrWorkflow = Processor | Workflow;

  export interface ProcessInputArgs { messages: Message[]; requestContext: RequestContext; abort(reason?: string, metadata?: Record<string, unknown>): never; [key: string]: any }
  export interface ProcessInputStepArgs extends ProcessInputArgs { step: any }
  export interface ProcessOutputStreamArgs { part: any; requestContext: RequestContext; abort(reason?: string, metadata?: Record<string, unknown>): never; [key: string]: any }
  export interface ProcessOutputResultArgs { result: AgentResult; requestContext: RequestContext; abort(reason?: string, metadata?: Record<string, unknown>): never; [key: string]: any }
  export interface ProcessOutputStepArgs extends ProcessOutputResultArgs { step: AgentStepResult }

  /** Tripwire shape raised when a processor aborts. */
  export interface TripwireResult<TMetadata = Record<string, unknown>> {
    reason: string;
    metadata?: TMetadata;
  }

  /** Normalize unicode in input; strip control chars. */
  export class UnicodeNormalizer implements Processor {
    constructor(opts?: { stripControlChars?: boolean; form?: "NFC" | "NFD" | "NFKC" | "NFKD" });
    readonly id: "unicode-normalizer";
    readonly name: string;
    processInput(args: ProcessInputArgs): Promise<void>;
  }

  /** LLM-based moderation gate. Tripwires on violations. */
  export class ModerationProcessor implements Processor {
    constructor(opts: {
      model: any;
      categories?: string[];
      strategy?: "block" | "warn" | "filter";
      threshold?: number;
      instructions?: string;
    });
    readonly id: "moderation";
    readonly name: string;
    processInput(args: ProcessInputArgs): Promise<void>;
  }

  /** Detect prompt-injection attempts. */
  export class PromptInjectionDetector implements Processor {
    constructor(opts: {
      model: any;
      strategy?: "block" | "rewrite" | "warn";
      threshold?: number;
      instructions?: string;
    });
    readonly id: "prompt-injection-detector";
    readonly name: string;
    processInput(args: ProcessInputArgs): Promise<void>;
  }

  /** Detect PII; redact / block / warn. */
  export class PIIDetector implements Processor {
    constructor(opts: {
      model: any;
      strategy?: "block" | "redact" | "warn";
      detectionTypes?: Array<"email" | "phone" | "name" | "address" | "ssn" | "creditcard" | string>;
      threshold?: number;
      instructions?: string;
    });
    readonly id: "pii-detector";
    readonly name: string;
    processInput(args: ProcessInputArgs): Promise<void>;
    processOutputResult(args: ProcessOutputResultArgs): Promise<void>;
  }

  /** Enforce language constraints on input. */
  export class LanguageDetector implements Processor {
    constructor(opts: {
      model: any;
      allowedLanguages?: string[];
      blockedLanguages?: string[];
      strategy?: "block" | "warn" | "translate";
      threshold?: number;
    });
    readonly id: "language-detector";
    readonly name: string;
    processInput(args: ProcessInputArgs): Promise<void>;
  }

  /** Coerce output into a schema during streaming. */
  export class StructuredOutputProcessor<OUTPUT extends {} = any> implements Processor {
    constructor(opts: {
      schema: import("ai").ZodType;
      model?: any;
      instructions?: string;
      errorStrategy?: "strict" | "warn" | "fallback";
      fallbackValue?: OUTPUT;
      jsonPromptInjection?: string;
      providerOptions?: Record<string, unknown>;
    });
    readonly id: "structured-output";
    readonly name: string;
    processOutputStream(args: ProcessOutputStreamArgs): Promise<any>;
  }

  /** Batch stream parts together to cut per-part overhead. */
  export class BatchPartsProcessor implements Processor {
    constructor(opts?: { batchSize?: number; flushIntervalMs?: number });
    readonly id: "batch-parts";
    readonly name: string;
    processOutputStream(args: ProcessOutputStreamArgs): AsyncIterable<any>;
  }

  /** Enforce an output token budget. */
  export class TokenLimiterProcessor implements Processor {
    constructor(opts: { maxTokens: number; strategy?: "truncate" | "error" });
    readonly id: "token-limiter";
    readonly name: string;
    processOutputStream(args: ProcessOutputStreamArgs): AsyncIterable<any>;
  }

  /** Scrub system-prompt leakage from output. */
  export class SystemPromptScrubber implements Processor {
    constructor(opts?: { model?: any; strategy?: "redact" | "rewrite" });
    readonly id: "system-prompt-scrubber";
    readonly name: string;
    processOutputResult(args: ProcessOutputResultArgs): Promise<void>;
  }

  /** Filter / rename tool calls at inference time. */
  export class ToolCallFilter implements Processor {
    constructor(opts: {
      allow?: string[];
      deny?: string[];
      rename?: Record<string, string>;
    });
    readonly id: "tool-call-filter";
    readonly name: string;
    processInput(args: ProcessInputArgs): Promise<void>;
  }

  /** Inject AGENTS.md / instruction-file reminders. */
  export class AgentsMDInjector implements Processor {
    constructor(opts: {
      reminderText?: string;
      maxTokens?: number;
      pathExists?: (path: string) => boolean;
      isDirectory?: (path: string) => boolean;
      readFile?: (path: string) => string;
      getIgnoredInstructionPaths?: (args: any) => string[];
    });
    readonly id: "agents-md-injector";
    readonly name: string;
    processInputStep(args: any): Promise<any>;
  }

  /** Semantic tool search — activate a subset of tools per turn. */
  export class ToolSearchProcessor implements Processor {
    constructor(opts: {
      tools: Record<string, any>;
      search?: { topK?: number; minScore?: number };
      ttl?: number;
    });
    readonly id: "tool-search";
    readonly name: string;
    processInputStep(args: any): Promise<{ tools: Record<string, any> }>;
  }

  /** Inject skills metadata into the system message. */
  export class SkillsProcessor implements Processor {
    constructor(opts: { workspace: any; format?: "xml" | "json" });
    readonly id: "skills-processor";
    readonly name: string;
    listSkills(): Promise<Array<{ name: string; description: string; license?: string }>>;
    processInputStep(args: any): Promise<void>;
  }

  /** On-demand skill discovery + loading. */
  export class SkillSearchProcessor implements Processor {
    constructor(opts: {
      workspace: any;
      search?: { topK?: number; minScore?: number };
      ttl?: number;
    });
    readonly id: "skill-search";
    readonly name: string;
    dispose(): void;
    processInputStep(args: any): Promise<{ tools: Record<string, unknown> | undefined }>;
  }

  /** Inject workspace filesystem / sandbox instructions. */
  export class WorkspaceInstructionsProcessor implements Processor {
    constructor(opts?: { position?: "system" | "prepend" | "append" });
    readonly id: "workspace-instructions";
    readonly name: string;
    processInputStep(args: any): Promise<void>;
  }

  // ── Storage + Vector abstract bases ────────────────────────────

  /**
   * Abstract base every Mastra storage backend extends. brainkit
   * users typically don't implement this — reach for one of the
   * concrete classes (`LibSQLStore`, `PostgresStore`, …). Declared
   * here so `StorageInstance` users can narrow against real methods
   * when they need to.
   */
  export abstract class MastraStorage implements StorageInstance {
    readonly __storageType: string;
    abstract getThreadById(opts: { threadId: string }): Promise<Thread | null>;
    abstract saveThread(opts: { thread: Thread }): Promise<Thread>;
    abstract saveMessages(opts: { threadId: string; messages: Message[] }): Promise<void>;
    abstract getMessagesByThreadId(opts: { threadId: string; limit?: number }): Promise<Message[]>;
    abstract deleteThread(threadId: string): Promise<void>;
    abstract listThreads(opts?: { resourceId?: string; page?: number; perPage?: number }): Promise<Thread[]>;
    abstract close?(): Promise<void>;
  }

  /**
   * Abstract base every Mastra vector backend extends.
   * `LibSQLVector`, `PgVector`, `MongoDBVector` implement this.
   */
  export abstract class MastraVector implements VectorStoreInstance {
    readonly __vectorType: string;
    abstract createIndex(opts: { indexName: string; dimension: number; metric?: "cosine" | "euclidean" | "dotproduct" }): Promise<void>;
    abstract listIndexes(): Promise<string[]>;
    abstract describeIndex(indexName: string): Promise<any>;
    abstract deleteIndex(indexName: string): Promise<void>;
    abstract upsert(opts: { indexName: string; vectors: VectorEntry[]; ids?: string[]; metadata?: Record<string, unknown> }): Promise<string[]>;
    abstract query(opts: { indexName: string; queryVector: number[]; topK?: number; filter?: VectorFilter; includeVector?: boolean }): Promise<QueryResult[]>;
    abstract updateIndexById?(indexName: string, id: string, update: { vector?: number[]; metadata?: Record<string, unknown> }): Promise<void>;
    abstract deleteIndexById?(indexName: string, id: string): Promise<void>;
  }

  // ── Observability ──────────────────────────────────────────────
  //
  // Mastra's observability stack. brainkit's own `modules/tracing`
  // + `modules/audit` sit alongside; these types declare the
  // Mastra-side surface so deployments can wire Mastra-native
  // exporters (Langfuse / Braintrust / etc.) when the host kit
  // doesn't own tracing.

  /** Core observability coordinator. */
  export class Observability {
    constructor(config?: ObservabilityConfig);
    readonly spans: any;
    readonly metrics: any;
    readonly exporters: any[];
    recordSpan?(span: Span): void;
    flush?(): Promise<void>;
    shutdown?(): Promise<void>;
  }

  export interface ObservabilityConfig {
    serviceName?: string;
    exporters?: any[];
    processors?: any[];
    sampler?: { type: "always-on" | "always-off" | "ratio"; ratio?: number };
    attributes?: Record<string, unknown>;
  }

  /** Span primitive — lifecycle is start → events → end. */
  export interface Span {
    readonly id: string;
    readonly name: string;
    readonly kind: SpanKind;
    readonly startTime: number;
    readonly endTime?: number;
    readonly attributes: Record<string, unknown>;
    readonly status: SpanStatus;
    readonly parentSpanId?: string;
    readonly traceId: string;
    setAttribute(key: string, value: unknown): void;
    setStatus(status: SpanStatus): void;
    addEvent(name: string, attributes?: Record<string, unknown>): void;
    end(error?: unknown): void;
  }

  export type SpanKind = "internal" | "server" | "client" | "producer" | "consumer";
  export interface SpanStatus {
    code: "unset" | "ok" | "error";
    message?: string;
  }

  export interface StartSpanOptions {
    kind?: SpanKind;
    attributes?: Record<string, unknown>;
    parentSpanId?: string;
  }

  /** Default exporter — prints spans to stdout / a Logger. */
  export class DefaultExporter {
    constructor(opts?: { logger?: any; format?: "json" | "pretty" });
    export(spans: Span[]): Promise<void>;
    shutdown(): Promise<void>;
  }

  /** Batch spans then export; reduces per-span overhead. */
  export class BatchSpanProcessor {
    constructor(exporter: DefaultExporter | any, opts?: { maxBatchSize?: number; scheduledDelayMs?: number });
    onStart(span: Span): void;
    onEnd(span: Span): void;
    shutdown(): Promise<void>;
  }

  /** Simple synchronous exporter — one span at a time. */
  export class SimpleSpanProcessor {
    constructor(exporter: DefaultExporter | any);
    onStart(span: Span): void;
    onEnd(span: Span): void;
    shutdown(): Promise<void>;
  }

  /** Per-call tracing overrides (kept loose — providers vary). */
  export interface TracingOptions {
    enabled?: boolean;
    serviceName?: string;
    attributes?: Record<string, unknown>;
    sampler?: ObservabilityConfig["sampler"];
  }

  /** Counter metric primitive. */
  export interface Counter {
    add(value: number, attributes?: Record<string, unknown>): void;
  }
  /** Gauge metric primitive. */
  export interface Gauge {
    set(value: number, attributes?: Record<string, unknown>): void;
  }
  /** Histogram metric primitive. */
  export interface Histogram {
    record(value: number, attributes?: Record<string, unknown>): void;
  }

  // ── Workspace (deeper) ─────────────────────────────────────────
  //
  // The Workspace class above wraps a filesystem + sandbox; the
  // types below cover the individual components so users can
  // compose their own (composite FS, mounted subtree, custom
  // sandbox).

  export abstract class MastraFilesystem {
    abstract readFile(path: string): Promise<Uint8Array | string>;
    abstract writeFile(path: string, data: Uint8Array | string): Promise<void>;
    abstract listFiles(path: string): Promise<string[]>;
    abstract stat(path: string): Promise<{ size: number; mtime: Date; isFile: boolean; isDirectory: boolean }>;
    abstract deleteFile(path: string): Promise<void>;
    abstract mkdir(path: string, opts?: { recursive?: boolean }): Promise<void>;
    abstract exists(path: string): Promise<boolean>;
  }

  /** Filesystem that routes paths to different backends by prefix. */
  export class CompositeFilesystem extends MastraFilesystem {
    constructor(mounts: Record<string, MastraFilesystem>);
    readFile(path: string): Promise<Uint8Array | string>;
    writeFile(path: string, data: Uint8Array | string): Promise<void>;
    listFiles(path: string): Promise<string[]>;
    stat(path: string): Promise<{ size: number; mtime: Date; isFile: boolean; isDirectory: boolean }>;
    deleteFile(path: string): Promise<void>;
    mkdir(path: string, opts?: { recursive?: boolean }): Promise<void>;
    exists(path: string): Promise<boolean>;
  }

  /** Abstract sandbox that exec'd commands run in. */
  export abstract class MastraSandbox {
    abstract spawn(command: string, args?: string[], options?: { cwd?: string; env?: Record<string, string> }): Promise<ProcessHandle>;
    abstract exec(command: string, options?: { cwd?: string; timeout?: number }): Promise<{ stdout: string; stderr: string; exitCode: number }>;
    abstract shutdown(): Promise<void>;
  }

  /** Handle for an in-flight sandboxed process. */
  export interface ProcessHandle {
    readonly pid: number;
    wait(): Promise<number>;
    kill(signal?: string | number): void;
    write(data: string | Uint8Array): Promise<void>;
    readLine(): Promise<string | null>;
    readonly stdout: AsyncIterable<Uint8Array>;
    readonly stderr: AsyncIterable<Uint8Array>;
  }

  /** Workspace tool factories — reach for these instead of hand-rolling. */
  export function createWorkspaceTools(config?: {
    filesystem?: MastraFilesystem;
    sandbox?: MastraSandbox;
    readOnly?: boolean;
  }): Record<string, Tool>;
  export function readFileTool(opts?: any): Tool;
  export function writeFileTool(opts?: any): Tool;
  export function editFileTool(opts?: any): Tool;
  export function listFilesTool(opts?: any): Tool;
  export function deleteFileTool(opts?: any): Tool;
  export function fileStatTool(opts?: any): Tool;
  export function mkdirTool(opts?: any): Tool;
  export function searchTool(opts?: any): Tool;
  export function indexContentTool(opts?: any): Tool;
  export function executeCommandTool(opts?: any): Tool;

  // ── AI-SDK voice bridge ────────────────────────────────────────

  /**
   * Adapt an AI-SDK voice-capable model into a MastraVoice.
   * Useful when a provider isn't shipped as a dedicated
   * `@mastra/voice-*` package but exposes speak/listen via the
   * AI SDK interface.
   */
  export function aisdkVoice(model: any, options?: { speaker?: string; [key: string]: any }): MastraVoice;

  /** No-op voice used when none is configured. */
  export class DefaultVoice implements MastraVoice {
    constructor();
    speak(input: string | VoiceAudioStream, options?: VoiceSpeakOptions): Promise<VoiceAudioStream>;
    listen(audio: VoiceAudioStream, options?: VoiceListenOptions): Promise<string>;
  }
}
